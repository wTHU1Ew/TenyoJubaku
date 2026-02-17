# Tasks: Add OKX Simulator Tool

## 1. Project Setup
- [x] 1.1 Create `cmd/okx-simulator/` directory
- [x] 1.2 Create `main.go` with CLI argument parsing
- [x] 1.3 Implement `-interval` flag (default: 3 minutes)
- [x] 1.4 Implement price file loading (`-file` flag)
- [x] 1.5 Implement `-pos` flag for initial position setup

## 2. Core Components
- [x] 2.1 Implement `PriceController` with thread-safe price management
- [x] 2.2 Implement timer-based price advancement
- [x] 2.3 Implement `TradingEngine` for orders, positions, TPSL, liquidation
- [x] 2.4 Implement order ID / algo ID generation
- [x] 2.5 Implement liquidation price calculation (`liqPx`)
- [x] 2.6 Implement TPSL trigger detection on each price tick
- [x] 2.7 Implement liquidation enforcement on each price tick
- [x] 2.8 Auto-infer `posSide` from order `side` if not provided

## 3. HTTP Server
- [x] 3.1 Create HTTP server bound to `127.0.0.1:8888`
- [x] 3.2 Implement `/api/v5/market/ticker` endpoint
- [x] 3.3 Implement `/api/v5/account/positions` endpoint
- [x] 3.4 Implement `/api/v5/account/balance` endpoint
- [x] 3.5 Implement `/api/v5/trade/order` endpoint (POST, regular order)
- [x] 3.6 Implement `/api/v5/trade/orders-pending` endpoint (GET)
- [x] 3.7 Implement `/api/v5/trade/orders-algo-pending` endpoint (GET)
- [x] 3.8 Implement `/api/v5/trade/order-algo` endpoint (POST, TPSL)
- [x] 3.9 Implement `/api/v5/trade/amend-algos` endpoint (POST)

## 4. OKX Response Format
- [x] 4.1 Create response structs matching OKX API format
- [x] 4.2 Implement JSON response helpers
- [x] 4.3 Ensure response format matches `internal/okx/types.go`

## 5. Testing
- [x] 5.1 Test price sequence from CLI arguments
- [x] 5.2 Test price sequence from file
- [x] 5.3 Test interval timing
- [x] 5.4 Test all API endpoints return correct format
- [x] 5.5 Test 1: CLI order → position created → query returns correct data
- [x] 5.6 Test 2: Two sequential orders each with TPSL, both TP triggers fire correctly
- [x] 5.7 Test 3: Order with TP only, price falls to liqPx, liquidation fires
- [x] 5.8 Integration test with TenyoJubaku V5.1 dynamic SL (4 tests: long/short × 1%/3% SL — all passed; report in docs/dynamic-sl-v51-integration-test-report.md)

## 6. Documentation
- [x] 6.1 Add usage comments in main.go
- [x] 6.2 Create example price files in `testdata/`
- [x] 6.3 Document integration steps in design.md
- [x] 6.4 Create automated test scripts `testdata/run_test2.sh`, `testdata/run_test3.sh`
- [x] 6.5 Sync proposal.md, design.md, spec.md to reflect actual TradingEngine implementation
