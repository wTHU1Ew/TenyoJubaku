# Phase 1B 实施待办清单 / Phase 1B Implementation TODO

**目标 / Goal**: 实现 OKX Client 和 Storage 的 Context 支持，完成 Phase 1 重构

**开始日期 / Start Date**: 2025-12-03
**状态 / Status**: ⏳ 未开始 / Not Started

---

## 📋 完整任务清单 / Complete Task List

### Phase 1B.1: OKX Client Context Methods (OKX 客户端 Context 方法)

#### 1.1 添加内部辅助方法 / Add Internal Helper Methods
- [ ] `doRequestWithContext(ctx context.Context, method, path string) ([]byte, error)`
- [ ] `doRequestWithBodyAndContext(ctx context.Context, method, path, body string) ([]byte, error)`

**参考代码 / Reference**: `REFACTORING_GUIDE.md` 行 143-225

#### 1.2 更新现有方法添加 Context / Update Existing Methods with Context
- [ ] `GetAccountBalance(ctx context.Context) (*AccountBalanceResponse, error)`
- [ ] `GetPositions(ctx context.Context) (*PositionsResponse, error)`
- [ ] `GetTicker(ctx context.Context, instId string) (*TickerResponse, error)`
- [ ] `GetPendingAlgoOrders(ctx context.Context, ordType string) (*PendingAlgoOrdersResponse, error)`
- [ ] `PlaceAlgoOrder(ctx context.Context, req AlgoOrderRequest) (*AlgoOrderResponse, error)`
- [ ] `HealthCheck(ctx context.Context) error`

**注意**: 保持方法签名与 `internal/okx/interface.go` 一致

#### 1.3 实现新的交易方法 / Implement New Trading Methods
- [ ] `PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)`
- [ ] `AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error)`
- [ ] `CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)`
- [ ] `GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error)`

**参考代码 / Reference**: `REFACTORING_GUIDE.md` 行 48-141

#### 1.4 添加 Context 导入 / Add Context Import
- [ ] 在 `internal/okx/client.go` 顶部添加 `"context"` 导入

#### 1.5 取消注释接口检查 / Uncomment Interface Check
- [ ] 在 `internal/okx/interface.go` 取消注释 `var _ Interface = (*Client)(nil)`

---

### Phase 1B.2: Storage Context Methods (存储层 Context 方法)

#### 2.1 创建数据库 Schema / Create Database Schema
- [ ] 创建 `order_history` 表 SQL
- [ ] 创建 `pending_confirmations` 表 SQL
- [ ] 创建索引（week_start, order_id, status 等）

**文件位置 / File Location**: `internal/storage/schema.go` 或直接在 `storage.go`

#### 2.2 实现 Order History 方法 / Implement Order History Methods
- [ ] `InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error`
- [ ] `GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error)`
- [ ] `GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error)`
- [ ] `GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error)`

#### 2.3 实现 Pending Confirmation 方法 / Implement Pending Confirmation Methods
- [ ] `InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error`
- [ ] `GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error)`
- [ ] `GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error)`
- [ ] `UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error`
- [ ] `DeletePendingConfirmation(ctx context.Context, orderID string) error`

#### 2.4 添加 Context 导入 / Add Context Import
- [ ] 在 `internal/storage/storage.go` 添加 `"context"` 导入

#### 2.5 取消注释接口检查 / Uncomment Interface Check
- [ ] 在 `internal/storage/interface.go` 取消注释 `var _ Interface = (*Storage)(nil)`

---

### Phase 1B.3: Update Mock Implementations (更新 Mock 实现)

#### 3.1 OKX Mock
- [ ] 更新 `internal/okx/mock.go` 的所有方法签名添加 `ctx context.Context`
- [ ] 实现新的 4 个交易方法（PlaceOrder, AmendOrder, CancelOrder, GetPendingOrders）

#### 3.2 Storage Mock
- [ ] 更新 `internal/storage/mock.go` 添加新的 9 个方法
- [ ] 实现基本的内存存储逻辑

---

### Phase 1B.4: Update Service Layer (更新服务层)

#### 4.1 Monitor Service
- [ ] 添加 `"context"` 导入到 `internal/monitor/monitor.go`
- [ ] `healthCheck()`: 添加 `ctx := context.Background()` 并传递给 `okxClient.HealthCheck(ctx)`
- [ ] `fetchAndStoreBalances()`: 添加 context 到 `GetAccountBalance(ctx)`
- [ ] `fetchAndStorePositions()`: 添加 context 到 `GetPositions(ctx)`

#### 4.2 TPSL Manager
- [ ] 添加 `"context"` 导入到 `internal/tpsl/manager.go`
- [ ] `AnalyzePositions()`: 添加 context 到 `GetPendingAlgoOrders(ctx, ...)`
- [ ] `getCurrentMarketPrice()`: 添加 context 到 `GetTicker(ctx, ...)`
- [ ] 所有 `PlaceAlgoOrder()` 调用: 添加 context（约 4-6 处）

#### 4.3 TPSL Scheduler
- [ ] 添加 `"context"` 导入到 `internal/tpsl/scheduler.go`
- [ ] `checkAndPlaceTPSL()`: 添加 context 到 `GetPositions(ctx)`

---

### Phase 1B.5: Update Main (更新主程序)

- [ ] 在 `cmd/main.go` 中创建根 context
- [ ] 传递 context 到各个服务（如果需要）

---

### Phase 1B.6: Verification (验证)

#### 6.1 编译检查 / Compile Check
```bash
go build ./...
```
- [ ] 无编译错误
- [ ] 所有接口实现检查通过

#### 6.2 运行测试 / Run Tests
```bash
go test ./...
```
- [ ] 所有测试通过

#### 6.3 代码检查 / Code Check
```bash
go vet ./...
go fmt ./...
```
- [ ] 通过 vet 检查
- [ ] 代码格式化

#### 6.4 功能测试 / Functional Test
- [ ] Feature 1 (Monitor) 正常工作
- [ ] Feature 2 (TPSL) 正常工作
- [ ] 无运行时错误

---

## 🔄 继续任务的命令 / Commands to Resume

### 如果 Token 用完了，下次对话时使用：

```
继续 PHASE1B_TODO.md 中的工作
```

或者更具体：

```
继续 Phase 1B.1: 实现 OKX Client Context 方法
```

### 我会做什么 / What I Will Do:

1. 读取 `PHASE1B_TODO.md` 查看进度
2. 从第一个未完成的任务开始
3. 逐步完成每个任务
4. 在完成后更新此文件的进度（在对应项前打勾 ✓）

---

## 📊 进度追踪 / Progress Tracking

### 总体进度 / Overall Progress
- **Phase 1B.1**: ⏳ 0/15 任务完成 (0%)
- **Phase 1B.2**: ⏳ 0/14 任务完成 (0%)
- **Phase 1B.3**: ⏳ 0/2 任务完成 (0%)
- **Phase 1B.4**: ⏳ 0/7 任务完成 (0%)
- **Phase 1B.5**: ⏳ 0/2 任务完成 (0%)
- **Phase 1B.6**: ⏳ 0/7 任务完成 (0%)

**总计 / Total**: 0/47 任务完成 (0%)

### 预计时间 / Estimated Time
- Phase 1B.1: ~2 小时
- Phase 1B.2: ~2 小时
- Phase 1B.3: ~30 分钟
- Phase 1B.4: ~1 小时
- Phase 1B.5: ~15 分钟
- Phase 1B.6: ~30 分钟

**总计 / Total**: ~6-7 小时

---

## 📝 当前状态 / Current Status

**最后更新 / Last Update**: 2025-12-03

**当前正在进行 / Currently Working On**: 无 / None

**下一个任务 / Next Task**: Phase 1B.1.1 - 添加内部辅助方法

**遇到的问题 / Issues Encountered**: 无 / None

---

## 💡 实施提示 / Implementation Tips

### OKX Client 实现提示
1. 所有代码示例都在 `REFACTORING_GUIDE.md` 中
2. 保持向后兼容：新方法签名必须与接口一致
3. Context 取消检查：在重试循环中添加 `select { case <-ctx.Done(): return ctx.Err() }`

### Storage 实现提示
1. 使用事务确保原子性
2. 添加适当的索引提高查询性能
3. 周查询使用 `WHERE week_start = ?` 而不是时间范围查询

### 服务层更新提示
1. 使用 `context.Background()` 作为默认 context
2. 如果有超时需求，使用 `context.WithTimeout()`
3. 保持现有日志和错误处理不变

---

## ✅ 完成标准 / Completion Criteria

Phase 1B 完成的标准：

1. ✅ 所有代码可以编译（`go build ./...`）
2. ✅ 所有接口实现检查通过（取消注释后无错误）
3. ✅ 所有测试通过（`go test ./...`）
4. ✅ Feature 1 和 Feature 2 功能正常
5. ✅ 代码通过 `go vet` 和 `go fmt` 检查
6. ✅ 所有新方法都有适当的文档注释

---

## 🎯 完成后的下一步 / Next Steps After Completion

Phase 1B 完成后，你可以：

1. **开始 Feature 3 开发** - 所有基础设施就绪
2. **编写集成测试** - 测试完整的工作流
3. **优化性能** - 使用 context timeout、并发等
4. **添加更多通知渠道** - Email、Telegram 等

---

**注意 / Note**:
- 此文件会在实施过程中持续更新
- 每完成一个任务，会在对应项前打勾 ✓
- 如果遇到问题，会在"遇到的问题"部分记录
