# TenyoJubaku CLI 使用指南 / CLI Usage Guide

## 概述 / Overview

TenyoJubaku 包含两个可执行文件：

1. **`tenyojubaku`** - 主服务（持续运行）
   - 监控账户余额和持仓
   - 自动管理止盈止损（TPSL）
   - 后台持续运行

2. **`tenyojubaku-cli`** - 命令行工具（一次性操作）
   - 手动下单（带频率限制和 Maker-only 检查）
   - 查看订单统计
   - 查看订单历史

---

## 构建 / Build

```bash
# 构建主服务
go build -o tenyojubaku ./cmd

# 构建 CLI 工具
go build -o tenyojubaku-cli ./cmd/cli
```

---

## 使用方式 / Usage

### 1. 启动主服务（后台监控）

在一个终端窗口中运行：

```bash
./tenyojubaku
```

主服务会持续运行，执行以下任务：
- 每 60 秒监控账户余额和持仓
- 每 5 分钟检查并设置止盈止损订单
- 记录所有数据到 SQLite 数据库

**保持这个窗口运行**，不要关闭。

---

### 2. 使用 CLI 工具（手动操作）

打开**另一个终端窗口**，使用 CLI 工具进行手动操作。

#### 查看帮助

```bash
./tenyojubaku-cli help
```

---

## CLI 命令详解 / CLI Commands

### 📊 查看本周订单统计

```bash
./tenyojubaku-cli order stats
```

**输出示例**：
```
Order Statistics for Week Starting 2025-12-08
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Orders:        15
Regular Orders:      12
Reduce-Only Orders:  3

By Status:
  Placed:   10
  Filled:   4
  Canceled: 1
  Failed:   0

Frequency Limit:
  Used:      12 / 20 (regular orders only)
  Remaining: 8
```

**说明**：
- 显示当前周（周一 00:00 UTC 开始）的订单统计
- 如果启用了频率限制，会显示已用/剩余配额
- 根据配置，可能只计算普通订单（排除 reduce-only）

---

### 📋 查看订单历史

```bash
# 查看最近 10 笔订单（默认）
./tenyojubaku-cli order list

# 查看最近 20 笔订单
./tenyojubaku-cli order list --limit 20
```

**输出示例**：
```
Recent Orders (Week Starting 2025-12-08)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

TIME          INSTRUMENT       SIDE  TYPE   SIZE   PRICE   STATUS
────          ──────────       ────  ────   ────   ─────   ──────
12-08 14:30   BTC-USDT-SWAP    buy   limit  0.01   49500   placed
12-08 14:25   ETH-USDT-SWAP    sell  limit  0.1    3200    placed ®
12-08 13:15   BTC-USDT-SWAP    buy   limit  0.01   49800   filled

(Showing 3 of 15 orders. Use --limit to show more)
```

**说明**：
- `®` 标记表示 reduce-only 订单（仅平仓）
- 只显示当前周的订单
- 按时间倒序排列（最新的在最上面）

---

### 📤 手动下单

#### 基本用法

```bash
./tenyojubaku-cli order place \
  --instrument <合约ID> \
  --side <买/卖> \
  --size <数量> \
  --price <价格> \
  [--type <订单类型>] \
  [--reduce-only]
```

#### 参数说明

| 参数 | 必需 | 说明 | 示例 |
|------|------|------|------|
| `--instrument` | ✓ | 合约 ID | `BTC-USDT-SWAP` |
| `--side` | ✓ | 方向：buy 或 sell | `buy` |
| `--size` | ✓ | 订单数量 | `0.01` |
| `--price` | * | 订单价格（limit/post_only 必需） | `50000` |
| `--type` | | 订单类型（默认：limit）<br>可选：limit, post_only, market | `limit` |
| `--reduce-only` | | 仅平仓（不开新仓） | （flag，无值） |

---

### 示例场景 / Examples

#### 1. 买入 BTC（限价单）

```bash
./tenyojubaku-cli order place \
  --instrument BTC-USDT-SWAP \
  --side buy \
  --size 0.01 \
  --price 50000 \
  --type limit
```

**执行流程**：
1. ✅ 检查频率限制（本周是否还有配额）
2. ✅ 检查 Maker-only（价格是否距离市场价足够远）
3. ✅ 调用 OKX API 下单
4. ✅ 记录订单历史到数据库

**成功输出**：
```
Placing order...
  Instrument: BTC-USDT-SWAP
  Side:       buy
  Type:       limit
  Size:       0.01
  Price:      50000
  Reduce-only: false

✓ Order placed successfully!
  Order ID: 123456789
```

**失败示例（频率限制）**：
```
✗ Order failed: frequency limit check failed: weekly order limit exceeded: 20/20 orders (week starting 2025-12-08)
```

**失败示例（Maker-only）**：
```
✗ Order failed: maker-only check failed: price distance 0.02% is less than minimum 0.05% (order: 50050.00, market: 50100.00)
```

---

#### 2. 卖出 BTC（平仓，reduce-only）

```bash
./tenyojubaku-cli order place \
  --instrument BTC-USDT-SWAP \
  --side sell \
  --size 0.01 \
  --price 51000 \
  --reduce-only
```

**说明**：
- `--reduce-only` 确保只能平仓，不能开新仓
- 如果配置了 `exclude_reduce_only: true`，此订单不计入频率限制

---

#### 3. Post-only 订单（保证 Maker）

```bash
./tenyojubaku-cli order place \
  --instrument ETH-USDT-SWAP \
  --side buy \
  --size 0.1 \
  --price 3200 \
  --type post_only
```

**说明**：
- `post_only` 订单保证是 Maker（不会立即成交）
- 如果价格会导致立即成交，订单会被拒绝
- 不需要检查价格距离（OKX 保证其为 Maker）

---

#### 4. 市价单（如果允许）

```bash
./tenyojubaku-cli order place \
  --instrument BTC-USDT-SWAP \
  --side buy \
  --size 0.01 \
  --type market
```

**注意**：
- 如果 `maker_only.enabled: true`，市价单会被拒绝
- 市价单总是 Taker（支付更高手续费）

---

## Order Control 功能 / Order Control Features

CLI 工具通过 Order Control 服务下单，自动执行以下检查：

### 1. 频率限制（Frequency Limiting）

配置：`configs/config.yaml`
```yaml
order_control:
  frequency_limit:
    enabled: true
    weekly_max_orders: 20           # 每周最多 20 单
    exclude_reduce_only: true       # 不计算平仓订单
```

**行为**：
- 统计本周（周一 00:00 UTC 起）的订单数量
- 超过限制时拒绝下单
- Reduce-only 订单可选择排除

---

### 2. Maker-only 模式

配置：`configs/config.yaml`
```yaml
order_control:
  maker_only:
    enabled: true
    min_price_distance_pct: 0.05    # 最小 0.05% 价格距离
```

**行为**：
- 拒绝市价单（market）
- 检查限价单价格距离：
  - **买单**：价格必须 ≤ 市场 ask - 0.05%
  - **卖单**：价格必须 ≥ 市场 bid + 0.05%
- `post_only` 订单直接通过

**目的**：
- 享受 Maker 费率优惠（-0.02% 到 0.02%）
- 避免 Taker 费率（0.05% 到 0.08%）

---

## 配置示例 / Configuration Example

### 启用所有 Order Control 功能

```yaml
order_control:
  enabled: true

  frequency_limit:
    enabled: true
    weekly_max_orders: 20
    exclude_reduce_only: true

  maker_only:
    enabled: true
    min_price_distance_pct: 0.05
```

### 只启用频率限制

```yaml
order_control:
  enabled: true

  frequency_limit:
    enabled: true
    weekly_max_orders: 20
    exclude_reduce_only: true

  maker_only:
    enabled: false
```

### 禁用 Order Control

```yaml
order_control:
  enabled: false
```

---

## 常见问题 / FAQ

### Q: 主服务和 CLI 会冲突吗？

**A:** 不会。它们共享同一个数据库，但：
- 主服务：持续运行 Monitor 和 TPSL
- CLI 工具：一次性操作，执行完就退出
- 两者互不干扰

---

### Q: CLI 下单会影响主服务吗？

**A:** 不会。CLI 工具：
- 只是调用 OKX API 下单
- 记录订单历史到数据库
- 不会中断或影响主服务

---

### Q: 如何查看主服务是否在运行？

**A:**
```bash
# macOS/Linux
ps aux | grep tenyojubaku

# 或者查看日志
tail -f logs/app.log
```

---

### Q: 订单被拒绝了，怎么办？

**A:** 检查错误信息：

1. **频率限制**：`weekly order limit exceeded`
   - 等到下周一 00:00 UTC
   - 或者修改 `weekly_max_orders` 配置

2. **Maker-only**：`price distance ... is less than minimum`
   - 调整订单价格，使其距离市场价更远
   - 或者使用 `--type post_only`
   - 或者禁用 `maker_only.enabled: false`

3. **OKX API 错误**：`Order rejected by OKX`
   - 检查账户余额
   - 检查订单参数（size, price 等）
   - 查看 OKX 错误代码

---

### Q: 如何禁用 Order Control？

**A:** 编辑 `configs/config.yaml`：
```yaml
order_control:
  enabled: false
```

此时 CLI 工具仍然可用，但不会执行频率和 Maker-only 检查。

---

## 技术细节 / Technical Details

### 数据库

- **位置**：`./data/tenyojubaku.db`（SQLite）
- **表**：`order_history`
- **字段**：
  - `order_id`: OKX 订单 ID
  - `inst_id`: 合约 ID
  - `side`: buy/sell
  - `ord_type`: limit/market/post_only
  - `size`: 订单数量
  - `price`: 订单价格
  - `reduce_only`: 是否仅平仓
  - `placed_at`: 下单时间
  - `week_start`: 周起始时间（用于频率统计）
  - `status`: placed/filled/canceled/failed

### 周计算逻辑

- **周起始**：每周一 00:00:00 UTC
- **函数**：`models.GetWeekStart(time.Now())`
- **示例**：
  - 2025-12-08 (周一) → week_start: 2025-12-08 00:00:00
  - 2025-12-10 (周三) → week_start: 2025-12-08 00:00:00
  - 2025-12-14 (周日) → week_start: 2025-12-08 00:00:00
  - 2025-12-15 (周一) → week_start: 2025-12-15 00:00:00

---

## 进阶使用 / Advanced Usage

### 脚本批量下单

创建 Shell 脚本：

```bash
#!/bin/bash
# place_orders.sh

# BTC 买入梯度订单
./tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 49000
./tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 48500
./tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 48000

# ETH 买入
./tenyojubaku-cli order place --instrument ETH-USDT-SWAP --side buy --size 0.1 --price 3100
```

**注意**：批量下单仍受频率限制约束。

---

### 定时任务

使用 cron 定时执行：

```bash
# 每天早上 9:00 查看订单统计
0 9 * * * cd /path/to/TenyoJubaku && ./tenyojubaku-cli order stats >> logs/daily_stats.log
```

---

## 总结 / Summary

- ✅ **主服务**（`tenyojubaku`）：后台持续运行，监控和 TPSL
- ✅ **CLI 工具**（`tenyojubaku-cli`）：手动下单和查询，一次性操作
- ✅ **互不干扰**：共享数据库，但各自独立运行
- ✅ **Order Control**：自动频率限制 + Maker-only 检查
- ✅ **灵活配置**：可随时启用/禁用各项功能

---

**祝交易顺利！Happy Trading! 🚀**
