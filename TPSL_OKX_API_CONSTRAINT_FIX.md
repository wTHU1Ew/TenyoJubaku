# OKX API Price Constraint Fix

## 发现日期 / Discovery Date
2025-12-02

## 问题描述 / Problem Description

### 用户报告 / User Report
程序尝试设置止盈订单时，OKX API 返回错误：

```
Error Code: 51277
Error Message: "TP trigger price cannot be higher than the last price"
```

### API 请求详情 / API Request Details

```json
POST /api/v5/trade/order-algo
{
  "instId": "BTC-USD-SWAP",
  "tdMode": "cross",
  "side": "buy",           // 买入平仓（平空）
  "posSide": "short",      // 空头持仓
  "ordType": "conditional",
  "sz": "2",
  "tpTriggerPx": "84829.8",   // ❌ 止盈触发价 = 当前价
  "tpOrdPx": "-1",
  "reduceOnly": true,
  "tpTriggerPxType": "last"
}
```

**当前市场价格**：$84,829.8

### 根本原因 / Root Cause

**OKX API 对止盈价格的限制**：

1. **做多持仓 (Long Position)**：
   - 止盈订单是 "sell"（卖出平仓）
   - 止盈触发价必须 **> 当前价格**
   - 原因：只有价格上涨到目标价以上才能止盈

2. **做空持仓 (Short Position)**：
   - 止盈订单是 "buy"（买入平仓）
   - 止盈触发价必须 **< 当前价格**
   - 原因：只有价格下跌到目标价以下才能止盈

### 问题场景 / Problem Scenario

```
持仓类型：空头 (Short)
入场价格：$91,610.47
预期止盈价：$85,197.73（计算得出）
当前价格：$84,829.8（已经低于预期）

程序逻辑（旧）：
- 检测到当前价 <= 预期止盈价
- 调整止盈价为当前价：$84,829.8
- 尝试下单：tpTriggerPx = $84,829.8

OKX API 检查：
- 当前价：$84,829.8
- 止盈触发价：$84,829.8
- 判断：触发价 >= 当前价（不满足要求）
- 返回错误：51277 ❌
```

## 修复方案 / Fix Solution

### 核心思路 / Core Idea

当当前价格已经超过预期止盈价时，不能直接使用当前价作为止盈触发价，而应该：

- **做多持仓**：设置触发价 = 当前价 × 1.001（略高 0.1%）
- **做空持仓**：设置触发价 = 当前价 × 0.999（略低 0.1%）

这样可以：
1. ✅ 满足 OKX API 的价格约束
2. ✅ 触发价仍然非常接近当前价（会很快触发）
3. ✅ 锁定大部分已实现的利润

### 代码修改 / Code Changes

**文件**：`internal/tpsl/manager.go`
**函数**：`adjustTPSLPricesWithCurrentPrice()`

#### 修改前 / Before (做空持仓部分)

```go
// Short position: TP below entry, SL above entry

// Check TP: if current price <= expected TP price
if currentPrice <= prices.TpPrice {
    m.logger.Warn("Position %s (short): Current price %.8f has reached or exceeded expected TP %.8f",
        position.Instrument, currentPrice, prices.TpPrice)
    m.logger.Info("Adjusting TP price to current market price: %.8f → %.8f",
        prices.TpPrice, currentPrice)
    adjustedPrices.TpPrice = currentPrice  // ❌ 直接使用当前价
    // Set TP at current price (will trigger immediately or very soon)
}
```

#### 修改后 / After (做空持仓部分)

```go
// Short position: TP below entry, SL above entry

// Check TP: if current price <= expected TP price
if currentPrice <= prices.TpPrice {
    m.logger.Warn("Position %s (short): Current price %.8f has reached or exceeded expected TP %.8f",
        position.Instrument, currentPrice, prices.TpPrice)
    // For short position: TP must be BELOW current price
    // Set TP slightly below current price (0.1% lower to ensure it's below)
    adjustedPrice := currentPrice * 0.999  // ✅ 略低于当前价
    m.logger.Info("Adjusting TP price to slightly below current price: %.8f → %.8f (current: %.8f)",
        prices.TpPrice, adjustedPrice, currentPrice)
    adjustedPrices.TpPrice = adjustedPrice
}
```

#### 做多持仓部分 / Long Position Part

```go
if isLong {
    // Long position: TP above entry, SL below entry

    // Check TP: if current price >= expected TP price
    if currentPrice >= prices.TpPrice {
        m.logger.Warn("Position %s (long): Current price %.8f has reached or exceeded expected TP %.8f",
            position.Instrument, currentPrice, prices.TpPrice)
        // For long position: TP must be ABOVE current price
        // Set TP slightly above current price (0.1% higher to ensure it's above)
        adjustedPrice := currentPrice * 1.001  // ✅ 略高于当前价
        m.logger.Info("Adjusting TP price to slightly above current price: %.8f → %.8f (current: %.8f)",
            prices.TpPrice, adjustedPrice, currentPrice)
        adjustedPrices.TpPrice = adjustedPrice
    }

    // ... SL logic
}
```

## 修复后的行为 / Behavior After Fix

### 场景：做空持仓，价格已达止盈目标

```
持仓：空头 (Short)
入场价：$91,610.47
预期止盈：$85,197.73
当前价：$84,829.8（已低于预期）

修复后的逻辑：
1. 检测到当前价 <= 预期止盈价
2. 计算调整价格：$84,829.8 × 0.999 = $84,745.03
3. 设置止盈触发价：$84,745.03
4. OKX API 验证：
   - 当前价：$84,829.8
   - 触发价：$84,745.03
   - 判断：触发价 < 当前价 ✅
   - 结果：订单接受

预期日志：
[WARN] Position BTC-USD-SWAP (short): Current price 84829.80 has reached or exceeded expected TP 85197.73
[INFO] Adjusting TP price to slightly below current price: 85197.73 → 84745.03 (current: 84829.80)
[INFO] Take-Profit order placed successfully for BTC-USD-SWAP (short), algoId: xxxxx, trigger: 84745.03
```

### 场景：做多持仓，价格已达止盈目标

```
持仓：多头 (Long)
入场价：$50,000
预期止盈：$52,500
当前价：$53,000（已高于预期）

修复后的逻辑：
1. 检测到当前价 >= 预期止盈价
2. 计算调整价格：$53,000 × 1.001 = $53,053
3. 设置止盈触发价：$53,053
4. OKX API 验证：
   - 当前价：$53,000
   - 触发价：$53,053
   - 判断：触发价 > 当前价 ✅
   - 结果：订单接受

预期日志：
[WARN] Position XXX (long): Current price 53000.00 has reached or exceeded expected TP 52500.00
[INFO] Adjusting TP price to slightly above current price: 52500.00 → 53053.00 (current: 53000.00)
[INFO] Take-Profit order placed successfully for XXX (long), algoId: xxxxx, trigger: 53053.00
```

## 调整幅度的选择 / Adjustment Percentage Choice

### 为什么选择 0.1% (0.001)？

1. **足够小**：
   - 不会显著影响止盈效果
   - 大部分利润已经锁定
   - 例如：$84,829 → $84,745，差异仅 $84

2. **足够大**：
   - 确保满足 OKX API 的价格约束
   - 避免因价格波动导致边界情况
   - 0.1% 远大于价格精度误差

3. **行业标准**：
   - 加密货币交易中常用的微小价差
   - Tick size 通常在 0.01-0.1 之间
   - 0.1% 是一个安全且合理的缓冲

### 可配置化（未来改进）

可以考虑将调整幅度添加到配置文件：

```yaml
tpsl:
  # 当价格超过预期时，止盈价格调整幅度（百分比）
  # TP price adjustment when current price exceeds target (percentage)
  # For long: TP = current_price × (1 + adjustment_pct)
  # For short: TP = current_price × (1 - adjustment_pct)
  # Default: 0.001 (0.1%)
  price_adjustment_pct: 0.001
```

## OKX API 价格约束总结 / OKX API Price Constraints Summary

### 止盈订单 (Take-Profit Orders)

| 持仓类型 | 平仓方向 | 触发价要求 | 示例 |
|---------|---------|----------|------|
| 做多 (Long) | Sell | TP > 当前价 | 当前 $100 → TP $101 ✅ |
| 做多 (Long) | Sell | TP ≤ 当前价 | 当前 $100 → TP $100 ❌ |
| 做空 (Short) | Buy | TP < 当前价 | 当前 $100 → TP $99 ✅ |
| 做空 (Short) | Buy | TP ≥ 当前价 | 当前 $100 → TP $100 ❌ |

### 止损订单 (Stop-Loss Orders)

| 持仓类型 | 平仓方向 | 触发价要求 | 示例 |
|---------|---------|----------|------|
| 做多 (Long) | Sell | SL < 当前价 | 当前 $100 → SL $99 ✅ |
| 做多 (Long) | Sell | SL ≥ 当前价 | 当前 $100 → SL $100 ❌ |
| 做空 (Short) | Buy | SL > 当前价 | 当前 $100 → SL $101 ✅ |
| 做空 (Short) | Buy | SL ≤ 当前价 | 当前 $100 → SL $100 ❌ |

### 错误代码参考 / Error Code Reference

- **51277**: TP trigger price cannot be higher than the last price
- **51278**: SL trigger price cannot be lower than the last price
- 其他价格相关错误码请参考 OKX API 文档

## 测试验证 / Testing

### 编译验证 / Build Verification

```bash
$ go build -o ./bin/tenyojubaku ./cmd/main.go
✅ Build successful
```

### 运行测试 / Runtime Test

```bash
$ ./bin/tenyojubaku
```

### 预期结果 / Expected Results

1. ✅ 程序检测到价格已超过预期止盈目标
2. ✅ 自动调整止盈价格（略低于/高于当前价）
3. ✅ 成功创建止盈和止损订单
4. ✅ 日志显示调整后的价格和原因

## 相关文档 / Related Documentation

1. **TPSL 修复总结**: `TPSL_FIX_SUMMARY.md`
   - 止盈止损分开下单的修复

2. **价格验证功能**: `TPSL_PRICE_VALIDATION_SUMMARY.md`
   - 当前价格与预期价格对比验证

3. **覆盖分析修复**: `TPSL_COVERAGE_BUG_FIX.md`
   - 必须同时有 TP 和 SL 才算覆盖

4. **OKX API 文档**: `document/markdown/OKX_API.md`
   - OKX 官方 API 规范和限制

## 历史问题回顾 / Historical Issues

这是 Feature 2 (Automatic TPSL) 的第**四次**重要修复：

1. **第一次** (2025-12-01): 分开下单
   - 问题：只设置 SL，不设置 TP
   - 原因：OKX API 限制

2. **第二次** (2025-12-02): 价格验证
   - 问题：当前价格可能已超过预期 TP 价格
   - 解决：获取实时市场价格并调整

3. **第三次** (2025-12-02): 覆盖分析
   - 问题：只有 SL 时系统认为已覆盖
   - 解决：要求必须同时有 TP 和 SL

4. **第四次** (2025-12-02): OKX 价格约束 ⬅️ 当前
   - 问题：调整后的 TP 价格等于当前价，违反 OKX API 约束
   - 原因：OKX 要求 TP 必须与当前价有价差
   - 解决：调整 TP 价格为略高于/低于当前价（0.1%）

## 经验教训 / Lessons Learned

1. **API 约束必须仔细研究**
   - 不能假设"等于当前价"就可以接受
   - 要明确理解"大于"和"小于"的严格要求

2. **错误信息是宝贵的反馈**
   - OKX 返回的 51277 错误码明确指出问题
   - debug 模式的 API 响应对调试至关重要

3. **边界情况需要额外处理**
   - "价格刚好等于"是典型的边界情况
   - 需要添加小的偏移量来处理

4. **日志要详细记录调整过程**
   - 记录原始价格、调整后价格、当前价格
   - 便于用户理解系统行为和验证正确性

## 实施人员 / Implementation
- **发现**: 用户报告（OKX API 错误）
- **分析**: Claude Code
- **修复**: Claude Code
- **测试**: 待用户验证 / Pending User Verification

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Feature 2 - Automatic TPSL Management (Bug Fix)
- **版本**: v1.1.2 (OKX API Price Constraint Fix)
- **日期**: 2025-12-02
