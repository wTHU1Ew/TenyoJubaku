package ordercontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/notifier"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Service Order Control 服务 / Order Control service
// 提供订单频率限制、Maker-only 检查等功能
// Provides order frequency limiting, maker-only checks, etc.
type Service struct {
	config    *config.OrderControlConfig
	okxClient okx.Interface
	storage   storage.Interface
	notifier  notifier.Interface
	logger    *logger.Logger
}

// New 创建 Order Control 服务 / Create Order Control service
// 初始化订单控制服务实例
// Initialize order control service instance
//
// Parameters:
//   - config: Order control configuration
//   - okxClient: OKX API client interface
//   - storage: Storage interface for order history
//   - notifier: Notifier interface for alerts
//   - logger: Logger instance
//
// Returns:
//   - *Service: Order Control 服务实例 / Order Control service instance
func New(
	config *config.OrderControlConfig,
	okxClient okx.Interface,
	storage storage.Interface,
	notifier notifier.Interface,
	logger *logger.Logger,
) *Service {
	return &Service{
		config:    config,
		okxClient: okxClient,
		storage:   storage,
		notifier:  notifier,
		logger:    logger,
	}
}

// PlaceOrder 下单（带频率和 Maker-only 检查）/ Place order with frequency and maker-only checks
// 执行订单前置检查，通过后调用 OKX API 下单，并记录订单历史
// Execute pre-order checks, call OKX API if passed, and record order history
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Order request
//
// Returns:
//   - *okx.OrderResponse: OKX 下单响应 / OKX order response
//   - error: 检查失败或下单失败时返回错误 / Error on check failure or order failure
func (s *Service) PlaceOrder(ctx context.Context, req *okx.OrderRequest) (*okx.OrderResponse, error) {
	// Check if order control is enabled
	if !s.config.Enabled {
		s.logger.Debug("Order control disabled, placing order directly")
		return s.okxClient.PlaceOrder(ctx, req)
	}

	s.logger.Info("Order control enabled, performing pre-order checks for %s %s %s",
		req.Side, req.Sz, req.InstId)

	// Phase 1: Frequency limit check
	if s.config.FrequencyLimit.Enabled {
		if err := s.checkFrequencyLimit(ctx, req); err != nil {
			s.logger.Warn("Frequency limit check failed: %v", err)
			// Log alert message
			s.logger.Info("Alert: Order blocked by frequency limit: %s %s %s - %v",
				req.Side, req.Sz, req.InstId, err)
			return nil, fmt.Errorf("frequency limit check failed: %w", err)
		}
	}

	// Phase 2: Maker-only check
	if s.config.MakerOnly.Enabled {
		if err := s.checkMakerOnly(ctx, req); err != nil {
			s.logger.Warn("Maker-only check failed: %v", err)
			// Log alert message
			s.logger.Info("Alert: Order blocked by maker-only check: %s %s %s - %v",
				req.Side, req.Sz, req.InstId, err)
			return nil, fmt.Errorf("maker-only check failed: %w", err)
		}
	}

	// All checks passed, place the order
	s.logger.Info("All pre-order checks passed, placing order")
	resp, err := s.okxClient.PlaceOrder(ctx, req)
	if err != nil {
		s.logger.Error("Failed to place order: %v", err)

		// Record failed order attempt
		s.recordOrderHistory(ctx, req, "", models.OrderStatusFailed)

		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	// Record successful order placement
	orderID := ""
	if len(resp.Data) > 0 {
		orderID = resp.Data[0].OrdId
	}
	s.recordOrderHistory(ctx, req, orderID, models.OrderStatusPlaced)

	s.logger.Info("Order placed successfully: %s", orderID)
	return resp, nil
}

// recordOrderHistory 记录订单历史 / Record order history
// 将订单信息写入数据库用于频率统计
// Write order information to database for frequency tracking
func (s *Service) recordOrderHistory(ctx context.Context, req *okx.OrderRequest, orderID string, status models.OrderStatus) {
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	order := &models.OrderHistory{
		OrderID:    orderID,
		InstId:     req.InstId,
		Side:       req.Side,
		OrdType:    req.OrdType,
		Size:       req.Sz,
		Price:      req.Px,
		ReduceOnly: req.ReduceOnly,
		PlacedAt:   now,
		WeekStart:  weekStart,
		Status:     string(status),
		CreatedAt:  now,
	}

	if err := s.storage.InsertOrderHistory(ctx, order); err != nil {
		s.logger.Error("Failed to record order history: %v", err)
		// Don't fail the order placement due to recording failure
	} else {
		s.logger.Debug("Order history recorded: %s", orderID)
	}
}
