# Change: Add OKX Simulator Tool for Testing

## Why
Real exchange price volatility is unpredictable and cannot be immediately reproduced for testing. The TenyoJubaku project needs a reliable way to test TPSL and dynamic stop-loss functionality with controlled price movements. A standalone OKX simulator will enable:
- Reproducible unit testing scenarios
- Testing edge cases (rapid price changes, breakeven triggers, trailing SL)
- CI/CD integration without real API calls

## What Changes
- Create a new standalone program `okx-simulator` that simulates OKX exchange behavior
- Implement OKX-compatible REST API endpoints for TPSL/dynamic SL testing
- Support configurable price sequences via CLI arguments or text file
- Support configurable price change intervals (default: 3 minutes)
- Bind to localhost:8888 only (not accessible from external networks)

## Impact
- New directory: `cmd/okx-simulator/`
- New package: `internal/simulator/` (optional, if needed for shared logic)
- No changes to existing code
- This is an **independent testing tool** that runs separately from the main application

## API Endpoints Implemented (OKX-compatible format)

Based on `internal/okx/interface.go` and extended for full trading simulation:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v5/market/ticker` | GET | Get current price (simulated) |
| `/api/v5/account/positions` | GET | Get live positions (created by order execution) |
| `/api/v5/account/balance` | GET | Get account balance (mock) |
| `/api/v5/trade/order` | POST | Place regular market/limit order (creates position) |
| `/api/v5/trade/orders-pending` | GET | Get pending regular orders |
| `/api/v5/trade/orders-algo-pending` | GET | Get pending algo orders |
| `/api/v5/trade/order-algo` | POST | Place TPSL algo order |
| `/api/v5/trade/amend-algos` | POST | Amend algo order (for dynamic SL) |

## Trading Engine Behavior

- Regular orders (`/trade/order`) execute immediately at current price, creating/modifying positions
- TPSL algo orders monitor price each tick and trigger when conditions are met
- Liquidation is checked on every price change: if price breaches `liqPx`, position is force-closed
- `posSide` is auto-inferred from order `side` if not provided (`buy` → `long`, `sell` → `short`)
- Liquidation price formula:
  - Long: `liqPx = avgPx × (1 - 1/leverage + mmr)` where `mmr = 0.01`
  - Short: `liqPx = avgPx × (1 + 1/leverage - mmr)`

## CLI Usage

```bash
# Using price arguments (space-separated)
./okx-simulator 100 101 102 103 104 105

# Using price file (one price per line, # for comments)
./okx-simulator -file=prices.txt

# Custom interval (in minutes)
./okx-simulator -interval=0.35 -file=prices.txt
```
