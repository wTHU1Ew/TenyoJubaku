# Quick Fix: GetLatestPositions Missing margin_mode Column

## 问题 / Problem
尽管数据库中正确存储了 `margin_mode = "isolated"`，但 TPSL 订单仍然使用 `tdMode = "cross"`。

## 根本原因 / Root Cause
`GetLatestPositions()` SQL 查询没有选择 `margin_mode` 列：

```go
// ❌ 错误：缺少 margin_mode
SELECT id, timestamp, instrument, position_side, position_size,
       average_price, unrealized_pnl, margin, leverage
FROM positions
```

导致 Position 对象的 `MarginMode` 字段为空值，然后在 TPSL manager 中被默认为 "cross"。

## 修复 / Fix
**文件**: `internal/storage/storage.go`

### SELECT 查询
```go
// ✅ 正确：包含 margin_mode
SELECT id, timestamp, instrument, position_side, position_size,
       average_price, unrealized_pnl, margin, leverage, margin_mode
FROM positions
```

### Scan 语句
```go
// ✅ 添加 &p.MarginMode
rows.Scan(&p.ID, &timestamp, &p.Instrument, &p.PositionSide,
          &p.PositionSize, &p.AveragePrice, &p.UnrealizedPnL,
          &p.Margin, &p.Leverage, &p.MarginMode)
```

## 验证 / Verification
重新编译并运行后，应该看到：

```json
// OKX API Debug
Request Body: {
  "tdMode":"isolated",  // ✅ 正确！
  "lever":"10",
  ...
}
```

## 日期 / Date
2025-12-02
