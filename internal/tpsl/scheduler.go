package tpsl

import (
	"context"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Scheduler TPSL调度器 / TPSL scheduler
// 负责定期检查持仓并触发TPSL管理
// Responsible for periodic position checking and triggering TPSL management
type Scheduler struct {
	manager  *Manager
	storage  *storage.Storage
	config   *config.TPSLConfig
	logger   *logger.Logger
	ticker   *time.Ticker
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewScheduler 创建TPSL调度器 / Create TPSL scheduler
// 初始化TPSL调度器实例
// Initialize TPSL scheduler instance
//
// Parameters:
//   - config: TPSL configuration
//   - storage: Storage instance
//   - okxClient: OKX API client
//   - logger: Logger instance
//
// Returns:
//   - *Scheduler: TPSL调度器实例 / TPSL scheduler instance
func NewScheduler(config *config.TPSLConfig, storage *storage.Storage, okxClient *okx.Client, logger *logger.Logger) *Scheduler {
	manager := New(config, okxClient, logger)
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		manager: manager,
		storage: storage,
		config:  config,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
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

	// Fetch current positions from database
	positionsSlice, err := s.storage.GetLatestPositions()
	if err != nil {
		s.logger.Error("Failed to fetch positions from storage: %v", err)
		return
	}

	// Convert to pointers for manager
	positions := make([]*models.Position, len(positionsSlice))
	for i := range positionsSlice {
		positions[i] = &positionsSlice[i]
	}

	// Run TPSL analysis and placement
	summary, err := s.manager.AnalyzeAndPlaceTPSL(positions)
	if err != nil {
		s.logger.Error("TPSL analysis failed: %v", err)
		return
	}

	s.logger.Info("TPSL check cycle completed: %d positions checked, %d orders placed, %d failures",
		summary.TotalChecked, summary.OrdersPlaced, summary.PlacementFailures)
}
