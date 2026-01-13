package tpsl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
	"gorm.io/gorm"
)

// LoadOrCreateTracker 加载或创建动态止损追踪器 / Load or create dynamic SL tracker
// 从数据库查询tracker，如果不存在则创建新的tracker并插入数据库
// Query tracker from database, if not exists create new tracker and insert into database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - storage: Storage interface for database operations
//   - position: Position data model with entry price and position details
//   - currentSlPrice: Current stop-loss price from existing algo order
//
// Returns:
//   - *models.DynamicSLTracker: 追踪器实例 / Tracker instance (loaded from DB or newly created)
//   - error: 数据库操作失败时返回错误 / Error on database operation failure
func LoadOrCreateTracker(ctx context.Context, storage storage.Interface, position *models.Position, currentSlPrice float64) (*models.DynamicSLTracker, error) {
	// Generate position key: "{instId}_{posSide}"
	positionKey := fmt.Sprintf("%s_%s", position.Instrument, position.PositionSide)

	// Try to load existing tracker from database
	tracker, err := storage.GetDynamicSLTracker(ctx, positionKey)
	if err == nil {
		// Tracker exists, return it
		return tracker, nil
	}

	// Check if error is "not found" (expected) or actual database error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to query tracker: %w", err)
	}

	// Tracker doesn't exist, create new one
	tracker = &models.DynamicSLTracker{
		PositionKey:        positionKey,
		InstId:             position.Instrument,
		PosSide:            string(position.PositionSide),
		EntryPrice:         position.AveragePrice,
		CurrentSlPrice:     currentSlPrice,
		FirstMoveTriggered: false,
		LastUpdatedAt:      time.Now().UTC(),
		CreatedAt:          time.Now().UTC(),
	}

	// Initialize highest/lowest price based on position side
	if position.PositionSide == models.PositionSideLong || position.PositionSide == models.PositionSideNet {
		// For long positions, track highest price
		tracker.HighestPriceReached = position.AveragePrice
		tracker.LowestPriceReached = 0
	} else {
		// For short positions, track lowest price
		tracker.HighestPriceReached = 0
		tracker.LowestPriceReached = position.AveragePrice
	}

	// Insert into database
	if err := storage.InsertDynamicSLTracker(ctx, tracker); err != nil {
		return nil, fmt.Errorf("failed to insert tracker: %w", err)
	}

	return tracker, nil
}

// UpdateTracker 更新追踪器状态 / Update tracker state
// 根据当前市场价格更新追踪器的最高/最低价格，检查firstMove阈值，并更新数据库
// Update tracker's highest/lowest price based on current market price, check firstMove threshold, and update database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - storage: Storage interface for database operations
//   - tracker: Dynamic SL tracker instance to update
//   - currentPrice: Current market price (from ticker API)
//   - config: Dynamic SL configuration with threshold parameters
//
// Returns:
//   - bool: true if tracker state changed and was updated in database
//   - error: 数据库更新失败时返回错误 / Error on database update failure
func UpdateTracker(ctx context.Context, storage storage.Interface, tracker *models.DynamicSLTracker, currentPrice float64, config *config.DynamicSLConfig) (bool, error) {
	updated := false

	// Determine position type
	isLong := tracker.PosSide == string(models.PositionSideLong) || tracker.PosSide == string(models.PositionSideNet)

	if isLong {
		// Long position: track highest price
		if currentPrice > tracker.HighestPriceReached {
			tracker.HighestPriceReached = currentPrice
			updated = true
		}

		// Check firstMove threshold for long positions
		// Formula: (currentPrice - entryPrice) / entryPrice >= firstMovePct
		if !tracker.FirstMoveTriggered {
			profitPct := (currentPrice - tracker.EntryPrice) / tracker.EntryPrice
			if profitPct >= config.FirstMovePct {
				tracker.FirstMoveTriggered = true
				updated = true
			}
		}
	} else {
		// Short position: track lowest price
		if tracker.LowestPriceReached == 0 || currentPrice < tracker.LowestPriceReached {
			tracker.LowestPriceReached = currentPrice
			updated = true
		}

		// Check firstMove threshold for short positions
		// Formula: (entryPrice - currentPrice) / entryPrice >= firstMovePct
		if !tracker.FirstMoveTriggered {
			profitPct := (tracker.EntryPrice - currentPrice) / tracker.EntryPrice
			if profitPct >= config.FirstMovePct {
				tracker.FirstMoveTriggered = true
				updated = true
			}
		}
	}

	// Update timestamp if state changed
	if updated {
		tracker.LastUpdatedAt = time.Now().UTC()

		// Persist changes to database
		if err := storage.UpdateDynamicSLTracker(ctx, tracker); err != nil {
			return false, fmt.Errorf("failed to update tracker: %w", err)
		}
	}

	return updated, nil
}

// CalculateDynamicSL 计算动态止损价格 / Calculate dynamic stop-loss price
// 根据入场价格、当前价格、最高/最低价格和配置参数，计算新的止损价格
// Calculate new stop-loss price based on entry price, current price, highest/lowest price, and config parameters
//
// This implements the core dynamic trailing stop-loss algorithm:
// Step 1: If profit >= firstMovePct, move SL to breakeven (entry * 1.001)
// Step 2: If profit continues, trail SL every trailingStepPct gain by stopMoveStepPct
//
// Parameters:
//   - entryPrice: Position entry price (average price)
//   - currentPrice: Current market price
//   - highestPrice: Highest price reached (for long) or lowest price reached (for short)
//   - currentSlPrice: Current stop-loss price
//   - firstMoveTriggered: Whether firstMove threshold has been triggered
//   - isLong: true for long/net positions, false for short positions
//   - config: Dynamic SL configuration with threshold parameters
//
// Returns:
//   - bool: true if SL should be adjusted
//   - float64: 新的止损价格 / New stop-loss price (only valid if bool is true)
//   - error: 计算失败时返回错误 / Error on calculation failure
func CalculateDynamicSL(entryPrice, currentPrice, highestPrice, currentSlPrice float64, firstMoveTriggered, isLong bool, config *config.DynamicSLConfig) (bool, float64, error) {
	// Validate inputs
	if entryPrice <= 0 {
		return false, 0, fmt.Errorf("entry price must be positive, got %.8f", entryPrice)
	}
	if currentPrice <= 0 {
		return false, 0, fmt.Errorf("current price must be positive, got %.8f", currentPrice)
	}
	if currentSlPrice <= 0 {
		return false, 0, fmt.Errorf("current SL price must be positive, got %.8f", currentSlPrice)
	}
	if config == nil {
		return false, 0, fmt.Errorf("config cannot be nil")
	}

	// Step 1: Check if firstMove threshold is reached
	if !firstMoveTriggered {
		var profitPct float64
		if isLong {
			// Long: profit = (current - entry) / entry
			profitPct = (currentPrice - entryPrice) / entryPrice
		} else {
			// Short: profit = (entry - current) / entry
			profitPct = (entryPrice - currentPrice) / entryPrice
		}

		// Check if profit reaches firstMove threshold
		if profitPct >= config.FirstMovePct {
			// Move SL to breakeven + 0.1% (to cover fees)
			var newSlPrice float64
			if isLong {
				newSlPrice = entryPrice * 1.001
			} else {
				newSlPrice = entryPrice * 0.999
			}
			return true, newSlPrice, nil
		}

		// Not profitable enough yet
		return false, 0, nil
	}

	// Step 2: Check if SL needs to move to breakeven (after firstMove triggered)
	if isLong {
		// For long: SL should be >= entry * 1.001
		if currentSlPrice < entryPrice*1.001 {
			newSlPrice := entryPrice * 1.001
			return true, newSlPrice, nil
		}
	} else {
		// For short: SL should be <= entry * 0.999
		if currentSlPrice > entryPrice*0.999 {
			newSlPrice := entryPrice * 0.999
			return true, newSlPrice, nil
		}
	}

	// Step 3: Check if price has gained enough to trail SL
	if isLong {
		// Long: check if (current - highest) / highest >= trailingStepPct
		if currentPrice > highestPrice {
			gainFromHighest := (currentPrice - highestPrice) / highestPrice
			if gainFromHighest >= config.TrailingStepPct {
				// Move SL up by stopMoveStepPct
				newSlPrice := currentSlPrice * (1 + config.StopMoveStepPct)
				return true, newSlPrice, nil
			}
		}
	} else {
		// Short: check if (lowest - current) / lowest >= trailingStepPct
		// Note: for short, highestPrice parameter actually contains lowestPrice
		if currentPrice < highestPrice {
			gainFromLowest := (highestPrice - currentPrice) / highestPrice
			if gainFromLowest >= config.TrailingStepPct {
				// Move SL down by stopMoveStepPct
				newSlPrice := currentSlPrice * (1 - config.StopMoveStepPct)
				return true, newSlPrice, nil
			}
		}
	}

	// No adjustment needed
	return false, 0, nil
}

// ShouldAdjustSL 检查是否需要调整止损 / Check if SL adjustment is needed
// 根据追踪器状态和配置，判断是否需要调整止损价格，并计算新的止损价格
// Based on tracker state and configuration, determine if SL adjustment is needed and calculate new SL price
//
// This is a convenience wrapper around CalculateDynamicSL that uses tracker state.
//
// Parameters:
//   - tracker: Dynamic SL tracker with current state
//   - currentPrice: Current market price (from ticker API)
//   - config: Dynamic SL configuration with threshold parameters
//
// Returns:
//   - bool: true if SL adjustment is needed
//   - float64: 新的止损价格 / New stop-loss price (only valid if bool is true)
//   - error: 计算失败时返回错误 / Error on calculation failure
func ShouldAdjustSL(tracker *models.DynamicSLTracker, currentPrice float64, config *config.DynamicSLConfig) (bool, float64, error) {
	// Validate tracker
	if err := tracker.Validate(); err != nil {
		return false, 0, fmt.Errorf("invalid tracker: %w", err)
	}

	// Determine position type
	isLong := tracker.PosSide == string(models.PositionSideLong) || tracker.PosSide == string(models.PositionSideNet)

	// Get the appropriate highest/lowest price for the position type
	var extremePrice float64
	if isLong {
		extremePrice = tracker.HighestPriceReached
	} else {
		extremePrice = tracker.LowestPriceReached
	}

	// Use CalculateDynamicSL for the actual calculation
	return CalculateDynamicSL(
		tracker.EntryPrice,
		currentPrice,
		extremePrice,
		tracker.CurrentSlPrice,
		tracker.FirstMoveTriggered,
		isLong,
		config,
	)
}
