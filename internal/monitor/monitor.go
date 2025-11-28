package monitor

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Monitor 监控服务 / Monitoring service
type Monitor struct {
	okxClient     *okx.Client
	storage       *storage.Storage
	logger        *logger.Logger
	interval      time.Duration
	stopChan      chan struct{}
	lastSuccess   time.Time
	errorCount    int64
	successCount  int64
}

// New 创建新的监控服务 / Create new monitoring service
// 初始化监控服务，配置OKX客户端、存储层、日志和轮询间隔
// Initialize monitoring service with OKX client, storage layer, logger, and polling interval
//
// Parameters:
//   - okxClient: OKX API client instance for fetching account data
//   - storage: Database storage layer instance for persisting data
//   - logger: Logger instance for logging operations
//   - intervalSeconds: Monitoring polling interval in seconds (e.g., 60 for 1 minute)
//
// Returns:
//   - *Monitor: 已配置的监控服务实例 / Configured monitoring service instance ready to start
func New(okxClient *okx.Client, storage *storage.Storage, logger *logger.Logger, intervalSeconds int) *Monitor {
	return &Monitor{
		okxClient: okxClient,
		storage:   storage,
		logger:    logger,
		interval:  time.Duration(intervalSeconds) * time.Second,
		stopChan:  make(chan struct{}),
	}
}

// Start 启动监控服务 / Start monitoring service
// 启动持续监控循环，按配置间隔获取并存储账户数据
// Start continuous monitoring loop, fetch and store account data at configured interval
//
// 监控流程 / Monitoring Flow:
// 1. 执行健康检查（验证OKX API和数据库连接）/ Perform health check (verify OKX API and DB connectivity)
// 2. 启动定时器，按interval间隔执行 / Start ticker, execute at interval
// 3. 每个周期: 获取余额 → 获取持仓 → 存储数据 / Each cycle: fetch balance → fetch positions → store data
// 4. 监听停止信号，优雅退出 / Listen for stop signal, graceful exit
//
// Returns:
//   - error: 初始健康检查失败时返回错误 / Error on initial health check failure
//     监控过程中的错误会被记录但不会停止服务 / Errors during monitoring are logged but don't stop service
func (m *Monitor) Start() error {
	m.logger.Info("Starting monitoring service with interval: %v", m.interval)

	// Perform initial health check
	if err := m.healthCheck(); err != nil {
		return fmt.Errorf("initial health check failed: %w", err)
	}

	// Start monitoring loop
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.logger.Debug("Monitoring cycle started")
			if err := m.fetchAndStore(); err != nil {
				m.errorCount++
				m.logger.Error("Monitoring cycle failed (error count: %d): %v", m.errorCount, err)
			} else {
				m.successCount++
				m.lastSuccess = time.Now()
				m.logger.Info("Monitoring cycle completed successfully (success count: %d)", m.successCount)
			}

		case <-m.stopChan:
			m.logger.Info("Monitoring service stopped")
			return nil
		}
	}
}

// Stop 停止监控服务 / Stop monitoring service
func (m *Monitor) Stop() {
	m.logger.Info("Stopping monitoring service...")
	close(m.stopChan)
}

// healthCheck 健康检查 / Perform health check
func (m *Monitor) healthCheck() error {
	m.logger.Info("Performing health check...")

	// Check OKX API connectivity
	if err := m.okxClient.HealthCheck(); err != nil {
		m.logger.Error("OKX API health check failed: %v", err)
		return fmt.Errorf("OKX API health check failed: %w", err)
	}
	m.logger.Info("OKX API connection successful")

	// Check database connectivity
	if err := m.storage.HealthCheck(); err != nil {
		m.logger.Error("Database health check failed: %v", err)
		return fmt.Errorf("database health check failed: %w", err)
	}
	m.logger.Info("Database connection successful")

	m.logger.Info("Health check passed")
	return nil
}

// fetchAndStore 获取并存储数据 / Fetch and store account data
func (m *Monitor) fetchAndStore() error {
	// Fetch account balances
	if err := m.fetchAndStoreBalances(); err != nil {
		return fmt.Errorf("failed to fetch balances: %w", err)
	}

	// Fetch positions
	if err := m.fetchAndStorePositions(); err != nil {
		return fmt.Errorf("failed to fetch positions: %w", err)
	}

	return nil
}

// fetchAndStoreBalances 获取并存储账户余额 / Fetch and store account balances
// 从OKX API获取账户余额，解析并存储到数据库
// Fetch account balances from OKX API, parse and store to database
//
// 处理流程 / Processing Flow:
// 1. 调用OKX API获取余额数据 / Call OKX API to get balance data
// 2. 遍历所有币种的余额详情 / Iterate through all currency balance details
// 3. 过滤：仅记录BTC、ETH、USDT / Filter: only record BTC, ETH, USDT
// 4. 解析字符串金额为浮点数 / Parse string amounts to float64
// 5. 创建AccountBalance模型并验证 / Create AccountBalance model and validate
// 6. 写入数据库 / Write to database
//
// Returns:
//   - error: API调用失败、数据解析失败或数据库写入失败时返回错误
//     Error on API call failure, data parsing failure, or database write failure
func (m *Monitor) fetchAndStoreBalances() error {
	m.logger.Debug("Fetching account balances from OKX API...")

	// Fetch balances from OKX API
	resp, err := m.okxClient.GetAccountBalance()
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	m.logger.Debug("Received account balance response from OKX API")

	// Parse and store balances
	timestamp := time.Now().UTC()
	storedCount := 0

	if len(resp.Data) == 0 {
		m.logger.Warn("No account data in balance response")
		return nil
	}

	for _, account := range resp.Data {
		for _, detail := range account.Details {
			// Filter: only record BTC, ETH, and USDT
			if detail.Ccy != "BTC" && detail.Ccy != "ETH" && detail.Ccy != "USDT" {
				continue
			}

			// Parse balance values
			balance, err := strconv.ParseFloat(detail.Eq, 64)
			if err != nil {
				m.logger.Warn("Failed to parse balance for %s: %v", detail.Ccy, err)
				continue
			}

			available, err := strconv.ParseFloat(detail.AvailBal, 64)
			if err != nil {
				m.logger.Warn("Failed to parse available balance for %s: %v", detail.Ccy, err)
				continue
			}

			frozen, err := strconv.ParseFloat(detail.FrozenBal, 64)
			if err != nil {
				m.logger.Warn("Failed to parse frozen balance for %s: %v", detail.Ccy, err)
				frozen = 0 // Default to 0 if parse fails
			}

			equity, err := strconv.ParseFloat(detail.EqUsd, 64)
			if err != nil {
				equity = 0 // Default to 0 if parse fails
			}

			// Create balance model
			balanceModel := &models.AccountBalance{
				Timestamp: timestamp,
				Currency:  detail.Ccy,
				Balance:   balance,
				Available: available,
				Frozen:    frozen,
				Equity:    equity,
			}

			// Insert into database
			if err := m.storage.InsertAccountBalance(balanceModel); err != nil {
				m.logger.Error("Failed to insert balance for %s: %v", detail.Ccy, err)
				return err
			}

			storedCount++
			m.logger.Debug("Stored balance for %s: %.8f", detail.Ccy, balance)
		}
	}

	m.logger.Info("Stored %d account balance records", storedCount)
	return nil
}

// fetchAndStorePositions 获取并存储持仓信息 / Fetch and store positions
// 从OKX API获取持仓信息，解析并存储到数据库
// Fetch position information from OKX API, parse and store to database
//
// 处理流程 / Processing Flow:
// 1. 调用OKX API获取持仓数据 / Call OKX API to get position data
// 2. 如果无持仓则记录日志并返回 / If no positions, log and return
// 3. 遍历所有持仓 / Iterate through all positions
// 4. 解析持仓数值（仓位、价格、盈亏等）/ Parse position values (size, price, PnL, etc.)
// 5. 跳过零仓位的记录 / Skip records with zero position size
// 6. 创建Position模型并验证 / Create Position model and validate
// 7. 写入数据库 / Write to database
//
// Returns:
//   - error: API调用失败、数据解析失败或数据库写入失败时返回错误
//     Error on API call failure, data parsing failure, or database write failure
func (m *Monitor) fetchAndStorePositions() error {
	m.logger.Debug("Fetching positions from OKX API...")

	// Fetch positions from OKX API
	resp, err := m.okxClient.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	m.logger.Debug("Received positions response from OKX API")

	// Check if there are any positions
	if len(resp.Data) == 0 {
		m.logger.Info("No open positions")
		return nil
	}

	// Parse and store positions
	timestamp := time.Now().UTC()
	storedCount := 0

	for _, pos := range resp.Data {
		// Parse position values
		posSize, err := strconv.ParseFloat(pos.Pos, 64)
		if err != nil || posSize == 0 {
			continue // Skip positions with zero size
		}

		avgPrice, err := strconv.ParseFloat(pos.AvgPx, 64)
		if err != nil {
			m.logger.Warn("Failed to parse average price for %s: %v", pos.InstId, err)
			continue
		}

		upl, err := strconv.ParseFloat(pos.Upl, 64)
		if err != nil {
			m.logger.Warn("Failed to parse unrealized PnL for %s: %v", pos.InstId, err)
			upl = 0
		}

		margin, err := strconv.ParseFloat(pos.Margin, 64)
		if err != nil {
			m.logger.Warn("Failed to parse margin for %s: %v", pos.InstId, err)
			margin = 0
		}

		leverage, err := strconv.ParseFloat(pos.Lever, 64)
		if err != nil {
			m.logger.Warn("Failed to parse leverage for %s: %v", pos.InstId, err)
			leverage = 0
		}

		// Normalize position side
		posSide := pos.PosSide
		if posSide == "" {
			posSide = "net" // Default for one-way mode
		}

		// Create position model
		positionModel := &models.Position{
			Timestamp:     timestamp,
			Instrument:    pos.InstId,
			PositionSide:  posSide,
			PositionSize:  posSize,
			AveragePrice:  avgPrice,
			UnrealizedPnL: upl,
			Margin:        margin,
			Leverage:      leverage,
		}

		// Insert into database
		if err := m.storage.InsertPosition(positionModel); err != nil {
			m.logger.Error("Failed to insert position for %s: %v", pos.InstId, err)
			return err
		}

		storedCount++
		m.logger.Debug("Stored position for %s: side=%s, size=%.8f", pos.InstId, posSide, posSize)
	}

	m.logger.Info("Stored %d position records", storedCount)
	return nil
}

// GetMetrics 获取监控指标 / Get monitoring metrics
func (m *Monitor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"last_success":  m.lastSuccess,
		"error_count":   m.errorCount,
		"success_count": m.successCount,
	}
}
