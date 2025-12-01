# maker-only-trading Specification

## Purpose
Enforce maker-only order placement to prevent FOMO trading and ensure better price execution, with exceptions for position-closing orders.

## ADDED Requirements

### Requirement: Maker-Only Configuration
The system SHALL load maker-only trading configuration from the YAML config file with validation.

#### Scenario: Valid maker-only configuration
- **GIVEN** a valid config.yaml with order_control.maker_only section
- **WHEN** the system starts
- **THEN** the system loads enabled flag (default: true)
- **AND** loads min_price_distance_pct (default: 0.01 for 1%)
- **AND** loads allow_taker_for_reduce_only flag (default: true)
- **AND** loads max_taker_pct (default: 0.5 for 50%)
- **AND** loads ticker_staleness_seconds (default: 60)
- **AND** logs "Maker-only configuration loaded: min_distance=1.0%, max_taker=50%" with INFO level

#### Scenario: Missing maker-only configuration
- **GIVEN** config.yaml exists but has no order_control.maker_only section
- **WHEN** the system starts
- **THEN** the system uses default maker-only configuration values
- **AND** logs "Using default maker-only trading configuration" with INFO level

#### Scenario: Invalid price distance percentage
- **GIVEN** config.yaml has min_price_distance_pct < 0 OR > 1.0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid min_price_distance_pct, must be between 0 and 1" with ERROR level
- **AND** exits with status code 1

#### Scenario: Invalid taker percentage
- **GIVEN** config.yaml has max_taker_pct <= 0 OR > 1.0
- **WHEN** the system validates configuration
- **THEN** the system logs "Invalid max_taker_pct, must be between 0 and 1" with ERROR level
- **AND** exits with status code 1

#### Scenario: Maker-only enforcement disabled
- **GIVEN** config.yaml has order_control.maker_only.enabled = false
- **WHEN** the system starts
- **THEN** the system logs "Maker-only enforcement disabled in configuration" with INFO level
- **AND** does not perform maker/taker validation on orders

### Requirement: Market Price Retrieval
The system SHALL fetch current market price from OKX ticker API for price distance validation.

#### Scenario: Successful ticker API call
- **GIVEN** the system needs to validate price distance for an order
- **WHEN** the system calls OKX ticker API GET /api/v5/market/ticker?instId=BTC-USDT-SWAP
- **THEN** the request includes proper authentication headers
- **AND** parses the JSON response to extract last traded price
- **AND** caches the price with timestamp for 5 seconds
- **AND** logs "Market price fetched for BTC-USDT-SWAP: 42000.5" with DEBUG level

#### Scenario: Ticker API failure with retry
- **GIVEN** the system calls OKX ticker API
- **WHEN** the API request fails (network error or HTTP error)
- **THEN** the system retries up to 3 times with exponential backoff (1s, 2s, 4s)
- **AND** logs each retry attempt with WARN level
- **AND** if all retries fail, checks for cached price

#### Scenario: Use cached price within staleness window
- **GIVEN** the ticker API failed to fetch fresh price
- **AND** a cached price exists from 30 seconds ago
- **AND** ticker_staleness_seconds = 60
- **WHEN** the system needs current market price
- **THEN** the system uses the cached price
- **AND** logs "Using cached market price (age: 30s) for BTC-USDT-SWAP: 42000.5" with INFO level

#### Scenario: Reject order with stale price
- **GIVEN** the ticker API failed to fetch fresh price
- **AND** the cached price is 90 seconds old
- **AND** ticker_staleness_seconds = 60
- **WHEN** the system needs current market price
- **THEN** the system rejects the order
- **AND** logs "Order rejected: unable to fetch current market price (cached price too stale: 90s)" with ERROR level
- **AND** returns error to caller with message "Cannot validate price distance: market price unavailable"

#### Scenario: Parse ticker response
- **GIVEN** OKX ticker API returns successful response
- **WHEN** the response JSON is {"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","last":"42123.4","askPx":"42124","bidPx":"42123"}]}
- **THEN** the system extracts last = "42123.4"
- **AND** converts to float64: 42123.4
- **AND** caches with current timestamp

#### Scenario: Handle ticker API error codes
- **GIVEN** OKX ticker API returns error response
- **WHEN** response is {"code":"51001","msg":"Instrument ID does not exist"}
- **THEN** the system logs "Invalid instrument ID: BTC-INVALID" with ERROR level
- **AND** rejects the order
- **AND** returns error to caller

### Requirement: Market Order Validation
The system SHALL reject market orders unless they are reduce-only with size within taker percentage limit.

#### Scenario: Reject non-reduce-only market order
- **GIVEN** user attempts to place a market order (ordType = "market")
- **AND** the order has reduce_only = false
- **WHEN** the system validates the order
- **THEN** the system rejects the order
- **AND** logs "Market order rejected: non-reduce-only market orders not allowed (maker-only mode)" with WARN level
- **AND** returns error to caller with message "Market orders not allowed. Use limit orders as maker to avoid FOMO trading."

#### Scenario: Allow reduce-only market order within taker limit
- **GIVEN** user attempts to place a market order (ordType = "market")
- **AND** the order has reduce_only = true (closing a position)
- **AND** current position size for BTC-USDT-SWAP long = 2.0
- **AND** order size = 1.0 (50% of position)
- **AND** max_taker_pct = 0.5
- **WHEN** the system validates the order
- **THEN** the system calculates taker_ratio = 1.0 / 2.0 = 0.5
- **AND** taker_ratio <= max_taker_pct (0.5 <= 0.5)
- **AND** the system allows the order
- **AND** logs "Reduce-only market order allowed: 50% of position (within 50% taker limit)" with INFO level

#### Scenario: Reject reduce-only market order exceeding taker limit
- **GIVEN** user attempts to place a market order (ordType = "market")
- **AND** the order has reduce_only = true
- **AND** current position size for BTC-USDT-SWAP long = 2.0
- **AND** order size = 1.5 (75% of position)
- **AND** max_taker_pct = 0.5
- **WHEN** the system validates the order
- **THEN** the system calculates taker_ratio = 1.5 / 2.0 = 0.75
- **AND** taker_ratio > max_taker_pct (0.75 > 0.5)
- **AND** the system rejects the order
- **AND** logs "Market order rejected: size 1.5 exceeds taker limit (75% > 50% max)" with WARN level
- **AND** returns error to caller with message "Market order size exceeds taker limit: 75% of position (max 50% allowed)"

#### Scenario: Query position size for taker validation
- **GIVEN** the system needs to validate taker percentage for a reduce-only market order
- **WHEN** the system queries current position
- **THEN** the system queries local database (positions table from monitoring) for latest position snapshot
- **AND** if position not found in database, queries OKX positions API as fallback
- **AND** extracts position size for matching instId and posSide
- **AND** caches position size for 60 seconds

#### Scenario: Handle missing position for reduce-only order
- **GIVEN** user attempts to place reduce-only market order
- **AND** no position exists for the instrument and side
- **WHEN** the system validates the order
- **THEN** the system logs "No position found for reduce-only order: BTC-USDT-SWAP long" with WARN level
- **AND** allows the order (OKX will reject if invalid)
- **AND** logs warning about potential inconsistency

### Requirement: Limit Order Price Distance Validation
The system SHALL validate that limit orders are placed at minimum distance from market price.

#### Scenario: Accept limit order with sufficient distance (buy below market)
- **GIVEN** user attempts to place a buy limit order for BTC-USDT-SWAP
- **AND** order price = 40000
- **AND** current market price = 42000
- **AND** min_price_distance_pct = 0.01 (1%)
- **WHEN** the system validates price distance
- **THEN** the system calculates distance = |40000 - 42000| / 42000 = 0.0476 (4.76%)
- **AND** distance >= min_price_distance_pct (4.76% >= 1%)
- **AND** the system allows the order
- **AND** logs "Limit order price distance valid: 4.76% (min 1%)" with INFO level

#### Scenario: Accept limit order with sufficient distance (sell above market)
- **GIVEN** user attempts to place a sell limit order for BTC-USDT-SWAP
- **AND** order price = 44000
- **AND** current market price = 42000
- **AND** min_price_distance_pct = 0.01 (1%)
- **WHEN** the system validates price distance
- **THEN** the system calculates distance = |44000 - 42000| / 42000 = 0.0476 (4.76%)
- **AND** distance >= min_price_distance_pct (4.76% >= 1%)
- **AND** the system allows the order
- **AND** logs "Limit order price distance valid: 4.76% (min 1%)" with INFO level

#### Scenario: Reject limit order with insufficient distance
- **GIVEN** user attempts to place a buy limit order for BTC-USDT-SWAP
- **AND** order price = 41800
- **AND** current market price = 42000
- **AND** min_price_distance_pct = 0.01 (1%)
- **WHEN** the system validates price distance
- **THEN** the system calculates distance = |41800 - 42000| / 42000 = 0.0048 (0.48%)
- **AND** distance < min_price_distance_pct (0.48% < 1%)
- **AND** the system rejects the order
- **AND** logs "Limit order rejected: price distance 0.48% < minimum 1% (market: 42000, order: 41800)" with WARN level
- **AND** returns error to caller with message "Order price too close to market (0.48% distance, min 1% required). Avoid FOMO trading."

#### Scenario: Price distance calculation precision
- **GIVEN** order price = 0.00012345
- **AND** market price = 0.00012468
- **WHEN** the system calculates price distance
- **THEN** the system uses high-precision decimal arithmetic (decimal.Decimal or float64 with proper rounding)
- **AND** distance = |0.00012345 - 0.00012468| / 0.00012468 = 0.0099 (0.99%)
- **AND** comparison handles floating-point precision correctly

#### Scenario: Reduce-only limit orders exempt from distance check
- **GIVEN** user attempts to place a reduce-only limit order (closing position)
- **AND** order price is very close to market price (0.1% distance)
- **AND** min_price_distance_pct = 0.01 (1%)
- **WHEN** the system validates the order
- **THEN** the system checks reduce_only flag
- **AND** skips price distance validation for reduce-only orders
- **AND** logs "Reduce-only limit order allowed (price distance check skipped)" with INFO level
- **AND** allows the order

### Requirement: Order Type Detection
The system SHALL correctly identify order types (market vs limit) for validation.

#### Scenario: Identify market order
- **GIVEN** order request has ordType = "market"
- **WHEN** the system processes the order
- **THEN** the system classifies it as market order (taker)
- **AND** applies market order validation rules

#### Scenario: Identify limit order
- **GIVEN** order request has ordType = "limit"
- **WHEN** the system processes the order
- **THEN** the system classifies it as limit order (maker)
- **AND** applies price distance validation rules

#### Scenario: Identify post-only limit order
- **GIVEN** order request has ordType = "post_only"
- **WHEN** the system processes the order
- **THEN** the system classifies it as limit order (maker)
- **AND** applies price distance validation rules
- **AND** logs "Post-only order detected (guaranteed maker)" with DEBUG level

#### Scenario: Handle optimal limit IOC as taker
- **GIVEN** order request has ordType = "optimal_limit_ioc"
- **WHEN** the system processes the order
- **THEN** the system classifies it as potentially taker order
- **AND** applies market order validation rules (reduce-only check)

### Requirement: Price Distance Caching
The system SHALL cache market prices to reduce API calls and improve performance.

#### Scenario: Cache market price after fetch
- **GIVEN** the system fetches market price for BTC-USDT-SWAP
- **WHEN** ticker API returns price = 42000
- **THEN** the system caches the price with current UTC timestamp
- **AND** cache key = "ticker:BTC-USDT-SWAP"
- **AND** cache TTL = 5 seconds

#### Scenario: Reuse cached price within TTL
- **GIVEN** market price for BTC-USDT-SWAP was fetched 3 seconds ago
- **AND** cache contains price = 42000
- **WHEN** the system needs market price for a new order
- **THEN** the system checks cache first
- **AND** finds valid cached price (age < 5 seconds)
- **AND** reuses cached price without API call
- **AND** logs "Using cached market price (age: 3s) for BTC-USDT-SWAP: 42000" with DEBUG level

#### Scenario: Refresh stale cache
- **GIVEN** market price for BTC-USDT-SWAP was fetched 6 seconds ago
- **AND** cache contains price = 42000
- **WHEN** the system needs market price for a new order
- **THEN** the system checks cache first
- **AND** finds stale cached price (age > 5 seconds)
- **AND** fetches fresh price from ticker API
- **AND** updates cache with new price and timestamp

#### Scenario: Handle concurrent cache access
- **GIVEN** two orders are being validated concurrently for the same instrument
- **WHEN** both need market price at the same time
- **THEN** the system uses mutex or similar synchronization
- **AND** only one API call is made to fetch price
- **AND** both orders reuse the same cached price

### Requirement: Logging and Auditing
The system SHALL log all maker-only validation decisions for auditing and debugging.

#### Scenario: Log market order rejection
- **GIVEN** a market order is rejected due to non-reduce-only
- **WHEN** validation fails
- **THEN** the system logs with WARN level:
  - Order type (market)
  - Instrument ID
  - Side and size
  - Reason for rejection (non-reduce-only)
- **AND** log message format: "Market order rejected for BTC-USDT-SWAP buy 0.5: non-reduce-only market orders not allowed (maker-only mode)"

#### Scenario: Log taker percentage violation
- **GIVEN** a reduce-only market order exceeds taker limit
- **WHEN** validation fails
- **THEN** the system logs with WARN level:
  - Order size and position size
  - Calculated taker percentage
  - Configured max taker percentage
- **AND** log message format: "Market order rejected for BTC-USDT-SWAP sell 1.5: taker percentage 75% exceeds max 50% (position size: 2.0)"

#### Scenario: Log price distance violation
- **GIVEN** a limit order is too close to market price
- **WHEN** validation fails
- **THEN** the system logs with WARN level:
  - Order price and market price
  - Calculated distance percentage
  - Minimum required distance
- **AND** log message format: "Limit order rejected for BTC-USDT-SWAP buy @ 41800: price distance 0.48% < minimum 1% (market: 42000)"

#### Scenario: Log successful validation
- **GIVEN** an order passes all maker-only validations
- **WHEN** validation succeeds
- **THEN** the system logs with INFO level:
  - Order type and details
  - Validation checks passed (price distance, taker limit, etc.)
- **AND** log message format: "Maker-only validation passed for BTC-USDT-SWAP buy @ 40000: limit order with 4.76% price distance (min 1%)"

### Requirement: Error Handling and Recovery
The system SHALL handle validation errors gracefully and provide clear error messages.

#### Scenario: Handle invalid instrument ID
- **GIVEN** user attempts to place order for non-existent instrument
- **WHEN** ticker API returns error for invalid instId
- **THEN** the system rejects the order
- **AND** logs "Invalid instrument ID: INVALID-SWAP" with ERROR level
- **AND** returns error to caller with message "Invalid instrument ID: INVALID-SWAP"

#### Scenario: Handle zero market price
- **GIVEN** ticker API returns price = 0 or empty string
- **WHEN** the system attempts to calculate price distance
- **THEN** the system detects invalid price
- **AND** logs "Invalid market price (zero or empty) for BTC-USDT-SWAP" with ERROR level
- **AND** rejects the order as fail-safe
- **AND** returns error to caller with message "Cannot validate price distance: invalid market price"

#### Scenario: Handle negative price distance calculation
- **GIVEN** calculation results in negative value due to data error
- **WHEN** the system validates price distance
- **THEN** the system takes absolute value |price_diff| / market_price
- **AND** ensures distance is always positive
- **AND** logs calculation with DEBUG level for debugging

#### Scenario: Database query failure for position size
- **GIVEN** the system queries database for position size
- **WHEN** database error occurs
- **THEN** the system retries query up to 3 times
- **AND** if retries fail, falls back to OKX positions API
- **AND** if both fail, logs error and allows order (fail-open for reduce-only)
- **AND** logs "Unable to verify position size, allowing reduce-only order" with WARN level

### Requirement: Configuration Validation
The system SHALL validate maker-only configuration values on startup.

#### Scenario: Validate price distance percentage range
- **GIVEN** config.yaml has min_price_distance_pct = 1.5 (150%)
- **WHEN** the system validates configuration
- **THEN** the system logs "Warning: min_price_distance_pct is very high (150%), most orders may be rejected" with WARN level
- **AND** continues with the configured value (user choice)

#### Scenario: Validate taker percentage range
- **GIVEN** config.yaml has max_taker_pct = 0.1 (10%)
- **WHEN** the system validates configuration
- **THEN** the system accepts the value (valid range 0-1)
- **AND** logs "Maker-only configuration loaded: max_taker=10%" with INFO level

#### Scenario: Validate staleness seconds
- **GIVEN** config.yaml has ticker_staleness_seconds = 300 (5 minutes)
- **WHEN** the system validates configuration
- **THEN** the system logs "Warning: ticker_staleness_seconds is high (300s), price validation may use outdated prices" with WARN level
- **AND** continues with the configured value

### Requirement: Reduce-Only Flag Detection
The system SHALL correctly determine if an order is reduce-only based on order parameters and current positions.

#### Scenario: Explicit reduce-only flag
- **GIVEN** order request has reduce_only = true
- **WHEN** the system checks reduce-only status
- **THEN** the system classifies it as reduce-only
- **AND** exempts from maker-only restrictions

#### Scenario: Infer reduce-only from order direction
- **GIVEN** current position is long 2.0 BTC-USDT-SWAP
- **AND** order is sell 1.0 BTC-USDT-SWAP
- **AND** order does not have explicit reduce_only flag
- **WHEN** the system checks reduce-only status
- **THEN** the system infers it's likely reduce-only (opposite direction of position)
- **AND** treats as reduce-only for validation purposes

#### Scenario: Opening order detected
- **GIVEN** current position is long 2.0 BTC-USDT-SWAP
- **AND** order is buy 1.0 BTC-USDT-SWAP (same direction)
- **WHEN** the system checks reduce-only status
- **THEN** the system classifies it as opening order (NOT reduce-only)
- **AND** applies full maker-only restrictions

#### Scenario: No position exists
- **GIVEN** no position exists for BTC-USDT-SWAP
- **AND** order is buy 1.0 BTC-USDT-SWAP
- **WHEN** the system checks reduce-only status
- **THEN** the system classifies it as opening order
- **AND** applies full maker-only restrictions
