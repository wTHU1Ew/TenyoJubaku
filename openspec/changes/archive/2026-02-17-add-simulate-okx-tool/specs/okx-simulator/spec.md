## ADDED Requirements

### Requirement: Price Sequence Input
The OKX simulator SHALL accept price sequences via command-line arguments or a text file.

#### Scenario: CLI price arguments
- **WHEN** the user runs `./okx-simulator 100 101 102 103`
- **THEN** the simulator starts with prices [100, 101, 102, 103]
- **AND** the initial price is 100

#### Scenario: Price file input
- **WHEN** the user runs `./okx-simulator -file=prices.txt`
- **AND** prices.txt contains one price per line
- **THEN** the simulator loads all prices from the file
- **AND** lines starting with `#` are treated as comments and ignored

#### Scenario: Empty or missing prices
- **WHEN** the user runs the simulator without any price arguments or file
- **THEN** the simulator exits with an error message explaining usage

### Requirement: Configurable Price Change Interval
The OKX simulator SHALL change to the next price in the sequence at configurable intervals.

#### Scenario: Default interval
- **WHEN** the user does not specify `-interval` flag
- **THEN** the simulator changes price every 3 minutes

#### Scenario: Custom interval
- **WHEN** the user specifies `-interval=1`
- **THEN** the simulator changes price every 1 minute

#### Scenario: End of price sequence
- **WHEN** the price sequence reaches its last value
- **THEN** the simulator stays at the last price (does not loop or exit)

### Requirement: Localhost-Only HTTP Server
The OKX simulator SHALL bind only to localhost to prevent external network access.

#### Scenario: Server binding
- **WHEN** the simulator starts
- **THEN** the HTTP server binds to `127.0.0.1:8888`
- **AND** the server is not accessible from external IP addresses

#### Scenario: Port in use
- **WHEN** port 8888 is already in use
- **THEN** the simulator exits with a clear error message

### Requirement: OKX-Compatible Ticker API
The simulator SHALL provide a `/api/v5/market/ticker` endpoint returning the current simulated price.

#### Scenario: Get ticker
- **WHEN** a GET request is made to `/api/v5/market/ticker?instId=BTC-USDT-SWAP`
- **THEN** the response contains the current simulated price
- **AND** the response format matches OKX API specification (code, msg, data array)

### Requirement: OKX-Compatible Positions API
The simulator SHALL provide a `/api/v5/account/positions` endpoint returning live positions created by order execution.

#### Scenario: Get positions after order placement
- **WHEN** a buy order has been placed via `/api/v5/trade/order`
- **AND** a GET request is made to `/api/v5/account/positions`
- **THEN** the response contains the created position with correct `posSide`, `avgPx`, `liqPx`

#### Scenario: Positions reflect liquidation price
- **WHEN** a position is created with `avgPx=100` and `lever=10`
- **THEN** `liqPx` in the response equals `91` (long) or `109` (short)

#### Scenario: No positions
- **WHEN** no orders have been placed or all positions are closed
- **AND** a GET request is made to `/api/v5/account/positions`
- **THEN** the response contains an empty data array

### Requirement: OKX-Compatible Algo Order APIs
The simulator SHALL provide endpoints for managing algo orders (TPSL).

#### Scenario: Place algo order
- **WHEN** a POST request is made to `/api/v5/trade/order-algo` with valid parameters
- **THEN** the order is stored in memory
- **AND** a unique algoId is returned in the response

#### Scenario: Get pending algo orders
- **WHEN** a GET request is made to `/api/v5/trade/orders-algo-pending?ordType=conditional`
- **THEN** the response contains all pending algo orders that were placed

#### Scenario: Amend algo order
- **WHEN** a POST request is made to `/api/v5/trade/amend-algos` with valid algoId
- **THEN** the order's trigger prices are updated
- **AND** the response indicates success

#### Scenario: Amend non-existent order
- **WHEN** a POST request is made to `/api/v5/trade/amend-algos` with invalid algoId
- **THEN** the response contains error code and message

### Requirement: Regular Order Placement and Position Creation
The simulator SHALL implement `/api/v5/trade/order` to execute orders and create positions.

#### Scenario: Place market buy order
- **WHEN** a POST request is made to `/api/v5/trade/order` with `side=buy`, `ordType=market`
- **THEN** the order fills immediately at current price
- **AND** a long position is created (or existing long position size is increased)
- **AND** a unique `ordId` is returned

#### Scenario: Auto-infer posSide
- **WHEN** a POST request is made to `/api/v5/trade/order` without `posSide` field
- **AND** `side=buy`
- **THEN** the simulator treats `posSide` as `long`
- **AND** position key uses `long` suffix

#### Scenario: Get pending orders
- **WHEN** a GET request is made to `/api/v5/trade/orders-pending`
- **THEN** the response contains any unfilled orders (market orders are empty after fill)

### Requirement: TPSL Trigger on Price Change
The simulator SHALL check all TPSL orders and trigger them when conditions are met.

#### Scenario: Take-profit trigger
- **WHEN** a TPSL algo order has `tpTriggerPx=110`
- **AND** the price advances to 110
- **THEN** the algo order triggers, closing the associated position
- **AND** the algo order is removed from pending

#### Scenario: Stop-loss trigger
- **WHEN** a TPSL algo order has `slTriggerPx=95`
- **AND** the price drops to 95
- **THEN** the algo order triggers, closing the associated position

### Requirement: Liquidation on Price Breach
The simulator SHALL liquidate positions when price reaches the liquidation price.

#### Scenario: Long position liquidation
- **WHEN** a long position has `liqPx=91`
- **AND** the price drops to 91 or below
- **THEN** the position is force-closed (liquidated)
- **AND** any associated TPSL algo orders remain in pending state (liquidation takes priority)

#### Scenario: Liquidation price formula
- Long: `liqPx = avgPx × (1 - 1/leverage + 0.01)`
- Short: `liqPx = avgPx × (1 + 1/leverage - 0.01)`
