# Design: Automatic Stop-Loss and Take-Profit Management

## Architecture Overview

The TPSL management system follows a layered architecture consistent with the existing codebase:

```
┌─────────────────────────────────────────────────┐
│            Main Application (cmd/main.go)        │
│  - Initializes TPSL Scheduler                   │
│  - Manages graceful shutdown                     │
└─────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│      TPSL Scheduler (internal/tpsl/scheduler.go) │
│  - Runs periodic TPSL checks                    │
│  - Coordinates between components                │
└─────────────────────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          ▼                       ▼
┌──────────────────────┐  ┌──────────────────────┐
│  TPSL Manager        │  │  OKX Client          │
│  (internal/tpsl/     │  │  (internal/okx/      │
│   manager.go)        │  │   client.go)         │
│                      │  │                      │
│ - Analyze positions  │  │ - Get algo orders    │
│ - Calculate TPSL     │  │ - Place algo orders  │
│ - Determine coverage │  │ - Authenticate       │
└──────────────────────┘  └──────────────────────┘
          │                       │
          └───────────┬───────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│         Storage Layer (internal/storage/)       │
│  - Fetch positions from database                │
└─────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────┐
│          SQLite Database (data/*.db)            │
└─────────────────────────────────────────────────┘
```

## Component Design

### 1. TPSL Scheduler (`internal/tpsl/scheduler.go`)

**Responsibility**: Orchestrate periodic TPSL management operations

**Key Methods**:
- `New(config, storage, okxClient, logger)` - Create scheduler instance
- `Start()` - Start periodic TPSL checks
- `Stop()` - Graceful shutdown with context cancellation
- `runCheck()` - Execute one TPSL check cycle

**Behavior**:
- Runs in goroutine with configurable interval (default 5 minutes)
- Uses ticker for periodic execution
- Respects context cancellation for graceful shutdown
- Logs each check cycle start/completion
- Handles errors without crashing

### 2. TPSL Manager (`internal/tpsl/manager.go`)

**Responsibility**: Core TPSL business logic

**Key Methods**:
- `New(config, okxClient, logger)` - Create manager instance
- `AnalyzeAndPlaceTPSL(positions)` - Main entry point
- `getExistingAlgoOrders(positions)` - Fetch algo orders from OKX
- `calculateCoverage(position, algoOrders)` - Determine coverage gap
- `calculateTPSLPrices(position)` - Calculate TP/SL trigger prices
- `placeTPSLOrder(position, size, tpPrice, slPrice)` - Place algo order

**TPSL Calculation Logic**:

For **LONG positions**:
```
SL_distance = entry_price × volatility_pct × leverage
SL_price = entry_price - SL_distance

TP_distance = SL_distance × pl_ratio
TP_price = entry_price + TP_distance
```

For **SHORT positions**:
```
SL_distance = entry_price × volatility_pct × leverage
SL_price = entry_price + SL_distance

TP_distance = SL_distance × pl_ratio
TP_price = entry_price - TP_distance
```

**Coverage Analysis**:
1. Group algo orders by (instrument, position_side)
2. Sum TPSL order sizes for each position
3. Calculate uncovered size: `position_size - covered_size`
4. If uncovered size > 0, place additional TPSL order

### 3. OKX Client Extensions (`internal/okx/client.go`)

**New Methods**:
- `GetPendingAlgoOrders(instType, instId, ordType)` - Fetch pending algo orders
- `PlaceAlgoOrder(request)` - Place conditional TPSL order

**Types** (`internal/okx/types.go`):
```go
type AlgoOrderRequest struct {
    InstId        string  `json:"instId"`
    TdMode        string  `json:"tdMode"`
    Side          string  `json:"side"`
    PosSide       string  `json:"posSide"`
    OrdType       string  `json:"ordType"`
    Sz            string  `json:"sz"`
    TpTriggerPx   string  `json:"tpTriggerPx"`
    TpOrdPx       string  `json:"tpOrdPx"`
    SlTriggerPx   string  `json:"slTriggerPx"`
    SlOrdPx       string  `json:"slOrdPx"`
    ReduceOnly    bool    `json:"reduceOnly"`
}

type PendingAlgoOrdersResponse struct {
    Code string `json:"code"`
    Msg  string `json:"msg"`
    Data []struct {
        AlgoId       string `json:"algoId"`
        InstId       string `json:"instId"`
        PosSide      string `json:"posSide"`
        Side         string `json:"side"`
        Sz           string `json:"sz"`
        OrdType      string `json:"ordType"`
        State        string `json:"state"`
        TpTriggerPx  string `json:"tpTriggerPx"`
        SlTriggerPx  string `json:"slTriggerPx"`
        // ... other fields
    } `json:"data"`
}
```

### 4. Configuration (`internal/config/config.go`)

**New Config Section**:
```go
type TPSLConfig struct {
    Enabled         bool    `yaml:"enabled"`
    CheckInterval   int     `yaml:"check_interval"`    // seconds
    VolatilityPct   float64 `yaml:"volatility_pct"`    // e.g., 0.01 for 1%
    ProfitLossRatio float64 `yaml:"profit_loss_ratio"` // e.g., 5.0 for 5:1
}
```

**Config Template Addition** (`configs/config.template.yaml`):
```yaml
# TPSL Management Configuration
tpsl:
  # Enable automatic TPSL management
  enabled: true

  # TPSL check interval in seconds
  check_interval: 300

  # Volatility percentage for stop-loss calculation (0.01 = 1%)
  volatility_pct: 0.01

  # Profit-loss ratio for take-profit calculation (5.0 = 5:1)
  profit_loss_ratio: 5.0
```

## Data Flow

### TPSL Check Cycle

```
1. Timer triggers TPSL check
   ↓
2. Scheduler calls Manager.AnalyzeAndPlaceTPSL()
   ↓
3. Manager fetches current positions from Storage
   ↓
4. Manager fetches pending algo orders from OKX API
   ↓
5. For each position:
   a. Find matching algo orders (same instId + posSide)
   b. Calculate total covered size
   c. Calculate uncovered size
   d. If uncovered > 0:
      - Calculate TP/SL prices based on config
      - Place algo order for uncovered amount
      - Log placement details
   ↓
6. Log summary (positions checked, orders placed)
   ↓
7. Wait for next interval
```

### Position Coverage Matching

Algo orders match a position when:
- `instId` matches (e.g., "BTC-USDT-SWAP")
- `posSide` matches ("long", "short", or "net")
- `ordType` is "conditional" (TPSL order)
- `state` is "live" (pending/active)
- Order is `reduceOnly` = true

## Error Handling Strategy

### Recoverable Errors (Log and Continue)
- API rate limits → Log warning, wait for next interval
- Temporary network errors → Retry with backoff
- Specific position placement fails → Log error, continue to next position

### Critical Errors (Log and Stop)
- Invalid configuration → Exit on startup
- Storage connection lost → Graceful shutdown
- Authentication failure → Stop TPSL operations

### Error Logging Format
```
ERROR: Failed to place TPSL for position BTC-USDT-SWAP (long)
  Position Size: 0.5
  Uncovered Size: 0.3
  TP Price: 45000
  SL Price: 42500
  Error: rate limit exceeded
  Action: Will retry next cycle
```

## Concurrency Considerations

1. **No Shared State**: TPSL scheduler operates independently from monitoring loop
2. **Read-Only Database Access**: Only reads positions, no writes to avoid conflicts
3. **Goroutine Safety**: Single goroutine per scheduler instance
4. **Context-Based Cancellation**: Uses context for clean shutdown

## Testing Strategy

### Unit Tests
- TPSL price calculation (long/short, various leverage)
- Coverage calculation logic
- Configuration validation
- Error handling paths

### Integration Tests
- Mock OKX API responses for algo orders
- Test full TPSL cycle with mock data
- Verify correct API requests are generated

### Manual Testing (with small positions)
- Place test position without TPSL
- Verify auto-TPSL placement within check interval
- Verify TPSL prices are calculated correctly
- Test partial coverage scenario

## Security Considerations

1. **API Permissions**: Requires OKX API key with "Trade" permission
2. **Configuration Security**: TPSL config in config.yaml (gitignored)
3. **Logging**: Mask sensitive data (API keys, secrets)
4. **Order Validation**: Validate calculated prices are reasonable before placement

## Performance Considerations

1. **API Call Efficiency**:
   - One API call to get all pending algo orders (filtered by ordType)
   - One API call per uncovered position to place TPSL
   - Typical load: 2-5 API calls per 5-minute cycle

2. **Database Queries**:
   - Reuse positions already fetched by monitoring loop
   - Read-only access, no complex queries

3. **Memory Footprint**:
   - Minimal: only current positions and algo orders in memory
   - No historical data storage for TPSL module

## Migration Path

Since this is a new feature:
1. No database schema changes needed (reads existing positions table)
2. Add TPSL config section to config.template.yaml
3. Users update their config.yaml with TPSL settings
4. Feature is opt-in via `tpsl.enabled` flag
