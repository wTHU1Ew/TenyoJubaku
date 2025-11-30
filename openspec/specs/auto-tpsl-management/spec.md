# auto-tpsl-management Specification

## Purpose
TBD - created by archiving change add-auto-tpsl-management. Update Purpose after archive.
## Requirements
### Requirement: TPSL Configuration Management
The system SHALL load TPSL configuration from the YAML config file with validation and sensible defaults.

#### Scenario: Valid TPSL configuration
- **GIVEN** a valid config.yaml with TPSL section
- **WHEN** the system starts
- **THEN** the system loads TPSL enabled flag (default: true)
- **AND** loads check interval in seconds (default: 300)
- **AND** loads volatility percentage (default: 0.01)
- **AND** loads profit-loss ratio (default: 5.0)
- **AND** logs "TPSL configuration loaded successfully" with INFO level

#### Scenario: Missing TPSL configuration
- **GIVEN** config.yaml exists but has no TPSL section
- **WHEN** the system starts
- **THEN** the system uses default TPSL configuration values
- **AND** logs "Using default TPSL configuration" with INFO level
- **AND** TPSL management is enabled with default settings

#### Scenario: Invalid volatility percentage
- **GIVEN** config.yaml has volatility_pct <= 0 or > 1.0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid volatility_pct, must be between 0 and 1" with ERROR level
- **AND** exits with status code 1

#### Scenario: Invalid profit-loss ratio
- **GIVEN** config.yaml has profit_loss_ratio <= 0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid profit_loss_ratio, must be positive" with ERROR level
- **AND** exits with status code 1

#### Scenario: TPSL disabled in config
- **GIVEN** config.yaml has tpsl.enabled = false
- **WHEN** the system starts
- **THEN** the system does not start TPSL scheduler
- **AND** logs "TPSL management disabled in configuration" with INFO level

### Requirement: TPSL Scheduler Lifecycle
The system SHALL run a separate TPSL scheduler that periodically checks positions and places necessary TPSL orders.

#### Scenario: TPSL scheduler startup
- **GIVEN** TPSL is enabled in configuration
- **WHEN** the system starts
- **THEN** the TPSL scheduler starts in a separate goroutine
- **AND** performs an initial health check cycle
- **AND** logs "TPSL scheduler started with interval 300s" with INFO level

#### Scenario: Periodic TPSL checks
- **GIVEN** TPSL scheduler is running
- **WHEN** the check interval elapses (e.g., 300 seconds)
- **THEN** the scheduler triggers a TPSL analysis cycle
- **AND** logs "Starting TPSL check cycle" with DEBUG level
- **AND** logs "TPSL check cycle completed" with INFO level when done

#### Scenario: Graceful TPSL scheduler shutdown
- **GIVEN** the TPSL scheduler is running
- **WHEN** the system receives SIGINT or SIGTERM signal
- **THEN** the scheduler completes the current check cycle (if any)
- **AND** cancels the context to stop future cycles
- **AND** waits for goroutine to exit cleanly
- **AND** logs "TPSL scheduler stopped" with INFO level
- **AND** exits with status code 0

#### Scenario: TPSL scheduler error handling
- **GIVEN** an error occurs during a TPSL check cycle
- **WHEN** the error is recoverable (e.g., API rate limit)
- **THEN** the scheduler logs the error with WARN level
- **AND** continues running and waits for next interval
- **AND** does not crash or exit

### Requirement: Position Coverage Analysis
The system SHALL analyze each open position to determine if it has adequate TPSL coverage by querying pending algo orders.

#### Scenario: Fetch pending algo orders from OKX
- **GIVEN** TPSL check cycle starts
- **WHEN** the system queries OKX API for pending algo orders
- **THEN** the system calls GET /api/v5/trade/orders-algo-pending with ordType=conditional
- **AND** includes proper authentication headers
- **AND** parses the response to extract algo order details (algoId, instId, posSide, sz, state)

#### Scenario: Position with no TPSL coverage
- **GIVEN** an open position exists (e.g., BTC-USDT-SWAP long, size 1.5)
- **WHEN** the system analyzes this position
- **AND** no pending conditional algo orders exist for this instId and posSide
- **THEN** the system identifies this position as uncovered
- **AND** uncovered size = 1.5 (full position size)
- **AND** logs "Position BTC-USDT-SWAP (long) has no TPSL coverage, size: 1.5" with INFO level

#### Scenario: Position with full TPSL coverage
- **GIVEN** an open position exists (BTC-USDT-SWAP long, size 1.0)
- **WHEN** the system analyzes this position
- **AND** a pending conditional algo order exists for the same instId and posSide with sz = 1.0
- **THEN** the system identifies this position as fully covered
- **AND** uncovered size = 0
- **AND** does not place any new TPSL orders
- **AND** logs "Position BTC-USDT-SWAP (long) fully covered by TPSL" with DEBUG level

#### Scenario: Position with partial TPSL coverage
- **GIVEN** an open position exists (BTC-USDT-SWAP long, size 2.0)
- **WHEN** the system analyzes this position
- **AND** a pending conditional algo order exists for the same instId and posSide with sz = 0.8
- **THEN** the system calculates uncovered size = 2.0 - 0.8 = 1.2
- **AND** logs "Position BTC-USDT-SWAP (long) partially covered, uncovered size: 1.2" with INFO level

#### Scenario: Multiple algo orders for same position
- **GIVEN** an open position exists (BTC-USDT-SWAP long, size 3.0)
- **WHEN** the system analyzes this position
- **AND** two pending conditional algo orders exist: sz = 1.0 and sz = 0.5
- **THEN** the system sums the algo order sizes: 1.0 + 0.5 = 1.5
- **AND** uncovered size = 3.0 - 1.5 = 1.5

#### Scenario: Only live algo orders count toward coverage
- **GIVEN** an open position exists (BTC-USDT-SWAP long, size 2.0)
- **WHEN** the system queries algo orders
- **AND** one algo order has state = "live" with sz = 0.5
- **AND** another algo order has state = "canceled" with sz = 1.0
- **THEN** the system only counts the "live" order toward coverage
- **AND** uncovered size = 2.0 - 0.5 = 1.5

### Requirement: TPSL Price Calculation
The system SHALL calculate stop-loss and take-profit trigger prices based on position entry price, leverage, configured volatility percentage, and profit-loss ratio.

#### Scenario: Calculate stop-loss for long position
- **GIVEN** a long position with entry price = 40000, leverage = 5, volatility_pct = 0.01
- **WHEN** the system calculates stop-loss price
- **THEN** SL_distance = 40000 × 0.01 × 5 = 2000
- **AND** SL_price = 40000 - 2000 = 38000
- **AND** the system logs "Calculated SL for long position: entry=40000, leverage=5, SL=38000" with DEBUG level

#### Scenario: Calculate take-profit for long position
- **GIVEN** a long position with entry price = 40000, leverage = 5, volatility_pct = 0.01, pl_ratio = 5.0
- **WHEN** the system calculates take-profit price
- **THEN** SL_distance = 40000 × 0.01 × 5 = 2000
- **AND** TP_distance = 2000 × 5.0 = 10000
- **AND** TP_price = 40000 + 10000 = 50000
- **AND** the system logs "Calculated TP for long position: entry=40000, leverage=5, TP=50000" with DEBUG level

#### Scenario: Calculate stop-loss for short position
- **GIVEN** a short position with entry price = 40000, leverage = 10, volatility_pct = 0.01
- **WHEN** the system calculates stop-loss price
- **THEN** SL_distance = 40000 × 0.01 × 10 = 4000
- **AND** SL_price = 40000 + 4000 = 44000
- **AND** the system logs "Calculated SL for short position: entry=40000, leverage=10, SL=44000" with DEBUG level

#### Scenario: Calculate take-profit for short position
- **GIVEN** a short position with entry price = 40000, leverage = 10, volatility_pct = 0.01, pl_ratio = 5.0
- **WHEN** the system calculates take-profit price
- **THEN** SL_distance = 40000 × 0.01 × 10 = 4000
- **AND** TP_distance = 4000 × 5.0 = 20000
- **AND** TP_price = 40000 - 20000 = 20000
- **AND** the system logs "Calculated TP for short position: entry=40000, leverage=10, TP=20000" with DEBUG level

#### Scenario: Handle net position side
- **GIVEN** a position with posSide = "net" (one-way mode) and side = "buy"
- **WHEN** the system calculates TPSL prices
- **THEN** the system treats it as a long position (SL below entry, TP above entry)

#### Scenario: Validate calculated prices are reasonable
- **GIVEN** calculated SL_price or TP_price is negative or zero
- **WHEN** the system validates the calculation
- **THEN** the system logs "Invalid TPSL calculation result" with ERROR level
- **AND** does not place the algo order
- **AND** continues to next position

### Requirement: Automatic TPSL Order Placement
The system SHALL place conditional algo orders via OKX API to add TPSL protection for uncovered positions.

#### Scenario: Place TPSL order for uncovered position
- **GIVEN** a long position BTC-USDT-SWAP with uncovered size = 1.2
- **AND** calculated SL_price = 38000, TP_price = 50000
- **WHEN** the system places a TPSL order
- **THEN** the system calls POST /api/v5/trade/order-algo with:
  - instId = "BTC-USDT-SWAP"
  - tdMode = position's margin mode (e.g., "cross")
  - side = opposite of position ("sell" for long position)
  - posSide = "long"
  - ordType = "conditional"
  - sz = "1.2"
  - tpTriggerPx = "50000"
  - tpOrdPx = "-1" (market order)
  - slTriggerPx = "38000"
  - slOrdPx = "-1" (market order)
  - reduceOnly = true
- **AND** includes proper authentication headers

#### Scenario: Successful TPSL order placement
- **GIVEN** the system places a TPSL algo order
- **WHEN** OKX API returns code = "0" (success)
- **THEN** the system logs "TPSL order placed successfully for BTC-USDT-SWAP (long), algoId: 123456" with INFO level
- **AND** continues to next position

#### Scenario: TPSL order placement rate limit
- **GIVEN** the system places a TPSL algo order
- **WHEN** OKX API returns HTTP 429 (rate limit exceeded)
- **THEN** the system logs "Rate limit exceeded while placing TPSL, will retry next cycle" with WARN level
- **AND** does not retry immediately
- **AND** continues to next position
- **AND** will retry in the next check cycle

#### Scenario: TPSL order placement failure
- **GIVEN** the system places a TPSL algo order
- **WHEN** OKX API returns an error (code != "0" or HTTP error)
- **THEN** the system logs the error details with ERROR level
- **AND** logs position details (instId, posSide, size, TP, SL)
- **AND** continues to next position without crashing

#### Scenario: API authentication failure during placement
- **GIVEN** the system attempts to place a TPSL order
- **WHEN** OKX API returns authentication error (code 50111 or 50113)
- **THEN** the system logs "TPSL order placement authentication failed" with ERROR level
- **AND** stops TPSL scheduler to avoid repeated failures
- **AND** logs "Stopping TPSL scheduler due to authentication failure" with ERROR level

#### Scenario: Network error during TPSL placement
- **GIVEN** the system places a TPSL algo order
- **WHEN** a network error occurs (timeout, connection refused)
- **THEN** the system retries up to 3 times with exponential backoff (1s, 2s, 4s)
- **AND** logs each retry attempt with WARN level
- **AND** logs final failure with ERROR level if all retries fail
- **AND** continues to next position

### Requirement: TPSL Operation Logging and Auditing
The system SHALL log all TPSL analysis and placement operations with sufficient detail for auditing and troubleshooting.

#### Scenario: Log TPSL check cycle summary
- **GIVEN** a TPSL check cycle completes
- **WHEN** the cycle has processed all positions
- **THEN** the system logs a summary with INFO level including:
  - Total positions checked
  - Positions with no coverage
  - Positions with partial coverage
  - Positions with full coverage
  - Number of TPSL orders placed
  - Number of placement failures

#### Scenario: Log individual position analysis
- **GIVEN** the system analyzes a position
- **WHEN** analysis completes
- **THEN** the system logs position details with DEBUG level:
  - Instrument ID
  - Position side
  - Position size
  - Covered size
  - Uncovered size
  - Action taken (place order, skip, error)

#### Scenario: Log TPSL calculation details
- **GIVEN** the system calculates TPSL prices
- **WHEN** calculation completes
- **THEN** the system logs with DEBUG level:
  - Entry price
  - Leverage
  - Volatility percentage
  - Profit-loss ratio
  - Calculated SL price
  - Calculated TP price

#### Scenario: Log API request details
- **GIVEN** the system makes an API request (get algo orders or place order)
- **WHEN** the request completes
- **THEN** the system logs with DEBUG level:
  - Endpoint called
  - HTTP status code
  - Response time
  - Success/failure status
- **AND** masks sensitive data (API keys, secrets) in logs

### Requirement: OKX Algo Order API Integration
The system SHALL integrate with OKX algo order APIs to query pending orders and place conditional TPSL orders.

#### Scenario: Query pending algo orders
- **GIVEN** TPSL check cycle starts
- **WHEN** the system queries pending algo orders
- **THEN** the system calls GET /api/v5/trade/orders-algo-pending?ordType=conditional
- **AND** includes authentication headers (OK-ACCESS-KEY, OK-ACCESS-SIGN, OK-ACCESS-TIMESTAMP, OK-ACCESS-PASSPHRASE)
- **AND** parses JSON response into PendingAlgoOrdersResponse struct

#### Scenario: Parse algo order response
- **GIVEN** OKX API returns algo orders response
- **WHEN** the system parses the response
- **THEN** the system extracts for each algo order:
  - algoId (unique identifier)
  - instId (instrument)
  - posSide (position side)
  - side (order side)
  - sz (order size)
  - ordType (order type)
  - state (order state: live, canceled, effective, etc.)
  - tpTriggerPx (take-profit trigger price)
  - slTriggerPx (stop-loss trigger price)
- **AND** filters orders with ordType = "conditional" and state = "live"

#### Scenario: Place conditional algo order
- **GIVEN** the system needs to place a TPSL order
- **WHEN** the system calls OKX place algo order API
- **THEN** the system sends POST /api/v5/trade/order-algo
- **AND** request body is JSON with AlgoOrderRequest structure
- **AND** includes authentication headers
- **AND** parses response to extract algoId on success

#### Scenario: Handle algo order API error codes
- **GIVEN** OKX API returns an error code
- **WHEN** code = "51000" (parameter error)
- **THEN** logs "Invalid TPSL order parameters" with ERROR level
- **WHEN** code = "51008" (insufficient balance)
- **THEN** logs "Insufficient balance for TPSL order" with WARN level
- **WHEN** code = "51020" (order amount too small)
- **THEN** logs "TPSL order size below minimum" with WARN level

### Requirement: Margin Mode Compatibility
The system SHALL correctly handle different margin modes (cross, isolated) when placing TPSL orders.

#### Scenario: TPSL for cross margin position
- **GIVEN** a position with margin mode = "cross"
- **WHEN** the system places a TPSL order
- **THEN** the algo order request has tdMode = "cross"

#### Scenario: TPSL for isolated margin position
- **GIVEN** a position with margin mode = "isolated"
- **WHEN** the system places a TPSL order
- **THEN** the algo order request has tdMode = "isolated"

#### Scenario: Determine margin mode from position data
- **GIVEN** the position data from monitoring includes margin mode field
- **WHEN** the system prepares TPSL order request
- **THEN** the system uses the position's margin mode in tdMode parameter

### Requirement: Position Side Handling
The system SHALL correctly handle different position side configurations (long/short mode vs net mode).

#### Scenario: TPSL for long position in hedge mode
- **GIVEN** a position with posSide = "long"
- **WHEN** the system places a TPSL order
- **THEN** algo order has posSide = "long" and side = "sell" (close long)

#### Scenario: TPSL for short position in hedge mode
- **GIVEN** a position with posSide = "short"
- **WHEN** the system places a TPSL order
- **THEN** algo order has posSide = "short" and side = "buy" (close short)

#### Scenario: TPSL for net position in one-way mode
- **GIVEN** a position with posSide = "net"
- **WHEN** the system determines position direction from position size (positive = long, negative = short)
- **THEN** algo order has posSide = "net" and appropriate side

### Requirement: Error Recovery and Resilience
The system SHALL handle errors gracefully and continue TPSL operations without crashing.

#### Scenario: Continue after individual position failure
- **GIVEN** the system is processing 5 positions
- **WHEN** TPSL placement fails for position 2
- **THEN** the system logs the error
- **AND** continues processing positions 3, 4, and 5
- **AND** does not crash or stop the scheduler

#### Scenario: Handle empty positions list
- **GIVEN** the monitoring system has no open positions
- **WHEN** TPSL check cycle runs
- **THEN** the system logs "No open positions, skipping TPSL check" with INFO level
- **AND** completes the cycle normally
- **AND** waits for next interval

#### Scenario: Handle empty algo orders response
- **GIVEN** the system queries pending algo orders
- **WHEN** OKX API returns empty data array
- **THEN** the system treats all positions as uncovered
- **AND** proceeds with TPSL placement for all positions

#### Scenario: Retry after transient storage error
- **GIVEN** the system queries positions from storage
- **WHEN** a transient database error occurs
- **THEN** the system retries up to 3 times with backoff
- **AND** logs retry attempts
- **AND** if all retries fail, skips this check cycle and waits for next interval

