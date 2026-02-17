# Design: OKX Simulator Tool

## Context
TenyoJubaku's TPSL and dynamic stop-loss features depend on real-time price data from OKX. Testing these features requires:
- Controlled price movements
- Reproducible scenarios
- No dependency on real exchange

## Goals
- Standalone executable that simulates OKX API behavior
- Controllable price sequences for testing
- OKX-compatible REST API responses
- Localhost-only binding for security

## Non-Goals
- Full OKX API coverage (only TPSL/trading-related endpoints)
- WebSocket support (REST API sufficient for testing)
- Authentication simulation (not needed for local testing)
- Persistent state (in-memory only, resets on restart)

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    okx-simulator                         │
├──────────────────────────────────────────────────────────┤
│  CLI Parser                                              │
│  - Parse price args or load from file                    │
│  - Parse -interval flag                                  │
├──────────────────────────────────────────────────────────┤
│  Price Controller                                        │
│  - Holds price sequence                                  │
│  - Timer-based price advancement                         │
│  - Thread-safe current price access                      │
│  - Triggers TPSL check and liquidation on each advance   │
├──────────────────────────────────────────────────────────┤
│  Trading Engine                                          │
│  - Order execution (market/limit → fills immediately)    │
│  - Position management (open/close by instId+posSide)    │
│  - Liquidation price calculation and enforcement         │
│  - TPSL trigger detection on each price tick             │
│  - Auto-infer posSide from order side if not provided    │
├──────────────────────────────────────────────────────────┤
│  HTTP Server (localhost:8888)                            │
│  - /api/v5/market/ticker                  GET            │
│  - /api/v5/account/positions              GET            │
│  - /api/v5/account/balance               GET            │
│  - /api/v5/trade/order                    POST           │
│  - /api/v5/trade/orders-pending           GET            │
│  - /api/v5/trade/orders-algo-pending      GET            │
│  - /api/v5/trade/order-algo               POST           │
│  - /api/v5/trade/amend-algos             POST           │
└──────────────────────────────────────────────────────────┘
```

## Decisions

### Decision 1: Use Standard Library HTTP Server
**Why:** Simple requirements, no external dependencies, easy to maintain.
**Alternative:** gin/echo frameworks - overkill for this use case.

### Decision 2: In-Memory State Only
**Why:** Simulator is for testing, state doesn't need persistence.
**Trade-off:** State lost on restart, but that's acceptable for testing.

### Decision 3: Localhost-Only Binding
**Why:** Security requirement - prevent external access.
**Implementation:** Bind to `127.0.0.1:8888` explicitly, not `0.0.0.0:8888`.

### Decision 4: Price File Format
```
# prices.txt - one price per line, comments allowed
100.00
101.50
# spike scenario
150.00
102.00
```

### Decision 5: Positions Created by Order Execution (Not Flags)
Positions are created via the `/api/v5/trade/order` endpoint (same as real OKX), not via CLI flags. This enables testing the full order→position→TPSL→close flow that TenyoJubaku actually uses.

### Decision 6: Auto-Infer posSide
TenyoJubaku CLI doesn't always send `posSide`. The simulator auto-infers it:
- `side=buy` → `posSide=long`
- `side=sell` → `posSide=short`

This prevents position key mismatches when the CLI omits `posSide`.

## API Response Format

All responses follow OKX API format:
```json
{
    "code": "0",
    "msg": "",
    "data": [...]
}
```

### GET /api/v5/market/ticker?instId=XXX
```json
{
    "code": "0",
    "data": [{
        "instId": "BTC-USDT-SWAP",
        "last": "50000",
        "bidPx": "49999",
        "askPx": "50001"
    }]
}
```

### GET /api/v5/account/positions
Returns configured positions with current unrealized PnL calculated.

### POST /api/v5/trade/order
Executes immediately at current price. Creates or adds to position. Returns `ordId`.

### GET /api/v5/trade/orders-pending
Returns all active regular orders (filled orders are removed from pending).

### POST /api/v5/trade/order-algo
Stores TPSL algo order in memory, returns generated `algoId`.

### POST /api/v5/trade/amend-algos
Updates existing algo order's trigger prices. Used by dynamic SL feature.

## Price Advancement Logic

Every interval tick:
1. Advance to next price in sequence (stay at last price when exhausted)
2. Check all TPSL algo orders for trigger conditions
3. Check all positions for liquidation (price ≤ liqPx for long, price ≥ liqPx for short)
4. Log all events for debugging

## Liquidation Price Calculation

```
mmr = 0.01  (maintenance margin ratio)
Long:  liqPx = avgPx × (1 - 1/leverage + mmr)
Short: liqPx = avgPx × (1 + 1/leverage - mmr)

Example: entry=100, leverage=10x (long)
liqPx = 100 × (1 - 0.1 + 0.01) = 91
```

## Integration with TenyoJubaku

In config.yaml for testing:
```yaml
okx:
  api_url: "http://localhost:8888"  # Point to simulator
  api_key: "test"
  api_secret: "test"
  passphrase: "test"
```

## Example Test Scenarios

```bash
# Test TPSL: price rises to trigger TP at 110
./okx-simulator -file=testdata/test2_prices.txt -interval=0.35

# Test liquidation: price falls to liqPx=91
./okx-simulator -file=testdata/test3_prices.txt -interval=0.35

# Test dynamic SL (V5.1): price rises, dynamic SL follows trailing
./okx-simulator -interval=0.35 100 102 105 108 110 112 115 112 108 105
```

See `testdata/run_test2.sh` and `testdata/run_test3.sh` for full automated test scripts.
