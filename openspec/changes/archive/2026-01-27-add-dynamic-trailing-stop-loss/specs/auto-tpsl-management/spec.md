# auto-tpsl-management Spec Delta

## ADDED Requirements

### Requirement: Dynamic Trailing Stop-Loss Configuration
The system SHALL load dynamic trailing stop-loss configuration from the YAML config file with validation and defaults.

#### Scenario: Valid dynamic SL configuration
- **GIVEN** config.yaml has `tpsl.dynamic_sl` section with all parameters
- **WHEN** the system starts
- **THEN** the system loads `enabled` flag (default: false)
- **AND** loads `first_move_pct` (default: 0.01 for 1%)
- **AND** loads `trailing_step_pct` (default: 0.005 for 0.5%)
- **AND** loads `stop_move_step_pct` (default: 0.001 for 0.1%)
- **AND** logs "Dynamic SL configuration loaded successfully" with INFO level

#### Scenario: Dynamic SL disabled in config
- **GIVEN** config.yaml has `tpsl.dynamic_sl.enabled = false`
- **WHEN** the TPSL check cycle runs
- **THEN** the system skips all dynamic SL logic
- **AND** behavior is identical to static TPSL (V4.5)
- **AND** logs "Dynamic SL disabled" with DEBUG level

#### Scenario: Invalid dynamic SL parameters
- **GIVEN** config.yaml has `first_move_pct <= 0` or `>= 1.0`
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid dynamic_sl configuration" with ERROR level
- **AND** exits with status code 1

#### Scenario: Missing dynamic SL configuration
- **GIVEN** config.yaml has no `dynamic_sl` section under `tpsl`
- **WHEN** the system starts
- **THEN** the system uses default values with `enabled = false`
- **AND** logs "Using default dynamic SL configuration (disabled)" with INFO level

### Requirement: Position Profit Monitoring for Dynamic SL
The system SHALL monitor unrealized profit for each open position to determine when to adjust stop-loss levels.

#### Scenario: Track highest price reached for long position
- **GIVEN** a long position with entry price = 40000
- **WHEN** TPSL check cycle runs with current mark price = 40500
- **THEN** the system updates `highestPriceReached = 40500` for this position
- **AND** stores this in the dynamic SL tracker (in-memory)

#### Scenario: Track lowest price reached for short position
- **GIVEN** a short position with entry price = 40000
- **WHEN** TPSL check cycle runs with current mark price = 39500
- **THEN** the system updates `lowestPriceReached = 39500` for this position
- **AND** stores this in the dynamic SL tracker (in-memory)

#### Scenario: Update tracker only when price improves
- **GIVEN** a long position with `highestPriceReached = 41000`
- **WHEN** check cycle runs with current mark price = 40800
- **THEN** the system does NOT update `highestPriceReached`
- **AND** keeps `highestPriceReached = 41000`

#### Scenario: Initialize tracker on first cycle
- **GIVEN** a position exists but has no dynamic SL tracker in database yet
- **WHEN** TPSL check cycle runs with dynamic SL enabled
- **THEN** the system creates a new tracker for this position
- **AND** initializes `entryPrice` from position average price
- **AND** initializes `highestPriceReached = currentMarkPrice` (for long)
- **AND** sets `firstMoveTriggered = false`
- **AND** INSERTs the tracker into `dynamic_sl_tracking` table via GORM

### Requirement: FirstMove Threshold Detection
The system SHALL detect when a position reaches the configured profit threshold (firstMove) and trigger stop-loss adjustment to breakeven.

#### Scenario: FirstMove triggered for long position
- **GIVEN** a long position with entry = 40000, firstMove = 1%
- **WHEN** mark price reaches 40400 (1% profit)
- **THEN** the system detects firstMove threshold met
- **AND** sets `firstMoveTriggered = true` in tracker
- **AND** calculates new SL = 40000 * 1.001 = 40040 (breakeven + fee coverage)
- **AND** logs "FirstMove triggered for long position: moving SL to breakeven" with INFO level

#### Scenario: FirstMove triggered for short position
- **GIVEN** a short position with entry = 40000, firstMove = 1%
- **WHEN** mark price drops to 39600 (1% profit)
- **THEN** the system detects firstMove threshold met
- **AND** sets `firstMoveTriggered = true` in tracker
- **AND** calculates new SL = 40000 * 0.999 = 39960 (breakeven - fee coverage)
- **AND** logs "FirstMove triggered for short position: moving SL to breakeven" with INFO level

#### Scenario: FirstMove not triggered yet
- **GIVEN** a long position with entry = 40000, firstMove = 1%
- **WHEN** mark price is 40300 (0.75% profit, below threshold)
- **THEN** the system does NOT trigger firstMove
- **AND** `firstMoveTriggered` remains false
- **AND** no SL adjustment is made
- **AND** logs "Monitoring position profit: 0.75%, waiting for firstMove (1%)" with DEBUG level

#### Scenario: FirstMove triggers only once
- **GIVEN** a position with `firstMoveTriggered = true`
- **WHEN** mark price continues to increase
- **THEN** the system does NOT re-trigger firstMove logic
- **AND** proceeds to trailing SL logic instead

### Requirement: Trailing Stop-Loss Adjustment
The system SHALL trail the stop-loss upward (for longs) or downward (for shorts) as price continues to move favorably after firstMove is triggered.

#### Scenario: Trailing SL for long position
- **GIVEN** a long position with:
  - entry = 40000
  - firstMoveTriggered = true
  - currentSlPrice = 40040 (breakeven)
  - highestPriceReached = 40400
  - trailingStep = 0.5%, stopMoveStep = 0.1%
- **WHEN** mark price increases to 40604 (0.5% gain from highest)
- **THEN** the system calculates new SL = 40040 * 1.001 = 40080.04
- **AND** updates `highestPriceReached = 40604`
- **AND** logs "Trailing SL adjusted for long: SL 40040 → 40080" with INFO level

#### Scenario: Trailing SL for short position
- **GIVEN** a short position with:
  - entry = 40000
  - firstMoveTriggered = true
  - currentSlPrice = 39960 (breakeven)
  - lowestPriceReached = 39600
  - trailingStep = 0.5%, stopMoveStep = 0.1%
- **WHEN** mark price decreases to 39402 (0.5% gain from lowest)
- **THEN** the system calculates new SL = 39960 * 0.999 = 39920.04
- **AND** updates `lowestPriceReached = 39402`
- **AND** logs "Trailing SL adjusted for short: SL 39960 → 39920" with INFO level

#### Scenario: Trailing condition not met
- **GIVEN** a long position with:
  - highestPriceReached = 41000
  - trailingStep = 0.5%
- **WHEN** mark price increases to 41100 (0.24% gain, below 0.5% threshold)
- **THEN** the system does NOT adjust SL
- **AND** logs "Price gain 0.24% below trailing threshold (0.5%)" with DEBUG level

#### Scenario: Multiple trailing adjustments
- **GIVEN** a long position with SL trailing active
- **WHEN** price increases 2% in steps (0.6%, 0.7%, 0.7%)
- **THEN** the system makes 3 separate SL adjustments
- **AND** each adjustment moves SL up by 0.1%
- **AND** final SL reflects 3 × 0.1% = 0.3% total movement

### Requirement: Stop-Loss Order Amendment via OKX API
The system SHALL amend existing conditional algo orders to update stop-loss trigger prices when dynamic SL adjustment is needed.

#### Scenario: Amend SL order after firstMove
- **GIVEN** a long position with matching pending algo order (algoId = 123456)
- **WHEN** firstMove is triggered, new SL = 40040
- **THEN** the system calls `POST /api/v5/trade/amend-algos` with:
  - instId = position instrument ID
  - algoId = 123456
  - newSlTriggerPx = "40040"
- **AND** includes proper authentication headers
- **AND** logs "Amending algo order 123456: SL 38000 → 40040" with INFO level

#### Scenario: Successful SL amendment
- **GIVEN** the system calls amend algo order API
- **WHEN** OKX API returns code = "0" (success)
- **THEN** the system updates tracker's `currentSlPrice = 40040`
- **AND** logs "SL amendment successful for position BTC-USDT-SWAP" with INFO level
- **AND** continues to next position

#### Scenario: SL amendment failure due to invalid price
- **GIVEN** the system attempts to amend SL
- **WHEN** OKX API returns error code 51024 (SL price too close to market)
- **THEN** the system logs "SL amendment rejected: price too close to market" with WARN level
- **AND** does NOT update tracker's `currentSlPrice`
- **AND** retries calculation next cycle with updated market price

#### Scenario: SL amendment rate limit
- **GIVEN** the system amends multiple SL orders in quick succession
- **WHEN** OKX API returns HTTP 429 (rate limit exceeded)
- **THEN** the system logs "Rate limit exceeded during SL amendment" with WARN level
- **AND** implements exponential backoff (1s, 2s, 4s)
- **AND** retries up to 3 times
- **AND** if all retries fail, skips this adjustment and tries next cycle

#### Scenario: Network error during amendment
- **GIVEN** the system calls amend algo order API
- **WHEN** network timeout or connection error occurs
- **THEN** the system retries up to 3 times with exponential backoff
- **AND** logs each retry attempt with WARN level
- **AND** logs final failure with ERROR level if all retries fail
- **AND** position keeps current SL (static protection still active)

#### Scenario: Amend multiple SL orders for same position
- **GIVEN** a position has 2 pending conditional algo orders (edge case)
- **WHEN** dynamic SL adjustment is needed
- **THEN** the system identifies all matching orders by instId and posSide
- **AND** calls amend API for each order
- **AND** logs "Amended 2 SL orders for position BTC-USDT-SWAP" with INFO level

### Requirement: Dynamic SL State Management
The system SHALL persist dynamic SL tracker state for each position in SQLite database using GORM, ensuring state survives service restarts.

#### Scenario: Create tracker on position open
- **GIVEN** a new position is detected in check cycle
- **WHEN** dynamic SL is enabled and no tracker exists in database for this position
- **THEN** the system creates a new `DynamicSLTracker` with:
  - positionKey = "{instId}_{posSide}"
  - instId = position.Instrument
  - posSide = position.PositionSide
  - entryPrice = position.AveragePrice
  - currentSlPrice = calculated static SL price
  - highestPriceReached = currentMarkPrice (for long)
  - lowestPriceReached = currentMarkPrice (for short)
  - firstMoveTriggered = false
  - createdAt = NOW()
  - lastUpdatedAt = NOW()
- **AND** INSERTs tracker into `dynamic_sl_tracking` table using GORM

#### Scenario: Update tracker on each check cycle
- **GIVEN** a tracker exists in database for an open position
- **WHEN** TPSL check cycle runs
- **THEN** the system queries tracker from `dynamic_sl_tracking` table by position_key
- **AND** updates tracker with current mark price
- **AND** updates highest/lowest price if improved
- **AND** checks if adjustment conditions are met
- **AND** UPDATEs the tracker in database with new state using GORM

#### Scenario: Remove tracker on position close
- **GIVEN** a tracker exists in database for a position
- **WHEN** TPSL check cycle runs and position no longer exists in monitoring data
- **THEN** the system DELETEs tracker from `dynamic_sl_tracking` table using GORM
- **AND** logs "Position closed, removing dynamic SL tracker from database" with DEBUG level

#### Scenario: Tracker state persists across service restarts
- **GIVEN** dynamic SL trackers exist in database
- **WHEN** the TPSL manager restarts
- **THEN** all existing trackers remain in `dynamic_sl_tracking` table
- **AND** logs "Loading dynamic SL trackers from database" with INFO level
- **AND** trackers are queried from database on first check cycle
- **AND** state continues from where it left off (highestPrice, firstMove status preserved)

### Requirement: Dynamic SL Operation Logging
The system SHALL log all dynamic SL operations with sufficient detail for monitoring and troubleshooting.

#### Scenario: Log firstMove detection
- **GIVEN** firstMove threshold is reached
- **WHEN** SL adjustment is triggered
- **THEN** the system logs with INFO level:
  - Position instrument ID and side
  - Entry price
  - Current mark price
  - Profit percentage
  - Old SL price
  - New SL price (breakeven)
- **Example**: "FirstMove triggered: BTC-USDT-SWAP long, entry=40000, mark=40400, profit=1.00%, SL 38000 → 40040"

#### Scenario: Log trailing SL adjustment
- **GIVEN** trailing condition is met
- **WHEN** SL is adjusted upward/downward
- **THEN** the system logs with INFO level:
  - Position details
  - Highest/lowest price reached
  - Price gain from highest/lowest
  - Old and new SL prices
- **Example**: "Trailing SL: BTC-USDT-SWAP long, highest=40604, gain=0.50%, SL 40040 → 40080"

#### Scenario: Log amendment API calls
- **GIVEN** the system calls amend algo order API
- **WHEN** API call completes
- **THEN** the system logs with DEBUG level:
  - Algo order ID being amended
  - Old and new SL trigger prices
  - HTTP status code
  - Response time
  - Success or error message

#### Scenario: Log check cycle summary with dynamic SL metrics
- **GIVEN** TPSL check cycle completes with dynamic SL enabled
- **WHEN** cycle summary is logged
- **THEN** the system includes:
  - Total positions tracked for dynamic SL
  - Number of firstMove triggers this cycle
  - Number of trailing adjustments this cycle
  - Number of amendment API calls made
  - Number of amendment failures
- **Example**: "Dynamic SL: 5 positions tracked, 1 firstMove, 2 trailing adjustments, 3 amendments (0 failures)"

### Requirement: Integration with Static TPSL System
The system SHALL integrate dynamic SL functionality with existing static TPSL placement and management without breaking existing behavior.

#### Scenario: Dynamic SL adjusts existing static TPSL orders
- **GIVEN** a position has static TPSL orders placed by existing system
- **WHEN** dynamic SL conditions are met
- **THEN** the system amends the stop-loss trigger price of existing algo order
- **AND** take-profit trigger price remains unchanged (static TP)
- **AND** both dynamic SL and static TP coexist on same position

#### Scenario: Static TPSL placement when dynamic SL is enabled
- **GIVEN** dynamic SL is enabled in configuration
- **WHEN** a new position is detected without TPSL coverage
- **THEN** the system places static TPSL orders using existing logic (V4.5)
- **AND** creates a dynamic SL tracker for this position
- **AND** both systems work together (static placement + dynamic adjustment)

#### Scenario: Dynamic SL skipped when disabled
- **GIVEN** `tpsl.dynamic_sl.enabled = false` in configuration
- **WHEN** TPSL check cycle runs
- **THEN** the system skips all dynamic SL logic
- **AND** no trackers are created
- **AND** no amendments are made
- **AND** behavior is identical to V4.5 (static TPSL only)

#### Scenario: Position closed by dynamic SL trigger
- **GIVEN** a position with dynamic SL adjusted to 40080
- **WHEN** market price drops to 40080 and SL is triggered
- **THEN** OKX automatically closes the position
- **AND** cancels the take-profit order (OKX automatic behavior)
- **AND** next TPSL check cycle removes the tracker
- **AND** logs "Position closed by SL trigger" with INFO level

## MODIFIED Requirements

### Requirement: TPSL Configuration Management
The system SHALL load TPSL configuration from the YAML config file with validation and sensible defaults, including dynamic trailing stop-loss settings.

#### Scenario: Valid TPSL configuration
- **GIVEN** a valid config.yaml with TPSL section
- **WHEN** the system starts
- **THEN** the system loads TPSL enabled flag (default: true)
- **AND** loads check interval in seconds (default: 300)
- **AND** loads volatility percentage (default: 0.01)
- **AND** loads profit-loss ratio (default: 5.0)
- **AND** loads dynamic SL configuration (enabled, first_move_pct, trailing_step_pct, stop_move_step_pct)
- **AND** logs "TPSL configuration loaded successfully" with INFO level

#### Scenario: Missing TPSL configuration
- **GIVEN** config.yaml exists but has no TPSL section
- **WHEN** the system starts
- **THEN** the system uses default TPSL configuration values
- **AND** uses default dynamic SL configuration (enabled=false, first_move_pct=0.01, trailing_step_pct=0.005, stop_move_step_pct=0.001)
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
- **AND** dynamic SL is also disabled (cannot run without TPSL scheduler)

### Requirement: TPSL Operation Logging and Auditing
The system SHALL log all TPSL analysis and placement operations, including dynamic SL adjustments, with sufficient detail for auditing and troubleshooting.

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
  - **[NEW]** Number of positions tracked for dynamic SL
  - **[NEW]** Number of dynamic SL adjustments made
  - **[NEW]** Number of amendment failures

#### Scenario: Log individual position analysis
- **GIVEN** the system analyzes a position
- **WHEN** analysis completes
- **THEN** the system logs position details with DEBUG level:
  - Instrument ID
  - Position side
  - Position size
  - Covered size
  - Uncovered size
  - **[NEW]** Current profit percentage
  - **[NEW]** Dynamic SL state (firstMove triggered, trailing active)
  - Action taken (place order, skip, adjust SL, error)

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
- **GIVEN** the system makes an API request (get algo orders, place order, or **[NEW]** amend order)
- **WHEN** the request completes
- **THEN** the system logs with DEBUG level:
  - Endpoint called
  - HTTP status code
  - Response time
  - Success/failure status
- **AND** masks sensitive data (API keys, secrets) in logs
