package tpsl

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Scheduler TPSL调度器 / TPSL scheduler
// 负责定期检查持仓并触发TPSL管理
// Responsible for periodic position checking and triggering TPSL management
type Scheduler struct {
	manager   *Manager
	okxClient *okx.Client
	config    *config.TPSLConfig
	logger    *logger.Logger
	ticker    *time.Ticker
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewScheduler 创建TPSL调度器 / Create TPSL scheduler
// 初始化TPSL调度器实例
// Initialize TPSL scheduler instance
//
// Parameters:
//   - config: TPSL configuration
//   - okxClient: OKX API client
//   - logger: Logger instance
//
// Returns:
//   - *Scheduler: TPSL调度器实例 / TPSL scheduler instance
func NewScheduler(config *config.TPSLConfig, okxClient *okx.Client, logger *logger.Logger) *Scheduler {
	manager := New(config, okxClient, logger)
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		manager:   manager,
		okxClient: okxClient,
		config:    config,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
	}
}

// Start 启动TPSL调度器 / Start TPSL scheduler
// 开始定期执行TPSL检查
// Start periodic TPSL checks
func (s *Scheduler) Start() {
	interval := time.Duration(s.config.CheckInterval) * time.Second
	s.ticker = time.NewTicker(interval)

	s.logger.Info("TPSL scheduler started with interval %d seconds", s.config.CheckInterval)

	go s.run()
}

// Stop 停止TPSL调度器 / Stop TPSL scheduler
// 优雅地停止调度器
// Gracefully stop the scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping TPSL scheduler...")
	s.cancel()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	<-s.done // Wait for run goroutine to finish
	s.logger.Info("TPSL scheduler stopped")
}

// run 运行调度循环 / Run scheduler loop
// 执行定期TPSL检查的主循环
// Main loop for periodic TPSL checks
func (s *Scheduler) run() {
	defer close(s.done)

	// Run initial check immediately
	s.runCheck()

	// Then run periodic checks
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.runCheck()
		}
	}
}

// runCheck 执行一次TPSL检查 / Run one TPSL check
// 执行一次完整的TPSL检查周期
// Execute one complete TPSL check cycle
func (s *Scheduler) runCheck() {
	// Use defer/recover to prevent panics from crashing the scheduler
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic in TPSL check: %v", r)
		}
	}()

	s.logger.Debug("Starting TPSL check cycle")

	// Fetch current positions directly from OKX API (real-time data)
	s.logger.Info("Fetching positions from OKX API...")
	positionsResp, err := s.okxClient.GetPositions()
	if err != nil {
		s.logger.Error("Failed to fetch positions from OKX API: %v", err)
		return
	}

	// Check if there are any open positions
	if len(positionsResp.Data) == 0 {
		s.logger.Info("No open positions, skipping TPSL check")
		return
	}

	// Convert OKX response to Position models
	positions := make([]*models.Position, 0, len(positionsResp.Data))
	for _, pos := range positionsResp.Data {
		position, err := s.convertOKXPosition(&pos)
		if err != nil {
			s.logger.Warn("Failed to convert position %s: %v", pos.InstId, err)
			continue
		}
		positions = append(positions, position)
	}

	if len(positions) == 0 {
		s.logger.Info("No valid positions after conversion, skipping TPSL check")
		return
	}

	s.logger.Info("Found %d open position(s)", len(positions))

	// Run TPSL analysis and placement
	summary, err := s.manager.AnalyzeAndPlaceTPSL(positions)
	if err != nil {
		s.logger.Error("TPSL analysis failed: %v", err)
		return
	}

	s.logger.Info("TPSL check cycle completed: %d positions checked, %d orders placed, %d failures",
		summary.TotalChecked, summary.OrdersPlaced, summary.PlacementFailures)
}

// convertOKXPosition 将 OKX API 响应转换为 Position 模型
// Convert OKX API response to Position model
func (s *Scheduler) convertOKXPosition(pos *okx.PositionData) (*models.Position, error) {
	// Parse position size
	posSize, err := strconv.ParseFloat(pos.Pos, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse position size: %w", err)
	}
	if posSize == 0 {
		return nil, fmt.Errorf("position size is zero")
	}

	// Parse average price
	avgPrice, err := strconv.ParseFloat(pos.AvgPx, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse average price: %w", err)
	}

	// Parse unrealized PnL
	upl, err := strconv.ParseFloat(pos.Upl, 64)
	if err != nil {
		s.logger.Warn("Failed to parse unrealized PnL for %s: %v", pos.InstId, err)
		upl = 0
	}

	// Parse margin
	margin, err := strconv.ParseFloat(pos.Margin, 64)
	if err != nil {
		s.logger.Warn("Failed to parse margin for %s: %v", pos.InstId, err)
		margin = 0
	}

	// Parse leverage
	leverage, err := strconv.ParseFloat(pos.Lever, 64)
	if err != nil {
		s.logger.Warn("Failed to parse leverage for %s: %v", pos.InstId, err)
		leverage = 0
	}

	// Get margin mode (cross or isolated)
	marginMode := models.MarginMode(pos.MgnMode)
	if marginMode == "" {
		marginMode = models.MarginModeCross
	}

	// Normalize position side
	posSide := models.PositionSide(pos.PosSide)
	if posSide == "" {
		posSide = models.PositionSideNet
	}

	// Create position model
	position := &models.Position{
		Timestamp:     time.Now().UTC(),
		Instrument:    pos.InstId,
		PositionSide:  posSide,
		PositionSize:  posSize,
		AveragePrice:  avgPrice,
		UnrealizedPnL: upl,
		Margin:        margin,
		Leverage:      leverage,
		MarginMode:    marginMode,
	}

	return position, nil
}
