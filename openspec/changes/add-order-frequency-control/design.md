# Design: Order Frequency Control System

## Architecture Overview

The order frequency control system introduces a new layer between the trader and the OKX API, enforcing trading discipline through validation, tracking, and confirmation workflows.

```
┌─────────────────────────────────────────────────────────────┐
│                         Trader                              │
│                  (Manual or Future CLI)                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Order Control Layer (NEW)                      │
│  ┌─────────────────┐  ┌──────────────┐  ┌─────────────────┐│
│  │   Frequency     │  │  Maker-Only  │  │  Confirmation   ││
│  │   Validator     │  │  Validator   │  │   Manager       ││
│  └─────────────────┘  └──────────────┘  └─────────────────┘│
│           │                   │                  │          │
│           └───────────────────┴──────────────────┘          │
│                           │                                 │
└───────────────────────────┼─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Storage Layer                            │
│  ┌─────────────────┐  ┌──────────────────────────────────┐ │
│  │  order_history  │  │  pending_confirmations           │ │
│  │   (tracking)    │  │    (workflow state)              │ │
│  └─────────────────┘  └──────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     OKX Client                              │
│   Place Order | Get Ticker | Amend Order | Cancel Order    │
└─────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. Order Control Service

**Responsibility**: Orchestrate all order validation and placement logic

**Key Components**:
- `OrderControlService` (new): Main service coordinating validation and placement
- `FrequencyValidator` (new): Check order count against weekly limit
- `MakerValidator` (new): Validate maker-only rules and price distance
- `ConfirmationManager` (new): Manage pending order confirmation workflow

**Interfaces**:
```go
type OrderControlService interface {
    // PlaceOrder validates and places an order through all control checks
    PlaceOrder(ctx context.Context, req OrderRequest) (*OrderResult, error)

    // CheckPendingConfirmations runs periodic confirmation checks
    CheckPendingConfirmations(ctx context.Context) error
}

type OrderRequest struct {
    InstId     string  // Instrument ID
    Side       string  // buy or sell
    OrdType    string  // limit or market
    Px         string  // Price (for limit orders)
    Sz         string  // Size
    TdMode     string  // Trade mode (cross, isolated, cash)
    ReduceOnly bool    // Whether this is a position-reducing order
}

type OrderResult struct {
    OrdId      string  // Order ID from OKX
    Status     string  // Order status
    Warnings   []string // Any warnings during validation
}
```

### 2. Database Schema

**New Tables**:

```sql
-- Track order placement history for frequency limiting
CREATE TABLE order_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    inst_id TEXT NOT NULL,
    side TEXT NOT NULL,
    ord_type TEXT NOT NULL,
    size TEXT NOT NULL,
    price TEXT,
    reduce_only BOOLEAN NOT NULL DEFAULT 0,
    placed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    week_start DATE NOT NULL,  -- Start of the week (Monday UTC) for quick filtering
    status TEXT NOT NULL,  -- placed, filled, canceled, failed
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_order_history_week (week_start),
    INDEX idx_order_history_placed_at (placed_at),
    INDEX idx_order_history_order_id (order_id)
);

-- Track pending orders requiring confirmation
CREATE TABLE pending_confirmations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL UNIQUE,
    inst_id TEXT NOT NULL,
    side TEXT NOT NULL,
    ord_type TEXT NOT NULL,
    original_size TEXT NOT NULL,
    current_size TEXT NOT NULL,
    price TEXT,
    placed_at DATETIME NOT NULL,
    last_confirmation_at DATETIME,
    next_confirmation_due DATETIME NOT NULL,
    confirmation_count INTEGER NOT NULL DEFAULT 0,
    timeout_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,  -- pending, confirmed, timeout, canceled
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_pending_conf_status (status),
    INDEX idx_pending_conf_next_due (next_confirmation_due),
    INDEX idx_pending_conf_order_id (order_id)
);
```

### 3. Frequency Limiting Algorithm

**Strategy**: Rolling weekly window based on UTC timezone

**Algorithm**:
1. Determine current week start (Monday 00:00:00 UTC)
2. Query `order_history` for orders where `week_start = current_week_start AND status = 'placed'`
3. Count non-reduce-only orders (TP/SL don't count toward limit)
4. If count >= configured weekly limit, reject new order
5. Otherwise, allow order and record in `order_history`

**Edge Cases**:
- Week boundary: Orders placed just before midnight Monday UTC
- Database recovery: Backfill from OKX order history if db is lost
- Reduce-only orders: Always excluded from frequency count

### 4. Maker-Only Validation

**Strategy**: Validate order type and price distance before placement

**Algorithm**:
1. Check if order is market order:
   - If YES and ReduceOnly=false → REJECT
   - If YES and ReduceOnly=true → Check taker percentage limit
2. Check if order is limit order:
   - Query current market price from OKX ticker API (`GET /api/v5/market/ticker`)
   - Calculate price distance: `|order_price - market_price| / market_price`
   - If distance < configured minimum (default 1%) → REJECT
   - Otherwise → ALLOW
3. Log validation decision with reasoning

**Price Staleness Check**:
- If ticker API fails, use last known price if < 60 seconds old
- If no recent price available, REJECT order with error

**Taker Percentage Validation**:
- For reduce-only market orders, check current position size
- Calculate: `order_size / position_size`
- If ratio > configured max taker percentage (default 50%) → REJECT
- Otherwise → ALLOW

### 5. Confirmation Workflow

**Strategy**: Async scheduler checking pending confirmations at regular intervals

**Workflow**:
```
Order Placed → Record in pending_confirmations (status=pending)
                        │
                        ▼
                  [Wait 12 hours]
                        │
                        ▼
              Notification: "Confirm order?"
                        │
                ┌───────┴───────┐
                │               │
           User confirms   [Wait 4 hours]
                │               │
                │               ▼
                │         Timeout occurred
                │               │
                │               ▼
                │    Amend order size to 50%
                │    (via OKX amend API)
                │               │
                │               ▼
                │    Update current_size in DB
                │    timeout_count++
                │               │
                └───────────────┤
                                │
                                ▼
                     timeout_count >= max?
                        │           │
                       YES         NO
                        │           │
                        ▼           ▼
                  Cancel order   Continue loop
                  (via OKX)      (wait 12h again)
```

**Scheduler Design**:
- Separate goroutine (similar to TPSL scheduler)
- Check interval: every 5 minutes (configurable)
- Query `pending_confirmations` where `next_confirmation_due <= NOW() AND status = 'pending'`
- For each pending confirmation:
  - Check if timeout occurred (current_time - last_confirmation_at > waiting_period)
  - If timeout: amend order size
  - If no timeout but due: log notification (future: send actual notification)
  - Update `next_confirmation_due` for next cycle

**Confirmation Methods** (Phase 1: Logging only):
- Log to console and file with `WARN` level
- Include order details (instId, side, size, price)
- Future phases: Email, SMS, push notification, CLI command

### 6. Configuration Structure

```yaml
order_control:
  enabled: true  # Master switch for order control

  frequency_limit:
    enabled: true
    weekly_max_orders: 5  # Max orders per week (Monday-Sunday UTC)
    exclude_reduce_only: true  # Don't count TP/SL toward limit

  maker_only:
    enabled: true
    min_price_distance_pct: 0.01  # 1% minimum distance from market price
    allow_taker_for_reduce_only: true  # Allow market orders for closing positions
    max_taker_pct: 0.5  # Max 50% of position size for taker orders
    ticker_staleness_seconds: 60  # Max age of cached ticker price

  confirmation:
    enabled: true
    check_interval_seconds: 300  # Check every 5 minutes
    confirmation_interval_hours: 12  # Send confirmation request every 12 hours
    waiting_period_hours: 4  # How long to wait for confirmation before timeout
    timeout_size_reduction_pct: 0.5  # Reduce to 50% of current size on timeout
    max_timeouts: 3  # Cancel order after 3 timeouts
    notification_method: "log"  # Future: email, sms, webhook
```

## Error Handling

| Error Scenario | Handling Strategy |
|---------------|-------------------|
| Database write failure during order tracking | Retry 3 times with exponential backoff; if all fail, reject order (fail-safe) |
| Ticker API failure | Use cached price if < 60s old; otherwise reject order with clear error message |
| Order placement failure after passing validation | Log error, do NOT record in order_history, return error to caller |
| Amend order failure during timeout | Log error with ERROR level, mark confirmation as 'failed', retry next cycle |
| Cancel order failure after max timeouts | Log critical error, mark as 'failed', manual intervention required |

## Testing Strategy

### Unit Tests
- `FrequencyValidator`: Test week boundary calculations, order counting, reduce-only exclusion
- `MakerValidator`: Test price distance calculations, taker percentage validation, staleness checks
- `ConfirmationManager`: Test timeout calculations, size reduction logic, notification trigger

### Integration Tests
- End-to-end order placement with all validations
- Confirmation workflow from placement to timeout to cancellation
- Database transaction rollback on validation failure
- Frequency limit across multiple orders in same week

### Manual Testing (with 5 USDT limit)
- Place 5 orders in one week, verify 6th is rejected
- Place limit order 0.5% away from market, verify rejected
- Place limit order 1.5% away from market, verify accepted
- Wait for confirmation timeout, verify size reduction

## Deployment Considerations

1. **Database Migration**: Add new tables to existing SQLite database
2. **Configuration Update**: Add `order_control` section to config.yaml and config.template.yaml
3. **Graceful Shutdown**: Confirmation scheduler must complete current cycle before exit
4. **Backward Compatibility**: System works without order control (enabled=false fallback)
5. **Logging Volume**: Confirmation checks may generate significant logs; ensure log rotation is configured

## Performance Considerations

- **Frequency Query**: Index on `week_start` ensures fast weekly order count
- **Ticker Caching**: Cache market price for 5 seconds to reduce API calls
- **Confirmation Checks**: Limit to max 100 pending confirmations per cycle to avoid long-running queries
- **Database Connections**: Reuse existing connection pool, no additional connections needed

## Security Considerations

- **Order History Privacy**: Contains trading activity, must not be committed to git
- **Confirmation Notifications**: Future email/SMS must not expose sensitive data in plaintext
- **Database Permissions**: order_history and pending_confirmations tables require same protection as positions table

## Future Enhancements (Not in this proposal)

- Web UI for confirming pending orders
- Email/SMS/Push notification integration
- Manual override API for frequency limit
- Analytics dashboard for order frequency trends
- Integration with order entry notes (Feature 5)
