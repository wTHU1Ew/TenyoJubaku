# Implementation Tasks: Order Frequency Control

## Task Breakdown

### Phase 1: Database Schema and Configuration (Foundation)

1. **Add database schema for order tracking**
   - Create migration for `order_history` table with all columns and indexes
   - Create migration for `pending_confirmations` table with all columns and indexes
   - Update storage initialization to run migrations
   - Add database schema version tracking
   - Write unit tests for schema creation
   - **Validation**: Run migrations on fresh database, verify tables created with correct schema

2. **Extend configuration structure**
   - Add `OrderControlConfig` struct to `internal/config/config.go`
   - Add nested structs: `FrequencyLimitConfig`, `MakerOnlyConfig`, `ConfirmationConfig`
   - Add validation functions for all config values
   - Update `config.template.yaml` with all new configuration sections and comments
   - Write unit tests for config parsing and validation
   - **Validation**: Load config with various valid/invalid values, verify validation works

3. **Update storage layer with new methods**
   - Add `InsertOrderHistory()` method to storage interface
   - Add `GetWeeklyOrderCount(weekStart time.Time)` method
   - Add `InsertPendingConfirmation()` method
   - Add `GetDueConfirmations(now time.Time)` method
   - Add `UpdatePendingConfirmation()` method
   - Add `GetPendingConfirmationByOrderID()` method
   - Implement all methods with proper error handling and transactions
   - Write unit tests for all storage methods
   - **Validation**: Test CRUD operations on both tables, verify indexes improve query performance

### Phase 2: OKX Client Extensions (API Integration)

4. **Add OKX ticker API support**
   - Add `GetTicker(instId string)` method to OKX client
   - Add `TickerResponse` struct to `internal/okx/types.go`
   - Implement with retry logic and error handling
   - Write unit tests with mocked responses
   - **Validation**: Call ticker API with real credentials, verify price returned

5. **Add OKX order management API support**
   - Add `PlaceOrder(req OrderRequest)` method to OKX client
   - Add `AmendOrder(ordId, instId, newSz string)` method
   - Add `CancelOrder(ordId, instId string)` method
   - Add `GetOrderDetails(ordId, instId string)` method
   - Add corresponding request/response structs to types.go
   - Implement with authentication and retry logic
   - Write unit tests with mocked responses
   - **Validation**: Test with 5 USDT orders on testnet/live (place, amend, cancel, query)

### Phase 3: Core Validation Logic (Business Rules)

6. **Implement frequency limit validator**
   - Create `internal/ordercontrol/frequency.go` with `FrequencyValidator` struct
   - Implement `ValidateFrequency(ctx context.Context, order OrderRequest)` method
   - Implement week boundary calculation (Monday 00:00:00 UTC)
   - Implement order counting query with reduce-only exclusion
   - Add comprehensive logging for all validation decisions
   - Write unit tests for week calculations and frequency counting
   - Write integration tests with database
   - **Validation**: Place 5 orders in test week, verify 6th is rejected

7. **Implement maker-only validator**
   - Create `internal/ordercontrol/maker.go` with `MakerValidator` struct
   - Implement `ValidateMakerOnly(ctx context.Context, order OrderRequest)` method
   - Implement market order rejection logic with reduce-only exception
   - Implement price distance calculation with ticker integration
   - Implement taker percentage validation with position query
   - Add ticker price caching (5 second TTL)
   - Add comprehensive logging for all validation decisions
   - Write unit tests for all validation scenarios
   - Write integration tests with ticker API and positions
   - **Validation**: Test market order rejection, limit order distance validation, taker percentage

8. **Implement reduce-only detection**
   - Create `internal/ordercontrol/position.go` with helper functions
   - Implement `IsReduceOnly(order OrderRequest, currentPosition Position)` function
   - Query position from database (monitoring table) or OKX API fallback
   - Handle long/short position direction matching
   - Write unit tests for all position scenarios
   - **Validation**: Verify correct reduce-only detection for various order/position combinations

### Phase 4: Order Control Service (Orchestration)

9. **Create order control service**
   - Create `internal/ordercontrol/service.go` with `OrderControlService` struct
   - Implement `PlaceOrder(ctx context.Context, req OrderRequest)` method
   - Orchestrate all validations: frequency → maker-only → placement
   - Record order in `order_history` table on successful placement
   - Record pending orders in `pending_confirmations` table if applicable
   - Implement comprehensive error handling and rollback
   - Add detailed logging for entire order flow
   - Write unit tests for service orchestration
   - Write integration tests for end-to-end order placement
   - **Validation**: Place test orders through service, verify all validations executed

10. **Add order status tracking**
    - Implement background sync to query OKX order status
    - Update `order_history` table with status changes (filled, canceled)
    - Update `pending_confirmations` status for filled/canceled orders
    - Run sync every 5 minutes (configurable)
    - Write unit tests for status update logic
    - **Validation**: Place order, wait for fill, verify status updated in database

### Phase 5: Confirmation System (Automated Workflow)

11. **Implement confirmation scheduler**
    - Create `internal/ordercontrol/confirmation_scheduler.go`
    - Implement `ConfirmationScheduler` struct similar to TPSL scheduler
    - Implement `Start()` and `Stop()` methods with graceful shutdown
    - Implement periodic check cycle (every 5 minutes)
    - Query due confirmations from database
    - Write unit tests for scheduler lifecycle
    - **Validation**: Start scheduler, verify periodic cycles run

12. **Implement confirmation notification logic**
    - Implement `ProcessDueConfirmations(ctx context.Context)` method
    - Detect orders due for confirmation (next_confirmation_due <= now)
    - Log confirmation notifications with WARN level
    - Update `last_confirmation_at` and `next_confirmation_due`
    - Increment `confirmation_count`
    - Write unit tests for notification logic
    - **Validation**: Create pending confirmation with past due date, verify notification logged

13. **Implement timeout detection and size reduction**
    - Implement `ProcessTimeouts(ctx context.Context)` method
    - Detect orders past waiting period (now - last_confirmation_at > waiting_period)
    - Calculate reduced size (current_size * timeout_size_reduction_pct)
    - Call OKX amend order API to update size
    - Update `current_size` and increment `timeout_count` in database
    - Handle minimum order size constraint (cancel if below minimum)
    - Write unit tests for timeout detection and size calculation
    - Write integration tests with OKX amend API
    - **Validation**: Create pending order, trigger timeout, verify size amended on OKX

14. **Implement order cancellation after max timeouts**
    - Extend timeout processing to check max_timeouts limit
    - Call OKX cancel order API when limit exceeded
    - Update pending_confirmations status to 'canceled'
    - Write unit tests for cancellation logic
    - Write integration tests with OKX cancel API
    - **Validation**: Create order with timeout_count=2, trigger timeout, verify order canceled

15. **Implement order status synchronization**
    - Implement `SyncOrderStatus(ctx context.Context)` method in scheduler
    - Query OKX order details for all pending confirmations
    - Update status for filled/canceled orders
    - Remove from active tracking (update status in database)
    - Run sync as part of each confirmation check cycle
    - Write unit tests for sync logic
    - **Validation**: Fill order manually on OKX, wait for sync, verify removed from tracking

### Phase 6: Integration and Main Entry Point

16. **Integrate order control service into main application**
    - Update `cmd/main.go` to initialize OrderControlService
    - Pass all dependencies (config, storage, OKX client, logger)
    - Start confirmation scheduler if enabled
    - Add graceful shutdown for confirmation scheduler
    - Write integration tests for full application startup
    - **Validation**: Start application, verify order control initialized, schedulers running

17. **Add CLI command or API endpoint for placing orders (future)**
    - This task is a placeholder for future CLI/API development
    - For initial implementation, orders can be placed via direct service calls in tests
    - Document how to integrate order placement into UI/CLI layer
    - **Note**: Not required for this proposal, but document integration points

### Phase 7: Testing and Validation

18. **Write comprehensive integration tests**
    - Test frequency limit enforcement across multiple orders
    - Test maker-only validation with various order types and distances
    - Test confirmation workflow from placement to timeout to cancellation
    - Test week boundary rollover for frequency counting
    - Test concurrent order placement
    - Use real OKX testnet or live with 5 USDT limit
    - **Validation**: All integration tests pass, order controls work as specified

19. **Write end-to-end scenario tests**
    - Scenario: Place 5 orders, verify 6th rejected
    - Scenario: Place market order, verify rejected (non-reduce-only)
    - Scenario: Place limit order 0.5% away, verify rejected
    - Scenario: Place limit order 2% away, verify accepted
    - Scenario: Place order, wait 12 hours, verify confirmation notification
    - Scenario: Place order, ignore confirmation for 4 hours, verify size reduced
    - Scenario: Order reaches 3 timeouts, verify canceled
    - **Validation**: All scenarios execute successfully with expected outcomes

20. **Performance testing**
    - Test frequency query performance with 10,000 order_history records
    - Test confirmation query performance with 1,000 pending_confirmations
    - Verify indexes are used (EXPLAIN QUERY PLAN)
    - Measure memory usage of schedulers
    - Test concurrent order placement (10 simultaneous orders)
    - **Validation**: All queries < 10ms, no memory leaks, no race conditions

### Phase 8: Documentation and Configuration

21. **Update configuration template and documentation**
    - Add all order_control configuration to `config.template.yaml` with detailed comments
    - Add example values and explanations for each parameter
    - Document what each setting controls
    - Add warnings for sensitive settings (e.g., high price distance)
    - **Validation**: Configuration template is clear and self-documenting

22. **Update README with order control features**
    - Add section explaining order frequency control
    - Document maker-only trading rules
    - Explain confirmation system workflow
    - Add configuration examples
    - Add troubleshooting guide for common issues
    - **Validation**: Documentation is clear and helpful

23. **Add logging documentation**
    - Document all log levels used (DEBUG, INFO, WARN, ERROR)
    - Document key log messages to watch for
    - Document how to interpret confirmation notifications
    - Document how to debug order rejections
    - **Validation**: Log messages are understandable and actionable

### Phase 9: Deployment and Rollout

24. **Database migration for existing installations**
    - Create migration script to add new tables to existing database
    - Test migration on copy of production database
    - Document migration procedure
    - Add rollback procedure in case of issues
    - **Validation**: Migration runs successfully on test database

25. **Configuration migration**
    - Document how to update config.yaml for existing installations
    - Provide sensible defaults for all new settings
    - Test with existing config files (backward compatibility)
    - **Validation**: Existing installations can upgrade smoothly

26. **Gradual rollout plan**
    - Phase 1: Enable order_control but keep all sub-features disabled
    - Phase 2: Enable frequency_limit with high limit (e.g., 50/week)
    - Phase 3: Enable maker_only with lenient settings
    - Phase 4: Enable confirmation with long intervals
    - Phase 5: Adjust to final strict settings (5/week, 1% distance, 12h confirmation)
    - **Validation**: Rollout can be done incrementally with low risk

## Dependencies Between Tasks

- Tasks 1-3 must complete before any other tasks (foundation)
- Tasks 4-5 can run in parallel (OKX API extensions)
- Tasks 6-8 depend on tasks 1-5 (validation logic needs DB and API)
- Task 9 depends on tasks 6-8 (service orchestrates validators)
- Task 10 depends on tasks 4-5 (needs OKX order API)
- Tasks 11-15 depend on task 3 (confirmation needs DB schema)
- Task 16 depends on tasks 9-15 (integration needs all components)
- Tasks 18-20 depend on task 16 (testing needs integrated system)
- Tasks 21-23 can run in parallel with testing (documentation)
- Tasks 24-26 depend on all previous tasks (deployment is last)

## Parallelizable Work

- Tasks 4 and 5 can be done by one developer while another works on tasks 1-3
- Tasks 6, 7, 8 can be split among multiple developers after tasks 1-5 complete
- Tasks 11, 12, 13, 14, 15 can be done in parallel by multiple developers
- Tasks 21, 22, 23 can be done by technical writer while developers do tasks 18-20

## Estimated Effort

- Phase 1 (Foundation): 8-12 hours
- Phase 2 (API Integration): 6-8 hours
- Phase 3 (Validation Logic): 10-14 hours
- Phase 4 (Service): 6-8 hours
- Phase 5 (Confirmation): 12-16 hours
- Phase 6 (Integration): 4-6 hours
- Phase 7 (Testing): 10-14 hours
- Phase 8 (Documentation): 4-6 hours
- Phase 9 (Deployment): 4-6 hours

**Total Estimated Effort**: 64-90 hours (approximately 2-3 weeks for one developer)

## Success Criteria

All tasks must meet their validation criteria before moving to next phase. Integration tests must pass with real OKX API (testnet or live with 5 USDT maximum). All code must have >90% unit test coverage. Documentation must be clear and complete.
