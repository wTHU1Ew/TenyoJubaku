# Change: Add Dynamic Trailing Stop-Loss (动态移动止损)

## Why

The current TPSL system uses **static** stop-loss levels calculated at position entry. This approach has a critical weakness: once a position becomes profitable, the stop-loss remains at the original level, exposing profits to full reversal risk.

**Problem**: A position that gains 5% profit still has the same stop-loss level as when it was opened, meaning a reversal could eliminate all gains and still trigger the original loss.

**Solution**: Implement dynamic trailing stop-loss with **leverage-aware gradual breakeven approach** (V5.1):
1. **Phase A - Gradual Move to Breakeven**: When account profit reaches **1% × leverage**, move SL by **1.01% × leverage**
   - Continue step-by-step until SL fully covers entry price (breakeven)
   - Example: 10x leverage with initial SL at -2% → need moves until SL >= entry
2. **Phase B - Trailing Mode**: Only after reaching breakeven, trail SL upward
   - Every 0.5% price gain → SL moves up 0.1%
3. Allows profitable positions to run while providing leverage-proportional protection

This is the **MVP of Feature 5** (left-side trading strategy) as specified in project.md, with planned trading to be implemented in a future phase.

## What Changes

**Core Functionality (V5.1 - Leverage-Aware):**
- Add dynamic stop-loss adjustment logic with **two-phase algorithm**:
  - **Phase A**: Gradual move to breakeven using leverage-scaled thresholds
  - **Phase B**: Trailing mode (only after breakeven reached)
- Monitor position unrealized PNL and account profit in real-time
- Read position leverage for proportional threshold calculation
- Automatically amend algo order stop-loss prices via OKX API when profit thresholds are met
- Configuration-driven parameters (profitStepPct, slMoveStepPct, trailingStep, stopMoveStep)

**Specific Changes:**
1. **TPSL Configuration Extension (V5.1)**
   - Add `dynamic_sl` section to TPSL config
   - Parameters: `enabled`, `profit_step_pct`, `sl_move_step_pct`, `trailing_step_pct`, `stop_move_step_pct`
   - Default: profitStepPct=1% (×leverage), slMoveStepPct=1.01% (×leverage), trailingStep=0.5%, stopMoveStep=0.1%

2. **Dynamic SL Tracking (Database Persistence)**
   - Create new table `dynamic_sl_tracking` to store tracker state
   - Track current stop-loss price for each position in database
   - Store highest price reached after firstMove triggered
   - Calculate new stop-loss price when trailing conditions met
   - Use GORM for all database operations (complies with project.md ORM requirement)

3. **OKX API Integration**
   - Use existing or add `AmendAlgoOrder` API to modify stop-loss trigger price
   - Handle amendment failures gracefully (retry logic, logging)

4. **TPSL Manager Enhancement**
   - Add periodic check for profit thresholds (runs in same cycle as coverage check)
   - Calculate when to move stop-loss based on current price vs entry price
   - Amend algo orders when stop-loss needs adjustment

5. **Logging and Monitoring**
   - Log every stop-loss adjustment with reason
   - Track adjustment history for analysis
   - Alert on amendment failures

**Co-existence with Static TPSL:**
- Dynamic SL works alongside existing static TP
- When dynamic SL triggers, all TPSL orders for that position are cancelled automatically (OKX behavior)
- Static TP remains unchanged and will trigger at original calculated price

## Impact

**Affected Specs:**
- `auto-tpsl-management`: Add dynamic stop-loss requirements

**Affected Code:**
- `internal/config/config.go`: Add DynamicSLConfig struct
- `internal/tpsl/manager.go`: Add dynamic SL logic and amendment calls
- `internal/okx/client.go`: Add or verify AmendAlgoOrder method exists
- `configs/config.template.yaml`: Add dynamic_sl configuration section
- `docs/VERSION_HISTORY.md`: Document as V4.6 or V5.0 (new feature)

**Breaking Changes:**
- None (feature is opt-in via configuration)

**Database Changes (V5.1 - Updated Schema):**
- **MODIFIED TABLE**: `dynamic_sl_tracking` with columns:
  - `id` (primary key, auto-increment)
  - `position_key` (unique index, format: "{instId}_{posSide}")
  - `inst_id` (instrument ID)
  - `pos_side` (position side: long/short/net)
  - `entry_price` (position entry price)
  - `current_sl_price` (current stop-loss trigger price)
  - `initial_sl_price` (original SL price for calculating distance to breakeven) **NEW**
  - `leverage` (position leverage for threshold calculation) **NEW**
  - `highest_price_reached` (for long positions)
  - `lowest_price_reached` (for short positions)
  - `breakeven_reached` (boolean, true when Phase B starts) **NEW (replaces first_move_triggered)**
  - `move_count` (number of SL adjustments made) **NEW**
  - `last_updated_at` (timestamp)
  - `created_at` (timestamp)
- Uses GORM for all database operations (complies with project.md convention)
- Automatic cleanup: Delete trackers when positions close

**Configuration Changes:**
- **ADDED**: `tpsl.dynamic_sl.*` configuration section
- **BACKWARD COMPATIBLE**: If `dynamic_sl.enabled = false`, behavior is identical to current system

**Testing Impact:**
- New unit tests for dynamic SL calculation logic
- Integration tests with mock OKX API for amendment calls
- Manual testing with real positions (max 5 USDT as per project conventions)
