# order-confirmation Specification

## Purpose
Implement multi-confirmation system for pending orders with timeout-based automatic size reduction to prevent forgotten orders from executing at unintended sizes.

## ADDED Requirements

### Requirement: Confirmation Configuration
The system SHALL load order confirmation configuration from the YAML config file with validation.

#### Scenario: Valid confirmation configuration
- **GIVEN** a valid config.yaml with order_control.confirmation section
- **WHEN** the system starts
- **THEN** the system loads enabled flag (default: true)
- **AND** loads check_interval_seconds (default: 300 for 5 minutes)
- **AND** loads confirmation_interval_hours (default: 12)
- **AND** loads waiting_period_hours (default: 4)
- **AND** loads timeout_size_reduction_pct (default: 0.5 for 50%)
- **AND** loads max_timeouts (default: 3)
- **AND** loads notification_method (default: "log")
- **AND** logs "Order confirmation configuration loaded: interval=12h, waiting_period=4h, size_reduction=50%, max_timeouts=3" with INFO level

#### Scenario: Missing confirmation configuration
- **GIVEN** config.yaml exists but has no order_control.confirmation section
- **WHEN** the system starts
- **THEN** the system uses default confirmation configuration values
- **AND** logs "Using default order confirmation configuration" with INFO level

#### Scenario: Invalid confirmation interval
- **GIVEN** config.yaml has confirmation_interval_hours <= 0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid confirmation_interval_hours, must be positive" with ERROR level
- **AND** exits with status code 1

#### Scenario: Invalid waiting period
- **GIVEN** config.yaml has waiting_period_hours <= 0 OR > confirmation_interval_hours
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid waiting_period_hours, must be positive and less than confirmation_interval" with ERROR level
- **AND** exits with status code 1

#### Scenario: Invalid size reduction percentage
- **GIVEN** config.yaml has timeout_size_reduction_pct <= 0 OR > 1.0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid timeout_size_reduction_pct, must be between 0 and 1" with ERROR level
- **AND** exits with status code 1

#### Scenario: Confirmation disabled
- **GIVEN** config.yaml has order_control.confirmation.enabled = false
- **WHEN** the system starts
- **THEN** the system logs "Order confirmation system disabled in configuration" with INFO level
- **AND** does not start confirmation scheduler

### Requirement: Pending Confirmations Database Schema
The system SHALL create and maintain the pending_confirmations table for tracking order confirmation state.

#### Scenario: Database initialization with pending_confirmations table
- **GIVEN** the system starts for the first time or database is missing pending_confirmations table
- **WHEN** the database initialization runs
- **THEN** the system creates the pending_confirmations table with columns:
  - id (INTEGER PRIMARY KEY AUTOINCREMENT)
  - order_id (TEXT NOT NULL UNIQUE)
  - inst_id (TEXT NOT NULL)
  - side (TEXT NOT NULL)
  - ord_type (TEXT NOT NULL)
  - original_size (TEXT NOT NULL)
  - current_size (TEXT NOT NULL)
  - price (TEXT, nullable)
  - placed_at (DATETIME NOT NULL)
  - last_confirmation_at (DATETIME, nullable)
  - next_confirmation_due (DATETIME NOT NULL)
  - confirmation_count (INTEGER NOT NULL DEFAULT 0)
  - timeout_count (INTEGER NOT NULL DEFAULT 0)
  - status (TEXT NOT NULL)
  - created_at (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)
  - updated_at (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)
- **AND** creates index idx_pending_conf_status on status
- **AND** creates index idx_pending_conf_next_due on next_confirmation_due
- **AND** creates index idx_pending_conf_order_id on order_id
- **AND** logs "Pending confirmations table initialized successfully" with INFO level

#### Scenario: Record new pending order
- **GIVEN** an order is successfully placed via OKX API
- **AND** the order is a limit order (pending, not immediately filled)
- **WHEN** the system receives order confirmation
- **THEN** the system inserts a record into pending_confirmations table
- **AND** order_id = OKX order ID
- **AND** original_size = current_size = order size
- **AND** placed_at = current UTC timestamp
- **AND** next_confirmation_due = placed_at + confirmation_interval_hours
- **AND** confirmation_count = 0
- **AND** timeout_count = 0
- **AND** status = 'pending'
- **AND** logs "Pending order added to confirmation tracking: order_id=123, next_due=2025-12-01T12:00:00Z" with INFO level

#### Scenario: Update existing pending order
- **GIVEN** a pending confirmation record exists with order_id = "123"
- **WHEN** the system updates the record (e.g., after timeout)
- **THEN** the system uses UPDATE statement
- **AND** updates current_size, timeout_count, next_confirmation_due as needed
- **AND** updates updated_at = current UTC timestamp
- **AND** logs "Pending confirmation updated: order_id=123" with DEBUG level

#### Scenario: Remove completed order from tracking
- **GIVEN** a pending confirmation record exists with order_id = "123"
- **WHEN** the order is filled or canceled (detected via OKX order query)
- **THEN** the system updates status = 'filled' or 'canceled'
- **AND** logs "Order removed from confirmation tracking: order_id=123, reason=filled" with INFO level
- **AND** the record remains in database for audit history

### Requirement: Confirmation Scheduler Lifecycle
The system SHALL run a confirmation scheduler that periodically checks pending orders and triggers notifications/timeouts.

#### Scenario: Confirmation scheduler startup
- **GIVEN** order confirmation is enabled in configuration
- **WHEN** the system starts
- **THEN** the confirmation scheduler starts in a separate goroutine
- **AND** performs an initial health check cycle
- **AND** logs "Confirmation scheduler started with check interval 300s" with INFO level

#### Scenario: Periodic confirmation checks
- **GIVEN** confirmation scheduler is running
- **WHEN** the check interval elapses (e.g., 300 seconds)
- **THEN** the scheduler triggers a confirmation check cycle
- **AND** logs "Starting confirmation check cycle" with DEBUG level
- **AND** queries pending_confirmations for orders needing attention
- **AND** logs "Confirmation check cycle completed: processed N orders" with INFO level

#### Scenario: Graceful scheduler shutdown
- **GIVEN** the confirmation scheduler is running
- **WHEN** the system receives SIGINT or SIGTERM signal
- **THEN** the scheduler completes the current check cycle (if any)
- **AND** cancels the context to stop future cycles
- **AND** waits for goroutine to exit cleanly
- **AND** logs "Confirmation scheduler stopped" with INFO level
- **AND** exits with status code 0

#### Scenario: Scheduler error handling
- **GIVEN** an error occurs during a confirmation check cycle
- **WHEN** the error is recoverable (e.g., database timeout)
- **THEN** the scheduler logs the error with WARN level
- **AND** continues running and waits for next interval
- **AND** does not crash or exit

### Requirement: Confirmation Notification Trigger
The system SHALL trigger confirmation notifications when orders reach their confirmation due time without timeout.

#### Scenario: Trigger notification when due
- **GIVEN** a pending confirmation exists with next_confirmation_due = "2025-12-01T12:00:00Z"
- **AND** current UTC time = "2025-12-01T12:05:00Z" (5 minutes past due)
- **AND** last_confirmation_at is NULL (first notification) OR more than waiting_period ago
- **WHEN** the confirmation scheduler runs
- **THEN** the system triggers a confirmation notification
- **AND** logs "CONFIRMATION REQUIRED: Order BTC-USDT-SWAP buy @ 40000, size: 1.0, placed: 2025-12-01T00:00:00Z, order_id: 123" with WARN level
- **AND** updates last_confirmation_at = current UTC timestamp
- **AND** updates next_confirmation_due = current time + confirmation_interval_hours
- **AND** increments confirmation_count

#### Scenario: Log-based notification
- **GIVEN** notification_method = "log"
- **WHEN** a confirmation is triggered
- **THEN** the system writes a log entry with WARN level
- **AND** log includes order details: instrument, side, price, size, order_id, time since placement
- **AND** log message format: "⚠️  CONFIRMATION REQUIRED: Order [order_id: 123] BTC-USDT-SWAP buy @ 40000, size: 1.0, placed: 12 hours ago. Please confirm or order will be reduced to 50% in 4 hours."

#### Scenario: Multiple pending confirmations in one cycle
- **GIVEN** 3 pending confirmations are all due within the same check cycle
- **WHEN** the confirmation scheduler processes them
- **THEN** the system triggers notifications for all 3 orders
- **AND** logs each notification separately
- **AND** updates all 3 records in database
- **AND** logs summary: "Confirmation check cycle completed: 3 notifications sent, 0 timeouts processed" with INFO level

### Requirement: Timeout Detection and Size Reduction
The system SHALL detect confirmation timeouts and automatically reduce order size via OKX amend API.

#### Scenario: Detect timeout after waiting period
- **GIVEN** a pending confirmation exists with:
  - last_confirmation_at = "2025-12-01T12:00:00Z"
  - waiting_period_hours = 4
- **AND** current UTC time = "2025-12-01T16:05:00Z" (4 hours 5 minutes since notification)
- **WHEN** the confirmation scheduler runs
- **THEN** the system detects timeout occurred
- **AND** logs "Timeout detected for order_id=123 (4.08 hours since notification)" with WARN level

#### Scenario: Reduce order size on timeout
- **GIVEN** timeout is detected for order_id = "123"
- **AND** current_size = "1.0"
- **AND** timeout_size_reduction_pct = 0.5
- **WHEN** the system processes the timeout
- **THEN** the system calculates new_size = 1.0 * 0.5 = 0.5
- **AND** calls OKX amend order API: POST /api/v5/trade/amend-order
- **AND** request body includes: {"instId": "BTC-USDT-SWAP", "ordId": "123", "newSz": "0.5"}
- **AND** includes proper authentication headers

#### Scenario: Successful order amendment
- **GIVEN** OKX amend order API is called to reduce size
- **WHEN** OKX returns success response: {"code":"0","msg":"","data":[{"ordId":"123","reqId":""}]}
- **THEN** the system updates pending_confirmations record
- **AND** sets current_size = "0.5"
- **AND** increments timeout_count
- **AND** sets next_confirmation_due = current time + confirmation_interval_hours
- **AND** sets last_confirmation_at = NULL (reset for next cycle)
- **AND** logs "Order size reduced on timeout: order_id=123, 1.0 → 0.5 (timeout 1/3)" with WARN level

#### Scenario: Order amendment failure
- **GIVEN** OKX amend order API is called to reduce size
- **WHEN** OKX returns error response: {"code":"51400","msg":"Order does not exist"}
- **THEN** the system logs "Failed to amend order_id=123: Order does not exist" with ERROR level
- **AND** updates status = 'failed'
- **AND** removes from active confirmation tracking
- **AND** logs "Order removed from confirmation tracking due to amendment failure" with ERROR level

#### Scenario: Order amendment network error
- **GIVEN** OKX amend order API call encounters network error
- **WHEN** the request fails (timeout, connection error)
- **THEN** the system retries up to 3 times with exponential backoff
- **AND** logs each retry attempt with WARN level
- **AND** if all retries fail, logs "Failed to amend order_id=123 after 3 retries" with ERROR level
- **AND** updates status = 'failed'
- **AND** continues to next pending order

#### Scenario: Calculate reduced size with precision
- **GIVEN** current_size = "1.23456789"
- **AND** timeout_size_reduction_pct = 0.5
- **WHEN** the system calculates new size
- **THEN** new_size = 1.23456789 * 0.5 = 0.61728395 (rounded to exchange precision)
- **AND** uses appropriate decimal precision for the instrument
- **AND** ensures new_size meets minimum order size for the instrument

#### Scenario: New size below minimum order size
- **GIVEN** current_size = "0.0002" (very small)
- **AND** timeout_size_reduction_pct = 0.5
- **AND** minimum order size for BTC-USDT-SWAP = 0.0001
- **WHEN** the system calculates new_size = 0.0001
- **AND** new_size equals minimum order size
- **THEN** the system proceeds with amendment to minimum size
- **AND** logs "Order size reduced to minimum: 0.0001" with WARN level
- **GIVEN** new_size would be below minimum
- **THEN** the system cancels the order instead
- **AND** logs "Order canceled: size reduction would go below minimum" with WARN level

### Requirement: Order Cancellation After Max Timeouts
The system SHALL cancel orders that exceed the maximum number of timeouts.

#### Scenario: Cancel order after max timeouts
- **GIVEN** a pending confirmation exists with timeout_count = 3
- **AND** max_timeouts = 3
- **WHEN** the system processes this order
- **THEN** the system logs "Order reached max timeouts (3/3), canceling order_id=123" with WARN level
- **AND** calls OKX cancel order API: POST /api/v5/trade/cancel-order
- **AND** request body includes: {"instId": "BTC-USDT-SWAP", "ordId": "123"}
- **AND** includes proper authentication headers

#### Scenario: Successful order cancellation
- **GIVEN** OKX cancel order API is called
- **WHEN** OKX returns success response: {"code":"0","msg":"","data":[{"ordId":"123","sCode":"0"}]}
- **THEN** the system updates pending_confirmations record
- **AND** sets status = 'canceled'
- **AND** logs "Order canceled successfully: order_id=123 (max timeouts reached)" with INFO level
- **AND** removes from active confirmation tracking

#### Scenario: Order cancellation failure
- **GIVEN** OKX cancel order API is called
- **WHEN** OKX returns error response: {"code":"51400","msg":"Order does not exist"}
- **THEN** the system logs "Order cancellation failed for order_id=123: Order does not exist" with WARN level
- **AND** updates status = 'failed'
- **AND** removes from active confirmation tracking

#### Scenario: Order already filled before cancellation
- **GIVEN** order reaches max timeouts and cancellation is attempted
- **WHEN** OKX returns error: "Order already filled"
- **THEN** the system logs "Order order_id=123 already filled, removing from tracking" with INFO level
- **AND** updates status = 'filled'
- **AND** removes from active confirmation tracking

### Requirement: Order Status Synchronization
The system SHALL periodically synchronize order status with OKX to detect filled/canceled orders.

#### Scenario: Detect filled order during sync
- **GIVEN** a pending confirmation exists with order_id = "123"
- **WHEN** the scheduler queries OKX order status
- **AND** OKX returns state = "filled"
- **THEN** the system updates pending_confirmations record
- **AND** sets status = 'filled'
- **AND** logs "Order filled: order_id=123, removing from confirmation tracking" with INFO level
- **AND** stops further confirmation/timeout processing for this order

#### Scenario: Detect canceled order during sync
- **GIVEN** a pending confirmation exists with order_id = "123"
- **WHEN** the scheduler queries OKX order status
- **AND** OKX returns state = "canceled"
- **THEN** the system updates pending_confirmations record
- **AND** sets status = 'canceled'
- **AND** logs "Order canceled externally: order_id=123, removing from tracking" with INFO level

#### Scenario: Order status query failure
- **GIVEN** the scheduler attempts to query OKX order status
- **WHEN** the API call fails (network error, rate limit)
- **THEN** the system logs error with WARN level
- **AND** continues with existing pending confirmation record
- **AND** will retry in next check cycle

#### Scenario: Batch order status query
- **GIVEN** 10 pending confirmations exist
- **WHEN** the scheduler syncs order status
- **THEN** the system batches requests if possible (OKX API limits)
- **AND** queries order status for all pending orders
- **AND** updates status for all changed orders in a single database transaction

### Requirement: Confirmation Notification Format
The system SHALL format confirmation notifications with all necessary order details.

#### Scenario: Format confirmation notification message
- **GIVEN** a confirmation notification is triggered for order_id = "123"
- **AND** order details: BTC-USDT-SWAP, buy, price=40000, size=1.0, placed_at=2025-12-01T00:00:00Z
- **WHEN** the system formats the notification
- **THEN** the message includes:
  - Clear indicator: "CONFIRMATION REQUIRED"
  - Order ID: 123
  - Instrument: BTC-USDT-SWAP
  - Side: buy
  - Price: 40000
  - Size: 1.0
  - Time since placement: "12 hours ago"
  - Timeout warning: "Order will be reduced to 50% in 4 hours if not confirmed"
- **AND** message is human-readable and includes all context needed for decision

#### Scenario: Format timeout notification message
- **GIVEN** a timeout occurs and size is reduced
- **WHEN** the system logs the timeout
- **THEN** the message includes:
  - Order ID
  - Original size and new size
  - Timeout count (e.g., "1/3")
  - Next confirmation due time
  - Warning about cancellation if max timeouts reached
- **AND** message format: "Order size reduced on timeout: order_id=123, 1.0 → 0.5 (timeout 1/3), next confirmation due in 12 hours. Order will be canceled after 3 timeouts."

### Requirement: Logging and Auditing
The system SHALL log all confirmation events and timeout actions for complete audit trail.

#### Scenario: Log confirmation check cycle summary
- **GIVEN** a confirmation check cycle completes
- **WHEN** the cycle has processed all pending confirmations
- **THEN** the system logs a summary with INFO level:
  - Total pending confirmations checked
  - Notifications sent
  - Timeouts processed
  - Orders amended
  - Orders canceled
  - Errors encountered
- **AND** log message format: "Confirmation check cycle completed: checked=5, notifications=2, timeouts=1, amended=1, canceled=0, errors=0"

#### Scenario: Log individual confirmation notification
- **GIVEN** a confirmation notification is triggered
- **WHEN** the notification is sent
- **THEN** the system logs with WARN level (high visibility)
- **AND** log includes all order details
- **AND** log is written to both file and console (if console enabled)

#### Scenario: Log timeout processing
- **GIVEN** a timeout is processed and order is amended
- **WHEN** amendment succeeds
- **THEN** the system logs with WARN level:
  - Order ID
  - Size change (old → new)
  - Timeout count
  - Next steps
- **AND** log message is clear about what action was taken

#### Scenario: Log errors with context
- **GIVEN** an error occurs during confirmation processing
- **WHEN** the error is logged
- **THEN** the system logs with ERROR level
- **AND** includes full context: order_id, operation attempted, error details
- **AND** includes stack trace if critical error

### Requirement: Confirmation Count Tracking
The system SHALL track the number of confirmations sent for each order.

#### Scenario: Increment confirmation count on notification
- **GIVEN** a confirmation notification is triggered
- **WHEN** the notification is sent
- **THEN** the system increments confirmation_count
- **AND** updates the pending_confirmations record
- **AND** logs current confirmation count with each notification

#### Scenario: Track confirmation history
- **GIVEN** an order has received 5 confirmations
- **WHEN** viewing confirmation statistics
- **THEN** confirmation_count = 5
- **AND** provides audit trail of how many reminders were sent
- **AND** helps identify orders that require excessive confirmations

### Requirement: Performance and Scalability
The system SHALL handle confirmation checking efficiently even with many pending orders.

#### Scenario: Efficient query for due confirmations
- **GIVEN** pending_confirmations table contains 1000 records
- **AND** only 10 orders are due for confirmation in current cycle
- **WHEN** the scheduler queries for due confirmations
- **THEN** the query uses idx_pending_conf_next_due index
- **AND** query filters: `next_confirmation_due <= NOW() AND status = 'pending'`
- **AND** query execution time is < 10ms

#### Scenario: Limit confirmation processing per cycle
- **GIVEN** 500 pending confirmations are all due (edge case)
- **WHEN** the scheduler processes confirmations
- **THEN** the system processes maximum 100 confirmations per cycle
- **AND** prioritizes oldest confirmations first (ORDER BY next_confirmation_due ASC)
- **AND** logs "Processed 100/500 due confirmations, remaining will be handled in next cycle" with INFO level

#### Scenario: Database transaction for batch updates
- **GIVEN** 20 confirmations are processed in one cycle
- **WHEN** updating database records
- **THEN** the system uses a single database transaction for all updates
- **AND** commits transaction only after all updates succeed
- **AND** rolls back on any error to maintain consistency

### Requirement: Error Recovery
The system SHALL recover gracefully from errors during confirmation processing.

#### Scenario: Continue processing after individual failure
- **GIVEN** confirmation processing fails for order_id = "123" (e.g., network error)
- **WHEN** the scheduler is processing 10 pending confirmations
- **THEN** the system logs the error for order_id = "123"
- **AND** continues processing remaining 9 orders
- **AND** does not crash or stop scheduler
- **AND** will retry order_id = "123" in next cycle

#### Scenario: Handle database connection loss
- **GIVEN** database connection is lost during confirmation cycle
- **WHEN** the scheduler attempts to query pending_confirmations
- **THEN** the system detects connection error
- **AND** logs "Database connection lost during confirmation check" with ERROR level
- **AND** skips current cycle
- **AND** waits for next interval and retries
- **AND** database connection pool should auto-reconnect

#### Scenario: Handle OKX API rate limiting
- **GIVEN** multiple order amendments trigger OKX rate limit (429)
- **WHEN** rate limit is exceeded
- **THEN** the system backs off and waits
- **AND** logs "Rate limited during confirmation processing, will retry in next cycle" with WARN level
- **AND** marks affected orders for retry in next cycle
- **AND** does not lose track of pending confirmations
