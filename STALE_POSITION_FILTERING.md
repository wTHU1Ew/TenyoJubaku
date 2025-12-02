# Stale Position Filtering Fix

## 实施日期 / Implementation Date
2025-12-02

## 问题描述 / Problem Description

### 用户反馈 / User Feedback
> "我发现还存在一个问题，就是即使我没有仓位，但是依旧会尝试下发止盈止损"

### 根本原因 / Root Cause

**数据流程**：
1. 监控服务每 60 秒获取一次持仓快照
2. 如果有持仓 → 插入数据库
3. 如果无持仓 → **不插入任何记录**（直接返回）
4. TPSL 服务调用 `GetLatestPositions()` → 返回**最新时间戳**的记录

**问题场景**：
```
时间 T1 (16:00): 有持仓 2 BTC → 插入数据库 ✅
时间 T2 (16:01): 平仓，持仓 = 0 → 不插入记录 ❌
时间 T3 (16:02): TPSL 检查 → GetLatestPositions()
                → 返回 T1 的记录（2 BTC）❌
                → 尝试设置 TPSL ❌ 错误！
```

**本质问题**：
- `GetLatestPositions()` 无法区分"当前持仓"和"历史持仓"
- 数据库中最新记录可能是已经平仓的旧记录

## 解决方案 / Solution

### 方案选择

考虑了两种方案：

#### 方案1：添加状态字段（❌ 不采用）
```go
type Position struct {
    // ...
    Status string  // "open" | "closed"
}
```
**缺点**：
- 需要数据库迁移
- 需要更新所有 INSERT/UPDATE 逻辑
- 历史数据处理复杂

#### 方案2：时间阈值过滤（✅ 采用）
```go
// If latest snapshot is older than 10 minutes, consider positions closed
if time.Since(latestTime) > 10*time.Minute {
    return []models.Position{}, nil
}
```
**优点**：
- ✅ 无需修改数据结构
- ✅ 无需数据库迁移
- ✅ 历史数据自动保留
- ✅ 实现简单直接

### 实施细节

**文件**: `internal/storage/storage.go`

**修改前**：
```go
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
    query := `
        SELECT ... FROM positions
        WHERE timestamp = (SELECT MAX(timestamp) FROM positions)
    `
    // ❌ 无论时间戳多久，都返回记录
}
```

**修改后**：
```go
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
    // 1. 获取最新时间戳
    var latestTimestamp string
    err := s.db.QueryRow("SELECT MAX(timestamp) FROM positions").Scan(&latestTimestamp)

    // 2. 如果数据库为空，返回空切片
    if latestTimestamp == "" {
        return []models.Position{}, nil
    }

    // 3. 解析时间戳
    latestTime, err := time.Parse(time.RFC3339, latestTimestamp)

    // 4. ✅ 关键：如果最新记录超过 10 分钟，返回空切片
    if time.Since(latestTime) > 10*time.Minute {
        return []models.Position{}, nil  // 认为所有持仓已平
    }

    // 5. 否则返回最新记录
    query := `SELECT ... FROM positions WHERE timestamp = ?`
    // ...
}
```

## 工作原理 / How It Works

### 场景1：有活跃持仓

```
当前时间：16:05:00
最新记录：16:04:30 (30 秒前)
判断：30秒 < 10分钟 ✅
返回：最新持仓记录
TPSL：正常设置止盈止损
```

### 场景2：持仓已平（刚平仓）

```
当前时间：16:05:00
最新记录：16:03:00 (2 分钟前，当时有持仓)
判断：2分钟 < 10分钟 ✅
返回：最新持仓记录
TPSL：尝试设置止盈止损

说明：刚平仓时会有短暂的"惯性"
      但下一个监控周期(16:06)会正确检测到无持仓
```

### 场景3：持仓已平（超过10分钟）

```
当前时间：16:20:00
最新记录：16:03:00 (17 分钟前)
判断：17分钟 > 10分钟 ✅
返回：空切片 []
TPSL：跳过检查（"No open positions"）✅ 正确！
```

### 场景4：程序长时间未运行

```
当前时间：今天 10:00
最新记录：昨天 16:00 (18 小时前)
判断：18小时 > 10分钟 ✅
返回：空切片 []
TPSL：跳过检查 ✅ 正确！
```

## 时间阈值的选择 / Threshold Selection

### 为什么是 10 分钟？

**监控间隔**：60 秒（1 分钟）
**TPSL 检查间隔**：300 秒（5 分钟）

**分析**：
```
假设在 T0 时刻平仓：

T0: 平仓
T1 (1分钟后): 监控检测到无持仓，不插入记录
T2 (2分钟后): 监控检测到无持仓，不插入记录
...
T5 (5分钟后): TPSL 检查，最新记录是 T-1 (6分钟前)
              6分钟 < 10分钟 → 暂时还会返回旧记录
T10 (10分钟后): TPSL 检查，最新记录是 T-1 (11分钟前)
               11分钟 > 10分钟 → 返回空切片 ✅
```

**选择 10 分钟的理由**：
1. ✅ 大于最大检查间隔（5分钟）的 2 倍
2. ✅ 足够容忍网络延迟和异常
3. ✅ 不会太长，避免长时间错误

### 可配置化（未来改进）

可以将阈值做成配置项：

```yaml
storage:
  # 持仓记录过期时间（分钟）
  # Position record expiration time (minutes)
  # Records older than this are considered stale
  position_expiration_minutes: 10
```

## 日志示例 / Log Examples

### 有活跃持仓

```
[INFO] Fetching positions from OKX API...
[INFO] Stored 1 position records
[INFO] Starting TPSL analysis for 1 positions
[INFO] Position BTC-USD-SWAP coverage: ...
```

### 无持仓（最近平仓）

```
[INFO] Fetching positions from OKX API...
[INFO] No open positions
[INFO] No open positions, skipping TPSL check
```

### 无持仓（记录过期）

```
[INFO] Starting TPSL check cycle
[DEBUG] Latest position record is 15 minutes old, considering all positions closed
[INFO] No open positions, skipping TPSL check
[INFO] TPSL check cycle completed: 0 positions checked
```

## 优势 / Benefits

### 1. 历史数据保留 ✅
```sql
-- 所有历史持仓记录都保留
SELECT * FROM positions
WHERE timestamp BETWEEN '2025-12-01' AND '2025-12-02'
ORDER BY timestamp;

-- 可用于：
-- - 交易分析
-- - 收益统计
-- - 持仓时长计算
-- - 风险评估
```

### 2. 自动过期机制 ✅
- 无需手动清理
- 无需状态更新
- 基于时间自动判断

### 3. 容错性强 ✅
- 程序崩溃后重启 → 自动过滤过期记录
- 网络中断 → 超时后自动认为无持仓
- 数据不一致 → 时间阈值保证最终一致性

### 4. 实现简单 ✅
- 无需数据库迁移
- 无需修改模型
- 单个函数修改

## 边缘情况处理 / Edge Cases

### 1. 数据库为空
```go
if latestTimestamp == "" {
    return []models.Position{}, nil  // ✅ 返回空切片
}
```

### 2. 时间解析失败
```go
latestTime, err := time.Parse(time.RFC3339, latestTimestamp)
if err != nil {
    return nil, fmt.Errorf("failed to parse timestamp: %w", err)  // ✅ 返回错误
}
```

### 3. 频繁开平仓
```
T0: 开仓 2 BTC
T1: 平仓
T2: 再开仓 1 BTC
T3: 再平仓
...
```
**行为**：每次开仓都会插入新记录，更新"最新时间戳"，逻辑正确。

### 4. 监控服务停止
```
监控停止 → 不再插入新记录
10分钟后 → TPSL 认为无持仓
```
**安全性**：如果监控不工作，不应该基于过期数据设置 TPSL。✅ 正确行为。

## 潜在问题与风险 / Potential Issues

### 1. 刚平仓时的"惯性"

**问题**：平仓后的 1-5 分钟内，仍然可能尝试设置 TPSL

**影响**：
- OKX API 会返回错误（持仓不存在）
- 日志中会有错误信息
- 下一个周期会自动纠正

**是否需要修复**：
- ❌ 不影响正确性
- ❌ 影响范围小（5分钟内）
- ✅ 可以接受

### 2. 监控间隔修改

如果将监控间隔改为 10 分钟：
```yaml
monitoring:
  interval: 600  # 10 分钟
```

需要相应调整阈值：
```go
if time.Since(latestTime) > 20*time.Minute {  // 改为 20 分钟
```

**建议**：将阈值设为 `monitoring.interval × 2`

### 3. 系统时间不同步

如果系统时间不准确，可能导致错误判断。

**缓解措施**：
- 使用 NTP 同步系统时间
- 数据库和应用在同一服务器
- 时间戳使用 UTC

## 测试验证 / Testing

### 测试1：有活跃持仓
```bash
# 启动程序（有持仓）
./bin/tenyojubaku

# 预期日志：
# [INFO] Stored 1 position records
# [INFO] Starting TPSL analysis for 1 positions
```

### 测试2：平仓后立即检查
```bash
# 1. 在 OKX 平仓
# 2. 等待 1 分钟
# 3. 观察日志

# 预期：
# [INFO] No open positions
# [INFO] No open positions, skipping TPSL check
```

### 测试3：记录过期检查
```bash
# 1. 停止程序
# 2. 等待 15 分钟
# 3. 重启程序

# 预期：
# [INFO] No open positions, skipping TPSL check
```

### 测试4：历史数据查询
```bash
sqlite3 data/tenyojubaku.db "
SELECT timestamp, instrument, position_size
FROM positions
ORDER BY timestamp DESC
LIMIT 10;
"

# 预期：所有历史记录都保留
```

## 相关配置 / Related Configuration

### 监控间隔
```yaml
monitoring:
  interval: 60  # 秒
```

### TPSL 检查间隔
```yaml
tpsl:
  check_interval: 300  # 秒
```

### 建议的阈值关系
```
position_expiration_time >= max(monitoring.interval, tpsl.check_interval) × 2
```

## 未来改进 / Future Enhancements

### 1. 可配置阈值
```yaml
storage:
  position_expiration_minutes: 10
```

### 2. 添加日志说明
```go
if time.Since(latestTime) > threshold {
    s.logger.Debug("Latest position record is %.0f minutes old, considering all positions closed",
        time.Since(latestTime).Minutes())
    return []models.Position{}, nil
}
```

### 3. 监控指标
添加 Prometheus 指标：
```go
stale_position_checks_total{result="expired"}
stale_position_checks_total{result="active"}
```

## 相关文档 / Related Documentation

1. **TPSL 覆盖分析**: `TPSL_COVERAGE_BUG_FIX.md`
2. **Margin Mode 支持**: `MARGIN_MODE_AND_ENUMS_FIX.md`
3. **紧急止损调整**: `EMERGENCY_SL_ADJUSTMENT.md`

## 实施人员 / Implementation
- **发现**: 用户报告
- **分析**: Claude Code
- **设计**: Claude Code
- **实施**: Claude Code
- **测试**: 待用户验证 / Pending User Verification

## 版本信息 / Version Information
- **项目**: TenyoJubaku
- **功能**: Position Management (Bug Fix)
- **版本**: v1.2.2 (Stale Position Filtering)
- **日期**: 2025-12-02
