package ordercontrol

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
)

// checkMakerOnly 检查 Maker-only 限制 / Check maker-only restriction
// 验证订单类型和价格距离，确保只下 Maker 订单
// Verify order type and price distance to ensure maker-only orders
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Order request to check
//
// Returns:
//   - error: 不符合 Maker-only 要求时返回错误 / Error if maker-only requirements not met
func (s *Service) checkMakerOnly(ctx context.Context, req *okx.OrderRequest) error {
	// Check if maker-only is enabled
	if !s.config.MakerOnly.Enabled {
		s.logger.Debug("Maker-only disabled, skipping check")
		return nil
	}

	s.logger.Debug("Checking maker-only requirements for %s %s %s",
		req.Side, req.Sz, req.InstId)

	// Check 1: Order type must be limit or post_only
	if req.OrdType != "limit" && req.OrdType != "post_only" {
		return fmt.Errorf("maker-only mode: order type must be 'limit' or 'post_only', got '%s'", req.OrdType)
	}

	// post_only is always maker, no need to check price distance
	if req.OrdType == "post_only" {
		s.logger.Debug("Order type is post_only, maker-only check passed")
		return nil
	}

	// Check 2: For limit orders, verify price distance
	if req.OrdType == "limit" {
		if err := s.checkPriceDistance(ctx, req); err != nil {
			return fmt.Errorf("maker-only price distance check failed: %w", err)
		}
	}

	s.logger.Info("Maker-only check passed")
	return nil
}

// checkPriceDistance 检查价格距离 / Check price distance
// 验证限价单的价格与市场价格的距离是否满足要求
// Verify limit order price distance from market price meets requirements
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Order request with limit price
//
// Returns:
//   - error: 价格距离不足时返回错误 / Error if price distance insufficient
func (s *Service) checkPriceDistance(ctx context.Context, req *okx.OrderRequest) error {
	// Get current market price
	ticker, err := s.okxClient.GetTicker(ctx, req.InstId)
	if err != nil {
		s.logger.Error("Failed to get ticker for %s: %v", req.InstId, err)
		// If we can't get market price, reject the order (fail closed for safety)
		return fmt.Errorf("cannot verify price distance: failed to get market price")
	}

	if len(ticker.Data) == 0 {
		return fmt.Errorf("cannot verify price distance: no ticker data available")
	}

	// Parse market price (use best bid/ask as reference)
	marketPrice := 0.0
	if req.Side == "buy" {
		// For buy orders, compare against best ask (selling price)
		marketPrice, err = strconv.ParseFloat(ticker.Data[0].AskPx, 64)
		if err != nil {
			return fmt.Errorf("failed to parse ask price: %w", err)
		}
	} else {
		// For sell orders, compare against best bid (buying price)
		marketPrice, err = strconv.ParseFloat(ticker.Data[0].BidPx, 64)
		if err != nil {
			return fmt.Errorf("failed to parse bid price: %w", err)
		}
	}

	// Parse order price
	orderPrice, err := strconv.ParseFloat(req.Px, 64)
	if err != nil {
		return fmt.Errorf("failed to parse order price: %w", err)
	}

	// Calculate price distance percentage
	// For buy orders: (marketPrice - orderPrice) / marketPrice * 100
	// For sell orders: (orderPrice - marketPrice) / marketPrice * 100
	var distancePercent float64
	if req.Side == "buy" {
		// Buy order must be below market price
		distancePercent = (marketPrice - orderPrice) / marketPrice * 100
		if distancePercent < 0 {
			return fmt.Errorf("buy order price %.2f is above or at market ask %.2f (would be taker)",
				orderPrice, marketPrice)
		}
	} else {
		// Sell order must be above market price
		distancePercent = (orderPrice - marketPrice) / marketPrice * 100
		if distancePercent < 0 {
			return fmt.Errorf("sell order price %.2f is below or at market bid %.2f (would be taker)",
				orderPrice, marketPrice)
		}
	}

	// Check if distance meets minimum requirement
	minDistance := s.config.MakerOnly.MinPriceDistancePct
	if distancePercent < minDistance {
		return fmt.Errorf("price distance %.4f%% is less than minimum %.4f%% (order: %.2f, market: %.2f)",
			distancePercent, minDistance, orderPrice, marketPrice)
	}

	s.logger.Info("Price distance check passed: %.4f%% (min: %.4f%%, order: %.2f, market: %.2f)",
		distancePercent, minDistance, orderPrice, marketPrice)

	return nil
}
