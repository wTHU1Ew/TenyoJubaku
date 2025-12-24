# TPSL Infinite Loop Bug Fix

**Version:** V3.0
**Date:** 2025-12-25
**Type:** Critical Bug Fix
**Feature:** Feature 1 - TPSL System

## Problem Description

When a position size increased (e.g., from 30 to 32 contracts) after TPSL orders were already set, the system would continuously create new TPSL orders every monitoring cycle until hitting OKX's order limit.

### Timeline from Debug Logs

```
07:58:48 - Position: 30, TP(30) + SL(30), fully covered ✓
08:03:48 - Position: 32, TP(30) + SL(30), uncovered: 2 ✗
08:08:48 - Position: 32, TP(30+2) + SL(30+2), uncovered: 2 ✗ (still!)
08:13:48 - Position: 32, TP(30+2+2) + SL(30+2+2), uncovered: 2 ✗ (infinite loop!)
...
```

### Root Cause

In `internal/tpsl/manager.go:154-220`, the `analyzeCoverage()` function used `maxTpSize` and `maxSlSize` to track the **maximum** order size instead of **accumulating all** order sizes.

**Buggy Code:**
```go
if hasTp {
    tpCount++
    if size > maxTpSize {
        maxTpSize = size  // Only keeps the max value!
    }
}
```

**Impact:**
- When position increased from 30 to 32
- Existing orders: TP(30) + SL(30)
- System creates: TP(2) + SL(2)
- Next cycle: `maxTpSize = max(30, 2) = 30` (not 32!)
- System thinks 2 contracts still uncovered
- Creates another TP(2) + SL(2)
- Infinite loop... 🔄

## Solution

### Approach: Accumulative Coverage Strategy

Changed the logic to **accumulate all matching order sizes** instead of taking the maximum.

**Fixed Code:**
```go
if hasTp {
    tpCount++
    totalTpSize += size  // Accumulate all TP order sizes
}
if hasSl {
    slCount++
    totalSlSize += size  // Accumulate all SL order sizes
}
```

### Changes Made

**File:** `internal/tpsl/manager.go`

1. **Line 157-158:** Changed variable names from `maxTpSize/maxSlSize` to `totalTpSize/totalSlSize`
2. **Line 179, 185:** Changed from `if size > maxTpSize` to `totalTpSize += size`
3. **Line 198-202:** Updated coverage calculation to use total sizes
4. **Line 214:** Updated logging to show total sizes

### Why This Approach?

We chose the accumulative strategy because:
- ✅ Minimal code changes
- ✅ No need to implement order cancellation API
- ✅ Supports multiple TPSL orders covering the same position
- ✅ Preserves existing orders and only adds for uncovered portions

**Trade-off:**
- Multiple TPSL orders will exist for the same position
- Trigger prices may differ slightly if average entry price changed
- This is acceptable as all orders will trigger correctly

## Testing

### Build Verification
```bash
go build -o bin/tenyojubaku ./cmd
# ✓ Compilation successful
```

### Expected Behavior After Fix

When position increases from 30 to 32:
```
First cycle: totalTpSize=30, totalSlSize=30, uncovered=2
  → Creates: TP(2) + SL(2)

Second cycle: totalTpSize=32, totalSlSize=32, uncovered=0
  → No new orders created ✓
```

### Log Validation

After deployment, monitor logs for:
```
Position BTC-USD-SWAP coverage: total=32.00000000, TP_covered=32.00000000 (count:2), SL_covered=32.00000000 (count:2), final_covered=32.00000000, uncovered=0.00000000
```

Should see `uncovered=0.00000000` instead of continuous `uncovered=2.00000000`.

## Deployment

### Steps

1. **Stop service:**
   ```bash
   sudo systemctl stop tenyojubaku
   ```

2. **Deploy new binary:**
   ```bash
   # Local: Build and upload
   go build -o bin/tenyojubaku ./cmd
   scp -i OkxWatching.pem bin/tenyojubaku ubuntu@<ec2-ip>:/path/to/deployment/
   ```

3. **Start service:**
   ```bash
   sudo systemctl start tenyojubaku
   sudo systemctl status tenyojubaku
   ```

4. **Monitor logs:**
   ```bash
   sudo journalctl -u tenyojubaku -f
   ```

## Related Issues

This fix addresses the critical bug reported on 2025-12-25 where TPSL orders were created infinitely when position size increased.

## Future Enhancements

Consider implementing a full coverage strategy in the future:
- Add `CancelAlgoOrder` API support
- When position size changes, cancel all old TPSL orders
- Create new TPSL orders covering the entire position
- This would ensure consistent trigger prices based on current average price

For now, the accumulative strategy is sufficient and stable.
