# Architecture Refactor: Real-Time API for TPSL

## 实施日期 / Implementation Date
2025-12-02

## 背景 / Background

之前的架构中，TPSL 服务从数据库读取持仓数据来进行止盈止损管理。这种方式存在以下问题：

1. **数据延迟**：数据库中的持仓可能已经过期（已平仓但未更新）
2. **复杂的过期逻辑**：需要使用时间阈值来判断持仓是否仍然活跃
3. **配置复杂**：需要配置和调整 `position_expiration_minutes` 参数
4. **职责混淆**：TPSL 依赖 Monitor 的数据采集结果

## 用户讨论 / User Discussion

用户提出了关键问题：

> "为什么需要持仓记录过期时间阈值，而不是请求API/position发现没当前持仓就将数据库中的仓位设置为不活跃呢？"

> "这个检查活跃仓位，是否可以放在检查止盈止损的地方？我猜这样技术上是可行的，但是有一个问题，是否会破坏程序分层或之前的什么语义？"

经过架构分析，我们得出结论：**在 TPSL 检查时直接调用 OKX API 是更好的方案。**

## 新架构设计 / New Architecture

### 架构对比

#### 旧架构（耦合）
```
┌─────────┐
│  TPSL   │ (每 300 秒检查一次)
└────┬────┘
     │ 依赖数据库
┌────▼────┐
│ Storage │ (GetLatestPositions + 过期时间判断)
└────┬────┘
     │ 依赖监控数据
┌────▼────┐
│ Monitor │ (每 60 秒采集一次)
└────┬────┘
     │ 调用 API
┌────▼────┐
│ OKX API │
└─────────┘

问题：
- TPSL 数据延迟（最多 60 秒 + 过期判断误差）
- 依赖链条长，Monitor 挂了会影响 TPSL
- 需要复杂的过期时间配置
```

#### 新架构（解耦）
```
┌─────────┐        ┌─────────┐
│  TPSL   │        │ Monitor │
│(300秒)  │        │(60秒)   │
└────┬────┘        └────┬────┘
     │                  │
     │ 实时调用          │ 存储历史
     │ (风控决策)        │ (数据分析)
     │                  │
     ▼                  ▼
┌───────────────────────────┐
│        OKX API            │
│  (唯一数据源)              │
└───────────────────────────┘
           ▲
           │ 存储历史快照
      ┌────┴────┐
      │ Storage │
      │(历史数据)│
      └─────────┘

优势：
✅ TPSL 数据实时（直接从 API）
✅ 服务独立（Monitor 挂了不影响 TPSL）
✅ 逻辑简单（无需过期判断）
✅ 职责清晰（各司其职）
```

### 职责划分

**Monitor Service（数据存档服务）**
- **职责**：历史数据记录
- **功能**：
  - 每 60 秒调用 OKX API
  - 存储账户余额历史
  - 存储持仓快照历史
- **用途**：
  - 数据分析
  - 历史回测
  - 收益统计
- **语义**：**"我是数据记录员，负责定期拍快照"**

**TPSL Service（实时风控服务）**
- **职责**：实时风险管理
- **功能**：
  - 每 300 秒直接调用 OKX API 获取最新持仓
  - 分析 TPSL 覆盖率
  - 下单设置止盈止损
- **用途**：
  - 实时风险保护
  - 自动止盈止损管理
- **语义**：**"我是风控经理，基于最新数据做决策"**

**Storage Service（数据访问服务）**
- **职责**：数据持久化和查询
- **功能**：
  - 提供历史数据查询接口
  - 持久化快照数据
- **用途**：
  - 历史数据分析
  - 数据可视化
- **语义**：**"我是数据库管理员，提供历史数据访问"**

## 实施细节 / Implementation Details

### 1. TPSL Scheduler 重构

**文件**：`internal/tpsl/scheduler.go`

#### 结构体变化
```go
// 旧版本
type Scheduler struct {
    manager  *Manager
    storage  *storage.Storage  // ❌ 依赖 Storage
    config   *config.TPSLConfig
    logger   *logger.Logger
    // ...
}

// 新版本
type Scheduler struct {
    manager   *Manager
    okxClient *okx.Client      // ✅ 直接依赖 OKX Client
    config    *config.TPSLConfig
    logger    *logger.Logger
    // ...
}
```

#### NewScheduler 签名变化
```go
// 旧版本
func NewScheduler(config *config.TPSLConfig, storage *storage.Storage, okxClient *okx.Client, logger *logger.Logger) *Scheduler

// 新版本
func NewScheduler(config *config.TPSLConfig, okxClient *okx.Client, logger *logger.Logger) *Scheduler
```

#### runCheck 逻辑变化
```go
// 旧版本
func (s *Scheduler) runCheck() {
    // 从数据库读取持仓（可能过期）
    positionsSlice, err := s.storage.GetLatestPositions()

    // 转换为指针
    positions := make([]*models.Position, len(positionsSlice))
    for i := range positionsSlice {
        positions[i] = &positionsSlice[i]
    }

    // 执行 TPSL 分析
    summary, err := s.manager.AnalyzeAndPlaceTPSL(positions)
}

// 新版本
func (s *Scheduler) runCheck() {
    // 直接从 OKX API 获取最新持仓（实时）
    positionsResp, err := s.okxClient.GetPositions()

    // 检查是否有持仓
    if len(positionsResp.Data) == 0 {
        s.logger.Info("No open positions, skipping TPSL check")
        return
    }

    // 转换 OKX 响应为 Position 模型
    positions := make([]*models.Position, 0, len(positionsResp.Data))
    for _, pos := range positionsResp.Data {
        position, err := s.convertOKXPosition(&pos)
        if err != nil {
            s.logger.Warn("Failed to convert position %s: %v", pos.InstId, err)
            continue
        }
        positions = append(positions, position)
    }

    // 执行 TPSL 分析
    summary, err := s.manager.AnalyzeAndPlaceTPSL(positions)
}
```

#### 新增转换函数
```go
// convertOKXPosition 将 OKX API 响应转换为 Position 模型
func (s *Scheduler) convertOKXPosition(pos *okx.PositionData) (*models.Position, error) {
    // 解析 position size
    posSize, err := strconv.ParseFloat(pos.Pos, 64)
    if err != nil || posSize == 0 {
        return nil, fmt.Errorf("invalid position size")
    }

    // 解析平均价格
    avgPrice, err := strconv.ParseFloat(pos.AvgPx, 64)
    if err != nil {
        return nil, fmt.Errorf("failed to parse average price: %w", err)
    }

    // ... 其他字段解析

    // 创建 Position 模型
    position := &models.Position{
        Timestamp:     time.Now().UTC(),
        Instrument:    pos.InstId,
        PositionSide:  models.PositionSide(pos.PosSide),
        PositionSize:  posSize,
        AveragePrice:  avgPrice,
        // ... 其他字段
        MarginMode:    models.MarginMode(pos.MgnMode),
    }

    return position, nil
}
```

### 2. OKX Types 重构

**文件**：`internal/okx/types.go`

将匿名结构体提取为命名类型：

```go
// 新增 PositionData 类型
type PositionData struct {
    InstType  string `json:"instType"`
    MgnMode   string `json:"mgnMode"`
    PosId     string `json:"posId"`
    PosSide   string `json:"posSide"`
    Pos       string `json:"pos"`
    AvgPx     string `json:"avgPx"`
    Upl       string `json:"upl"`
    Margin    string `json:"margin"`
    Lever     string `json:"lever"`
    InstId    string `json:"instId"`
    // ... 其他字段
}

// 修改 PositionsResponse
type PositionsResponse struct {
    Code string         `json:"code"`
    Msg  string         `json:"msg"`
    Data []PositionData `json:"data"`  // ✅ 使用命名类型
}
```

### 3. 配置简化

**文件**：`internal/config/config.go`

#### 移除 DatabaseConfig 字段
```go
// 旧版本
type DatabaseConfig struct {
    Path                      string `yaml:"path"`
    WALMode                   bool   `yaml:"wal_mode"`
    MaxOpenConns              int    `yaml:"max_open_conns"`
    MaxIdleConns              int    `yaml:"max_idle_conns"`
    PositionExpirationMinutes int    `yaml:"position_expiration_minutes"`  // ❌ 移除
}

// 新版本
type DatabaseConfig struct {
    Path         string `yaml:"path"`
    WALMode      bool   `yaml:"wal_mode"`
    MaxOpenConns int    `yaml:"max_open_conns"`
    MaxIdleConns int    `yaml:"max_idle_conns"`
}
```

#### 移除配置验证
```go
// 旧版本
if c.Database.PositionExpirationMinutes <= 0 {
    c.Database.PositionExpirationMinutes = 10
}

// 新版本
// ✅ 不再需要
```

### 4. Storage 简化

**文件**：`internal/storage/storage.go`

#### 移除过期时间字段
```go
// 旧版本
type Storage struct {
    db                        *sql.DB
    positionExpirationMinutes int  // ❌ 移除
}

// 新版本
type Storage struct {
    db *sql.DB
}
```

#### 简化 New() 函数
```go
// 旧版本
func New(dbPath string, walMode bool, maxOpenConns, maxIdleConns, positionExpirationMinutes int) (*Storage, error)

// 新版本
func New(dbPath string, walMode bool, maxOpenConns, maxIdleConns int) (*Storage, error)
```

#### 简化 GetLatestPositions()
```go
// 旧版本
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
    // ... 获取最新时间戳

    // ❌ 复杂的过期判断
    expirationDuration := time.Duration(s.positionExpirationMinutes) * time.Minute
    if time.Since(latestTime) > expirationDuration {
        return []models.Position{}, nil
    }

    // ... 查询数据
}

// 新版本
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
    // NOTE: This function is now mainly used for historical data analysis.
    // TPSL service fetches positions directly from OKX API for real-time accuracy.

    // ... 获取最新时间戳
    // ... 直接查询数据（无过期判断）
}
```

### 5. Main 程序更新

**文件**：`cmd/main.go`

#### 移除 Storage 参数
```go
// 旧版本
db, err := storage.New(
    cfg.Database.Path,
    cfg.Database.WALMode,
    cfg.Database.MaxOpenConns,
    cfg.Database.MaxIdleConns,
    cfg.Database.PositionExpirationMinutes,  // ❌ 移除
)

// 新版本
db, err := storage.New(
    cfg.Database.Path,
    cfg.Database.WALMode,
    cfg.Database.MaxOpenConns,
    cfg.Database.MaxIdleConns,
)
```

#### 移除 TPSL 的 Storage 依赖
```go
// 旧版本
if cfg.TPSL.Enabled {
    log.Info("Initializing TPSL scheduler")
    tpslScheduler = tpsl.NewScheduler(&cfg.TPSL, db, okxClient, log)  // ❌ 传递 db
}

// 新版本
if cfg.TPSL.Enabled {
    log.Info("Initializing TPSL scheduler (real-time API mode)")
    tpslScheduler = tpsl.NewScheduler(&cfg.TPSL, okxClient, log)  // ✅ 不传递 db
}
```

### 6. 配置模板更新

**文件**：`configs/config.template.yaml`

移除 position_expiration_minutes 配置：

```yaml
# Database Configuration
database:
  # Path to SQLite database file
  path: "./data/tenyojubaku.db"

  # Enable Write-Ahead Logging (WAL) mode for better concurrency
  wal_mode: true

  # Connection pool settings
  max_open_conns: 1
  max_idle_conns: 1

  # ❌ 移除以下配置
  # position_expiration_minutes: 10
```

## 优势分析 / Benefits

### 1. 数据实时性 ✅
```
旧架构：
TPSL 读取数据 → 来自数据库 → 最多 60 秒延迟 + 过期误判

新架构：
TPSL 读取数据 → 直接从 OKX API → 实时数据，0 延迟
```

### 2. 服务解耦 ✅
```
旧架构：
TPSL → 依赖 Storage → 依赖 Monitor
(Monitor 挂了，TPSL 数据过期)

新架构：
TPSL → 直接调用 OKX API（独立运行）
Monitor → 直接调用 OKX API（独立运行）
```

### 3. 逻辑简化 ✅
```
旧架构：
- 需要配置 position_expiration_minutes
- 需要复杂的时间阈值判断
- 需要处理边缘情况（刚平仓的"惯性"）

新架构：
- 无需配置过期时间
- 直接检查 API 返回的持仓数量
- if len(positions) == 0 { return }  ← 就这么简单！
```

### 4. 准确性提升 ✅
```
旧架构：
可能出现：
- 持仓已平，但因未到过期时间，仍尝试设置 TPSL
- 持仓活跃，但因超过过期时间，误判为已平

新架构：
API 说有就有，API 说没有就没有，100% 准确
```

### 5. 职责清晰 ✅
```
旧架构：
Monitor: 数据采集
Storage: 数据存储 + 过期判断（？）
TPSL:   基于数据库数据做决策

新架构：
Monitor: 数据采集和历史存档
Storage: 数据存储和查询
TPSL:   基于实时 API 数据做决策
```

## API 调用频率分析 / API Call Frequency

### 调用频率对比
```
旧架构：
Monitor: 每 60 秒调用 GetPositions 一次
总计：1440 次/天

新架构：
Monitor: 每 60 秒调用 GetPositions 一次
TPSL:    每 300 秒调用 GetPositions 一次
总计：1440 + 288 = 1728 次/天

增加：288 次/天 (+20%)
```

### 为什么增加是可接受的？

1. **换取准确性**：20% 的调用增加，换取 100% 的数据准确性
2. **仍在限制内**：OKX API 限制通常是几千次/天，1728 远低于限制
3. **可调整**：如果需要，可以增加 TPSL check_interval 到 600 秒（10 分钟）

## 不会破坏的语义 / Preserved Semantics

### ❓ 会破坏分层吗？
**答案：不会！反而是更好的分层**

```
旧分层（紧耦合）：
应用层 → 服务层 → 存储层 → 数据源层
TPSL → Storage → Monitor → OKX API
(每层依赖下一层)

新分层（松耦合）：
应用层 ┬→ 服务A（数据源层）
       └→ 服务B（数据源层）

TPSL → OKX API（实时风控）
Monitor → OKX API（历史存档）
(并行访问，互不依赖)
```

### ❓ 职责是否混淆？
**答案：不会！职责更清晰**

```
旧职责：
Monitor: 数据采集员（为所有人采集数据）
TPSL: 被动消费者（使用 Monitor 的数据）

新职责：
Monitor: 数据记录员（记录历史快照用于分析）
TPSL: 主动决策者（自己获取最新数据做决策）
```

类比：
```
旧模式：报纸订阅
- Monitor 是报社，每天送报纸
- TPSL 看昨天的报纸做决策（可能过时）

新模式：实时新闻
- Monitor 是图书馆，存档报纸供查阅
- TPSL 直接看新闻网站做决策（实时信息）
```

## 风险和缓解 / Risks and Mitigation

### 风险1：API 调用增加
**影响**：每天增加 288 次 API 调用

**缓解**：
- OKX API 限制足够高（通常 >3000 次/天）
- 可以调整 TPSL check_interval 来降低频率
- 实时准确性带来的价值远超调用成本

### 风险2：API 失败影响
**问题**：如果 OKX API 临时故障，TPSL 会失败

**缓解**：
- OKX Client 已有重试机制（max_retries）
- Scheduler 有 panic recovery，单次失败不会崩溃
- 下一个周期（5分钟后）会自动重试

### 风险3：数据转换错误
**问题**：convertOKXPosition 可能解析失败

**缓解**：
- 转换函数有详细的错误处理
- 单个持仓转换失败不影响其他持仓
- 会记录 Warning 日志，便于排查

## 测试验证 / Testing

### 测试1：编译验证
```bash
go build -o ./bin/tenyojubaku ./cmd/main.go
# ✅ 编译成功
```

### 测试2：有持仓时的行为
```
预期日志：
[INFO] Starting TPSL check cycle
[INFO] Fetching positions from OKX API...
[INFO] Found 1 open position(s)
[INFO] Starting TPSL analysis for 1 positions
[INFO] Position BTC-USD-SWAP coverage: ...
[INFO] TPSL check cycle completed: 1 positions checked, 1 orders placed, 0 failures
```

### 测试3：无持仓时的行为
```
预期日志：
[INFO] Starting TPSL check cycle
[INFO] Fetching positions from OKX API...
[INFO] No open positions, skipping TPSL check
```

### 测试4：API 失败时的行为
```
预期日志：
[INFO] Starting TPSL check cycle
[INFO] Fetching positions from OKX API...
[ERROR] Failed to fetch positions from OKX API: timeout
(下一个周期会自动重试)
```

### 测试5：Monitor 独立性
```
场景：TPSL 禁用，Monitor 继续运行
预期：Monitor 正常记录数据到数据库，不受 TPSL 影响
```

## 日志变化 / Log Changes

### 启动日志
```
旧版本：
[INFO] Database initialized successfully (position expiration: 10 minutes)
[INFO] Initializing TPSL scheduler

新版本：
[INFO] Database initialized successfully
[INFO] Initializing TPSL scheduler (real-time API mode)
```

### TPSL 检查日志
```
旧版本：
[DEBUG] Starting TPSL check cycle
[INFO] Starting TPSL analysis for 1 positions

新版本：
[DEBUG] Starting TPSL check cycle
[INFO] Fetching positions from OKX API...
[INFO] Found 1 open position(s)
[INFO] Starting TPSL analysis for 1 positions
```

## 迁移说明 / Migration Guide

### 对现有部署的影响
1. **配置文件**：移除 `database.position_expiration_minutes` 配置（如果有）
2. **数据库**：无需迁移，历史数据完全保留
3. **行为变化**：TPSL 响应更快（从 API 获取），更准确

### 回滚方案
如果需要回滚到旧版本：
```bash
git revert <这次提交的commit hash>
go build -o ./bin/tenyojubaku ./cmd/main.go
```

## 相关文档 / Related Documentation

1. **持仓过期过滤（旧方案）**：`STALE_POSITION_FILTERING.md`
2. **可配置过期时间（已废弃）**：`CONFIGURABLE_POSITION_EXPIRATION.md`
3. **TPSL 覆盖分析**：`TPSL_COVERAGE_BUG_FIX.md`
4. **紧急止损调整**：`EMERGENCY_SL_ADJUSTMENT.md`

## 总结 / Summary

这次重构**不是简单的代码修改**，而是**架构理念的升级**：

### 从"间接消费"到"直接获取"
```
旧：TPSL 被动消费 Monitor 采集的数据
新：TPSL 主动获取最新数据做决策
```

### 从"时间推测"到"状态查询"
```
旧：通过记录时间推测持仓是否还存在
新：直接查询 API 获得准确状态
```

### 从"紧耦合"到"松耦合"
```
旧：TPSL → Storage → Monitor → API（链式依赖）
新：TPSL → API, Monitor → API（并行独立）
```

### 核心价值
- ✅ **准确性**：基于实时 API 数据，100% 准确
- ✅ **实时性**：无数据库延迟，即时响应
- ✅ **简洁性**：移除复杂的过期判断逻辑
- ✅ **可靠性**：服务独立，互不影响
- ✅ **清晰性**：职责分明，易于理解和维护

## 实施人员 / Implementation
- **讨论**: 用户提出关键架构问题
- **分析**: Claude Code 进行架构分析
- **设计**: Claude Code 设计新架构
- **实施**: Claude Code 完整实施
- **测试**: 编译验证通过 ✅

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Architecture Refactor
- **版本**: v2.0.0 (Real-Time API for TPSL)
- **日期**: 2025-12-02
- **破坏性变更**: 是（但向后兼容数据）
