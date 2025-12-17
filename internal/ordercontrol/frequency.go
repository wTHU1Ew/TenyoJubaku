package ordercontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// checkFrequencyLimit 检查订单频率限制 / Check order frequency limit
// 检查当前周的订单数量是否超过限制
// Check if current week's order count exceeds the limit
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - req: Order request to check
//
// Returns:
//   - error: 超过频率限制时返回错误 / Error if frequency limit exceeded
func (s *Service) checkFrequencyLimit(ctx context.Context, req *okx.OrderRequest) error {
	// Check if frequency limit is enabled
	if !s.config.FrequencyLimit.Enabled {
		s.logger.Debug("Frequency limit disabled, skipping check")
		return nil
	}

	// Get current week start
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	s.logger.Debug("Checking frequency limit for week starting %s", weekStart.Format("2006-01-02"))

	// Get current week's order count
	count, err := s.storage.GetOrderCountForWeek(ctx, weekStart)
	if err != nil {
		s.logger.Error("Failed to get order count for week: %v", err)
		// If we can't check, allow the order (fail open)
		s.logger.Warn("Allowing order due to frequency check failure (fail-open policy)")
		return nil
	}

	// Check if reduce-only orders should be counted
	isReduceOnly := req.ReduceOnly
	if isReduceOnly && s.config.FrequencyLimit.ExcludeReduceOnly {
		s.logger.Debug("Reduce-only order, skipping frequency limit check (exclude_reduce_only=true)")
		return nil
	}

	// Get weekly order statistics for detailed logging
	stats, err := s.storage.GetWeeklyOrderStats(ctx, weekStart)
	if err != nil {
		s.logger.Warn("Failed to get weekly stats: %v", err)
	} else if stats != nil {
		s.logger.Info("Weekly stats: %s", stats.String())
	}

	// Calculate the count to compare against limit
	// If ExcludeReduceOnly is true, we need to use RegularOrders count
	countToCheck := count
	if s.config.FrequencyLimit.ExcludeReduceOnly && stats != nil {
		countToCheck = stats.RegularOrders
	}

	maxOrders := s.config.FrequencyLimit.WeeklyMaxOrders
	s.logger.Info("Frequency limit check: %d/%d orders this week (exclude_reduce_only: %v)",
		countToCheck, maxOrders, s.config.FrequencyLimit.ExcludeReduceOnly)

	// Check if limit exceeded
	if countToCheck >= maxOrders {
		return fmt.Errorf("weekly order limit exceeded: %d/%d orders (week starting %s)",
			countToCheck, maxOrders, weekStart.Format("2006-01-02"))
	}

	remaining := maxOrders - countToCheck
	s.logger.Info("Frequency limit check passed: %d orders remaining this week", remaining)

	return nil
}
