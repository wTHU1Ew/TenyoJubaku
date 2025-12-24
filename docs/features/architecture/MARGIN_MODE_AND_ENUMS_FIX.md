# Margin Mode Support and Type-Safe Enums Implementation

## 实施日期 / Implementation Date
2025-12-02

## 问题描述 / Problem Description

### 用户反馈 / User Feedback
用户发现止盈止损订单被创建为**全仓 5x 杠杆**，但实际持仓是**逐仓 10x 杠杆**。

### 根本原因 / Root Cause

1. **缺少 MarginMode 字段**：
   - Position 模型没有存储 `margin_mode`（交易模式）
   - OKX API 返回了 `"mgnMode": "isolated"`，但我们没有保存
   - 代码中硬编码了 `TdMode: "cross"`

2. **类型安全问题**：
   - 使用字符串字面量容易拼写错误
   - 没有编译时类型检查
   - 例如：`"cross"`, `"isolated"`, `"long"`, `"short"` 等都是字符串

## 解决方案 / Solution

### 1. 创建类型安全的枚举 / Type-Safe Enums

创建了新文件 `pkg/models/enums.go`，定义了以下枚举类型：

#### MarginMode（保证金模式）
```go
type MarginMode string

const (
    MarginModeCross    MarginMode = "cross"     // 全仓
    MarginModeIsolated MarginMode = "isolated"  // 逐仓
)

func (m MarginMode) String() string
func (m MarginMode) IsValid() bool
```

#### PositionSide（持仓方向）
```go
type PositionSide string

const (
    PositionSideLong  PositionSide = "long"   // 多头
    PositionSideShort PositionSide = "short"  // 空头
    PositionSideNet   PositionSide = "net"    // 单向持仓
)

func (p PositionSide) String() string
func (p PositionSide) IsValid() bool
```

#### OrderSide（订单方向）
```go
type OrderSide string

const (
    OrderSideBuy  OrderSide = "buy"   // 买入
    OrderSideSell OrderSide = "sell"  // 卖出
)

func (o OrderSide) String() string
func (o OrderSide) IsValid() bool
```

#### OrderType（订单类型）
```go
type OrderType string

const (
    OrderTypeConditional OrderType = "conditional"  // 条件单
    OrderTypeMarket      OrderType = "market"       // 市价单
    OrderTypeLimit       OrderType = "limit"        // 限价单
)

func (o OrderType) String() string
func (o OrderType) IsValid() bool
```

#### TriggerPriceType（触发价格类型）
```go
type TriggerPriceType string

const (
    TriggerPriceTypeLast  TriggerPriceType = "last"   // 最新价
    TriggerPriceTypeIndex TriggerPriceType = "index"  // 指数价
    TriggerPriceTypeMark  TriggerPriceType = "mark"   // 标记价
)

func (t TriggerPriceType) String() string
func (t TriggerPriceType) IsValid() bool
```

### 2. 更新 Position 模型

**文件**：`pkg/models/position.go`

```go
type Position struct {
    ID            int64        `json:"id" db:"id"`
    Timestamp     time.Time    `json:"timestamp" db:"timestamp"`
    Instrument    string       `json:"instrument" db:"instrument"`
    PositionSide  PositionSide `json:"position_side" db:"position_side"`  // 使用枚举
    PositionSize  float64      `json:"position_size" db:"position_size"`
    AveragePrice  float64      `json:"average_price" db:"average_price"`
    UnrealizedPnL float64      `json:"unrealized_pnl" db:"unrealized_pnl"`
    Margin        float64      `json:"margin" db:"margin"`
    Leverage      float64      `json:"leverage" db:"leverage"`
    MarginMode    MarginMode   `json:"margin_mode" db:"margin_mode"`      // 新增字段，使用枚举
}
```

**验证改进**：
```go
func (p *Position) Validate() error {
    // ...
    if !p.PositionSide.IsValid() {
        return fmt.Errorf("invalid position_side: %s", p.PositionSide)
    }
    if p.MarginMode != "" && !p.MarginMode.IsValid() {
        return fmt.Errorf("invalid margin_mode: %s", p.MarginMode)
    }
    return nil
}
```

### 3. 更新数据库 Schema

**文件**：`internal/storage/storage.go`

```sql
CREATE TABLE IF NOT EXISTS positions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    instrument VARCHAR(50) NOT NULL,
    position_side VARCHAR(10) NOT NULL,
    position_size REAL NOT NULL,
    average_price REAL NOT NULL,
    unrealized_pnl REAL NOT NULL,
    margin REAL NOT NULL,
    leverage REAL,
    margin_mode VARCHAR(10) DEFAULT 'cross'  -- 新增列
);
```

**INSERT 语句更新**：
```go
query := `
    INSERT INTO positions (timestamp, instrument, position_side, position_size,
                          average_price, unrealized_pnl, margin, leverage, margin_mode)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`
```

### 4. 更新监控服务

**文件**：`internal/monitor/monitor.go`

```go
// Get margin mode from OKX API response
marginMode := models.MarginMode(pos.MgnMode)
if marginMode == "" {
    marginMode = models.MarginModeCross // Default
}

// Normalize position side
posSide := models.PositionSide(pos.PosSide)
if posSide == "" {
    posSide = models.PositionSideNet // Default for one-way mode
}

positionModel := &models.Position{
    // ...
    PositionSide: posSide,
    MarginMode:   marginMode,
}
```

### 5. 更新 TPSL Manager

**文件**：`internal/tpsl/manager.go`

#### 使用持仓的 MarginMode

```go
// Determine trade mode from position (not hardcoded!)
tdMode := position.MarginMode.String()
if tdMode == "" {
    tdMode = models.MarginModeCross.String()
}

// Place orders with correct margin mode
tpReq := okx.AlgoOrderRequest{
    InstId:  position.Instrument,
    TdMode:  tdMode,  // 使用持仓的实际模式
    // ...
}
```

#### 使用枚举类型比较

```go
// Old way (string literal - error prone)
if position.PositionSide == "long" {  // ❌ 拼写错误风险
    return true
}

// New way (enum - type safe)
if position.PositionSide == models.PositionSideLong {  // ✅ 编译时检查
    return true
}
```

#### 转换为字符串

```go
// When passing to OKX API
PosSide: position.PositionSide.String(),
```

## 优势 / Benefits

### 1. 类型安全 / Type Safety
```go
// ❌ 编译通过，但运行时错误
position.PositionSide = "longg"  // 拼写错误

// ✅ 编译时错误
position.PositionSide = models.PositionSideLongg  // 不存在该常量
```

### 2. IDE 自动补全 / IDE Auto-completion
```go
// 输入 models.PositionSide
// IDE 会提示：
// - PositionSideLong
// - PositionSideShort
// - PositionSideNet
```

### 3. 代码可读性 / Code Readability
```go
// Before
if position.PositionSide == "long" {  // 普通字符串

// After
if position.PositionSide == models.PositionSideLong {  // 明确的枚举值
```

### 4. 重构安全性 / Refactoring Safety
如果将来需要更改枚举值（如 OKX API 更新），只需修改一处常量定义，所有使用处都会自动更新。

### 5. 运行时验证 / Runtime Validation
```go
if !marginMode.IsValid() {
    return fmt.Errorf("invalid margin mode: %s", marginMode)
}
```

## 修复前后对比 / Before and After

### 修复前 / Before

```go
// TPSL Manager (硬编码)
tpReq := okx.AlgoOrderRequest{
    InstId:  position.Instrument,
    TdMode:  "cross",  // ❌ 硬编码，无论实际持仓是什么模式
    PosSide: position.PositionSide,  // string 类型
    // ...
}

// Position Model
type Position struct {
    // ...
    PositionSide string  // ❌ 无类型约束
    // MarginMode字段不存在 ❌
}

// 比较 (容易拼写错误)
if position.PositionSide == "long" {  // ❌ 字符串字面量
    // ...
}
```

### 修复后 / After

```go
// TPSL Manager (使用持仓实际模式)
tdMode := position.MarginMode.String()
if tdMode == "" {
    tdMode = models.MarginModeCross.String()
}

tpReq := okx.AlgoOrderRequest{
    InstId:  position.Instrument,
    TdMode:  tdMode,  // ✅ 使用持仓实际的交易模式
    PosSide: position.PositionSide.String(),  // ✅ 枚举类型
    // ...
}

// Position Model
type Position struct {
    // ...
    PositionSide PositionSide  // ✅ 枚举类型
    MarginMode   MarginMode    // ✅ 新字段，枚举类型
}

// 比较 (类型安全)
if position.PositionSide == models.PositionSideLong {  // ✅ 枚举常量
    // ...
}
```

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

**预期行为**：
1. 监控服务从 OKX 获取持仓时，会正确存储 `margin_mode`
2. TPSL 管理器创建订单时，会使用持仓的实际 `margin_mode`
3. 逐仓 10x 持仓 → 创建逐仓止盈止损订单
4. 全仓 5x 持仓 → 创建全仓止盈止损订单

### 验证方法 / Verification Methods

1. **查看数据库**：
```bash
sqlite3 data/tenyojubaku.db "SELECT instrument, position_side, margin_mode, leverage FROM positions ORDER BY timestamp DESC LIMIT 5;"
```

2. **查看日志**：
```bash
tail -f logs/app.log | grep -E "(margin_mode|MarginMode|tdMode)"
```

3. **查看 OKX API 响应**（debug 模式）：
```json
{
  "mgnMode": "isolated",
  "lever": "10",
  ...
}
```

4. **验证止盈止损订单**：
   - 登录 OKX 交易所
   - 查看条件单
   - 确认交易模式与持仓一致

## 枚举类型使用指南 / Enum Usage Guide

### 创建持仓 / Creating Position
```go
position := &models.Position{
    PositionSide: models.PositionSideLong,    // 使用枚举
    MarginMode:   models.MarginModeIsolated,  // 使用枚举
    // ...
}
```

### 比较 / Comparison
```go
if position.MarginMode == models.MarginModeCross {
    // 全仓模式逻辑
}

switch position.PositionSide {
case models.PositionSideLong:
    // 多头逻辑
case models.PositionSideShort:
    // 空头逻辑
case models.PositionSideNet:
    // 单向持仓逻辑
}
```

### 转换为字符串 / Converting to String
```go
// 传递给 API
apiRequest := SomeRequest{
    Side: position.PositionSide.String(),
    Mode: position.MarginMode.String(),
}

// 日志输出（自动调用 String()）
log.Printf("Position side: %s, margin mode: %s",
    position.PositionSide, position.MarginMode)
```

### 从字符串创建 / Creating from String
```go
// 从 API 响应
marginMode := models.MarginMode(apiResponse.MgnMode)
if !marginMode.IsValid() {
    marginMode = models.MarginModeCross  // 使用默认值
}

posSide := models.PositionSide(apiResponse.PosSide)
if !posSide.IsValid() {
    return fmt.Errorf("invalid position side: %s", apiResponse.PosSide)
}
```

### 验证 / Validation
```go
if !position.MarginMode.IsValid() {
    return fmt.Errorf("invalid margin mode")
}

if !position.PositionSide.IsValid() {
    return fmt.Errorf("invalid position side")
}
```

## 未来扩展 / Future Enhancements

### 1. 更多枚举类型 / Additional Enum Types
可以考虑为以下字段添加枚举：
- `InstrumentType`：SPOT, SWAP, FUTURES, OPTION
- `OrderState`：live, partially_filled, filled, canceled
- `AlgoOrderType`：trigger, oco, conditional, etc.

### 2. 字符串到枚举的辅助函数 / Helper Functions
```go
func ParseMarginMode(s string) (MarginMode, error) {
    mode := MarginMode(s)
    if !mode.IsValid() {
        return "", fmt.Errorf("invalid margin mode: %s", s)
    }
    return mode, nil
}
```

### 3. JSON/DB 自定义序列化 / Custom Serialization
```go
func (m *MarginMode) Scan(value interface{}) error {
    // 从数据库读取时的验证
}

func (m MarginMode) Value() (driver.Value, error) {
    // 写入数据库时的验证
}
```

## 相关文档 / Related Documentation

1. **TPSL 修复总结**: `TPSL_FIX_SUMMARY.md`
2. **价格验证功能**: `TPSL_PRICE_VALIDATION_SUMMARY.md`
3. **覆盖分析修复**: `TPSL_COVERAGE_BUG_FIX.md`
4. **OKX API 约束**: `TPSL_OKX_API_CONSTRAINT_FIX.md`

## 历史问题回顾 / Historical Issues

这是 Feature 2 (Automatic TPSL) 的第**五次**重要修复：

1. **第一次** (2025-12-01): 分开下单
2. **第二次** (2025-12-02): 价格验证
3. **第三次** (2025-12-02): 覆盖分析
4. **第四次** (2025-12-02): OKX 价格约束
5. **第五次** (2025-12-02): Margin Mode + 枚举类型 ⬅️ 当前

## 经验教训 / Lessons Learned

1. **读取完整的 API 响应**
   - OKX API 返回了很多字段，不能只选择部分
   - `mgnMode`, `lever` 等重要字段不应忽略

2. **避免硬编码配置**
   - 不要假设所有持仓都是全仓模式
   - 应该从实际数据中获取配置

3. **类型安全的价值**
   - 枚举类型可以在编译时捕获错误
   - IDE 自动补全大大提高开发效率
   - 代码可读性和可维护性显著提升

4. **数据模型的重要性**
   - 数据模型应该反映实际业务需求
   - 遗漏重要字段会导致功能缺陷

## 实施人员 / Implementation
- **发现**: 用户报告
- **分析**: Claude Code
- **设计**: Claude Code (基于用户建议使用 enum)
- **实施**: Claude Code
- **测试**: 待用户验证 / Pending User Verification

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Feature 2 - Automatic TPSL Management (Enhancement)
- **版本**: v1.2.0 (Margin Mode Support + Type-Safe Enums)
- **日期**: 2025-12-02
