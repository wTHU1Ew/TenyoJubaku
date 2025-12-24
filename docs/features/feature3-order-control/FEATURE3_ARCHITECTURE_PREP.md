# Feature 3 Architecture Preparation

**Date**: 2025-12-03
**Status**: Planning
**Goal**: Prepare architecture for Order Frequency Control implementation

## Overview

Feature 3 (Order Control) introduces a new service layer that requires significant architectural extensions. This document outlines the necessary changes to maintain low coupling and proper layering.

## Current Architecture Gaps

### Gap 1: OKX Client Interface - Missing Trading Operations (CRITICAL)

**Current State**: `okx.Interface` only supports read operations
**Required**: Add trading operations for Order Control

**Missing Methods**:
```go
type Interface interface {
    // Existing methods...
    GetAccountBalance() (*AccountBalanceResponse, error)
    GetPositions() (*PositionsResponse, error)
    GetTicker(instId string) (*TickerResponse, error)

    // NEW: Required for Feature 3
    PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)
    AmendOrder(ctx context.Context, orderId string, req *AmendOrderRequest) (*AmendOrderResponse, error)
    CancelOrder(ctx context.Context, orderId string) (*CancelOrderResponse, error)
    GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error)
}
```

**Impact**: Order Control Service cannot place/manage orders without these methods

---

### Gap 2: Storage Interface - Missing Order History (CRITICAL)

**Current State**: `storage.Interface` only supports account/position data
**Required**: Add order tracking for frequency limiting and confirmation workflow

**Missing Methods**:
```go
type Interface interface {
    // Existing methods...
    InsertAccountBalance(balance *models.AccountBalance) error
    InsertPosition(position *models.Position) error

    // NEW: Required for Feature 3
    InsertOrderHistory(order *models.OrderHistory) error
    GetOrderCountForWeek(weekStart time.Time) (int, error)
    GetOrdersForWeek(weekStart time.Time) ([]models.OrderHistory, error)

    InsertPendingConfirmation(conf *models.PendingConfirmation) error
    GetPendingConfirmationsDue(now time.Time) ([]models.PendingConfirmation, error)
    UpdatePendingConfirmation(orderId string, updates *ConfirmationUpdates) error
    DeletePendingConfirmation(orderId string) error
}
```

**New Database Tables**:
```sql
-- Track order history for frequency limiting
CREATE TABLE order_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL,
    inst_id TEXT NOT NULL,
    side TEXT NOT NULL,
    ord_type TEXT NOT NULL,
    size TEXT NOT NULL,
    price TEXT,
    reduce_only BOOLEAN NOT NULL DEFAULT 0,
    placed_at DATETIME NOT NULL,
    week_start DATE NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Track pending confirmations
CREATE TABLE pending_confirmations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id TEXT NOT NULL UNIQUE,
    inst_id TEXT NOT NULL,
    original_size TEXT NOT NULL,
    current_size TEXT NOT NULL,
    placed_at DATETIME NOT NULL,
    last_confirmation_at DATETIME,
    next_confirmation_due DATETIME NOT NULL,
    timeout_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

**Impact**: Cannot track order frequency or manage confirmation workflow

---

### Gap 3: Missing Context Propagation (HIGH)

**Current State**: All API methods lack `context.Context` parameter
**Required**: Add context for cancellation, timeouts, and tracing

**Changes Needed**:
```go
// Before
GetPositions() (*PositionsResponse, error)
PlaceAlgoOrder(req AlgoOrderRequest) (*AlgoOrderResponse, error)

// After
GetPositions(ctx context.Context) (*PositionsResponse, error)
PlaceAlgoOrder(ctx context.Context, req AlgoOrderRequest) (*AlgoOrderResponse, error)
```

**Impact**:
- Cannot cancel long-running requests
- Cannot set per-request timeouts
- Difficult graceful shutdown
- No request tracing

---

### Gap 4: Missing Notifier Abstraction (MEDIUM)

**Required**: Abstract notification mechanism for confirmation workflow

**Interface Design**:
```go
package notifier

type Interface interface {
    // SendConfirmationRequest sends a confirmation request to the user
    SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error
}

type ConfirmationInfo struct {
    OrderID       string
    Instrument    string
    Side          string
    Size          string
    Price         string
    PlacedAt      time.Time
    TimeoutIn     time.Duration
}

// Phase 1 Implementation
type LogNotifier struct {
    logger *logger.Logger
}

func (n *LogNotifier) SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error {
    n.logger.Warn("CONFIRMATION REQUIRED: Order %s for %s %s @ %s (timeout in %v)",
        info.OrderID, info.Side, info.Instrument, info.Price, info.TimeoutIn)
    return nil
}
```

**Future Implementations**:
- `EmailNotifier` - Send email
- `SMSNotifier` - Send SMS
- `WebhookNotifier` - Call webhook
- `CLINotifier` - Display in CLI prompt

---

### Gap 5: Missing Order Control Service Layer (CRITICAL)

**Required**: New service layer for order validation and placement

**Module Structure**:
```
internal/
├── ordercontrol/
│   ├── interface.go           # Service interface
│   ├── service.go             # Main service implementation
│   ├── frequency.go           # Frequency validator
│   ├── maker.go               # Maker-only validator
│   ├── confirmation.go        # Confirmation manager
│   ├── mock.go                # Mock for testing
│   └── ordercontrol_test.go  # Unit tests
```

**Service Interface**:
```go
package ordercontrol

type Interface interface {
    // ValidateAndPlaceOrder validates all rules and places order if valid
    ValidateAndPlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResult, error)

    // CheckPendingConfirmations runs periodic confirmation workflow
    CheckPendingConfirmations(ctx context.Context) error

    // Start begins the confirmation scheduler
    Start() error

    // Stop gracefully stops the scheduler
    Stop()
}

type OrderRequest struct {
    InstId     string
    Side       string  // buy, sell
    OrdType    string  // limit, market
    Px         string  // Price (for limit orders)
    Sz         string  // Size
    TdMode     string  // Trade mode
    ReduceOnly bool
}

type OrderResult struct {
    OrderID   string
    Status    string
    Warnings  []string
}
```

---

## Recommended Implementation Plan

### Phase 1: Interface Extensions (CRITICAL - Do First)

**Estimated Time**: 3-4 hours

1. **Extend OKX Client Interface**
   - Add `PlaceOrder()`, `AmendOrder()`, `CancelOrder()`, `GetPendingOrders()`
   - Add context.Context to ALL methods
   - Implement in `internal/okx/client.go`
   - Update mock in `internal/okx/mock.go`
   - Update existing usages

2. **Extend Storage Interface**
   - Add order history methods
   - Add pending confirmation methods
   - Create migration for new tables
   - Implement in `internal/storage/storage.go`
   - Update mock in `internal/storage/mock.go`

3. **Add Context Throughout**
   - Update all okx.Interface method signatures
   - Update all storage.Interface method signatures
   - Update Monitor service to use context
   - Update TPSL services to use context
   - Update main.go to create root context

**Deliverables**:
- [ ] `internal/okx/interface.go` updated
- [ ] `internal/okx/client.go` with new methods
- [ ] `internal/okx/types.go` with new request/response types
- [ ] `internal/okx/mock.go` updated
- [ ] `internal/storage/interface.go` updated
- [ ] `internal/storage/storage.go` with new tables/methods
- [ ] `internal/storage/mock.go` updated
- [ ] `pkg/models/order.go` with new models
- [ ] All existing services updated for context
- [ ] All tests passing

---

### Phase 2: Foundation Components (CRITICAL - Do Second)

**Estimated Time**: 2-3 hours

1. **Create Notifier Abstraction**
   ```
   internal/notifier/
   ├── interface.go
   ├── log_notifier.go
   └── mock.go
   ```

2. **Add Configuration**
   ```yaml
   order_control:
     enabled: true
     frequency_limit:
       weekly_max_orders: 5
     maker_only:
       min_price_distance_pct: 0.01
     confirmation:
       check_interval_seconds: 300
   ```

3. **Create Domain Models**
   ```
   pkg/models/
   ├── order.go              # OrderHistory, PendingConfirmation
   └── order_test.go
   ```

**Deliverables**:
- [ ] Notifier interface and log implementation
- [ ] Configuration structure extended
- [ ] Order domain models
- [ ] Unit tests for models

---

### Phase 3: Order Control Service (Can Start Feature 3)

**Estimated Time**: 8-10 hours

1. **Implement Core Service**
2. **Implement Validators**
3. **Implement Confirmation Manager**
4. **Integration with main.go**

---

## Architecture Principles to Maintain

### 1. Dependency Inversion ✅
```go
// Good - depend on abstractions
type OrderControlService struct {
    okxClient okx.Interface        // ✅ Interface
    storage   storage.Interface    // ✅ Interface
    notifier  notifier.Interface   // ✅ Interface
}

// Bad - depend on implementations
type OrderControlService struct {
    okxClient *okx.Client          // ❌ Concrete
    storage   *storage.Storage     // ❌ Concrete
}
```

### 2. Interface Segregation ✅
```go
// Good - focused interfaces
type FrequencyChecker interface {
    CheckWeeklyLimit(ctx context.Context) (bool, error)
}

type MakerValidator interface {
    ValidatePrice(ctx context.Context, instId string, price float64) error
}

// Bad - fat interface
type Validator interface {
    CheckWeeklyLimit(ctx context.Context) (bool, error)
    ValidatePrice(ctx context.Context, instId string, price float64) error
    SendNotification(info ConfirmationInfo) error
    // ... too many responsibilities
}
```

### 3. Single Responsibility ✅
Each component has ONE reason to change:
- `FrequencyValidator` - frequency limiting rules change
- `MakerValidator` - maker-only rules change
- `ConfirmationManager` - confirmation workflow changes
- `OrderControlService` - orchestration logic changes

### 4. Layering ✅
```
Application Layer (main.go)
    ↓
Service Layer (ordercontrol, monitor, tpsl)
    ↓
Infrastructure Layer (okx, storage, notifier)
    ↓
Domain Layer (models)
```

No layer should skip levels or depend on layers below it in ways that violate abstraction.

---

## Testing Strategy

### Unit Tests (Fast, Isolated)
```go
func TestFrequencyValidator_WeeklyLimit(t *testing.T) {
    mockStorage := &storage.MockStorage{
        GetOrderCountForWeekFunc: func(weekStart time.Time) (int, error) {
            return 4, nil  // 4 orders already placed
        },
    }

    validator := NewFrequencyValidator(mockStorage, 5)
    allowed, err := validator.Check(context.Background())

    if !allowed {
        t.Error("Expected order to be allowed (4 < 5)")
    }
}
```

### Integration Tests (Slower, Comprehensive)
```go
func TestOrderControl_FullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Use real database and mock OKX
    db, _ := storage.New("/tmp/test.db", true, 5, 2)
    defer db.Close()

    mockOKX := &okx.MockClient{}

    service := ordercontrol.New(mockOKX, db, notifier, config)

    // Test complete flow
    result, err := service.ValidateAndPlaceOrder(ctx, orderReq)
    // ... assertions
}
```

---

## Migration Path

### Step 1: Extend Interfaces (Non-Breaking)
Add new methods to interfaces without removing old ones

### Step 2: Update Implementations (Non-Breaking)
Implement new methods in concrete types

### Step 3: Add Context (Breaking, but Controlled)
- Create feature branch
- Update all method signatures
- Update all call sites
- Run full test suite
- Merge when green

### Step 4: Build New Service (Additive)
Build Order Control service using extended interfaces

### Step 5: Integrate (Additive)
Wire up in main.go with feature flag

---

## Success Criteria

Before starting Feature 3 implementation, verify:

- [ ] ✅ All interfaces support context.Context
- [ ] ✅ OKX client supports order placement/amendment/cancellation
- [ ] ✅ Storage supports order history tracking
- [ ] ✅ Storage supports pending confirmation state
- [ ] ✅ Notifier abstraction exists with log implementation
- [ ] ✅ Configuration structure ready
- [ ] ✅ Domain models defined
- [ ] ✅ All existing tests pass
- [ ] ✅ New mock implementations available
- [ ] ✅ Architecture documentation updated

---

## Conclusion

Current architecture is **NOT ready** for Feature 3 due to:
1. Missing OKX trading operations
2. Missing order history storage
3. Lack of context propagation
4. No notifier abstraction

**Recommendation**: Complete Phase 1 & 2 (5-7 hours) before starting Feature 3 implementation.

**Benefit**: Proper preparation ensures:
- Low coupling between Order Control and infrastructure
- Easy testing with mocks
- Future extensibility (email notifications, webhooks, etc.)
- Clean architecture maintained
