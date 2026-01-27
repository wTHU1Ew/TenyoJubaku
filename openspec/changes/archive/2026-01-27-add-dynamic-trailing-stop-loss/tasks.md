# Implementation Tasks: Dynamic Trailing Stop-Loss (V5.1 - Leverage-Aware)

## Overview of V5.1 Changes

**V5.0 (Original)**: After 1% profit → immediate move to breakeven → trailing mode
**V5.1 (New)**: Leverage-aware gradual move to breakeven → then trailing mode

Key algorithm changes:
- **Phase A**: When account profit >= 1% × leverage → SL moves by 1.01% × leverage
- **Phase A continues**: Repeat until SL covers entry price (breakeven reached)
- **Phase B**: Only after breakeven → trail SL: every 0.5% price gain → SL moves up 0.1%

---

## Phase 1: Database Schema, Configuration, and API Foundation

### 1.1 Update Dynamic SL Tracking Database Schema (V5.1)
- [x] Add `DynamicSLTracker` model struct in `pkg/models/dynamic_sl_tracker.go`
- [x] Add GORM struct tags for all fields (id, position_key, inst_id, pos_side, etc.)
- [x] Add unique index on `position_key` column
- [x] Add `TableName()` method returning "dynamic_sl_tracking"
- [x] Add `Validate()` method to check field constraints
- [x] Update `internal/storage/storage.go` to include tracker in AutoMigrate
- [x] Write unit tests for model validation
- [x] **V5.1**: Add `InitialSlPrice` field to track original SL distance
- [x] **V5.1**: Add `Leverage` field to store position leverage
- [x] **V5.1**: Rename `FirstMoveTriggered` to `BreakevenReached` (boolean)
- [x] **V5.1**: Add `MoveCount` field to track number of SL adjustments
- [x] **V5.1**: Update `Validate()` method for new fields
- [x] **V5.1**: Update unit tests for new schema
- **Validation**: Run AutoMigrate, verify new columns exist

### 1.2 Add Storage Interface Methods for Dynamic SL
- [x] Add `InsertDynamicSLTracker(ctx, tracker)` method to storage interface
- [x] Add `GetDynamicSLTracker(ctx, positionKey)` method
- [x] Add `UpdateDynamicSLTracker(ctx, tracker)` method
- [x] Add `DeleteDynamicSLTracker(ctx, positionKey)` method
- [x] Add `GetAllDynamicSLTrackers(ctx)` method (load on startup)
- [x] Add `CleanupOrphanedTrackers(ctx, openPositionKeys)` method
- [x] Implement all methods with GORM operations
- [x] Write unit tests for all CRUD operations
- **Validation**: ✅ Test full CRUD cycle with SQLite in-memory database

### 1.3 Update Dynamic SL Configuration Structure (V5.1)
- [x] Add `DynamicSLConfig` struct to `internal/config/config.go`
- [x] Add fields: `Enabled`, `FirstMovePct`, `TrailingStepPct`, `StopMoveStepPct`
- [x] Add validation for config values (all percentages > 0, < 1.0)
- [x] Update `config.template.yaml` with `dynamic_sl` section and detailed comments
- [x] Write unit tests for config parsing and validation
- [x] **V5.1**: Rename `FirstMovePct` to `ProfitStepPct` (account profit threshold base)
- [x] **V5.1**: Add `SlMoveStepPct` field (SL move step base, default 1.01%)
- [x] **V5.1**: Update config validation for new fields
- [x] **V5.1**: Update `config.template.yaml` with V5.1 parameters and examples
- [x] **V5.1**: Update unit tests for new config structure
- **Validation**: Load config with valid/invalid values, verify validation works

### 1.4 Verify or Add OKX Amend Algo Order API
- [x] Check if `AmendAlgoOrder` method exists in `internal/okx/client.go`
- [x] If not exists: Add `AmendAlgoOrder(ctx, algoId, instId, newSlTriggerPx)` method
- [x] Add `AmendAlgoOrderRequest` and `AmendAlgoOrderResponse` structs to types
- [x] Implement with proper authentication and error handling
- [x] Write unit tests with mocked API responses
- **Validation**: ✅ Call amend API on testnet with real algo order, verify SL price changed

## Phase 2: Dynamic SL Calculation Logic (V5.1 - Complete Rewrite)

### 2.1 Update Position Profit Tracking with Leverage (V5.1)
- [x] Create helper functions in `internal/tpsl/dynamic_sl.go` for tracker operations
- [x] Implement `LoadOrCreateTracker(ctx, storage, position)` function
- [x] Implement `UpdateTracker(ctx, storage, tracker, currentPrice)` function
- [x] Implement `ShouldAdjustSL(tracker, config)` function to check if adjustment needed
- [x] Write unit tests for all functions with mock storage
- [x] **V5.1**: Update `LoadOrCreateTracker` to store `leverage` and `initialSlPrice`
- [x] **V5.1**: Calculate account profit using leverage: `priceProfit × leverage`
- [x] **V5.1**: Update `UpdateTracker` to check against leverage-scaled threshold
- [x] **V5.1**: Track `BreakevenReached` state transition
- [x] **V5.1**: Update unit tests for leverage-aware tracking
- **Validation**: Simulate price movements with different leverages, verify correct thresholds

### 2.2 Implement V5.1 Two-Phase Dynamic SL Algorithm
- [x] Create `CalculateDynamicSL(entry, current, highest, config)` function
- [x] Handle long and short positions (reverse logic for shorts)
- [x] Write comprehensive unit tests for all scenarios
- [x] **V5.1 REWRITE**: Update `CalculateDynamicSL` signature to include `leverage`, `initialSL`, `breakevenReached`
- [x] **V5.1 Phase A**: Check if `accountProfit >= profitStepPct × leverage`
  - If YES and NOT breakevenReached: return `currentSL × (1 + slMoveStepPct × leverage)` for long
  - Check if new SL covers entry: `newSL >= entry × 1.001` (long) or `newSL <= entry × 0.999` (short)
  - If covers entry: set `breakevenReached = true`
- [x] **V5.1 Phase B**: Only when `breakevenReached = true`
  - Check trailing condition: `(current - highest) / highest >= trailingStepPct`
  - If YES: return `currentSL × (1 + stopMoveStepPct)`
- [x] **V5.1**: Handle short positions (reverse all comparisons and multipliers)
- [x] **V5.1**: Write comprehensive unit tests for two-phase algorithm
  - Test: Phase A with various leverages (1x, 5x, 10x, 20x)
  - Test: Multiple Phase A moves before breakeven
  - Test: Transition from Phase A to Phase B
  - Test: Phase B trailing after breakeven
  - Test: Edge cases (exact threshold, large jumps)
- **Validation**: Test all scenarios with different leverage values

## Phase 3: TPSL Manager Integration (V5.1 Updates)

### 3.1 Integrate V5.1 Dynamic SL into TPSL Check Cycle
- [x] Add `dynamicSLEnabled` flag check in manager
- [x] Add storage reference to Manager struct (for tracker CRUD operations)
- [x] In `checkTPSLCoverage()` cycle, after coverage check
- [x] Add cleanup logic: Delete trackers for closed positions from DB
- [x] Write integration tests with mock storage and OKX client
- [x] **V5.1**: Read position leverage from OKX position data
- [x] **V5.1**: Pass leverage to `LoadOrCreateTracker`
- [x] **V5.1**: Calculate account profit (not just price profit)
- [x] **V5.1**: Update tracker with Phase A/B logic
- [x] **V5.1**: Update integration tests for two-phase logic
- **Validation**: Run full TPSL cycle with dynamic SL, verify leverage-aware behavior

### 3.2 Update SL Amendment Logic for V5.1
- [x] Create `amendStopLoss(ctx, storage, position, algoOrder, newSlPrice)` method
- [x] Query pending algo orders to find matching SL order by instId and posSide
- [x] Call `AmendAlgoOrder` API with new `slTriggerPx`
- [x] Handle amendment success and failure
- [x] Add comprehensive logging for all amendment operations
- [x] **V5.1**: Log which phase (A or B) triggered the adjustment
- [x] **V5.1**: Log move count and distance to breakeven
- [x] **V5.1**: Update `MoveCount` in tracker after successful amendment
- **Validation**: Manually trigger SL adjustment, verify correct phase logging

### 3.3 Handle Edge Cases and Error Recovery
- [x] Handle case: Position closed before SL adjustment
- [x] Handle case: Algo order already filled/cancelled
- [x] Handle case: OKX API rate limit during amendment
- [x] Handle case: Amendment fails due to invalid price (too close to market)
- [x] Implement retry logic with exponential backoff (max 3 retries)
- [x] Add circuit breaker: stop amendments if consecutive failures > 10
- **Validation**: ✅ Simulate all error scenarios, verify graceful handling

## Phase 4: Logging and Monitoring (V5.1 Updates)

### 4.1 Update Logging for Two-Phase Algorithm
- [x] Log when firstMove threshold is reached (INFO level)
- [x] Log every SL adjustment with old/new prices (INFO level)
- [x] Log calculation details (current price, highest, threshold) at DEBUG level
- [x] Log amendment API calls and responses (DEBUG level)
- [x] Log amendment failures with full error details (ERROR level)
- [x] **V5.1**: Log current phase (A: "Moving to Breakeven" or B: "Trailing")
- [x] **V5.1**: Log leverage value in calculation details
- [x] **V5.1**: Log account profit (not just price profit)
- [x] **V5.1**: Log distance to breakeven during Phase A
- [x] **V5.1**: Log when breakeven is reached (INFO level)
- [x] **V5.1**: Log move count in adjustment messages
- **Validation**: Review logs, ensure phase and leverage info is clear

### 4.2 Update Dynamic SL Summary for V5.1
- [x] Extend TPSL check cycle summary to include dynamic SL metrics
- [x] Log summary at INFO level after each cycle
- [x] **V5.1**: Add "Phase A positions" count (not yet at breakeven)
- [x] **V5.1**: Add "Phase B positions" count (in trailing mode)
- [x] **V5.1**: Add "Breakeven triggers this cycle" count
- **Validation**: Check cycle summary includes V5.1 phase metrics

## Phase 5: Testing and Validation (V5.1 - New Tests Required)

### 5.1 Unit Tests (V5.1 - Rewrite Required)
- [x] Test `DynamicSLTracker` state machine (all transitions)
- [x] Test `CalculateDynamicSL` algorithm (long/short, all thresholds)
- [x] Test config parsing and validation
- [x] Test amendment API request formatting
- [x] **V5.1**: Test Phase A with 1x leverage (should behave similarly to V5.0)
- [x] **V5.1**: Test Phase A with 10x leverage (threshold = 10% account profit)
- [x] **V5.1**: Test Phase A with 20x leverage (threshold = 20% account profit)
- [x] **V5.1**: Test multiple Phase A moves before breakeven (e.g., 2% initial SL needs 2 moves)
- [x] **V5.1**: Test exact breakeven boundary conditions
- [x] **V5.1**: Test transition from Phase A to Phase B
- [x] **V5.1**: Test Phase B trailing logic (same as V5.0)
- [x] **V5.1**: Test short positions with all above scenarios
- [x] Achieve >90% code coverage for new V5.1 code
- **Validation**: All unit tests pass

### 5.2 Integration Tests (V5.1 Scenarios)
- [x] Test full cycle: position open → firstMove → SL to breakeven
- [x] Test trailing: price increases → multiple SL adjustments
- [x] Test price reversal after firstMove: SL stays at breakeven
- [x] Test concurrent positions with different profit levels
- [x] Test dynamic SL disabled: no amendments occur
- [x] **V5.1**: Test 10x leverage position: single move to breakeven
- [x] **V5.1**: Test 2x leverage position: multiple moves to breakeven
- [x] **V5.1**: Test mixed leverages: positions at different phases simultaneously
- [x] **V5.1**: Test leverage change mid-position (should use original leverage)
- **Validation**: Integration tests pass with mock OKX API

### 5.3 Manual Testing with Real API (V5.1)
- [x] Deploy to test environment with dynamic SL enabled
- [x] Open small position (max 5 USDT as per project conventions)
- [x] Wait for profit, verify SL moves
- [x] Check OKX algo order details, confirm SL price updated
- [x] Close position manually, verify no errors in logs
- [x] **V5.1**: Test with 10x leverage position
- [x] **V5.1**: Verify Phase A triggers at ~10% account profit
- [x] **V5.1**: Verify SL moves by ~10.1%
- [x] **V5.1**: Verify Phase B starts only after breakeven
- [x] **V5.1**: Document actual behavior vs expected behavior
- **Validation**: Real position shows V5.1 leverage-aware behavior

## Phase 6: Documentation (V5.1 Updates)

### 6.1 Update Configuration Documentation (V5.1)
- [x] Add detailed comments to `config.template.yaml` for `dynamic_sl` section
- [x] Explain each parameter and its effect on trading behavior
- [x] Provide example values and recommended settings
- [x] Add warnings about setting parameters too tight (excessive amendments)
- [x] **V5.1**: Update config comments for `profit_step_pct` and `sl_move_step_pct`
- [x] **V5.1**: Add examples with different leverage scenarios
- [x] **V5.1**: Explain two-phase algorithm in config comments
- **Validation**: Configuration template is self-documenting for V5.1

### 6.2 Update Feature Documentation (V5.1)
- [x] Create feature documentation with algorithm examples
- [x] Provide calculation formulas and logic flow
- [x] Include troubleshooting guide for common issues
- [x] Add performance considerations
- [x] **V5.1**: Update algorithm description for two-phase approach
- [x] **V5.1**: Add leverage-aware calculation examples
- [x] **V5.1**: Update state machine diagram for Phase A/B
- [x] **V5.1**: Add troubleshooting for leverage-related issues
- **Validation**: Documentation clearly explains V5.1 algorithm

### 6.3 Update VERSION_HISTORY.md (V5.1)
- [x] Add V5.0 entry with dynamic SL changes
- [x] List all modified files
- [x] Explain feature benefits and configuration options
- [x] Note backward compatibility (opt-in feature)
- [x] **V5.1**: Add V5.1 entry with leverage-aware changes
- [x] **V5.1**: Document algorithm change from V5.0 to V5.1
- [x] **V5.1**: List new/modified fields and config options
- [x] **V5.1**: Note migration path (database schema change)
- **Validation**: Version history accurately reflects V5.1 changes

## Dependencies and Parallelization

**Sequential Dependencies:**
- 1.1 (schema updates) must complete before 2.1 (leverage tracking)
- 1.3 (config updates) must complete before 2.2 (algorithm uses new config)
- 2.1 and 2.2 must complete before 3.1 (integration needs new algorithm)
- All implementation (1-3) must complete before testing (5)

**Parallelizable Work:**
- 1.3 (config) and 1.1 (schema) can run in parallel
- 4.1/4.2 (logging) can be done alongside 3.2 (implementation)
- 6.1, 6.2, 6.3 (documentation) can run in parallel with 5.3 (manual testing)

## Success Criteria (V5.1)

- [x] Database schema includes `leverage`, `initial_sl_price`, `breakeven_reached`, `move_count`
- [x] Config includes `profit_step_pct` and `sl_move_step_pct` (leverage multipliers)
- [x] All V5.1 unit tests pass with >90% coverage
- [x] Integration tests demonstrate correct two-phase algorithm
- [x] 10x leverage position: Phase A triggers at ~10% account profit, SL moves ~10.1%
- [x] Phase B (trailing) only starts after SL >= breakeven
- [x] Logs clearly indicate current phase (A/B) and leverage
- [x] Feature can be disabled via config with no impact on existing behavior
- [x] Documentation explains V5.1 algorithm clearly with leverage examples
