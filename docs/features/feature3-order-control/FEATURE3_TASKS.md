# Feature 3 (Order Control) 任务清单 / Task Checklist

**功能目标 / Feature Goal**: 实现订单频率控制、Maker-only 限制、订单确认机制

**创建日期 / Created**: 2025-12-08
**状态 / Status**: 📋 规划中 / Planning

---

## 📊 任务概览 / Task Overview

| 阶段 / Phase | 任务数 / Tasks | 预计时间 / Est. Time | 优先级 / Priority |
|-------------|---------------|---------------------|------------------|
| **Phase A: 数据库层** | 5 | 1-2 小时 | 🔴 必须 |
| **Phase B: 核心服务** | 8 | 3-4 小时 | 🔴 必须 |
| **Phase C: 调度器** | 3 | 1 小时 | 🟡 重要 |
| **Phase D: 集成** | 4 | 1 小时 | 🔴 必须 |
| **Phase E: 测试** | 4 | 1-2 小时 | 🟢 建议 |
| **总计 / Total** | **24** | **7-10 小时** | - |

---

## 🎯 Phase A: 数据库层实现 (必须)

### 现状 / Current State
- ✅ Models 已创建 (`pkg/models/order.go`)
- ✅ Storage 接口已定义 (`internal/storage/interface.go`)
- ⚠️ Storage 实现是占位符（返回 "not implemented"）

### 任务清单 / Task List

#### A1. 创建数据库表 Schema
**文件**: `internal/storage/storage.go` (在 New() 函数中)

**任务**:
- [ ] 创建 `order_history` 表
  ```sql
  CREATE TABLE IF NOT EXISTS order_history (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      order_id TEXT NOT NULL,
      inst_id TEXT NOT NULL,
      side TEXT NOT NULL,
      ord_type TEXT NOT NULL,
      size TEXT NOT NULL,
      price TEXT,
      reduce_only BOOLEAN NOT NULL DEFAULT 0,
      placed_at DATETIME NOT NULL,
      week_start DATE NOT NULL,
      status TEXT NOT NULL,
      created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      INDEX idx_week_start (week_start),
      INDEX idx_order_id (order_id)
  )
  ```

- [ ] 创建 `pending_confirmations` 表
  ```sql
  CREATE TABLE IF NOT EXISTS pending_confirmations (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      order_id TEXT NOT NULL UNIQUE,
      inst_id TEXT NOT NULL,
      side TEXT NOT NULL,
      ord_type TEXT NOT NULL,
      original_size TEXT NOT NULL,
      current_size TEXT NOT NULL,
      price TEXT,
      placed_at DATETIME NOT NULL,
      last_confirmation_at DATETIME,
      next_confirmation_due DATETIME NOT NULL,
      confirmation_count INTEGER NOT NULL DEFAULT 0,
      timeout_count INTEGER NOT NULL DEFAULT 0,
      status TEXT NOT NULL,
      created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
      INDEX idx_next_due (next_confirmation_due),
      INDEX idx_status (status)
  )
  ```

**预计时间**: 30 分钟
**复杂度**: ⭐ 低

---

#### A2. 实现 Order History 存储方法
**文件**: `internal/storage/storage.go`

**任务**:
- [ ] `InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error`
  - 验证 order 数据
  - INSERT 到 order_history 表
  - 返回插入的 ID

- [ ] `GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error)`
  - SELECT COUNT(*) WHERE week_start = ?
  - 可选：排除 reduce_only 订单（根据配置）

- [ ] `GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error)`
  - SELECT * WHERE week_start = ? ORDER BY placed_at

- [ ] `GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error)`
  - SELECT 聚合统计（total, reduce_only, by status）

**预计时间**: 1 小时
**复杂度**: ⭐⭐ 中等
**参考**: 现有的 `InsertAccountBalance` 方法

---

#### A3. 实现 Pending Confirmation 存储方法
**文件**: `internal/storage/storage.go`

**任务**:
- [ ] `InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error`
  - 验证 conf 数据
  - INSERT 到 pending_confirmations 表

- [ ] `GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error)`
  - SELECT * WHERE next_confirmation_due <= ? AND status = 'pending'

- [ ] `GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error)`
  - SELECT * WHERE order_id = ?

- [ ] `UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error`
  - 动态构建 UPDATE SQL（只更新非 nil 字段）
  - 更新 updated_at = CURRENT_TIMESTAMP

- [ ] `DeletePendingConfirmation(ctx context.Context, orderID string) error`
  - DELETE WHERE order_id = ?

**预计时间**: 1 小时
**复杂度**: ⭐⭐ 中等

---

## 🎯 Phase B: 核心 Order Control 服务 (必须)

### 现状 / Current State
- ✅ Config 结构已创建 (`OrderControlConfig`)
- ✅ Notifier 已实现 (`LogNotifier`)
- ❌ Order Control Service 不存在

### 任务清单 / Task List

#### B1. 创建 Order Control Service 结构
**文件**: `internal/ordercontrol/service.go` (新文件)

**任务**:
- [ ] 创建 `Service` 结构体
  ```go
  type Service struct {
      config    *config.OrderControlConfig
      okxClient okx.Interface
      storage   storage.Interface
      notifier  notifier.Interface
      logger    *logger.Logger
  }
  ```

- [ ] 实现 `New()` 构造函数
- [ ] 实现基本的验证逻辑框架

**预计时间**: 20 分钟
**复杂度**: ⭐ 低

---

#### B2. 实现 Frequency Limit (频率限制)
**文件**: `internal/ordercontrol/frequency.go` (新文件)

**任务**:
- [ ] `CheckFrequencyLimit(ctx context.Context, order *OrderRequest) error`
  - 获取本周起始时间（Monday 00:00 UTC）
  - 查询本周订单数量
  - 如果 `ExcludeReduceOnly`，过滤 reduce_only 订单
  - 检查是否超过 `WeeklyMaxOrders`
  - 返回错误或 nil

**逻辑**:
```go
func (s *Service) CheckFrequencyLimit(ctx context.Context, isReduceOnly bool) error {
    if !s.config.FrequencyLimit.Enabled {
        return nil // 未启用
    }

    weekStart := models.GetWeekStart(time.Now())
    count, err := s.storage.GetOrderCountForWeek(ctx, weekStart)
    if err != nil {
        return err
    }

    // 排除 reduce-only？
    if s.config.FrequencyLimit.ExcludeReduceOnly && isReduceOnly {
        return nil
    }

    if count >= s.config.FrequencyLimit.WeeklyMaxOrders {
        return fmt.Errorf("weekly order limit reached: %d/%d", count, s.config.FrequencyLimit.WeeklyMaxOrders)
    }

    return nil
}
```

**预计时间**: 30 分钟
**复杂度**: ⭐⭐ 中等

---

#### B3. 实现 Maker-Only Check (Maker 限制)
**文件**: `internal/ordercontrol/makeronly.go` (新文件)

**任务**:
- [ ] `CheckMakerOnly(ctx context.Context, order *OrderRequest) error`
  - 检查订单类型（limit, post_only 允许；market, fok, ioc 拒绝）
  - 对于 limit 订单，检查价格距离
  - 获取市场 ticker
  - 计算价格距离百分比
  - 检查是否满足 `MinPriceDistancePct`
  - 如果是 reduce_only 且 `AllowTakerForReduceOnly = true`，允许通过

**逻辑**:
```go
func (s *Service) CheckMakerOnly(ctx context.Context, req *OrderRequest) error {
    if !s.config.MakerOnly.Enabled {
        return nil
    }

    // Taker 订单类型
    takerTypes := map[string]bool{"market": true, "fok": true, "ioc": true}

    if takerTypes[req.OrdType] {
        // 检查是否是 reduce_only 且允许 taker
        if req.ReduceOnly && s.config.MakerOnly.AllowTakerForReduceOnly {
            // TODO: 检查 taker 订单百分比
            return nil
        }
        return fmt.Errorf("taker order type '%s' not allowed", req.OrdType)
    }

    // Limit 订单检查价格距离
    if req.OrdType == "limit" {
        ticker, err := s.okxClient.GetTicker(ctx, req.InstId)
        if err != nil {
            return err
        }

        // 检查 ticker 是否过期
        // 计算价格距离
        // 检查是否满足最小距离
    }

    return nil
}
```

**预计时间**: 1 小时
**复杂度**: ⭐⭐⭐ 较高

---

#### B4. 实现 Order Placement Wrapper (订单下单包装)
**文件**: `internal/ordercontrol/service.go`

**任务**:
- [ ] `PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)`
  - 运行所有检查：
    1. CheckFrequencyLimit
    2. CheckMakerOnly
  - 如果通过，调用 `okxClient.PlaceOrder()`
  - 记录订单到 `order_history`
  - 如果启用确认机制，创建 `pending_confirmation`
  - 返回结果

**预计时间**: 45 分钟
**复杂度**: ⭐⭐ 中等

---

#### B5. 实现 Confirmation Workflow (确认工作流)
**文件**: `internal/ordercontrol/confirmation.go` (新文件)

**任务**:
- [ ] `CreatePendingConfirmation(ctx context.Context, orderID string, order *OrderRequest, placedAt time.Time) error`
  - 计算 `next_confirmation_due = placedAt + WaitingPeriodHours`
  - 插入到 `pending_confirmations`

- [ ] `CheckPendingConfirmations(ctx context.Context) error`
  - 查询所有到期的确认
  - 对每个订单：
    1. 发送通知（使用 Notifier）
    2. 等待确认（暂时跳过，用户手动确认）
    3. 如果超时，减少订单规模或取消

- [ ] `HandleConfirmationTimeout(ctx context.Context, conf *models.PendingConfirmation) error`
  - 增加 `timeout_count`
  - 如果 `timeout_count >= MaxTimeouts`，取消订单
  - 否则，减少订单规模（`current_size *= (1 - TimeoutSizeReductionPct)`）
  - 修改订单（`okxClient.AmendOrder`）
  - 更新 `next_confirmation_due`

**预计时间**: 1.5 小时
**复杂度**: ⭐⭐⭐⭐ 高

---

## 🎯 Phase C: Confirmation Scheduler (重要)

### 任务清单 / Task List

#### C1. 创建 Confirmation Scheduler
**文件**: `internal/ordercontrol/scheduler.go` (新文件)

**任务**:
- [ ] 创建 `Scheduler` 结构体
- [ ] 实现 `Start()` 和 `Stop()` 方法
- [ ] 定时调用 `service.CheckPendingConfirmations()`
- [ ] 使用 `CheckIntervalSeconds` 作为间隔

**预计时间**: 45 分钟
**复杂度**: ⭐⭐ 中等
**参考**: `internal/tpsl/scheduler.go`

---

## 🎯 Phase D: 系统集成 (必须)

### 任务清单 / Task List

#### D1. 更新配置文件
**文件**: `configs/config.yaml`

**任务**:
- [ ] 添加 `order_control` 配置节
  ```yaml
  order_control:
    enabled: false  # 默认关闭，测试后开启
    frequency_limit:
      enabled: true
      weekly_max_orders: 20
      exclude_reduce_only: true
    maker_only:
      enabled: true
      min_price_distance_pct: 0.001  # 0.1%
      allow_taker_for_reduce_only: true
      max_taker_pct: 0.2
      ticker_staleness_seconds: 60
    confirmation:
      enabled: false  # 第一版可以不启用
      check_interval_seconds: 3600  # 1 hour
      confirmation_interval_hours: 24
      waiting_period_hours: 48
      timeout_size_reduction_pct: 0.1
      max_timeouts: 3
      notification_method: log
  ```

**预计时间**: 10 分钟
**复杂度**: ⭐ 低

---

#### D2. 在 main.go 中初始化
**文件**: `cmd/main.go`

**任务**:
- [ ] 创建 `ordercontrol.Service` 实例
- [ ] 创建 `ordercontrol.Scheduler` 实例（如果启用确认）
- [ ] 启动 scheduler
- [ ] 在 shutdown 时停止

**预计时间**: 20 分钟
**复杂度**: ⭐ 低

---

#### D3. 创建 API/CLI 接口 (可选)
**文件**: 新文件或现有文件

**任务**:
- [ ] 提供一个方式让外部调用 `PlaceOrder`
  - 可以是 CLI 命令
  - 可以是 HTTP API
  - 可以是其他服务调用

**预计时间**: 30 分钟 - 1 小时
**复杂度**: ⭐⭐ 中等
**优先级**: 🟡 中 (取决于如何触发订单)

---

## 🎯 Phase E: 测试和验证 (建议)

### 任务清单 / Task List

#### E1. 单元测试
**文件**: `internal/ordercontrol/*_test.go`

**任务**:
- [ ] 测试 `CheckFrequencyLimit`
- [ ] 测试 `CheckMakerOnly`
- [ ] 测试 `PlaceOrder` 工作流
- [ ] 测试确认超时逻辑

**预计时间**: 1-2 小时
**复杂度**: ⭐⭐⭐ 较高

---

#### E2. 集成测试
**任务**:
- [ ] 测试完整流程：下单 → 检查 → 存储 → 确认
- [ ] 测试频率限制（本周订单数）
- [ ] 测试 Maker-only 验证

**预计时间**: 1 小时
**复杂度**: ⭐⭐⭐ 较高

---

## 📋 推荐的实施顺序 / Recommended Implementation Order

### 🎯 最小可行版本 (MVP) - 5-6 小时

专注核心功能，跳过复杂的确认机制：

1. **Phase A1-A2** (1.5h): 数据库 schema + Order History 方法
2. **Phase B1-B2** (50min): Service 结构 + 频率限制
3. **Phase B3** (1h): Maker-only 检查
4. **Phase B4** (45min): PlaceOrder 包装
5. **Phase D1-D2** (30min): 配置和集成
6. **Phase E1** (1h): 基本测试

**结果**: 可用的频率限制 + Maker-only 功能

---

### 🚀 完整版本 - 9-10 小时

包含确认机制：

1. **MVP (5-6h)** - 先完成上面的
2. **Phase A3** (1h): Pending Confirmation 存储
3. **Phase B5** (1.5h): 确认工作流
4. **Phase C1** (45min): Confirmation Scheduler
5. **Phase E2** (1h): 集成测试

**结果**: 完整的 Feature 3 功能

---

### 🔥 快速原型 - 3-4 小时

只实现频率限制，跳过其他：

1. **Phase A1** (30min): 只创建 order_history 表
2. **Phase A2** (1h): Order History 存储方法
3. **Phase B1-B2** (50min): Service + 频率限制
4. **Phase B4** (简化版, 30min): PlaceOrder（只检查频率）
5. **Phase D1-D2** (30min): 配置和集成

**结果**: 基本的订单频率控制

---

## 💡 建议 / Recommendations

### 我的推荐：从 MVP 开始

**理由**:
1. ✅ 频率限制和 Maker-only 是核心价值
2. ✅ 确认机制复杂，可以后续迭代
3. ✅ 5-6 小时可以在 2-3 个会话完成
4. ✅ 可以先验证效果，再决定是否需要确认机制

**分阶段实施**:
- **Session 1** (2h): Phase A1-A2 (数据库层)
- **Session 2** (2h): Phase B1-B3 (核心检查逻辑)
- **Session 3** (1.5h): Phase B4 + D1-D2 (集成和测试)

---

## ❓ 你的决定 / Your Decision

请告诉我你想：

### 选项 A: MVP 版本 (推荐)
- 频率限制 + Maker-only
- 5-6 小时
- 分 2-3 个会话完成

### 选项 B: 快速原型
- 只有频率限制
- 3-4 小时
- 1-2 个会话完成

### 选项 C: 完整版本
- 所有功能包括确认
- 9-10 小时
- 3-4 个会话完成

### 选项 D: 自定义
- 告诉我你想要哪些功能
- 我会调整任务清单

---

**当前建议**: 从 **Phase A1 (创建数据库表)** 开始，这是所有功能的基础。

你想从哪里开始？要做到什么程度？
