package okx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client OKX API客户端 / OKX API client
type Client struct {
	apiURL      string
	apiKey      string
	apiSecret   string
	passphrase  string
	httpClient  *http.Client
	maxRetries  int
	debugEnable bool
}

// New 创建新的OKX客户端 / Create new OKX client
// 初始化OKX API客户端，配置HTTP超时和重试策略
// Initialize OKX API client with HTTP timeout and retry strategy
//
// Parameters:
//   - apiURL: OKX API base URL (e.g., "https://www.okx.com")
//   - apiKey: API key from OKX account settings
//   - apiSecret: API secret corresponding to the API key
//   - passphrase: API passphrase set during key creation
//   - timeout: HTTP request timeout in seconds
//   - maxRetries: Maximum retry attempts on request failure
//   - debugEnable: Whether to print API responses for debugging
//
// Returns:
//   - *Client: 配置完成的OKX客户端实例 / Configured OKX client instance ready for API calls
func New(apiURL, apiKey, apiSecret, passphrase string, timeout, maxRetries int, debugEnable bool) *Client {
	return &Client{
		apiURL:      apiURL,
		apiKey:      apiKey,
		apiSecret:   apiSecret,
		passphrase:  passphrase,
		httpClient:  &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		maxRetries:  maxRetries,
		debugEnable: debugEnable,
	}
}

// generateSignature 生成API签名 / Generate API signature
// 使用HMAC-SHA256算法生成OKX API请求签名（复杂算法详解）
// Generate OKX API request signature using HMAC-SHA256 algorithm (complex algorithm explained)
//
// 算法详解 / Algorithm Details:
// 1. 构建预哈希字符串: timestamp + method + requestPath + body
//    Build prehash string: timestamp + method + requestPath + body
//    例如 / Example: "2023-01-01T12:00:00.000ZGET/api/v5/account/balance"
//
// 2. 使用HMAC-SHA256计算哈希值，密钥为API Secret
//    Calculate hash using HMAC-SHA256 with API Secret as key
//    HMAC提供消息认证，确保请求未被篡改
//    HMAC provides message authentication, ensuring request hasn't been tampered
//
// 3. 将结果编码为Base64字符串
//    Encode result as Base64 string
//    OKX API要求签名必须为Base64格式
//    OKX API requires signature to be in Base64 format
//
// Parameters:
//   - timestamp: UTC timestamp in ISO8601 format (e.g., "2023-01-01T12:00:00.000Z")
//   - method: HTTP method (GET, POST, etc.)
//   - requestPath: API path including query parameters (e.g., "/api/v5/account/balance")
//   - body: Request body content (empty string for GET requests)
//
// Returns:
//   - string: Base64编码的HMAC-SHA256签名 / Base64-encoded HMAC-SHA256 signature
//     用于OK-ACCESS-SIGN请求头 / Used in OK-ACCESS-SIGN request header
func (c *Client) generateSignature(timestamp, method, requestPath, body string) string {
	// Create prehash string: timestamp + method + requestPath + body
	prehash := timestamp + method + requestPath + body

	// Calculate HMAC SHA256
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(prehash))

	// Encode to base64
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// doRequest 执行HTTP请求（带重试机制的复杂算法）/ Execute HTTP request with retry mechanism (complex algorithm)
// 发送HTTP请求到OKX API，包含签名认证、指数退避重试和错误处理
// Send HTTP request to OKX API with signature authentication, exponential backoff retry, and error handling
//
// 重试算法详解 / Retry Algorithm Details:
// - 初始尝试 + 最多maxRetries次重试 / Initial attempt + up to maxRetries retries
// - 指数退避策略: 第n次重试等待 2^(n-1) 秒 / Exponential backoff: nth retry waits 2^(n-1) seconds
//   例如 / Example: 1st retry = 1s, 2nd retry = 2s, 3rd retry = 4s
// - 仅在可恢复错误时重试（网络错误、429限流）/ Retry only on recoverable errors (network errors, 429 rate limits)
// - 其他错误立即返回 / Other errors return immediately
//
// Parameters:
//   - method: HTTP method ("GET", "POST", etc.)
//   - path: API endpoint path (e.g., "/api/v5/account/balance")
//
// Returns:
//   - []byte: API响应的原始字节数据 / Raw byte data from API response
//   - error: 请求失败时返回错误（所有重试均失败后）/ Error on request failure (after all retries exhausted)
//     包含最后一次失败的具体原因 / Contains reason for the last failure
func (c *Client) doRequest(ctx context.Context, method, path string) ([]byte, error) {
	return c.doRequestWithBody(ctx, method, path, "")
}

// doRequestWithBody 执行带请求体的HTTP请求 / Execute HTTP request with body
// 支持POST请求，包含JSON请求体的签名认证
// Support POST requests with JSON body and signature authentication
//
// Parameters:
//   - method: HTTP method ("GET", "POST", etc.)
//   - path: API endpoint path (e.g., "/api/v5/trade/order-algo")
//   - body: 请求体内容（JSON字符串）/ Request body content (JSON string)
//
// Returns:
//   - []byte: API响应的原始字节数据 / Raw byte data from API response
//   - error: 请求失败时返回错误（所有重试均失败后）/ Error on request failure (after all retries exhausted)
func (c *Client) doRequestWithBody(ctx context.Context, method, path, body string) ([]byte, error) {
	url := c.apiURL + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		default:
		}
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("request canceled during backoff: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		// Generate timestamp (ISO8601 format)
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

		// Generate signature
		signature := c.generateSignature(timestamp, method, path, body)

		// Create HTTP request
		var reqBody io.Reader
		if body != "" {
			reqBody = strings.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set headers
		req.Header.Set("OK-ACCESS-KEY", c.apiKey)
		req.Header.Set("OK-ACCESS-SIGN", signature)
		req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
		req.Header.Set("OK-ACCESS-PASSPHRASE", c.passphrase)
		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Debug: print API response if enabled
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

		// Check status code
		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limited, retry with backoff
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		// Success
		return respBody, nil
	}

	// All retries exhausted
	return nil, fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}

// GetAccountBalance 获取账户余额 / Get account balance
// 从OKX API获取账户余额信息，包含所有币种的余额、可用余额、冻结余额等
// Fetch account balance information from OKX API, including balance, available, frozen for all currencies
//
// Returns:
//   - *AccountBalanceResponse: 账户余额响应对象 / Account balance response object
//     包含Data字段，其中Details数组列出每个币种的余额详情
//     Contains Data field with Details array listing balance details for each currency
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"
//     Possible causes: network error, authentication failure, API error code not "0"
func (c *Client) GetAccountBalance(ctx context.Context) (*AccountBalanceResponse, error) {
	path := "/api/v5/account/balance"

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp AccountBalanceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

// GetPositions 获取持仓信息 / Get positions
// 从OKX API获取所有持仓信息，包含合约、持仓量、未实现盈亏等
// Fetch all position information from OKX API, including contracts, position size, unrealized PnL, etc.
//
// Returns:
//   - *PositionsResponse: 持仓响应对象 / Positions response object
//     包含Data字段，数组中每个元素代表一个持仓
//     Contains Data field, each element in array represents one position
//     如果没有持仓，Data数组为空 / If no positions, Data array is empty
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"
//     Possible causes: network error, authentication failure, API error code not "0"
func (c *Client) GetPositions(ctx context.Context) (*PositionsResponse, error) {
	path := "/api/v5/account/positions"

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp PositionsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

// GetPositionsHistory 获取历史仓位（最近3个月）/ Get positions history (last 3 months)
// Retrieves closed positions from the last 3 months with complete PnL data.
// Perfect for analyzing trading performance at the position level.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - instType: Instrument type (MARGIN, SWAP, FUTURES, OPTION)
//   - limit: Number of results per request (max 100, default 100)
//
// Returns: Positions history response with closed position data, or error
func (c *Client) GetPositionsHistory(ctx context.Context, instType string, limit int) (*PositionsHistoryResponse, error) {
	// Validate and set default limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	// Build request path with parameters
	path := fmt.Sprintf("/api/v5/account/positions-history?instType=%s&limit=%d", instType, limit)

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	var resp PositionsHistoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

// HealthCheck 健康检查 / Health check by testing API connectivity
// 通过尝试获取账户余额来验证API连接和认证是否正常
// Verify API connectivity and authentication by attempting to fetch account balance
//
// Returns:
//   - error: 连接失败或认证失败时返回错误 / Error on connection failure or authentication failure
//     nil表示API连接正常，认证通过 / nil indicates API is reachable and authenticated
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.GetAccountBalance(ctx)
	return err
}

// GetPendingAlgoOrders 获取待处理算法订单 / Get pending algo orders
// 从OKX API获取待处理的算法订单（如条件单、止盈止损单等）
// Fetch pending algo orders from OKX API (e.g., conditional orders, TPSL orders, etc.)
//
// Parameters:
//   - ordType: 订单类型 / Order type, e.g., "conditional" for TPSL orders
//     可选值 / Valid values: "conditional", "oco", "trigger", "move_order_stop", etc.
//
// Returns:
//   - *PendingAlgoOrdersResponse: 待处理算法订单响应对象 / Pending algo orders response object
//     包含Data字段，数组中每个元素代表一个待处理的算法订单
//     Contains Data field with array of pending algo orders
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"
//     Possible causes: network error, authentication failure, API error code not "0"
func (c *Client) GetPendingAlgoOrders(ctx context.Context, ordType string) (*PendingAlgoOrdersResponse, error) {
	path := "/api/v5/trade/orders-algo-pending?ordType=" + ordType

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp PendingAlgoOrdersResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

// PlaceAlgoOrder 下单算法订单 / Place algo order
// 向OKX API下单算法订单（如条件单、止盈止损单等）
// Place algo order to OKX API (e.g., conditional orders, TPSL orders, etc.)
//
// Parameters:
//   - req: 算法订单请求对象 / Algo order request object
//     包含订单参数（交易对、订单类型、止盈止损价格等）
//     Contains order parameters (instrument, order type, TPSL prices, etc.)
//
// Returns:
//   - *AlgoOrderResponse: 算法订单响应对象 / Algo order response object
//     包含Data字段，其中包含algoId（订单ID）
//     Contains Data field with algoId (order ID)
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"、参数错误
//     Possible causes: network error, authentication failure, API error code not "0", invalid parameters
func (c *Client) PlaceAlgoOrder(ctx context.Context, req AlgoOrderRequest) (*AlgoOrderResponse, error) {
	path := "/api/v5/trade/order-algo"

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequestWithBody(ctx, "POST", path, string(reqBody))
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp AlgoOrderResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	// Check for order-specific errors
	if len(resp.Data) > 0 && resp.Data[0].SCode != "" && resp.Data[0].SCode != "0" {
		return nil, fmt.Errorf("order placement error: code=%s, msg=%s", resp.Data[0].SCode, resp.Data[0].SMsg)
	}

	return &resp, nil
}

// AmendAlgoOrder 修改算法订单 / Amend algo order
// 修改现有算法订单的参数（如止盈止损触发价格）
// Modify parameters of existing algo order (e.g., take-profit/stop-loss trigger prices)
// 这是实现动态追踪止损的关键方法
// This is the critical method for implementing dynamic trailing stop-loss
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: 修改算法订单请求 / Amend algo order request
//     包含algoId（要修改的订单）和新参数（如新止损价格）
//     Contains algoId (order to modify) and new parameters (e.g., new stop-loss price)
//
// Returns:
//   - *AmendAlgoOrderResponse: 修改响应对象 / Amend response object
//     包含Data字段，其中包含修改后的algoId
//     Contains Data field with amended algoId
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"、订单不存在或无法修改
//     Possible causes: network error, authentication failure, API error code not "0", order not found or cannot be amended
func (c *Client) AmendAlgoOrder(ctx context.Context, req *AmendAlgoOrderRequest) (*AmendAlgoOrderResponse, error) {
	path := "/api/v5/trade/amend-algos"

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequestWithBody(ctx, "POST", path, string(reqBody))
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp AmendAlgoOrderResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	// Check for order-specific errors
	if len(resp.Data) > 0 && resp.Data[0].SCode != "" && resp.Data[0].SCode != "0" {
		return nil, fmt.Errorf("order amendment error: code=%s, msg=%s", resp.Data[0].SCode, resp.Data[0].SMsg)
	}

	return &resp, nil
}

// GetTicker 获取行情数据 / Get ticker data
// 从OKX API获取指定交易对的行情数据，包含最新成交价、买卖价等
// Fetch ticker data for specified instrument from OKX API, including last price, bid/ask prices, etc.
//
// Parameters:
//   - instId: 交易对ID / Instrument ID (e.g., "BTC-USDT-SWAP")
//
// Returns:
//   - *TickerResponse: 行情响应对象 / Ticker response object
//     包含Data字段，其中包含最新价格、买卖价等信息
//     Contains Data field with last price, bid/ask prices, etc.
//   - error: API请求失败或响应解析失败时返回错误 / Error on API request failure or response parsing failure
//     可能原因包括: 网络错误、认证失败、API错误码非"0"、交易对不存在
//     Possible causes: network error, authentication failure, API error code not "0", invalid instrument
func (c *Client) GetTicker(ctx context.Context, instId string) (*TickerResponse, error) {
	path := fmt.Sprintf("/api/v5/market/ticker?instId=%s", instId)

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	// Parse response
	var resp TickerResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}

// PlaceOrder places a regular order on the exchange
// Used for opening new positions or closing existing positions
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Order request with order parameters (instrument, side, type, size, price, etc.)
//
// Returns the order response with ordId (order ID) on success.
// Returns error on API request failure, authentication failure, or invalid parameters.
func (c *Client) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	path := "/api/v5/trade/order"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequestWithBody(ctx, "POST", path, string(reqBody))
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

// AmendOrder modifies an existing pending order
// Can change order size and/or price for pending orders
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Amend order request with orderId and new parameters
//
// Returns the amended order response.
// Returns error on API request failure or if order cannot be amended.
func (c *Client) AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error) {
	path := "/api/v5/trade/amend-order"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequestWithBody(ctx, "POST", path, string(reqBody))
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

// CancelOrder cancels an existing pending order
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Cancel order request with orderId
//
// Returns the cancellation response.
// Returns error on API request failure or if order cannot be canceled.
func (c *Client) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	path := "/api/v5/trade/cancel-order"
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, err := c.doRequestWithBody(ctx, "POST", path, string(reqBody))
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

// GetPendingOrders retrieves all pending orders (not algo orders)
// Returns orders that are placed but not yet filled or canceled
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//
// Returns error on API request failure or response parsing failure.
func (c *Client) GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error) {
	path := "/api/v5/trade/orders-pending"
	respBody, err := c.doRequest(ctx, "GET", path)
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

// GetOrdersHistory 获取订单历史（最近7天）/ Get order history (last 7 days)
// Retrieves completed orders from the last 7 days, regardless of how they were placed
// (OKX app, different APIs, etc). This ensures all account orders are visible.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - instType: Instrument type (SPOT, MARGIN, SWAP, FUTURES, OPTION)
//   - limit: Number of results per request (max 100, default 100)
//
// Returns: Order history response with list of completed orders, or error
func (c *Client) GetOrdersHistory(ctx context.Context, instType string, limit int) (*OrderHistoryResponse, error) {
	// Validate and set default limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	// Build request path with parameters
	path := fmt.Sprintf("/api/v5/trade/orders-history?instType=%s&limit=%d", instType, limit)

	respBody, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return nil, err
	}

	var resp OrderHistoryResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("API error: code=%s, msg=%s", resp.Code, resp.Msg)
	}

	return &resp, nil
}
