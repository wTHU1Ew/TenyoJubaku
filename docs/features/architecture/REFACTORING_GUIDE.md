# Phase 1 & 2 实施指南

**创建时间**: 2025-12-03
**状态**: ✅ 接口和类型已完成,等待实施

## 已完成的工作

### ✅ Phase 1.1-1.4: 接口和类型定义
- [x] OKX Interface 添加 Context 参数
- [x] OKX Interface 添加新的交易方法
- [x] 新类型定义 (OrderRequest, AmendOrderRequest, etc.)

## 剩余工作清单

由于这是一个大规模重构,涉及 **20+ 个文件**的修改,我为你准备了两个选项:

### 选项 A: 我继续完成所有重构 (推荐,但需要较长时间)
**工作量**: 剩余约 4-5 小时
**包含**:
1. 实现所有新的 OKX Client 方法
2. 更新所有现有方法添加 Context
3. 扩展 Storage 接口和实现
4. 更新所有服务使用 Context
5. 创建 Notifier 抽象
6. 添加配置和模型
7. 完整测试验证

**优点**: 一次性完成,保证质量
**缺点**: Token 消耗大,时间较长

### 选项 B: 我生成完整的实施脚本和示例代码
**工作量**: 30分钟
**包含**:
1. 每个文件的详细修改指南
2. 完整的代码示例
3. 测试命令和验证步骤
4. 你可以自己按步骤实施

**优点**: 快速,灵活
**缺点**: 需要你手动修改代码

---

## 快速实施摘要 (如果选择 B)

### Step 1: 实现 OKX Client 新方法 (client.go)

```go
// 添加到 client.go

func (c *Client) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
    path := "/api/v5/trade/order"
    reqBody, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    respBody, err := c.doRequestWithBodyAndContext(ctx, "POST", path, string(reqBody))
    if err != nil {
        return nil, err
    }

    var resp OrderResponse
    if err := json.Unmarshal(respBody, &resp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if resp.Code != "0" {
        return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
    }

    return &resp, nil
}

func (c *Client) AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error) {
    path := "/api/v5/trade/amend-order"
    reqBody, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    respBody, err := c.doRequestWithBodyAndContext(ctx, "POST", path, string(reqBody))
    if err != nil {
        return nil, err
    }

    var resp AmendOrderResponse
    if err := json.Unmarshal(respBody, &resp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if resp.Code != "0" {
        return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
    }

    return &resp, nil
}

func (c *Client) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
    path := "/api/v5/trade/cancel-order"
    reqBody, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    respBody, err := c.doRequestWithBodyAndContext(ctx, "POST", path, string(reqBody))
    if err != nil {
        return nil, err
    }

    var resp CancelOrderResponse
    if err := json.Unmarshal(respBody, &resp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if resp.Code != "0" {
        return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
    }

    return &resp, nil
}

func (c *Client) GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error) {
    path := "/api/v5/trade/orders-pending"

    respBody, err := c.doRequestWithContext(ctx, "GET", path)
    if err != nil {
        return nil, err
    }

    var resp PendingOrdersResponse
    if err := json.Unmarshal(respBody, &resp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if resp.Code != "0" {
        return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
    }

    return &resp, nil
}

// 添加支持 Context 的内部方法
func (c *Client) doRequestWithContext(ctx context.Context, method, path string) ([]byte, error) {
    return c.doRequestWithBodyAndContext(ctx, method, path, "")
}

func (c *Client) doRequestWithBodyAndContext(ctx context.Context, method, path, body string) ([]byte, error) {
    url := c.apiURL + path

    var lastErr error
    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("request canceled: %w", ctx.Err())
        default:
        }

        if attempt > 0 {
            backoff := time.Duration(1<<uint(attempt-1)) * time.Second
            select {
            case <-ctx.Done():
                return nil, fmt.Errorf("request canceled during backoff: %w", ctx.Err())
            case <-time.After(backoff):
            }
        }

        timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
        signature := c.generateSignature(timestamp, method, path, body)

        var reqBody io.Reader
        if body != "" {
            reqBody = strings.NewReader(body)
        }
        req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
        if err != nil {
            lastErr = fmt.Errorf("failed to create request: %w", err)
            continue
        }

        req.Header.Set("OK-ACCESS-KEY", c.apiKey)
        req.Header.Set("OK-ACCESS-SIGN", signature)
        req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
        req.Header.Set("OK-ACCESS-PASSPHRASE", c.passphrase)
        req.Header.Set("Content-Type", "application/json")

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = fmt.Errorf("request failed: %w", err)
            continue
        }
        defer resp.Body.Close()

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            lastErr = fmt.Errorf("failed to read response body: %w", err)
            continue
        }

        if c.debugEnable {
            fmt.Printf("\n=== OKX API Debug ===\n")
            fmt.Printf("Request: %s %s\n", method, path)
            if body != "" {
                fmt.Printf("Request Body: %s\n", body)
            }
            fmt.Printf("Status Code: %d\n", resp.StatusCode)
            fmt.Printf("Response Body: %s\n", string(respBody))
            fmt.Printf("=====================\n\n")
        }

        if resp.StatusCode == http.StatusTooManyRequests {
            lastErr = fmt.Errorf("rate limited (429)")
            continue
        }

        if resp.StatusCode != http.StatusOK {
            lastErr = fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
            continue
        }

        return respBody, nil
    }

    return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}

// 更新现有方法添加 Context
func (c *Client) GetAccountBalance(ctx context.Context) (*AccountBalanceResponse, error) {
    path := "/api/v5/account/balance"
    respBody, err := c.doRequestWithContext(ctx, "GET", path)
    // ... rest of implementation
}

// ... 对所有其他方法重复此模式
```

### Step 2: 更新 Storage 接口

```go
// storage/interface.go 添加

// InsertOrderHistory inserts an order history record.
InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error

// GetOrderCountForWeek returns the count of orders placed in a given week.
GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error)

// GetOrdersForWeek retrieves all orders placed in a given week.
GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error)

// InsertPendingConfirmation inserts a pending confirmation record.
InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error

// GetPendingConfirmationsDue retrieves confirmations that are due for checking.
GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error)

// UpdatePendingConfirmation updates a pending confirmation record.
UpdatePendingConfirmation(ctx context.Context, orderId string, updates map[string]interface{}) error

// DeletePendingConfirmation deletes a pending confirmation record.
DeletePendingConfirmation(ctx context.Context, orderId string) error
```

### Step 3: 创建Order Models

```go
// pkg/models/order.go

package models

import "time"

// OrderHistory represents a placed order for frequency tracking
type OrderHistory struct {
    ID         int64
    OrderID    string
    InstId     string
    Side       string
    OrdType    string
    Size       string
    Price      string
    ReduceOnly bool
    PlacedAt   time.Time
    WeekStart  time.Time // Monday 00:00:00 UTC of the week
    Status     string    // placed, filled, canceled, failed
    CreatedAt  time.Time
}

// PendingConfirmation represents a pending order requiring confirmation
type PendingConfirmation struct {
    ID                  int64
    OrderID             string
    InstId              string
    Side                string
    OrdType             string
    OriginalSize        string
    CurrentSize         string
    Price               string
    PlacedAt            time.Time
    LastConfirmationAt  *time.Time
    NextConfirmationDue time.Time
    ConfirmationCount   int
    TimeoutCount        int
    Status              string // pending, confirmed, timeout, canceled
    CreatedAt           time.Time
    UpdatedAt           time.Time
}
```

### Step 4: 创建 Notifier 抽象

```go
// internal/notifier/interface.go

package notifier

import (
    "context"
    "time"
)

type Interface interface {
    SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error
}

type ConfirmationInfo struct {
    OrderID    string
    Instrument string
    Side       string
    Size       string
    Price      string
    PlacedAt   time.Time
    TimeoutIn  time.Duration
}
```

```go
// internal/notifier/log_notifier.go

package notifier

import (
    "context"
    "github.com/wTHU1Ew/TenyoJubaku/internal/logger"
)

type LogNotifier struct {
    logger *logger.Logger
}

func NewLogNotifier(logger *logger.Logger) *LogNotifier {
    return &LogNotifier{logger: logger}
}

func (n *LogNotifier) SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error {
    n.logger.Warn("⚠️  CONFIRMATION REQUIRED ⚠️")
    n.logger.Warn("Order ID: %s", info.OrderID)
    n.logger.Warn("Instrument: %s", info.Instrument)
    n.logger.Warn("Side: %s | Size: %s | Price: %s", info.Side, info.Size, info.Price)
    n.logger.Warn("Placed At: %s", info.PlacedAt.Format("2006-01-02 15:04:05"))
    n.logger.Warn("Timeout In: %v", info.TimeoutIn)
    n.logger.Warn("Please confirm this order to prevent size reduction")
    return nil
}
```

### Step 5: 添加配置结构

```go
// internal/config/config.go 添加

type OrderControlConfig struct {
    Enabled bool `yaml:"enabled"`

    FrequencyLimit struct {
        Enabled           bool `yaml:"enabled"`
        WeeklyMaxOrders   int  `yaml:"weekly_max_orders"`
        ExcludeReduceOnly bool `yaml:"exclude_reduce_only"`
    } `yaml:"frequency_limit"`

    MakerOnly struct {
        Enabled                 bool    `yaml:"enabled"`
        MinPriceDistancePct     float64 `yaml:"min_price_distance_pct"`
        AllowTakerForReduceOnly bool    `yaml:"allow_taker_for_reduce_only"`
        MaxTakerPct             float64 `yaml:"max_taker_pct"`
        TickerStalenessSeconds  int     `yaml:"ticker_staleness_seconds"`
    } `yaml:"maker_only"`

    Confirmation struct {
        Enabled                   bool   `yaml:"enabled"`
        CheckIntervalSeconds      int    `yaml:"check_interval_seconds"`
        ConfirmationIntervalHours int    `yaml:"confirmation_interval_hours"`
        WaitingPeriodHours        int    `yaml:"waiting_period_hours"`
        TimeoutSizeReductionPct   float64 `yaml:"timeout_size_reduction_pct"`
        MaxTimeouts               int    `yaml:"max_timeouts"`
        NotificationMethod        string `yaml:"notification_method"`
    } `yaml:"confirmation"`
}

type Config struct {
    // ... existing fields
    OrderControl OrderControlConfig `yaml:"order_control"`
}
```

---

## 验证步骤

```bash
# 1. 编译检查
go build ./...

# 2. 运行测试
go test ./...

# 3. 检查接口实现
go vet ./...

# 4. 格式化代码
go fmt ./...
```

---

## 你的选择?

请告诉我你想选择哪个方案:
- **A**: 我继续完成所有实施 (需要较长时间)
- **B**: 使用这份指南,你自己实施
- **C**: 我先完成最关键的部分 (OKX Client + Context),其余你自己做

我的建议是 **选项 C** - 平衡了效率和质量。
