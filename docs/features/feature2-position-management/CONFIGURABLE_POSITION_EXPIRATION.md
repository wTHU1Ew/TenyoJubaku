# Configurable Position Expiration Threshold

## 实施日期 / Implementation Date
2025-12-02

## 背景 / Background

在实现了持仓记录过期过滤机制（`STALE_POSITION_FILTERING.md`）后，过期时间阈值被硬编码为 10 分钟。用户提出应该将此阈值做成可配置项，以便根据不同的监控间隔和 TPSL 检查间隔进行调整。

After implementing the stale position filtering mechanism (`STALE_POSITION_FILTERING.md`), the expiration threshold was hardcoded as 10 minutes. The user requested to make this threshold configurable to allow adjustment based on different monitoring and TPSL check intervals.

## 用户需求 / User Request

> "这个检测持仓记录时间的具体时长应该也是可配置项"

用户希望：
1. ✅ 过期时间阈值可配置
2. ✅ 能够根据实际使用场景调整阈值
3. ✅ 保持默认值为 10 分钟

## 实施方案 / Implementation

### 1. 配置结构定义

**文件**: `internal/config/config.go`

```go
// DatabaseConfig 数据库配置 / Database configuration
type DatabaseConfig struct {
    Path                      string `yaml:"path"`
    WALMode                   bool   `yaml:"wal_mode"`
    MaxOpenConns              int    `yaml:"max_open_conns"`
    MaxIdleConns              int    `yaml:"max_idle_conns"`
    PositionExpirationMinutes int    `yaml:"position_expiration_minutes"` // NEW
}
```

### 2. 配置验证和默认值

**文件**: `internal/config/config.go` (Validate 函数)

```go
if c.Database.PositionExpirationMinutes <= 0 {
    c.Database.PositionExpirationMinutes = 10 // Default 10 minutes
}
```

### 3. 配置文件模板

**文件**: `configs/config.template.yaml`

```yaml
database:
  # Path to SQLite database file
  path: "./data/tenyojubaku.db"

  # Enable Write-Ahead Logging (WAL) mode for better concurrency
  wal_mode: true

  # Connection pool settings
  max_open_conns: 1
  max_idle_conns: 1

  # Position record expiration time (minutes)
  # Records older than this are considered stale/closed positions
  # Recommended: 2x the maximum of (monitoring.interval, tpsl.check_interval) in seconds / 60
  # Example: monitoring.interval=60s, tpsl.check_interval=300s → recommend 10 minutes
  # Default: 10 minutes
  position_expiration_minutes: 10
```

### 4. Storage 层实现

**文件**: `internal/storage/storage.go`

#### Storage 结构更新
```go
type Storage struct {
    db                        *sql.DB
    positionExpirationMinutes int  // NEW FIELD
}
```

#### New() 函数签名更新
```go
func New(dbPath string, walMode bool, maxOpenConns, maxIdleConns, positionExpirationMinutes int) (*Storage, error) {
    // ...
    storage := &Storage{
        db:                        db,
        positionExpirationMinutes: positionExpirationMinutes,
    }
    // ...
}
```

#### GetLatestPositions() 使用配置值
```go
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
    // ...

    // If latest snapshot is older than the configured expiration time, consider all positions closed
    expirationDuration := time.Duration(s.positionExpirationMinutes) * time.Minute
    if time.Since(latestTime) > expirationDuration {
        return []models.Position{}, nil
    }

    // ...
}
```

### 5. 主程序传递配置

**文件**: `cmd/main.go`

```go
// Initialize database
log.Info("Initializing database at %s", cfg.Database.Path)
db, err := storage.New(
    cfg.Database.Path,
    cfg.Database.WALMode,
    cfg.Database.MaxOpenConns,
    cfg.Database.MaxIdleConns,
    cfg.Database.PositionExpirationMinutes,  // NEW PARAMETER
)
if err != nil {
    log.Error("Failed to initialize database: %v", err)
    exitCode = 1
    return
}
defer db.Close()
log.Info("Database initialized successfully (position expiration: %d minutes)", cfg.Database.PositionExpirationMinutes)
```

## 推荐配置 / Recommended Configuration

### 基本公式

```
position_expiration_minutes >= max(monitoring.interval, tpsl.check_interval) × 2 / 60
```

其中：
- `monitoring.interval`: 监控间隔（秒）
- `tpsl.check_interval`: TPSL 检查间隔（秒）
- `÷ 60`: 转换为分钟

### 配置示例

#### 场景1：默认配置
```yaml
monitoring:
  interval: 60  # 1 分钟

tpsl:
  check_interval: 300  # 5 分钟

database:
  position_expiration_minutes: 10  # 推荐：max(60, 300) × 2 / 60 = 10 分钟 ✅
```

#### 场景2：快速响应
```yaml
monitoring:
  interval: 30  # 30 秒

tpsl:
  check_interval: 60  # 1 分钟

database:
  position_expiration_minutes: 2  # 推荐：max(30, 60) × 2 / 60 = 2 分钟 ✅
```

#### 场景3：低频检查
```yaml
monitoring:
  interval: 300  # 5 分钟

tpsl:
  check_interval: 600  # 10 分钟

database:
  position_expiration_minutes: 20  # 推荐：max(300, 600) × 2 / 60 = 20 分钟 ✅
```

## 配置原则 / Configuration Principles

### 1. 安全边际
设置为最大检查间隔的 **2 倍**，留出安全缓冲：
- ✅ 容忍网络延迟
- ✅ 容忍临时服务中断
- ✅ 避免误判活跃持仓

### 2. 不要太短
如果过期时间 < 最大检查间隔：
```
monitoring.interval = 60s
tpsl.check_interval = 300s
position_expiration_minutes = 4  # ❌ 4分钟 < 5分钟，太短！

风险：正常持仓可能被误判为已平仓
```

### 3. 不要太长
如果过期时间过长：
```
position_expiration_minutes = 60  # ❌ 60分钟，太长！

风险：已平仓位可能在长时间内仍被尝试设置 TPSL
```

## 使用场景 / Use Cases

### 场景1：高频交易
如果你经常开平仓，需要快速响应：
```yaml
monitoring:
  interval: 30  # 30 秒一次

tpsl:
  check_interval: 60  # 1 分钟一次

database:
  position_expiration_minutes: 2  # 2 分钟过期
```

### 场景2：长期持仓
如果你长期持有，不需要频繁检查：
```yaml
monitoring:
  interval: 300  # 5 分钟一次

tpsl:
  check_interval: 600  # 10 分钟一次

database:
  position_expiration_minutes: 20  # 20 分钟过期
```

### 场景3：节省 API 调用
如果你想减少 API 调用次数：
```yaml
monitoring:
  interval: 120  # 2 分钟一次

tpsl:
  check_interval: 300  # 5 分钟一次

database:
  position_expiration_minutes: 10  # 10 分钟过期
```

## 验证效果 / Verification

### 1. 启动时日志
```
[INFO] Database initialized successfully (position expiration: 10 minutes)
```

### 2. 运行时行为
```
场景：持仓记录 15 分钟前插入
配置：position_expiration_minutes = 10

结果：
[INFO] No open positions, skipping TPSL check  ✅ 正确！
```

### 3. 配置修改测试
```bash
# 1. 修改配置文件
vim configs/config.yaml
# 修改 position_expiration_minutes: 5

# 2. 重启程序
./bin/tenyojubaku

# 3. 观察启动日志
# [INFO] Database initialized successfully (position expiration: 5 minutes)
```

## 边缘情况 / Edge Cases

### 1. 配置为 0 或负数
```yaml
database:
  position_expiration_minutes: 0  # 或负数
```

**行为**：自动使用默认值 10 分钟
```go
if c.Database.PositionExpirationMinutes <= 0 {
    c.Database.PositionExpirationMinutes = 10
}
```

### 2. 配置为极小值
```yaml
database:
  position_expiration_minutes: 1  # 1 分钟
```

**风险**：如果监控间隔 > 1 分钟，可能导致活跃持仓被误判为已平仓

**建议**：不要小于推荐值

### 3. 配置为极大值
```yaml
database:
  position_expiration_minutes: 1440  # 24 小时
```

**影响**：已平仓位在 24 小时内仍会尝试设置 TPSL（虽然会失败，但会有错误日志）

## 相关文件 / Modified Files

1. **internal/config/config.go**
   - 添加 `DatabaseConfig.PositionExpirationMinutes` 字段
   - 添加配置验证和默认值

2. **internal/storage/storage.go**
   - 添加 `Storage.positionExpirationMinutes` 字段
   - 更新 `New()` 函数签名
   - 更新 `GetLatestPositions()` 使用配置值

3. **cmd/main.go**
   - 传递 `cfg.Database.PositionExpirationMinutes` 到 `storage.New()`
   - 添加日志输出配置值

4. **configs/config.template.yaml**
   - 添加 `position_expiration_minutes` 配置项和说明

5. **STALE_POSITION_FILTERING.md**
   - 更新文档，标注阈值已可配置
   - 更新配置示例

## 优势 / Benefits

### 1. 灵活性 ✅
- 可根据实际交易频率调整
- 可根据网络环境调整
- 可根据 API 调用限制调整

### 2. 可维护性 ✅
- 配置集中在配置文件
- 无需修改代码即可调整
- 易于测试不同配置

### 3. 向后兼容 ✅
- 默认值为 10 分钟（与之前硬编码一致）
- 不配置时使用默认值
- 不影响现有部署

### 4. 文档完善 ✅
- 配置文件有详细注释
- 提供推荐值计算公式
- 提供多种场景示例

## 测试验证 / Testing

### 测试1：默认配置
```bash
# 不设置 position_expiration_minutes
# 预期：使用默认值 10 分钟

./bin/tenyojubaku
# [INFO] Database initialized successfully (position expiration: 10 minutes)
```

### 测试2：自定义配置
```bash
# 设置 position_expiration_minutes: 5
# 预期：使用配置值 5 分钟

./bin/tenyojubaku
# [INFO] Database initialized successfully (position expiration: 5 minutes)
```

### 测试3：功能验证
```bash
# 1. 停止程序 15 分钟
# 2. 配置 position_expiration_minutes: 10
# 3. 重启程序
# 预期：跳过 TPSL 检查（记录过期）

./bin/tenyojubaku
# [INFO] No open positions, skipping TPSL check ✅
```

### 测试4：编译验证
```bash
go build -o ./bin/tenyojubaku ./cmd/main.go
# 预期：编译成功，无错误
```

## 相关文档 / Related Documentation

1. **持仓过期过滤机制**: `STALE_POSITION_FILTERING.md`
2. **TPSL 覆盖分析**: `TPSL_COVERAGE_BUG_FIX.md`
3. **紧急止损调整**: `EMERGENCY_SL_ADJUSTMENT.md`
4. **配置模板**: `configs/config.template.yaml`

## 实施人员 / Implementation
- **需求**: 用户提出
- **设计**: Claude Code
- **实施**: Claude Code
- **测试**: 编译验证通过 ✅

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Position Management (Enhancement)
- **版本**: v1.2.3 (Configurable Position Expiration)
- **日期**: 2025-12-02
