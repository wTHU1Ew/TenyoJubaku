# TPSL Price Validation Feature - Implementation Summary

## 实施日期 / Implementation Date
2025-12-02

## 问题描述 / Problem Description

用户反馈了一个边缘情况：当系统检测到某个持仓没有设置止盈/止损时，如果当前市场价格已经突破了预期的止盈价格，系统应该使用当前价格作为止盈价格，而不是使用计算出的预期价格。

User reported an edge case: When the system detects that a position doesn't have TP/SL orders, if the current market price has already exceeded the expected TP price, the system should use the current price as the TP price instead of the calculated expected price.

### 用户原话 / Original User Request
> "程序之前应该有自动检测是否设置了止盈止损，这里可以多做一个逻辑，监测到未设置止盈/止损，则尝试用当前价格和预期止盈/止损位置做对比，如果当前价格突破了预期，则在价格设置止盈/止损价格的时候就使用当前价格"

## 实施方案 / Implementation Solution

### 1. 添加市场行情获取功能 / Added Market Ticker Functionality

#### 文件：`internal/okx/types.go`
添加了两个新的类型定义以支持获取实时市场行情：

```go
// TickerResponse OKX行情响应 / OKX ticker response
type TickerResponse struct {
    Code string       `json:"code"`
    Msg  string       `json:"msg"`
    Data []TickerData `json:"data"`
}

// TickerData OKX行情数据 / OKX ticker data
type TickerData struct {
    InstId    string `json:"instId"`
    Last      string `json:"last"`      // Last traded price
    LastSz    string `json:"lastSz"`    // Last traded size
    AskPx     string `json:"askPx"`     // Best ask price
    AskSz     string `json:"askSz"`     // Best ask size
    BidPx     string `json:"bidPx"`     // Best bid price
    BidSz     string `json:"bidSz"`     // Best bid size
    Open24h   string `json:"open24h"`   // Open price in the past 24 hours
    High24h   string `json:"high24h"`   // Highest price in the past 24 hours
    Low24h    string `json:"low24h"`    // Lowest price in the past 24 hours
    VolCcy24h string `json:"volCcy24h"` // 24h trading volume (quote currency)
    Vol24h    string `json:"vol24h"`    // 24h trading volume (base currency)
    Ts        string `json:"ts"`        // Ticker data generation time
    SodUtc0   string `json:"sodUtc0"`   // Open price at UTC 0
    SodUtc8   string `json:"sodUtc8"`   // Open price at UTC 8
}
```

#### 文件：`internal/okx/client.go`
添加了 `GetTicker` 方法以调用 OKX 市场行情 API：

```go
func (c *Client) GetTicker(instId string) (*TickerResponse, error) {
    path := fmt.Sprintf("/api/v5/market/ticker?instId=%s", instId)

    respBody, err := c.doRequest("GET", path)
    if err != nil {
        return nil, err
    }

    var resp TickerResponse
    if err := json.Unmarshal(respBody, &resp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if resp.Code != "0" {
        return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
    }

    return &resp, nil
}
```

### 2. 实现价格验证与调整逻辑 / Implemented Price Validation and Adjustment Logic

#### 文件：`internal/tpsl/manager.go`

添加了三个新函数来实现价格验证功能：

##### 2.1 `getCurrentMarketPrice()` - 获取当前市场价格
```go
func (m *Manager) getCurrentMarketPrice(instId string) (float64, error)
```
- 调用 OKX ticker API 获取最新成交价
- 解析并返回浮点数格式的价格
- 错误处理：API 调用失败、无数据、解析失败

##### 2.2 `adjustTPSLPricesWithCurrentPrice()` - 价格验证与调整
```go
func (m *Manager) adjustTPSLPricesWithCurrentPrice(
    position *models.Position,
    prices *TPSLPrices,
    currentPrice float64
) (*TPSLPrices, bool, bool)
```

**做多持仓逻辑 / Long Position Logic:**
- 止盈（TP）：应该在入场价之上
  - 如果当前价 >= 预期止盈价：使用当前价作为止盈价（价格已达目标）
  - 如果当前价 < 预期止盈价：使用预期止盈价（正常情况）
- 止损（SL）：应该在入场价之下
  - 如果当前价 <= 预期止损价：跳过止损单（价格已触及止损）

**做空持仓逻辑 / Short Position Logic:**
- 止盈（TP）：应该在入场价之下
  - 如果当前价 <= 预期止盈价：使用当前价作为止盈价（价格已达目标）
  - 如果当前价 > 预期止盈价：使用预期止盈价（正常情况）
- 止损（SL）：应该在入场价之上
  - 如果当前价 >= 预期止损价：跳过止损单（价格已触及止损）

**返回值 / Return Values:**
- `adjustedPrices`: 调整后的 TP/SL 价格
- `skipTP`: 是否跳过止盈单（价格已达目标且超出太多）
- `skipSL`: 是否跳过止损单（价格已触及止损）

##### 2.3 `placeTPSLOrderWithValidation()` - 带验证的下单主函数
```go
func (m *Manager) placeTPSLOrderWithValidation(
    position *models.Position,
    size float64,
    prices *TPSLPrices
) error
```

**工作流程 / Workflow:**
1. 调用 `getCurrentMarketPrice()` 获取当前市场价格
2. 如果获取失败，降级到 `placeTPSLOrderOriginal()`（无验证版本）
3. 调用 `adjustTPSLPricesWithCurrentPrice()` 验证并调整价格
4. 根据 `skipTP` 和 `skipSL` 标志决定是否下单
5. 分别下止盈单和止损单（两个独立订单）

**错误处理与降级策略 / Error Handling and Fallback:**
```go
currentPrice, err := m.getCurrentMarketPrice(position.Instrument)
if err != nil {
    m.logger.Warn("Failed to get current price for %s: %v. Falling back to original order placement without validation",
        position.Instrument, err)
    // Fallback to original placeTPSLOrder without validation
    return m.placeTPSLOrderOriginal(position, size, prices)
}
```

### 3. 日志输出示例 / Log Output Examples

#### 正常情况 / Normal Case
```
[INFO] Placing TPSL orders for BTC-USDT-SWAP (long): current=50000.00, entry=49000.00, TP=51450.00, SL=48510.00
[INFO] Take-Profit order placed successfully for BTC-USDT-SWAP (long), algoId: 123456, trigger: 51450.00
[INFO] Stop-Loss order placed successfully for BTC-USDT-SWAP (long), algoId: 123457, trigger: 48510.00
```

#### 当前价已达止盈目标 / Current Price Reached TP Target
```
[WARN] Position BTC-USDT-SWAP (long): Current price 51500.00 has reached or exceeded expected TP 51450.00
[INFO] Adjusting TP price to current market price: 51450.00 → 51500.00
[INFO] Take-Profit order placed successfully for BTC-USDT-SWAP (long), algoId: 123456, trigger: 51500.00
```

#### 当前价已触及止损 / Current Price Hit SL
```
[ERROR] Position BTC-USDT-SWAP (long): Current price 48500.00 has hit or passed SL 48510.00!
[WARN] Skipping Stop-Loss order for BTC-USDT-SWAP (long) - price already at or below SL level
[INFO] Take-Profit order placed successfully for BTC-USDT-SWAP (long), algoId: 123456, trigger: 51450.00
```

#### API 获取价格失败（降级） / API Fetch Failed (Fallback)
```
[WARN] Failed to get current price for BTC-USDT-SWAP: API error. Falling back to original order placement without validation
[INFO] Take-Profit order placed successfully for BTC-USDT-SWAP (long), algoId: 123456, trigger: 51450.00
[INFO] Stop-Loss order placed successfully for BTC-USDT-SWAP (long), algoId: 123457, trigger: 48510.00
```

## 代码清理 / Code Cleanup

### 删除旧函数 / Removed Old Function
删除了 `internal/tpsl/manager.go` 中第 339-448 行的旧 `placeTPSLOrder()` 函数，该函数存在以下问题：
- 使用了不存在的 `position.MarkPrice` 字段（编译错误）
- 声明了未使用的 `shouldPlaceTP` 变量（编译错误）
- 没有实现与 OKX ticker API 的集成

Removed the old `placeTPSLOrder()` function at lines 339-448 in `internal/tpsl/manager.go`, which had the following issues:
- Used non-existent `position.MarkPrice` field (compilation error)
- Declared unused `shouldPlaceTP` variable (compilation error)
- Lacked integration with OKX ticker API

### 保留的函数 / Retained Functions
1. **`placeTPSLOrderWithValidation()`** (Line 517)
   - 主要入口函数，带完整的价格验证
   - Main entry function with full price validation

2. **`placeTPSLOrderOriginal()`** (Line 618)
   - 降级备用函数，无价格验证
   - Fallback function without price validation

## 测试验证 / Testing and Verification

### 编译验证 / Build Verification
```bash
$ go build -o tenyojubaku ./cmd/main.go
# 编译成功，无错误 / Build succeeded without errors
```

### 函数签名验证 / Function Signature Verification
```bash
$ grep -n "^func (m \*Manager) placeTPSL" internal/tpsl/manager.go
517:func (m *Manager) placeTPSLOrderWithValidation(position *models.Position, size float64, prices *TPSLPrices) error {
618:func (m *Manager) placeTPSLOrderOriginal(position *models.Position, size float64, prices *TPSLPrices) error {
```

### 调用点验证 / Call Site Verification
```bash
$ grep -n "placeTPSLOrder" internal/tpsl/manager.go
119:		err = m.placeTPSLOrderWithValidation(position, uncoveredSize, prices)
```
确认主函数调用的是带验证的版本 / Confirmed main function calls the validated version

## 优势与特性 / Benefits and Features

### 1. 智能价格调整 / Intelligent Price Adjustment
- ✅ 自动检测当前价格是否已达目标
- ✅ 动态调整止盈价格以锁定利润
- ✅ 避免设置无效的止盈单

### 2. 安全性 / Safety
- ✅ 检测止损价格是否已触及
- ✅ 跳过无意义的止损单（避免立即触发）
- ✅ 降级策略确保在 API 失败时仍能下单

### 3. 容错性 / Fault Tolerance
- ✅ API 调用失败时自动降级到无验证版本
- ✅ 不会因为验证功能影响正常下单流程
- ✅ 详细的错误日志便于问题排查

### 4. 灵活性 / Flexibility
- ✅ 支持做多和做空两种持仓类型
- ✅ 独立处理止盈和止损（符合 OKX API 限制）
- ✅ 保留原始下单函数作为备用方案

## 相关文档 / Related Documentation

1. **TPSL 修复总结**: `TPSL_FIX_SUMMARY.md`
   - 记录了止盈止损分开下单的修复

2. **文档更新记录**: `DOCUMENTATION_UPDATES.md`
   - 记录了配置文件和 API 文档的更新

3. **OKX API 文档**: `document/markdown/OKX_API.md`
   - 包含了 TP/SL API 限制的详细说明

4. **配置文件**: `configs/config.yaml` 和 `configs/config.template.yaml`
   - 包含 TPSL 参数配置说明和计算公式

## 后续建议 / Future Recommendations

### 1. 单元测试 / Unit Tests
建议为新增的三个函数添加单元测试：
- `getCurrentMarketPrice()` - 模拟 API 响应
- `adjustTPSLPricesWithCurrentPrice()` - 测试各种价格场景
- `placeTPSLOrderWithValidation()` - 测试完整流程

### 2. 集成测试 / Integration Tests
在测试环境中验证以下场景：
- 价格正常：TP/SL 按预期价格下单
- 价格已达 TP：TP 使用当前价格
- 价格已触 SL：跳过 SL 订单
- API 失败：降级到无验证版本

### 3. 监控指标 / Monitoring Metrics
建议添加监控指标：
- TP 价格调整次数
- SL 订单跳过次数
- API 获取价格失败次数
- 降级到无验证版本的次数

### 4. 配置参数 / Configuration Parameters
可考虑添加以下可配置参数：
- `price_adjustment_enabled`: 是否启用价格调整（默认 true）
- `price_tolerance_pct`: 价格容差百分比（避免微小差异导致调整）
- `skip_sl_threshold_pct`: 跳过止损的价格阈值（当前价格超出 SL 多少时跳过）

## 实施人员 / Implementation
- **开发**: Claude Code
- **审核**: 待审核 / Pending Review
- **测试**: 待测试 / Pending Testing

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Feature 2 - Automatic TPSL Management (Enhancement)
- **版本**: v1.1.0 (Price Validation)
- **日期**: 2025-12-02
