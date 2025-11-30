# Implementation Tasks: Automatic TPSL Management

This document outlines the implementation tasks for the automatic stop-loss and take-profit management feature. Tasks are ordered to deliver user-visible progress incrementally with validation at each step.

## Phase 1: Foundation and Configuration

### Task 1.1: Add TPSL configuration structure
**Description**: Extend configuration schema to support TPSL settings

**Implementation**:
- Add `TPSLConfig` struct to `internal/config/config.go`
- Add TPSL fields: `Enabled`, `CheckInterval`, `VolatilityPct`, `ProfitLossRatio`
- Add validation logic for TPSL config (volatility 0-1, ratio > 0, interval > 0)
- Update `config.template.yaml` with TPSL section and defaults

**Validation**:
- Unit test: config validation rejects invalid volatility/ratio
- Unit test: config loads defaults when TPSL section missing
- Manual: verify config.template.yaml has TPSL section with comments

**Dependencies**: None

**Estimated Complexity**: Low

---

### Task 1.2: Extend OKX client with algo order types
**Description**: Add Go types for OKX algo order API request/response

**Implementation**:
- Add `AlgoOrderRequest` struct to `internal/okx/types.go`
- Add `AlgoOrderResponse` struct for placement response
- Add `PendingAlgoOrdersResponse` struct for query response
- Add `AlgoOrder` struct to represent individual algo order data

**Validation**:
- Unit test: JSON marshaling/unmarshaling of new types
- Compare struct fields against OKX API documentation

**Dependencies**: None

**Estimated Complexity**: Low

---

### Task 1.3: Implement OKX get pending algo orders API
**Description**: Add method to query pending conditional algo orders

**Implementation**:
- Add `GetPendingAlgoOrders(ordType string)` method to `internal/okx/client.go`
- Build query string with ordType parameter
- Call `doRequest("GET", "/api/v5/trade/orders-algo-pending?ordType="+ordType)`
- Parse response into `PendingAlgoOrdersResponse`
- Handle API errors (code != "0")

**Validation**:
- Unit test: verify correct API path and query string
- Unit test: parse sample OKX response JSON
- Manual test: call API with real credentials, verify returns pending orders

**Dependencies**: Task 1.2

**Estimated Complexity**: Medium

---

### Task 1.4: Implement OKX place algo order API
**Description**: Add method to place conditional TPSL orders

**Implementation**:
- Add `PlaceAlgoOrder(req AlgoOrderRequest)` method to `internal/okx/client.go`
- Marshal request to JSON
- Update `doRequest` to support POST with body
- Call POST /api/v5/trade/order-algo with JSON body
- Parse response into `AlgoOrderResponse`
- Handle API errors and return algoId on success

**Validation**:
- Unit test: verify correct JSON serialization of request
- Unit test: parse sample success/error responses
- Manual test: place small test TPSL order (5 USDT position), verify created on OKX

**Dependencies**: Task 1.2

**Estimated Complexity**: Medium

---

## Phase 2: TPSL Core Logic

### Task 2.1: Create TPSL manager with coverage analysis
**Description**: Implement core logic to analyze position TPSL coverage

**Implementation**:
- Create `internal/tpsl/manager.go`
- Add `Manager` struct with config, OKX client, logger
- Add `New()` constructor
- Add `AnalyzeCoverage(position, algoOrders)` method
  - Filter algo orders matching position (instId, posSide)
  - Sum sizes of matching "live" conditional orders
  - Calculate uncovered size = position_size - covered_size
  - Return uncovered size
- Add helper `matchesPosition(algoOrder, position)` function

**Validation**:
- Unit test: no algo orders → uncovered = full position size
- Unit test: full coverage → uncovered = 0
- Unit test: partial coverage → correct uncovered calculation
- Unit test: multiple algo orders summed correctly
- Unit test: only "live" orders counted, "canceled" orders ignored

**Dependencies**: Task 1.3

**Estimated Complexity**: Medium

---

### Task 2.2: Implement TPSL price calculation
**Description**: Calculate stop-loss and take-profit trigger prices

**Implementation**:
- Add `CalculateTPSLPrices(position, config)` method to manager
- Extract position: entry price, leverage, position side
- Calculate SL_distance = entry_price × volatility_pct × leverage
- For long: SL = entry - distance, TP = entry + (distance × pl_ratio)
- For short: SL = entry + distance, TP = entry - (distance × pl_ratio)
- Handle "net" position side (determine direction from position size)
- Validate calculated prices > 0
- Return TPSLPrices struct with TP and SL values

**Validation**:
- Unit test: long position SL/TP calculation with various parameters
- Unit test: short position SL/TP calculation
- Unit test: net position handling
- Unit test: various leverage values (1x, 5x, 10x)
- Unit test: negative/zero price validation

**Dependencies**: None

**Estimated Complexity**: Medium

---

### Task 2.3: Implement TPSL order placement logic
**Description**: Place TPSL algo orders for uncovered positions

**Implementation**:
- Add `PlaceTPSL(position, uncoveredSize, prices)` method to manager
- Determine order side (opposite of position: sell for long, buy for short)
- Determine margin mode (tdMode) from position
- Build `AlgoOrderRequest`:
  - instId, tdMode, side, posSide from position
  - ordType = "conditional"
  - sz = uncoveredSize (as string)
  - tpTriggerPx, slTriggerPx from prices (as strings)
  - tpOrdPx = "-1", slOrdPx = "-1" (market orders)
  - reduceOnly = true
- Call OKX client PlaceAlgoOrder()
- Handle errors (log and return error)
- Log success with algoId

**Validation**:
- Unit test: correct AlgoOrderRequest constructed for long position
- Unit test: correct AlgoOrderRequest constructed for short position
- Unit test: cross vs isolated margin mode handling
- Manual test: place test TPSL, verify appears on OKX platform

**Dependencies**: Task 1.4, Task 2.2

**Estimated Complexity**: Medium

---

### Task 2.4: Implement main TPSL analysis workflow
**Description**: Orchestrate full TPSL check cycle

**Implementation**:
- Add `AnalyzeAndPlaceTPSL(positions)` method to manager
- Query pending algo orders from OKX API
- For each position:
  - Analyze coverage (Task 2.1)
  - If uncovered size > 0:
    - Calculate TPSL prices (Task 2.2)
    - Place TPSL order (Task 2.3)
    - Log result
  - If uncovered size == 0:
    - Log "position fully covered"
- Aggregate statistics (total checked, orders placed, failures)
- Return summary
- Handle individual position errors gracefully (log and continue)

**Validation**:
- Unit test: empty positions list handled
- Unit test: all positions fully covered → no orders placed
- Unit test: mixed coverage → orders placed only for uncovered
- Unit test: individual placement failure doesn't stop processing
- Integration test: mock OKX client, verify correct API calls

**Dependencies**: Task 2.1, Task 2.2, Task 2.3

**Estimated Complexity**: High

---

## Phase 3: Scheduler and Integration

### Task 3.1: Create TPSL scheduler
**Description**: Implement periodic TPSL check scheduler

**Implementation**:
- Create `internal/tpsl/scheduler.go`
- Add `Scheduler` struct with manager, storage, config, ticker, context
- Add `New(config, storage, okxClient, logger)` constructor
- Add `Start()` method:
  - Create ticker with check interval
  - Start goroutine
  - Run periodic checks in loop
  - Listen for context cancellation
- Add `Stop()` method for graceful shutdown
- Add `runCheck()` method:
  - Fetch current positions from storage
  - Call manager.AnalyzeAndPlaceTPSL()
  - Log summary

**Validation**:
- Unit test: scheduler starts and stops cleanly
- Unit test: context cancellation stops loop
- Integration test: verify runCheck called at intervals
- Manual test: observe logs showing periodic TPSL checks

**Dependencies**: Task 2.4

**Estimated Complexity**: Medium

---

### Task 3.2: Integrate TPSL scheduler into main application
**Description**: Wire TPSL scheduler into application startup

**Implementation**:
- Update `cmd/main.go`:
  - Load TPSL config
  - If TPSL enabled:
    - Create TPSL manager
    - Create TPSL scheduler
    - Start scheduler
  - Update shutdown logic to stop TPSL scheduler
- Ensure graceful shutdown waits for TPSL scheduler goroutine

**Validation**:
- Manual test: start application, verify TPSL scheduler starts
- Manual test: verify TPSL disabled flag prevents scheduler start
- Manual test: send SIGINT, verify clean shutdown with TPSL scheduler stopped

**Dependencies**: Task 3.1

**Estimated Complexity**: Low

---

### Task 3.3: Add position margin mode tracking
**Description**: Ensure position model includes margin mode field

**Implementation**:
- Check if `Position` model in `pkg/models/position.go` has margin mode field
- If missing, add `MarginMode string` field
- Update monitoring code to extract margin mode from OKX position response
- Update storage code to persist margin mode to database
- Add database migration if schema change needed (add column to positions table)

**Validation**:
- Unit test: position model validation includes margin mode
- Integration test: monitoring stores margin mode correctly
- Manual test: check database, verify margin mode populated

**Dependencies**: None (can be done in parallel with Phase 1-2)

**Estimated Complexity**: Medium

---

## Phase 4: Logging and Error Handling

### Task 4.1: Add comprehensive TPSL logging
**Description**: Implement detailed logging for TPSL operations

**Implementation**:
- Add DEBUG logs in manager:
  - Position analysis start/end
  - Coverage calculation details
  - TPSL price calculation (entry, leverage, SL, TP)
  - API request/response details
- Add INFO logs:
  - TPSL check cycle start/completion
  - Uncovered positions detected
  - Successful TPSL order placement with algoId
  - Check cycle summary (positions checked, orders placed)
- Add WARN logs:
  - Rate limit encountered
  - Transient errors during placement
- Add ERROR logs:
  - TPSL calculation validation failures
  - API authentication failures
  - Persistent placement failures
- Ensure sensitive data (API keys) masked in all logs

**Validation**:
- Manual test: review logs at DEBUG level, verify sufficient detail
- Manual test: verify no API keys/secrets in logs
- Manual test: trigger rate limit, verify WARN log
- Manual test: trigger auth error, verify ERROR log

**Dependencies**: Task 2.4, Task 3.1

**Estimated Complexity**: Low

---

### Task 4.2: Implement retry logic for transient errors
**Description**: Add retry with exponential backoff for recoverable errors

**Implementation**:
- Add retry wrapper function in `internal/tpsl/retry.go`
- Exponential backoff: 1s, 2s, 4s (max 3 retries)
- Apply retry to:
  - Get pending algo orders API call
  - Place algo order API call
  - Storage queries for positions
- Log each retry attempt with attempt number
- Return error after exhausting retries

**Validation**:
- Unit test: verify retry count and backoff timing
- Unit test: success on 2nd retry stops further retries
- Unit test: all retries fail returns error
- Integration test: mock API to fail twice then succeed, verify retries work

**Dependencies**: Task 1.3, Task 1.4

**Estimated Complexity**: Medium

---

### Task 4.3: Implement error recovery and resilience
**Description**: Ensure TPSL scheduler continues running after errors

**Implementation**:
- Wrap `runCheck()` in defer/recover to catch panics
- If panic occurs, log stack trace and continue to next interval
- Implement graceful degradation:
  - If storage unavailable, skip cycle and wait for next interval
  - If auth failure, stop scheduler (permanent error)
  - If rate limited, wait for next interval
  - If individual position fails, continue to next position
- Add health check: on repeated failures, log alert but keep running

**Validation**:
- Unit test: panic during check doesn't crash scheduler
- Manual test: simulate storage failure, verify scheduler continues
- Manual test: simulate auth failure, verify scheduler stops
- Manual test: one position placement fails, others still processed

**Dependencies**: Task 3.1, Task 4.2

**Estimated Complexity**: Medium

---

## Phase 5: Testing and Validation

### Task 5.1: Write unit tests for TPSL manager
**Description**: Comprehensive unit tests for TPSL business logic

**Implementation**:
- Create `internal/tpsl/manager_test.go`
- Test coverage analysis with various scenarios (no coverage, partial, full)
- Test TPSL price calculation for long/short positions, various leverage
- Test order placement request construction
- Test error handling paths
- Aim for >80% code coverage

**Validation**:
- Run `go test ./internal/tpsl/... -cover`
- Verify all tests pass
- Verify coverage > 80%

**Dependencies**: Task 2.1, Task 2.2, Task 2.3, Task 2.4

**Estimated Complexity**: High

---

### Task 5.2: Write integration tests for TPSL workflow
**Description**: End-to-end integration tests with mocked OKX API

**Implementation**:
- Create `internal/tpsl/integration_test.go`
- Mock OKX client with controllable responses
- Test full workflow:
  - Position with no TPSL → order placed
  - Position with partial TPSL → additional order placed
  - Position with full TPSL → no order placed
  - API errors → handled gracefully
- Verify correct API calls made with correct parameters

**Validation**:
- Run `go test ./internal/tpsl/... -tags=integration`
- Verify all integration tests pass

**Dependencies**: Task 2.4, Task 3.1

**Estimated Complexity**: High

---

### Task 5.3: Manual testing with live OKX API (testnet or small positions)
**Description**: Validate TPSL system with real API

**Implementation**:
- Use OKX demo trading environment OR small positions (≤5 USDT)
- Test scenarios:
  1. Open position without TPSL → verify auto-TPSL placed within check interval
  2. Open position with partial TPSL → verify additional TPSL order added
  3. Open position with full TPSL → verify no duplicate orders
  4. Verify TPSL prices calculated correctly (check against manual calculation)
  5. Test with long and short positions
  6. Test with different leverage (1x, 5x, 10x)
- Document test results

**Validation**:
- Checklist of scenarios tested and passed
- Screenshots or logs showing successful TPSL placement
- Verify TPSL orders appear on OKX platform

**Dependencies**: All previous tasks

**Estimated Complexity**: High (requires careful coordination with live API)

---

## Phase 6: Documentation and Finalization

### Task 6.1: Update README with TPSL feature
**Description**: Document TPSL feature in README.md

**Implementation**:
- Add section "Automatic Stop-Loss and Take-Profit Management"
- Describe feature purpose and behavior
- Document configuration options (enabled, interval, volatility, ratio)
- Provide examples of configuration
- Add troubleshooting section for common issues

**Validation**:
- Review README for clarity and completeness
- Verify code examples are correct

**Dependencies**: None (can be done in parallel)

**Estimated Complexity**: Low

---

### Task 6.2: Update config.template.yaml with detailed comments
**Description**: Ensure TPSL config template has clear documentation

**Implementation**:
- Add comments explaining each TPSL config field
- Provide examples of typical values
- Add warnings about API permissions required (Trade permission)
- Add notes about check interval trade-offs (longer = less API calls, slower response)

**Validation**:
- Review config.template.yaml for clarity
- Verify defaults are sensible for typical use case

**Dependencies**: Task 1.1

**Estimated Complexity**: Low

---

### Task 6.3: Create migration guide for existing users
**Description**: Document how existing users enable TPSL feature

**Implementation**:
- Create docs/tpsl-migration.md
- Steps:
  1. Pull latest code
  2. Update config.yaml with TPSL section (copy from template)
  3. Ensure OKX API key has "Trade" permission
  4. Restart application
  5. Verify TPSL scheduler started in logs
- Include troubleshooting tips

**Validation**:
- Review migration guide for clarity
- Test migration steps on clean system

**Dependencies**: All implementation tasks

**Estimated Complexity**: Low

---

## Summary

**Total Tasks**: 18

**Phases**:
1. Foundation and Configuration (4 tasks)
2. TPSL Core Logic (4 tasks)
3. Scheduler and Integration (3 tasks)
4. Logging and Error Handling (3 tasks)
5. Testing and Validation (3 tasks)
6. Documentation and Finalization (3 tasks)

**Parallel Work Opportunities**:
- Task 1.1, 1.2 can be done in parallel
- Task 1.3, 1.4 can be done in parallel after 1.2
- Phase 2 tasks have some dependencies but Task 2.1 and 2.2 can start simultaneously
- Task 3.3 can be done anytime in parallel with other tasks
- Phase 6 documentation tasks can be done in parallel with implementation

**Critical Path**:
1.2 → 1.3/1.4 → 2.1/2.2 → 2.3 → 2.4 → 3.1 → 3.2 → 5.3

**Key Milestones**:
- M1: OKX API integration complete (after Task 1.4)
- M2: TPSL core logic complete (after Task 2.4)
- M3: Scheduler integration complete (after Task 3.2)
- M4: Testing complete (after Task 5.3)
- M5: Production ready (after Task 6.3)
