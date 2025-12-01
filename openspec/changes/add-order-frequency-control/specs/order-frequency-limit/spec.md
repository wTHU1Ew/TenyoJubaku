# order-frequency-limit Specification

## Purpose
Limit the number of orders placed per week to prevent overtrading and enforce trading discipline.

## ADDED Requirements

### Requirement: Order Frequency Configuration
The system SHALL load order frequency limit configuration from the YAML config file with validation.

#### Scenario: Valid frequency configuration
- **GIVEN** a valid config.yaml with order_control.frequency_limit section
- **WHEN** the system starts
- **THEN** the system loads enabled flag (default: true)
- **AND** loads weekly_max_orders (default: 5)
- **AND** loads exclude_reduce_only flag (default: true)
- **AND** logs "Order frequency limit configuration loaded: weekly_max=5, exclude_reduce_only=true" with INFO level

#### Scenario: Missing frequency configuration
- **GIVEN** config.yaml exists but has no order_control.frequency_limit section
- **WHEN** the system starts
- **THEN** the system uses default frequency limit configuration values
- **AND** logs "Using default order frequency limit configuration" with INFO level

#### Scenario: Invalid weekly max orders
- **GIVEN** config.yaml has weekly_max_orders <= 0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid weekly_max_orders, must be positive integer" with ERROR level
- **AND** exits with status code 1

#### Scenario: Frequency limit disabled
- **GIVEN** config.yaml has order_control.frequency_limit.enabled = false
- **WHEN** the system starts
- **THEN** the system logs "Order frequency limit disabled in configuration" with INFO level
- **AND** does not enforce frequency limits on order placement

### Requirement: Order History Persistence
The system SHALL persist all order placements to a database table for frequency tracking.

#### Scenario: Record successful order placement
- **GIVEN** an order is successfully placed via OKX API
- **WHEN** the system receives order confirmation
- **THEN** the system inserts a record into the order_history table
- **AND** the record includes order_id, inst_id, side, ord_type, size, price, reduce_only, placed_at (UTC), week_start (Monday UTC), status='placed'
- **AND** week_start is calculated as the most recent Monday 00:00:00 UTC
- **AND** the insertion is logged with DEBUG level

#### Scenario: Order placement failure
- **GIVEN** an order placement attempt fails at OKX API
- **WHEN** the OKX API returns an error
- **THEN** the system does NOT insert a record into order_history
- **AND** logs the failure with ERROR level
- **AND** returns error to the caller

#### Scenario: Database write failure during tracking
- **GIVEN** an order is successfully placed via OKX API
- **WHEN** the system attempts to write to order_history table
- **AND** a database error occurs
- **THEN** the system retries the write up to 3 times with exponential backoff (1s, 2s, 4s)
- **AND** logs each retry attempt with WARN level
- **AND** if all retries fail, logs critical error with ERROR level
- **AND** the order remains placed on OKX (no rollback), but tracking is incomplete

#### Scenario: Week boundary calculation
- **GIVEN** the current time is Sunday 2025-12-07 23:59:59 UTC
- **WHEN** the system calculates week_start for a new order
- **THEN** week_start = "2025-12-01" (previous Monday)
- **GIVEN** the current time is Monday 2025-12-08 00:00:00 UTC
- **WHEN** the system calculates week_start for a new order
- **THEN** week_start = "2025-12-08" (current Monday)

### Requirement: Frequency Limit Enforcement
The system SHALL validate order frequency before placing new orders and reject orders exceeding the weekly limit.

#### Scenario: Order within weekly limit
- **GIVEN** the weekly limit is 5 orders
- **AND** 4 orders have been placed in the current week (Monday-Sunday UTC)
- **WHEN** the user attempts to place a new order
- **THEN** the system queries order_history for count where week_start = current_week_start AND status = 'placed' AND reduce_only = false
- **AND** count = 4 (within limit)
- **AND** the system allows the order to proceed
- **AND** logs "Order frequency check passed: 4/5 orders this week" with INFO level

#### Scenario: Order exceeding weekly limit
- **GIVEN** the weekly limit is 5 orders
- **AND** 5 orders have been placed in the current week
- **WHEN** the user attempts to place a new order
- **THEN** the system queries order_history for count
- **AND** count = 5 (limit reached)
- **AND** the system rejects the order
- **AND** logs "Order rejected: weekly limit exceeded (5/5 orders)" with WARN level
- **AND** returns error to caller with message "Weekly order limit exceeded: 5/5 orders placed this week"

#### Scenario: Reduce-only orders excluded from limit
- **GIVEN** the weekly limit is 5 orders
- **AND** 5 regular orders have been placed in the current week
- **AND** exclude_reduce_only = true
- **WHEN** the user attempts to place a reduce-only order (TP/SL)
- **THEN** the system counts only non-reduce-only orders
- **AND** the reduce-only order is NOT counted toward the limit
- **AND** the system allows the order to proceed
- **AND** logs "Reduce-only order allowed despite limit (excluded from count)" with INFO level

#### Scenario: Week rollover resets count
- **GIVEN** 5 orders were placed last week (ending Sunday 2025-11-30 23:59:59 UTC)
- **WHEN** the user attempts to place a new order on Monday 2025-12-01 00:00:01 UTC
- **THEN** the system calculates week_start = "2025-12-01"
- **AND** queries order_history where week_start = "2025-12-01"
- **AND** count = 0 (new week)
- **AND** the system allows the order to proceed
- **AND** logs "Order frequency check passed: 0/5 orders this week (new week)" with INFO level

#### Scenario: Database query failure during validation
- **GIVEN** the system attempts to query order_history for frequency check
- **WHEN** a database error occurs
- **THEN** the system retries the query up to 3 times with exponential backoff
- **AND** logs each retry with WARN level
- **AND** if all retries fail, rejects the order as a fail-safe measure
- **AND** logs "Order rejected: unable to verify frequency limit (database error)" with ERROR level
- **AND** returns error to caller

### Requirement: Order History Database Schema
The system SHALL create and maintain the order_history table with proper indexes for efficient querying.

#### Scenario: Database initialization with order_history table
- **GIVEN** the system starts for the first time or database is missing order_history table
- **WHEN** the database initialization runs
- **THEN** the system creates the order_history table with columns:
  - id (INTEGER PRIMARY KEY AUTOINCREMENT)
  - order_id (TEXT NOT NULL)
  - inst_id (TEXT NOT NULL)
  - side (TEXT NOT NULL)
  - ord_type (TEXT NOT NULL)
  - size (TEXT NOT NULL)
  - price (TEXT, nullable)
  - reduce_only (BOOLEAN NOT NULL DEFAULT 0)
  - placed_at (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)
  - week_start (DATE NOT NULL)
  - status (TEXT NOT NULL)
  - created_at (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)
- **AND** creates index idx_order_history_week on week_start
- **AND** creates index idx_order_history_placed_at on placed_at
- **AND** creates index idx_order_history_order_id on order_id
- **AND** logs "Order history table initialized successfully" with INFO level

#### Scenario: Query performance with week_start index
- **GIVEN** order_history table contains 10,000 records spanning 52 weeks
- **WHEN** the system queries for order count in current week
- **THEN** the query uses idx_order_history_week index
- **AND** query execution time is < 10ms
- **AND** the query plan shows index usage (verified in unit tests)

### Requirement: Logging and Auditing
The system SHALL log all frequency limit checks and enforcement actions for auditing.

#### Scenario: Log successful frequency check
- **GIVEN** the system validates order frequency before placement
- **WHEN** the order is within the limit
- **THEN** the system logs with INFO level:
  - Current order count (e.g., "4/5")
  - Week start date
  - Order details (instId, side, size)
- **AND** log message format: "Order frequency check passed: 4/5 orders this week (week starting 2025-12-01), placing order BTC-USDT-SWAP buy 0.1"

#### Scenario: Log frequency limit rejection
- **GIVEN** the system validates order frequency before placement
- **WHEN** the order exceeds the limit
- **THEN** the system logs with WARN level:
  - Current order count (e.g., "5/5")
  - Week start date
  - Rejected order details (instId, side, size)
  - Reason for rejection
- **AND** log message format: "Order rejected: weekly limit exceeded (5/5 orders, week starting 2025-12-01), order BTC-USDT-SWAP buy 0.1 not placed"

#### Scenario: Log reduce-only exclusion
- **GIVEN** the system processes a reduce-only order
- **WHEN** the order is excluded from frequency count
- **THEN** the system logs with INFO level:
  - Order details
  - Confirmation that it's reduce-only
  - Current limit status
- **AND** log message format: "Reduce-only order BTC-USDT-SWAP sell 0.1 allowed despite limit (5/5 orders this week, excluded from count)"

### Requirement: Weekly Limit Calculation
The system SHALL calculate the weekly order count based on UTC timezone with Monday as week start.

#### Scenario: Count orders in current week
- **GIVEN** current UTC time is Wednesday 2025-12-03 15:30:00
- **WHEN** the system calculates current week_start
- **THEN** week_start = "2025-12-01" (Monday 00:00:00 UTC)
- **WHEN** the system queries order_history for current week count
- **THEN** the query filters: `week_start = '2025-12-01' AND status = 'placed' AND reduce_only = 0`
- **AND** returns accurate count of orders placed since Monday 00:00:00 UTC

#### Scenario: Handle timezone edge cases
- **GIVEN** user is in timezone UTC+8 (Beijing)
- **AND** local time is Monday 07:00:00 (UTC Sunday 23:00:00)
- **WHEN** user attempts to place an order
- **THEN** the system uses UTC time for week calculation
- **AND** week_start = previous Monday (not current local Monday)
- **AND** order counts toward previous week's limit
- **AND** logs include UTC timestamp for clarity

#### Scenario: Handle leap seconds and DST
- **GIVEN** system time handling may encounter leap seconds or DST transitions
- **WHEN** calculating week boundaries
- **THEN** the system uses UTC (which has no DST)
- **AND** uses standard time.Time operations in Go (which handle leap seconds)
- **AND** week_start calculation remains consistent across time edge cases

### Requirement: Order Status Tracking
The system SHALL track order status changes and update order_history accordingly.

#### Scenario: Update order status to filled
- **GIVEN** an order exists in order_history with status='placed'
- **WHEN** the order is filled (detected via OKX order query or webhook)
- **THEN** the system updates the order_history record
- **AND** sets status='filled'
- **AND** logs "Order status updated: order_id=123, status=filled" with INFO level

#### Scenario: Update order status to canceled
- **GIVEN** an order exists in order_history with status='placed'
- **WHEN** the order is canceled by user or system
- **THEN** the system updates the order_history record
- **AND** sets status='canceled'
- **AND** canceled orders still count toward weekly limit (placement is what matters)
- **AND** logs "Order status updated: order_id=123, status=canceled" with INFO level

#### Scenario: Failed order not counted
- **GIVEN** an order placement attempt fails immediately at OKX API
- **WHEN** OKX returns error before order is created
- **THEN** the system does NOT insert into order_history
- **AND** the order does NOT count toward weekly limit
- **AND** logs "Order placement failed: [error details], not recorded in history" with ERROR level

### Requirement: Manual Override and Bypass
The system SHALL provide configuration option to bypass frequency limit for emergency trading.

#### Scenario: Bypass frequency limit via config
- **GIVEN** config.yaml has order_control.frequency_limit.enabled = false
- **WHEN** the user attempts to place any order
- **THEN** the system skips frequency validation
- **AND** does NOT query order_history for count
- **AND** allows order to proceed regardless of count
- **AND** still records order in order_history for auditing
- **AND** logs "Frequency limit bypassed (disabled in config)" with INFO level

#### Scenario: Emergency override flag (future enhancement)
- **GIVEN** this is a placeholder for future enhancement
- **WHEN** this feature is not implemented in this proposal
- **THEN** manual override must be done via config.yaml restart
- **AND** no runtime override API is available in this version

### Requirement: Database Recovery from OKX History
The system SHALL support backfilling order_history from OKX API if database is lost.

#### Scenario: Detect missing order history
- **GIVEN** the system starts and order_history table is empty
- **WHEN** the system performs initial health check
- **THEN** the system logs "Order history table is empty, consider backfilling from OKX" with WARN level
- **AND** continues normal operation (does not automatically backfill)

#### Scenario: Manual backfill trigger (future enhancement)
- **GIVEN** this is a placeholder for future CLI command
- **WHEN** user runs `./tenyojubaku backfill-orders --days=7`
- **THEN** the system queries OKX order history for last 7 days
- **AND** inserts historical orders into order_history table
- **AND** calculates correct week_start for each order
- **AND** logs "Backfilled N orders from OKX history" with INFO level
- **NOTE**: This feature is NOT implemented in this proposal, only the database schema supports it
