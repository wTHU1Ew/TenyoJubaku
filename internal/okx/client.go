package okx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client OKX API客户端 / OKX API client
type Client struct {
	apiURL     string
	apiKey     string
	apiSecret  string
	passphrase string
	httpClient *http.Client
	maxRetries int
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
//
// Returns:
//   - *Client: 配置完成的OKX客户端实例 / Configured OKX client instance ready for API calls
func New(apiURL, apiKey, apiSecret, passphrase string, timeout, maxRetries int) *Client {
	return &Client{
		apiURL:     apiURL,
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		passphrase: passphrase,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		maxRetries: maxRetries,
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
func (c *Client) doRequest(method, path string) ([]byte, error) {
	url := c.apiURL + path
	body := "" // GET requests have empty body

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		// Generate timestamp (ISO8601 format)
		timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

		// Generate signature
		signature := c.generateSignature(timestamp, method, path, body)

		// Create HTTP request
		req, err := http.NewRequest(method, url, nil)
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
func (c *Client) GetAccountBalance() (*AccountBalanceResponse, error) {
	path := "/api/v5/account/balance"

	respBody, err := c.doRequest("GET", path)
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
func (c *Client) GetPositions() (*PositionsResponse, error) {
	path := "/api/v5/account/positions"

	respBody, err := c.doRequest("GET", path)
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

// HealthCheck 健康检查 / Health check by testing API connectivity
// 通过尝试获取账户余额来验证API连接和认证是否正常
// Verify API connectivity and authentication by attempting to fetch account balance
//
// Returns:
//   - error: 连接失败或认证失败时返回错误 / Error on connection failure or authentication failure
//     nil表示API连接正常，认证通过 / nil indicates API is reachable and authenticated
func (c *Client) HealthCheck() error {
	_, err := c.GetAccountBalance()
	return err
}
