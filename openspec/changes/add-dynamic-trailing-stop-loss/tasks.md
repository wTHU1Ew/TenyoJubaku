# Implementation Tasks: Dynamic Trailing Stop-Loss

## Phase 1: Database Schema, Configuration, and API Foundation

### 1.1 Create Dynamic SL Tracking Database Schema
- [ ] Add `DynamicSLTracker` model struct in `pkg/models/dynamic_sl_tracker.go`
- [ ] Add GORM struct tags for all fields (id, position_key, inst_id, pos_side, etc.)
- [ ] Add unique index on `position_key` column
- [ ] Add `TableName()` method returning "dynamic_sl_tracking"
- [ ] Add `Validate()` method to check field constraints
- [ ] Update `internal/storage/storage.go` to include tracker in AutoMigrate
- [ ] Write unit tests for model validation
- **Validation**: Run AutoMigrate on fresh database, verify table created with correct schema

### 1.2 Add Storage Interface Methods for Dynamic SL
- [ ] Add `InsertDynamicSLTracker(ctx, tracker)` method to storage interface
- [ ] Add `GetDynamicSLTracker(ctx, positionKey)` method
- [ ] Add `UpdateDynamicSLTracker(ctx, tracker)` method
- [ ] Add `DeleteDynamicSLTracker(ctx, positionKey)` method
- [ ] Add `GetAllDynamicSLTrackers(ctx)` method (load on startup)
- [ ] Add `CleanupOrphanedTrackers(ctx, openPositionKeys)` method
- [ ] Implement all methods with GORM operations
- [ ] Write unit tests for all CRUD operations
- **Validation**: Test full CRUD cycle with SQLite in-memory database

### 1.3 Add Dynamic SL Configuration Structure
- [ ] Add `DynamicSLConfig` struct to `internal/config/config.go`
- [ ] Add fields: `Enabled`, `FirstMovePct`, `TrailingStepPct`, `StopMoveStepPct`
- [ ] Add validation for config values (all percentages > 0, < 1.0)
- [ ] Update `config.template.yaml` with `dynamic_sl` section and detailed comments
- [ ] Write unit tests for config parsing and validation
- **Validation**: Load config with valid/invalid values, verify validation works

### 1.4 Verify or Add OKX Amend Algo Order API
- [ ] Check if `AmendAlgoOrder` method exists in `internal/okx/client.go`
- [ ] If not exists: Add `AmendAlgoOrder(ctx, algoId, instId, newSlTriggerPx)` method
- [ ] Add `AmendAlgoOrderRequest` and `AmendAlgoOrderResponse` structs to types
- [ ] Implement with proper authentication and error handling
- [ ] Write unit tests with mocked API responses
- **Validation**: Call amend API on testnet with real algo order, verify SL price changed

## Phase 2: Dynamic SL Calculation Logic

### 2.1 Add Position Profit Tracking with Database Persistence
- [ ] Create helper functions in `internal/tpsl/dynamic_sl.go` for tracker operations
- [ ] Implement `LoadOrCreateTracker(ctx, storage, position)` function
  - Query tracker from DB by position_key
  - If not exists: Create new tracker with initial values, INSERT into DB
  - If exists: Load from DB
- [ ] Implement `UpdateTracker(ctx, storage, tracker, currentPrice)` function
  - Update highest/lowest price if improved
  - Check firstMove threshold
  - UPDATE database with new state
- [ ] Implement `ShouldAdjustSL(tracker, config)` function to check if adjustment needed
- [ ] Write unit tests for all functions with mock storage
- **Validation**: Simulate price movements, verify tracker updates in database

### 2.2 Implement Dynamic SL Calculation Algorithm
- [ ] Create `CalculateDynamicSL(entry, current, highest, config)` function
- [ ] **Step 1**: Check if `(current - entry) / entry >= firstMovePct`
  - If YES and not triggered before: return `entry * (1 + 0.001)` (breakeven + fees)
  - Mark `firstMoveTriggered = true`
- [ ] **Step 2**: After firstMove triggered, check price gain from highest
  - If `(current - highest) / highest >= trailingStepPct`:
    - Update `highest = current`
    - Calculate new SL: `currentSL * (1 + stopMoveStepPct)`
- [ ] Handle long and short positions (reverse logic for shorts)
- [ ] Write comprehensive unit tests for all scenarios
- **Validation**: Test edge cases (exact threshold, just below/above threshold, large jumps)

## Phase 3: TPSL Manager Integration

### 3.1 Integrate Dynamic SL into TPSL Check Cycle
- [ ] Add `dynamicSLEnabled` flag check in manager
- [ ] Add storage reference to Manager struct (for tracker CRUD operations)
- [ ] In `checkTPSLCoverage()` cycle, after coverage check:
  - Query current positions with unrealized PNL
  - For each position: Load or create tracker from database
  - Update tracker with current mark price (updates DB)
  - Check if SL adjustment needed via `ShouldAdjustSL()`
- [ ] Add cleanup logic: Delete trackers for closed positions from DB
- [ ] Write integration tests with mock storage and OKX client
- **Validation**: Run full TPSL cycle with dynamic SL enabled, verify trackers persisted in database

### 3.2 Implement SL Amendment Logic with Database Updates
- [ ] Create `amendStopLoss(ctx, storage, position, algoOrder, newSlPrice)` method
- [ ] Query pending algo orders to find matching SL order by instId and posSide
- [ ] Call `AmendAlgoOrder` API with new `slTriggerPx`
- [ ] Handle amendment success:
  - Update tracker's `currentSlPrice` field
  - UPDATE tracker in database
  - Log adjustment details
- [ ] Handle amendment failure: log error, retry next cycle (DB unchanged)
- [ ] Add comprehensive logging for all amendment operations
- **Validation**: Manually trigger SL adjustment, verify OKX order amended and DB updated

### 3.3 Handle Edge Cases and Error Recovery
- [ ] Handle case: Position closed before SL adjustment
- [ ] Handle case: Algo order already filled/cancelled
- [ ] Handle case: OKX API rate limit during amendment
- [ ] Handle case: Amendment fails due to invalid price (too close to market)
- [ ] Implement retry logic with exponential backoff (max 3 retries)
- [ ] Add circuit breaker: stop amendments if consecutive failures > 10
- **Validation**: Simulate all error scenarios, verify graceful handling

## Phase 4: Logging and Monitoring

### 4.1 Add Detailed Logging for Dynamic SL
- [ ] Log when firstMove threshold is reached (INFO level)
- [ ] Log every SL adjustment with old/new prices (INFO level)
- [ ] Log calculation details (current price, highest, threshold) at DEBUG level
- [ ] Log amendment API calls and responses (DEBUG level)
- [ ] Log amendment failures with full error details (ERROR level)
- **Validation**: Review logs during test run, ensure clarity and completeness

### 4.2 Add Dynamic SL Summary to Check Cycle
- [ ] Extend TPSL check cycle summary to include:
  - Positions with dynamic SL enabled
  - Number of SL adjustments made this cycle
  - Number of firstMove triggers this cycle
  - Number of amendment failures
- [ ] Log summary at INFO level after each cycle
- **Validation**: Check cycle summary includes dynamic SL metrics

## Phase 5: Testing and Validation

### 5.1 Unit Tests
- [ ] Test `DynamicSLTracker` state machine (all transitions)
- [ ] Test `CalculateDynamicSL` algorithm (long/short, all thresholds)
- [ ] Test config parsing and validation
- [ ] Test amendment API request formatting
- [ ] Achieve >90% code coverage for new code
- **Validation**: All unit tests pass, coverage report shows >90%

### 5.2 Integration Tests
- [ ] Test full cycle: position open → firstMove → SL to breakeven
- [ ] Test trailing: price increases → multiple SL adjustments
- [ ] Test price reversal after firstMove: SL stays at breakeven
- [ ] Test concurrent positions with different profit levels
- [ ] Test dynamic SL disabled: no amendments occur
- **Validation**: Integration tests pass with mock OKX API

### 5.3 Manual Testing with Real API
- [ ] Deploy to test environment with dynamic SL enabled
- [ ] Open small position (max 5 USDT as per project conventions)
- [ ] Wait for 1% profit, verify SL moves to breakeven
- [ ] If market allows, wait for 0.5% more gain, verify SL trails
- [ ] Check OKX algo order details, confirm SL price updated
- [ ] Close position manually, verify no errors in logs
- **Validation**: Real position shows dynamic SL working as expected

## Phase 6: Documentation

### 6.1 Update Configuration Documentation
- [ ] Add detailed comments to `config.template.yaml` for `dynamic_sl` section
- [ ] Explain each parameter and its effect on trading behavior
- [ ] Provide example values and recommended settings
- [ ] Add warnings about setting parameters too tight (excessive amendments)
- **Validation**: Configuration template is self-documenting

### 6.2 Create Feature Documentation
- [ ] Create `docs/features/feature1-tpsl/DYNAMIC_SL_V4.6_2026-01-XX.md`
- [ ] Document algorithm with examples (entry $100, firstMove triggers at $101, etc.)
- [ ] Provide calculation formulas and logic flow
- [ ] Include troubleshooting guide for common issues
- [ ] Add performance considerations (amendment frequency, API rate limits)
- **Validation**: Documentation is clear and includes real examples

### 6.3 Update VERSION_HISTORY.md
- [ ] Add V4.6 entry with dynamic SL changes
- [ ] List all modified files
- [ ] Explain feature benefits and configuration options
- [ ] Note backward compatibility (opt-in feature)
- **Validation**: Version history accurately reflects changes

## Dependencies and Parallelization

**Sequential Dependencies:**
- 1.1 (database schema) must complete before 1.2 (storage methods)
- 1.2 must complete before 2.1 (storage interface needed for tracker operations)
- 1.3 (config) must complete before 1.4 (config needed for API params)
- 2.1 and 2.2 must complete before 3.1 (tracker and algorithm needed)
- 3.1 must complete before 3.2 (integration needed before amendment)
- All implementation (1-3) must complete before testing (5)

**Parallelizable Work:**
- 1.3 (config) and 1.4 (API) can run in parallel with 1.1-1.2 (database work)
- 2.2 (calculation algorithm) can run in parallel with 1.4 (API work)
- 4.1 and 4.2 (logging) can be done alongside 3.2-3.3 (implementation)
- 6.1, 6.2, 6.3 (documentation) can run in parallel with 5.3 (manual testing)

## Success Criteria

- [ ] All unit tests pass with >90% coverage
- [ ] Integration tests demonstrate correct SL adjustment logic
- [ ] Manual test on real market shows SL moving to breakeven after 1% profit
- [ ] Configuration is clear and validated on load
- [ ] Logs provide full visibility into dynamic SL operations
- [ ] Feature can be disabled via config with no impact on existing TPSL behavior
- [ ] Documentation explains algorithm clearly with examples
