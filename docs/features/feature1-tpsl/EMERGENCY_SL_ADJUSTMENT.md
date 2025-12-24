# Emergency Stop-Loss Price Adjustment

## 实施日期 / Implementation Date
2025-12-02

## 需求描述 / Requirement

### 用户反馈 / User Feedback
> "设置止损之前也要和设置止盈一样先检查和当前价格的对比，如果当前价格突破了止损价格，则以当前价格为准进行止损，需要尽可能让止损触发，我可以接受额外亏一点点"

用户希望：
1. ✅ 当前价格已突破预期止损价时，仍然设置止损（不是跳过）
2. ✅ 使用略微不利的价格，确保止损能够触发
3. ✅ 愿意接受额外的小额损失，以确保持仓被保护

## 实施方案 / Implementation

### 修改前 / Before

```go
// 当价格触及止损时，直接跳过设置止损
if currentPrice <= prices.SlPrice {  // 做多
    m.logger.Error("Position has hit SL! Manual intervention required")
    skipSL = true  // ❌ 跳过止损，持仓没有保护
}
```

**问题**：
- 持仓没有任何止损保护
- 继续亏损无法自动平仓
- 需要人工干预

### 修改后 / After

```go
// 当价格触及止损时，设置紧急止损
if currentPrice <= prices.SlPrice {  // 做多
    m.logger.Warn("Position has hit expected SL!")
    // 设置紧急止损：略低于当前价 (0.1%)
    adjustedPrice := currentPrice * 0.999
    m.logger.Info("Adjusting SL to: %.8f → %.8f", prices.SlPrice, adjustedPrice)
    m.logger.Warn("ALERT: Setting emergency SL at current price")
    adjustedPrices.SlPrice = adjustedPrice  // ✅ 设置紧急止损
}
```

**优势**：
- ✅ 持仓有止损保护
- ✅ 会很快触发（价格略微波动就会触发）
- ✅ 自动平仓，无需人工干预
- ✅ 额外损失很小（约 0.1%）

## 详细逻辑 / Detailed Logic

### 做多持仓 (Long Position)

**正常情况**（当前价 > 止损价）：
```
入场价：$100
预期止损：$99 (1% 止损)
当前价：$100.5 ✅ 正常
操作：设置止损 = $99
```

**紧急情况**（当前价 ≤ 止损价）：
```
入场价：$100
预期止损：$99 (1% 止损)
当前价：$98.5 ⚠️ 已触及止损！
操作：设置紧急止损 = $98.5 × 0.999 = $98.40
说明：价格再跌 0.1% 就会触发止损
```

### 做空持仓 (Short Position)

**正常情况**（当前价 < 止损价）：
```
入场价：$100
预期止损：$101 (1% 止损)
当前价：$99.5 ✅ 正常
操作：设置止损 = $101
```

**紧急情况**（当前价 ≥ 止损价）：
```
入场价：$100
预期止损：$101 (1% 止损)
当前价：$101.5 ⚠️ 已触及止损！
操作：设置紧急止损 = $101.5 × 1.001 = $101.60
说明：价格再涨 0.1% 就会触发止损
```

## OKX API 价格约束 / OKX API Price Constraints

### 做多持仓止损
- **要求**：SL < 当前价
- **紧急止损**：当前价 × 0.999（略低 0.1%）
- **示例**：当前价 $98.5 → SL $98.40 ✅

### 做空持仓止损
- **要求**：SL > 当前价
- **紧急止损**：当前价 × 1.001（略高 0.1%）
- **示例**：当前价 $101.5 → SL $101.60 ✅

## 日志示例 / Log Examples

### 做空持仓紧急止损

```
[WARN] Position BTC-USD-SWAP (short): Current price 92600.00 has hit or passed expected SL 92526.57!
[INFO] Adjusting SL price to slightly above current price: 92526.57 → 92692.60 (current: 92600.00)
[WARN] ALERT: Setting emergency SL at current price - position already in loss beyond expected SL
[INFO] Placing TPSL orders for BTC-USD-SWAP (short): TP=85108.81, SL=92692.60, current=92600.00

=== OKX API Debug ===
Request Body: {
  "instId": "BTC-USD-SWAP",
  "tdMode": "isolated",
  "side": "buy",
  "slTriggerPx": "92692.60",  // 略高于当前价
  "slOrdPx": "-1",
  "slTriggerPxType": "last"
}
Response: {"code":"0", "data":[{"algoId":"xxx","sCode":"0"}]}
=====================

[INFO] Stop-Loss order placed successfully, algoId: xxx, trigger: 92692.60
```

### 做多持仓紧急止损

```
[WARN] Position ETH-USDT-SWAP (long): Current price 2450.00 has hit or passed expected SL 2475.00!
[INFO] Adjusting SL price to slightly below current price: 2475.00 → 2447.55 (current: 2450.00)
[WARN] ALERT: Setting emergency SL at current price - position already in loss beyond expected SL
[INFO] Placing TPSL orders for ETH-USDT-SWAP (long): TP=2600.00, SL=2447.55, current=2450.00

=== OKX API Debug ===
Request Body: {
  "instId": "ETH-USDT-SWAP",
  "tdMode": "cross",
  "side": "sell",
  "slTriggerPx": "2447.55",  // 略低于当前价
  "slOrdPx": "-1",
  "slTriggerPxType": "last"
}
Response: {"code":"0", "data":[{"algoId":"yyy","sCode":"0"}]}
=====================

[INFO] Stop-Loss order placed successfully, algoId: yyy, trigger: 2447.55
```

## 触发时机 / Trigger Timing

### 正常止损
- 价格朝不利方向移动到预期止损价
- 例如：做多持仓从 $100 跌到 $99

### 紧急止损
- 当检查时发现价格已经超过预期止损
- 设置的紧急止损会很快触发（0.1% 波动）
- 最小化进一步损失

## 额外损失计算 / Additional Loss Calculation

### 示例：做空持仓

```
持仓大小：2 BTC
入场价：$91,610
预期止损：$92,526 (1% 止损)
当前价：$92,600 (已超过预期止损 $74)

预期损失：2 × ($92,526 - $91,610) = $1,832
实际损失：2 × ($92,600 - $91,610) = $1,980
已发生额外损失：$148 (已经发生)

紧急止损设置：$92,600 × 1.001 = $92,692
如果触发紧急止损：2 × ($92,692 - $91,610) = $2,164
紧急止损额外损失：$2,164 - $1,832 = $332

总额外损失：$332
额外损失百分比：($92,692 - $92,526) / $92,526 ≈ 0.18%
```

### 用户接受的风险
- ✅ 约 0.1-0.2% 的额外损失
- ✅ 换取自动止损保护
- ✅ 避免继续无限制亏损

## 最佳实践 / Best Practices

### 1. 及时监控
建议缩短 TPSL 检查间隔：
```yaml
tpsl:
  check_interval: 60  # 1分钟检查一次（而不是5分钟）
```

### 2. 设置合理的止损距离
```yaml
tpsl:
  volatility_pct: 0.015  # 1.5% 止损（留出更多缓冲）
```

### 3. 启用价格告警
建议添加价格告警，当接近止损价时提前通知。

### 4. 紧急止损后的行动
看到紧急止损日志后：
1. 检查是否有其他持仓也需要保护
2. 评估市场趋势，是否需要调整策略
3. 考虑手动干预其他高风险持仓

## 相关配置 / Related Configuration

### 调整幅度配置（未来改进）
可以将 0.1% (0.001) 做成可配置参数：

```yaml
tpsl:
  # 紧急止损价格调整幅度（百分比）
  # Emergency SL price adjustment percentage
  # Higher value = more conservative (triggers easier, but more loss)
  # Lower value = more aggressive (may not trigger if price bounces back)
  emergency_sl_adjustment_pct: 0.001  # 0.1%
```

## 风险提示 / Risk Warning

### 优点 / Advantages
- ✅ 自动止损保护
- ✅ 避免继续亏损
- ✅ 无需人工干预
- ✅ 额外损失可控（~0.1%）

### 缺点 / Disadvantages
- ⚠️ 可能在短期回调后触发止损
- ⚠️ 承担额外的小额损失
- ⚠️ 价格剧烈波动时可能无法及时触发

### 适用场景 / Use Cases
- ✅ 持仓监控不够及时
- ✅ 接受小额额外损失换取保护
- ✅ 希望自动化风险管理
- ✅ 避免情绪化决策

### 不适用场景 / Not Suitable For
- ❌ 期望精确止损价格
- ❌ 不能接受任何额外损失
- ❌ 需要频繁手动调整策略
- ❌ 短期交易/剥头皮策略

## 相关文档 / Related Documentation

1. **止盈价格调整**: `TPSL_OKX_API_CONSTRAINT_FIX.md`
2. **价格验证功能**: `TPSL_PRICE_VALIDATION_SUMMARY.md`
3. **覆盖分析修复**: `TPSL_COVERAGE_BUG_FIX.md`
4. **Margin Mode支持**: `MARGIN_MODE_AND_ENUMS_FIX.md`

## 实施人员 / Implementation
- **需求**: 用户提出
- **设计**: Claude Code
- **实施**: Claude Code
- **测试**: 待用户验证 / Pending User Verification

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Feature 2 - Automatic TPSL Management (Enhancement)
- **版本**: v1.2.1 (Emergency SL Adjustment)
- **日期**: 2025-12-02
