# Design: Dynamic Trailing Stop-Loss

## Context

The current TPSL system (auto-tpsl-management) provides **static** stop-loss protection calculated at position entry based on:
- Entry price
- Configured volatility percentage (default 1%)
- Position leverage (not used in current calculation, see line 281 in manager.go)

**Limitation**: Once set, the stop-loss never moves, even as position becomes profitable. A 5% profitable position can still lose the original 1% if price reverses.

**User Strategy** (from project.md Feature 5 - Updated V5.1):
- **Leverage-Aware Gradual Move to Breakeven**:
  - When account profit reaches **1% × leverage**, move SL by **1.01% × leverage** (direction based on long/short)
  - Continue this step-by-step process until SL fully covers the entry price (breakeven)
  - Example: 10x leverage with initial SL at -2% (20% account loss potential)
    - First trigger: 10% account profit → SL moves up 10.1%
    - Second trigger: another 10% account profit → SL moves up another 10.1%, now at/above breakeven
- **Trailing Mode (only after breakeven is reached)**:
  - As price continues favorably, trail SL upward: every 0.5% price gain → SL moves up 0.1%
- This locks in profits while allowing position to run, with leverage-proportional protection

**Constraint**: Must work alongside existing static TPSL and be configuration-driven.

## Goals / Non-Goals

### Goals
1. **Protect Profits**: Automatically move stop-loss towards breakeven using leverage-aware gradual steps
2. **Leverage-Proportional**: Use position leverage to scale profit thresholds and SL movement steps
3. **Lock in Gains**: Trail stop-loss as price moves favorably to secure profits (only after breakeven)
4. **Configuration-Driven**: All parameters (profitStepPct, slMoveStepPct, trailingStep, stopMoveStep) configurable
5. **Co-exist with Static TPSL**: Dynamic SL adjusts existing algo orders, static TP unchanged
6. **Minimal Overhead**: Leverage existing TPSL check cycle, no new goroutines needed
7. **Backward Compatible**: Feature is opt-in; when disabled, system behaves identically to V4.5

### Non-Goals
1. **Historical Analysis**: Not building SL adjustment history table (only current state matters)
2. **Position-Specific Overrides**: All positions use same config (no per-position customization in MVP)
3. **Advanced Trailing Logic**: Not implementing ATR-based or time-based adjustments (future enhancement)
4. **Planned Trading**: Multi-entry left-side trading is Phase 2 (separate proposal)

## Decisions

### Decision 1: Database Persistence for State Tracking

**Choice**: Store dynamic SL state (highest price, firstMove triggered, current SL) in SQLite database using GORM

**Alternatives Considered**:
- **In-memory tracking only**: Track state in map within TPSL Manager
  - **Rejected**: State lost on service restart (unacceptable for AWS production deployment)
  - User explicitly stated: "考虑到在aws服务器运行这显然是需要数据库记录的"
- **Stateless recalculation**: Recalculate from scratch every cycle using only current position data
  - **Rejected**: Cannot track "highest price reached" without persistent state

**Rationale**:
- **Production Reliability**: Service restarts on AWS should not lose tracking state
- **Project Convention Compliance**: Project requires ORM for all database operations (see project.md)
- **State Criticality**: Losing highest price tracking means re-triggering firstMove, incorrect SL levels
- **GORM Integration**: Project already uses GORM v1.31.1, adding new table is straightforward

**Implementation**:
- Create new table: `dynamic_sl_tracking`
- Use GORM for all database operations (insert, update, query, delete)
- Load state from DB on manager startup
- Update DB after each SL adjustment

### Decision 2: Amendment vs New Orders

**Choice**: Amend existing algo orders rather than cancel + create new orders

**Alternatives Considered**:
- **Cancel + Create**: Cancel old TPSL order, create new one with updated SL
  - **Rejected**: Two API calls vs one, higher chance of race condition
  - Risk of position being unprotected between cancel and create

**Rationale**:
- OKX provides `POST /api/v5/trade/amend-algos` for modifying algo orders
- Atomic operation, position always protected
- Less API call volume (important for rate limits)

### Decision 3: Integration into Existing Check Cycle

**Choice**: Run dynamic SL logic within the existing TPSL check cycle (default 300s interval)

**Alternatives Considered**:
- **Separate goroutine with higher frequency**: Run dynamic SL checks every 60s
  - **Rejected**: Increases system complexity, more API calls
  - 5-minute interval sufficient for trailing stop strategy
- **Event-driven**: Only check on significant price movements
  - **Rejected**: Requires WebSocket price feeds, significant architecture change

**Rationale**:
- Leverages existing infrastructure (scheduler, error handling, logging)
- Consistent with project's layered design
- 5-minute granularity acceptable for swing trading strategy (user's typical style)

### Decision 4: Long/Short Position Handling (V5.1 - Leverage-Aware)

**Choice**: Use existing `isLongPosition()` method; reverse all logic for short positions; scale thresholds by leverage

**Implementation** (Two-Phase Algorithm):

**Phase A: Gradual Move to Breakeven (Leverage-Aware)**

- **Long**: SL moves up as account profits
  - Threshold: `accountProfit >= 1% × leverage` (e.g., 10x leverage → 10% account profit)
  - Action: `newSL = currentSL × (1 + 1.01% × leverage)` (e.g., 10x → SL moves up 10.1%)
  - Repeat: Continue until `currentSL >= entryPrice × 1.001` (breakeven + fees)
  - Example with 10x leverage, initial SL at entry - 2%:
    - Account profit 10% → SL moves from -2% to +8.1%
    - Already above breakeven, enter trailing mode

- **Short**: SL moves down as account profits
  - Threshold: `accountProfit >= 1% × leverage`
  - Action: `newSL = currentSL × (1 - 1.01% × leverage)`
  - Repeat: Continue until `currentSL <= entryPrice × 0.999` (breakeven - fees)

**Phase B: Trailing Mode (Only After Breakeven)**

- **Long**: Trail SL as price continues favorably
  - Trigger: `if (markPrice - highestPrice) / highestPrice >= 0.5%`
  - Action: SL moves up by 0.1%

- **Short**: Trail SL as price continues favorably
  - Trigger: `if (lowestPrice - markPrice) / lowestPrice >= 0.5%`
  - Action: SL moves down by 0.1%

**Rationale**:
- Leverage-proportional thresholds ensure consistent risk management across different leverage levels
- Gradual approach prevents premature breakeven trigger in volatile markets
- Clear two-phase separation (breakeven vs trailing) simplifies state machine

### Decision 5: Configuration Structure (V5.1 - Updated)

**Choice**: Nest `dynamic_sl` under existing `tpsl` config section

```yaml
tpsl:
  enabled: true
  check_interval: 300
  volatility_pct: 0.01
  profit_loss_ratio: 5.0

  # NEW: Dynamic trailing stop-loss (V5.1 - Leverage-Aware)
  dynamic_sl:
    enabled: true

    # Phase A: Gradual Move to Breakeven (Leverage-Aware)
    # Threshold: account_profit >= profit_step_pct × leverage
    profit_step_pct: 0.01      # 1% base (e.g., 10x leverage → 10% account profit triggers move)
    # Action: SL moves by sl_move_step_pct × leverage
    sl_move_step_pct: 0.0101   # 1.01% base (e.g., 10x leverage → SL moves 10.1%)

    # Phase B: Trailing Mode (Only After Breakeven)
    trailing_step_pct: 0.005   # Trail every 0.5% price gain
    stop_move_step_pct: 0.001  # Move SL up 0.1% each trail step
```

**Rationale**:
- Logical grouping (dynamic SL is extension of TPSL functionality)
- Clear hierarchy in config file
- Independent enable/disable from static TPSL
- Leverage scaling built into algorithm, config provides base percentages

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    TPSL Manager (existing)                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Check Cycle (every 300s)                              │ │
│  │  1. Query positions from monitoring                    │ │
│  │  2. Query pending algo orders from OKX                 │ │
│  │  3. Analyze coverage (existing)                        │ │
│  │  4. Place new TPSL orders if needed (existing)         │ │
│  │  5. [NEW] Check dynamic SL conditions                  │ │
│  │  6. [NEW] Amend algo orders if SL adjustment needed    │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  DynamicSLTracker (NEW - Database Model - V5.1)       │ │
│  │  - ID: int64 (primary key)                            │ │
│  │  - PositionKey: string (unique index)                 │ │
│  │  - InstId: string                                      │ │
│  │  - PosSide: string                                     │ │
│  │  - EntryPrice: float64                                 │ │
│  │  - CurrentSlPrice: float64                             │ │
│  │  - InitialSlPrice: float64 (original SL for calc)     │ │
│  │  - Leverage: float64 (position leverage)              │ │
│  │  - HighestPriceReached: float64 (for long)             │ │
│  │  - LowestPriceReached: float64 (for short)             │ │
│  │  - BreakevenReached: bool (Phase B starts here)        │ │
│  │  - MoveCount: int (how many times SL moved)            │ │
│  │  - LastUpdatedAt: time.Time                            │ │
│  │  - CreatedAt: time.Time                                │ │
│  │  + UpdateTracker(markPrice, leverage) [updates DB]    │ │
│  │  + ShouldAdjustSL() (bool, newSlPrice, phase)         │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  Loaded from DB: SELECT * FROM dynamic_sl_tracking         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
          ┌────────────────────────────────┐
          │      OKX Client (extended)     │
          │  + AmendAlgoOrder(...)         │
          │    POST /api/v5/trade/amend... │
          └────────────────────────────────┘
```

### Data Flow

```
1. TPSL Check Cycle Starts
   │
   ▼
2. Query Current Positions (with mark price and unrealized PNL)
   │
   ▼
3. For Each Position:
   │
   ├─► Check if dynamic SL enabled (config)
   │   │
   │   ▼ YES
   │   Query DynamicSLTracker from DB by position_key
   │   │   ├─► Exists: Load tracker from database
   │   │   └─► Not Exists: Create new tracker, INSERT into DB
   │   │
   │   ▼
   │   UpdateTracker(currentMarkPrice)
   │   │   ├─► Update highest/lowest price if needed
   │   │   ├─► Check firstMove threshold
   │   │   └─► UPDATE database with new state
   │   │
   │   ▼
   │   ShouldAdjustSL()?
   │   │
   │   ▼ YES (returns newSlPrice)
   │   Query pending algo orders for this position
   │   │
   │   ▼
   │   Find matching SL order (by instId, posSide, ordType=conditional)
   │   │
   │   ▼
   │   Call OKX AmendAlgoOrder(algoId, newSlTriggerPx)
   │   │
   │   ├─► SUCCESS: Update tracker.currentSlPrice, UPDATE DB, log adjustment
   │   │
   │   └─► FAILURE: Log error, retry next cycle (DB unchanged)
   │
   └─► Continue to next position

4. Log Dynamic SL Summary (positions tracked, adjustments made, failures)
```

### State Transitions (V5.1 - Two-Phase Algorithm)

```
Position State Machine for Dynamic SL:

[Position Opened]
      │
      ▼
[Static TPSL Placed] ──► Phase A: Gradual Move to Breakeven
      │                    │
      │                    ├─► Monitor account profit
      │                    │   every cycle
      │                    │
      │                    ├─► accountProfit >= 1% × leverage?
      │                    │
      │                    ▼ YES
      │              [Move SL by 1.01% × leverage]
      │                    │
      │                    ├─► currentSL >= entry × 1.001? (Long)
      │                    │   currentSL <= entry × 0.999? (Short)
      │                    │
      │                    ▼ NO (not yet at breakeven)
      │                    └─► Loop: wait for next profit threshold
      │                    │
      │                    ▼ YES (breakeven reached)
      │              [Phase B: Trailing Mode]
      │                    │
      │                    ├─► Update highest/lowest price
      │                    │
      │                    ├─► (markPrice - highest) / highest >= 0.5%?
      │                    │
      │                    ▼ YES
      │                    Amend SL up by 0.1%
      │                    │
      │                    └─► (loop: continue monitoring)
      │
      ▼
[Position Closed] ──► DELETE tracker from database

Example: 10x leverage, entry $100, initial SL $98 (2% loss = 20% account loss)
─────────────────────────────────────────────────────────────────────────────
1. Price rises to $101 (1% gain = 10% account profit)
   → SL moves from $98 to $98 × 1.101 = $107.898
   → Already above breakeven ($100.10), enter Phase B
2. Price continues to $102 (2% gain from $101 high)
   → Wait for 0.5% from high = need $101.505
3. Price hits $101.51 (0.5% above $101 high)
   → SL moves up 0.1%: $107.898 × 1.001 = $108.006
```

## Risks / Trade-offs

### Risk 1: Amendment API Rate Limits
**Risk**: Frequent SL adjustments could hit OKX rate limits (currently 60 requests/2s for trade endpoints)

**Mitigation**:
- Check cycle runs every 5 minutes (max 12 adjustments/hour per position)
- With 10 positions, max ~120 amendments/hour = 2/minute (well below limit)
- Implement retry with exponential backoff on rate limit error
- Add circuit breaker: pause amendments if consecutive failures > 10

### Risk 2: Price Gaps / Slippage
**Risk**: In fast-moving markets, price could gap past breakeven before SL is amended

**Mitigation**:
- Static SL remains in place until amendment succeeds (position always protected)
- 5-minute check interval is acceptable for swing trading (user's style)
- Future enhancement: Reduce check interval for highly volatile instruments

### Risk 3: Database Consistency on Concurrent Updates
**Risk**: Multiple TPSL check cycles (if overlapping) could cause race conditions on database updates

**Impact**: Tracker state could become inconsistent, incorrect SL calculations

**Mitigation**:
- TPSL check cycles run sequentially (not concurrent) by design
- Use GORM transactions for atomic read-modify-write operations
- Add unique constraint on position_key to prevent duplicate trackers
- Load state from DB at start of each cycle (fresh read)

### Risk 4: Amendment Timing Race Condition
**Risk**: Between checking price and amending order, price could reverse significantly

**Mitigation**:
- OKX validates amended SL price is reasonable (not too close to market)
- If amendment fails due to invalid price, retry next cycle with updated calculation
- Worst case: Static SL still protects position

### Risk 4: Interaction with Manual Order Amendments
**Risk**: User manually amends SL via OKX app → system overwrites it next cycle

**Mitigation**:
- Document in config.template.yaml: "Do not manually adjust SL when dynamic_sl is enabled"
- Future enhancement: Detect external amendments and disable dynamic SL for that position
- For MVP: User education via documentation

### Risk 5: Database Growth
**Risk**: `dynamic_sl_tracking` table grows unbounded if trackers aren't deleted when positions close

**Mitigation**:
- Delete tracker from DB immediately when position closes (checked every cycle)
- Add cleanup job: Delete trackers older than 7 days with no matching open position
- Expected table size: < 100 rows (typical user has < 10 concurrent positions)

## Migration Plan

### Deployment Steps

1. **Code Deployment**:
   - Deploy new code with `dynamic_sl.enabled = false` by default
   - Verify existing TPSL functionality unchanged (backward compatibility test)

2. **Configuration Update**:
   - Add `dynamic_sl` section to config.yaml
   - Set `enabled = true` with default parameters
   - Restart service

3. **Monitoring**:
   - Watch logs for "Dynamic SL adjustment" messages
   - Verify no errors in amendment API calls
   - Check OKX account for SL price changes in algo orders

4. **Gradual Rollout**:
   - Start with conservative parameters (firstMove=2%, trailingStep=1%, stopMoveStep=0.2%)
   - After 1 week of stable operation, adjust to target values (1%, 0.5%, 0.1%)

### Rollback Procedure

If issues arise:
1. Set `tpsl.dynamic_sl.enabled = false` in config
2. Restart service
3. All positions revert to static TPSL behavior
4. No data loss (no database changes)

### Configuration Validation

On startup, system validates:
- `first_move_pct` > 0 and < 1.0
- `trailing_step_pct` > 0 and < `first_move_pct`
- `stop_move_step_pct` > 0 and < `trailing_step_pct`
- If validation fails: Log ERROR and refuse to start (fail-fast)

## Open Questions

1. **Should we support position-specific dynamic SL parameters?**
   - Example: More aggressive trailing for BTC, conservative for altcoins
   - **Decision**: Not in MVP; add in future if user requests

2. **Should we persist SL adjustment history for analysis?**
   - Could help optimize parameters over time
   - **Decision**: Not in MVP; logs provide sufficient visibility

3. **What happens if position closes between cycles?**
   - **Decision**: Tracker is removed on next cycle when position query returns empty
   - No special handling needed

4. **Should dynamic SL work for both long and short positions?**
   - **Decision**: YES, full support for both (reverse logic for shorts)

5. **What if user has multiple TPSL orders for same position?**
   - **Decision**: Amend all matching orders (iterate through pending algo orders)
   - Edge case; unlikely given system only creates one TPSL per position
