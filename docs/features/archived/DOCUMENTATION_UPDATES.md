# 文档更新记录

## 更新时间
2025-12-01

## 更新原因
修复了自动止盈止损功能中只设置止损、未设置止盈的问题后，需要更新相关文档以准确反映：
1. OKX API 的实际行为和限制
2. 正确的止盈止损计算公式
3. 推荐的最佳实践

## 更新文件清单

### 1. `document/markdown/OKX_API.md`

#### 修改位置 1：请求示例（Line 15763-15803）

**修改前**：
```
# Place Take Profit / Stop Loss Order
POST /api/v5/trade/order-algo
body
{
    "instId":"BTC-USDT",
    "tdMode":"cross",
    "side":"buy",
    "ordType":"conditional",
    "sz":"2",
    "tpTriggerPx":"15",
    "tpOrdPx":"18"
}
```

**修改后**：添加了三个示例
1. **推荐方式 1**：只设置止盈（TP）的订单
2. **推荐方式 2**：只设置止损（SL）的订单
3. **⚠️ 警告示例**：同时设置 TP 和 SL 可能导致 TP 被忽略

#### 修改位置 2：API 限制说明（Line 15940-15951）

**修改前**：
```
When placing net TP/SL order (ordType=conditional) and both take-profit
and stop-loss parameters are sent, only stop-loss logic will be performed
and take-profit logic will be ignored.
```

**修改后**：
```
⚠️ IMPORTANT LIMITATION: When placing TP/SL order (ordType=conditional)
and both take-profit and stop-loss parameters are sent together, only
stop-loss logic will be performed and take-profit logic will be ignored.
This applies to:
- Net mode (single-direction position mode) - officially documented
- Long/short mode (hedge mode) - may also be affected in certain situations

RECOMMENDED SOLUTION: Place take-profit and stop-loss as TWO SEPARATE ORDERS
to ensure both are executed:
1. First order: Only include tpTriggerPx and tpOrdPx (no SL parameters)
2. Second order: Only include slTriggerPx and slOrdPx (no TP parameters)

This ensures both orders are active and will not interfere with each other.
```

**改进点**：
- ✅ 明确指出问题不仅限于 net 模式，hedge 模式也可能受影响
- ✅ 添加了醒目的警告标志 ⚠️
- ✅ 提供了明确的解决方案和步骤
- ✅ 格式化更清晰易读

### 2. `configs/config.template.yaml`

#### 修改位置：TPSL 配置注释（Line 84-102）

**修改前**：
```yaml
# Volatility percentage for stop-loss calculation (e.g., 0.01 = 1%)
# This is the base risk percentage before leverage adjustment
# Formula: SL_distance = entry_price × volatility_pct × leverage  ❌ 错误！
# Example: For 1% volatility with 5x leverage, actual position loss at SL is 5%
volatility_pct: 0.01

# Profit-loss ratio for take-profit calculation (e.g., 5.0 = 5:1 ratio)
# Take-profit distance is stop-loss distance multiplied by this ratio
# Example: If SL is 1%, TP will be 5% (with 5:1 ratio)
profit_loss_ratio: 5.0
```

**修改后**：
```yaml
# Volatility percentage for stop-loss calculation (e.g., 0.01 = 1%)
# This is the base risk percentage (NOT adjusted by leverage)
# Formula: SL_distance = entry_price × volatility_pct  ✅ 正确
# Example: For entry price $100, volatility 1%:
#   - Long position: SL = $99 (1% below entry)
#   - Short position: SL = $101 (1% above entry)
volatility_pct: 0.01

# Profit-loss ratio for take-profit calculation (e.g., 5.0 = 5:1 ratio)
# Formula: TP_distance = entry_price × volatility_pct × profit_loss_ratio
# Example: For entry price $100, volatility 1%, ratio 5:1:
#   - Long position: TP = $105 (5% above entry)
#   - Short position: TP = $95 (5% below entry)
#
# Note: TP and SL are placed as TWO SEPARATE ORDERS to ensure both work correctly
# (OKX API limitation: when both parameters sent together, only SL may execute)
profit_loss_ratio: 5.0
```

**改进点**：
- ✅ 修正了错误的公式（不包含杠杆）
- ✅ 提供了具体的数值示例（做多和做空）
- ✅ 添加了关于分开下单的说明
- ✅ 解释了为什么要分开下单（OKX API 限制）

### 3. `configs/config.yaml`

同上，应用了与 `config.template.yaml` 相同的修改。

## 关键更正

### 错误 1：止损计算公式包含杠杆
**错误公式**：
```
SL_distance = entry_price × volatility_pct × leverage
```

**正确公式**：
```
SL_distance = entry_price × volatility_pct
```

**说明**：
- 原公式会导致用户误以为杠杆会影响止损距离的计算
- 实际代码中从未使用杠杆参数计算止损距离
- 止损距离纯粹基于入场价和波动率百分比

### 错误 2：未充分说明 OKX API 的 TP/SL 限制
**原说明**：
- 仅提到 net 模式下的限制
- 没有提供解决方案
- 格式不够醒目

**改进后**：
- 明确说明 hedge 模式也可能受影响
- 提供了具体的解决方案（分开下单）
- 使用警告符号和醒目格式

### 错误 3：缺少实际计算示例
**原说明**：
- 只有抽象的公式
- 没有具体数值示例
- 不区分做多/做空

**改进后**：
- 提供了具体的数值示例（$100 入场价）
- 分别说明做多和做空的计算
- 更容易理解和验证

## 验证清单

更新后需要确认的事项：

- [x] OKX API 文档更新完成
- [x] 配置模板文件更新完成
- [x] 配置文件更新完成
- [ ] 用户手册需要相应更新（如果有）
- [ ] README 需要添加 TPSL 说明（建议）

## 后续建议

1. **创建用户指南**：
   - 详细说明 TPSL 功能的工作原理
   - 提供配置参数的调优建议
   - 添加常见问题解答

2. **示例配置**：
   - 提供不同风险偏好的配置示例
   - 保守型：volatility_pct=0.005, ratio=10:1
   - 激进型：volatility_pct=0.02, ratio=3:1

3. **监控建议**：
   - 建议用户在启用 debug 模式后观察几个周期
   - 在 OKX 交易所手动验证订单正确性
   - 记录实际触发的 TP/SL 价格与预期的对比

## 相关链接

- 代码修复：`internal/tpsl/manager.go`
- 修复总结：`TPSL_FIX_SUMMARY.md`
- OKX API 文档：`document/markdown/OKX_API.md`

---

**更新人**：Claude Code
**审核人**：待审核
