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

// PhaseType 动态止损阶段类型 / Dynamic SL phase type (V5.1)
type PhaseType string

const (
	// PhaseA 渐进保本阶段 / Gradual move to breakeven phase
	PhaseA PhaseType = "PhaseA"
	// PhaseB 追踪止损阶段 / Trailing stop-loss phase
	PhaseB PhaseType = "PhaseB"
)

// LoadOrCreateTracker 加载或创建动态止损追踪器 / Load or create dynamic SL tracker (V5.1)
// 从数据库查询tracker，如果不存在则创建新的tracker并插入数据库
// Query tracker from database, if not exists create new tracker and insert into database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - storage: Storage interface for database operations
//   - position: Position data model with entry price and position details
//   - currentSlPrice: Current stop-loss price from existing algo order
//   - leverage: Position leverage for threshold calculation (V5.1)
//
// Returns:
//   - *models.DynamicSLTracker: 追踪器实例 / Tracker instance (loaded from DB or newly created)
//   - error: 数据库操作失败时返回错误 / Error on database operation failure
func LoadOrCreateTracker(ctx context.Context, storage storage.Interface, position *models.Position, currentSlPrice float64, leverage float64) (*models.DynamicSLTracker, error) {
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

	// Ensure leverage is at least 1
	if leverage < 1 {
		leverage = 1
	}

	// Tracker doesn't exist, create new one (V5.1)
	tracker = &models.DynamicSLTracker{
		PositionKey:      positionKey,
		InstId:           position.Instrument,
		PosSide:          string(position.PositionSide),
		EntryPrice:       position.AveragePrice,
		CurrentSlPrice:   currentSlPrice,
		InitialSlPrice:   currentSlPrice, // V5.1: Store initial SL for distance calculation
		Leverage:         leverage,       // V5.1: Store leverage for threshold calculation
		BreakevenReached: false,          // V5.1: Start in Phase A
		MoveCount:        0,              // V5.1: No moves yet
		LastUpdatedAt:    time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
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

// UpdateTracker 更新追踪器状态 / Update tracker state (V5.1)
// 根据当前市场价格更新追踪器的最高/最低价格，并更新数据库
// Update tracker's highest/lowest price based on current market price and update database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - storage: Storage interface for database operations
//   - tracker: Dynamic SL tracker instance to update
//   - currentPrice: Current market price (from ticker API)
//
// Returns:
//   - bool: true if tracker state changed and was updated in database
//   - error: 数据库更新失败时返回错误 / Error on database update failure
func UpdateTracker(ctx context.Context, storage storage.Interface, tracker *models.DynamicSLTracker, currentPrice float64) (bool, error) {
	updated := false

	// Determine position type
	isLong := tracker.PosSide == string(models.PositionSideLong) || tracker.PosSide == string(models.PositionSideNet)

	if isLong {
		// Long position: track highest price
		if currentPrice > tracker.HighestPriceReached {
			tracker.HighestPriceReached = currentPrice
			updated = true
		}
	} else {
		// Short position: track lowest price
		if tracker.LowestPriceReached == 0 || currentPrice < tracker.LowestPriceReached {
			tracker.LowestPriceReached = currentPrice
			updated = true
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

// CalculateDynamicSL 计算动态止损价格 / Calculate dynamic stop-loss price (V5.1 - Two-Phase Algorithm)
// 实现V5.1两阶段动态止损算法：
// Phase A: 渐进保本 - 当账户盈利达到 profitStepPct × leverage，止损移动 slMoveStepPct × leverage
// Phase B: 追踪止损 - 只有在到达保本位后，每0.5%价格上涨，止损上移0.1%
//
// Implements V5.1 two-phase dynamic SL algorithm:
// Phase A: Gradual breakeven - When account profit >= profitStepPct × leverage, SL moves by slMoveStepPct × leverage
// Phase B: Trailing - Only after breakeven, every 0.5% price gain → SL moves 0.1%
//
// Parameters:
//   - entryPrice: Position entry price (average price)
//   - currentPrice: Current market price
//   - extremePrice: Highest price reached (for long) or lowest price reached (for short)
//   - currentSlPrice: Current stop-loss price
//   - initialSlPrice: Initial stop-loss price (V5.1)
//   - leverage: Position leverage (V5.1)
//   - breakevenReached: Whether breakeven has been reached (V5.1)
//   - isLong: true for long/net positions, false for short positions
//   - cfg: Dynamic SL configuration with threshold parameters
//
// Returns:
//   - bool: true if SL should be adjusted
//   - float64: 新的止损价格 / New stop-loss price (only valid if bool is true)
//   - PhaseType: 当前阶段 / Current phase (PhaseA or PhaseB)
//   - error: 计算失败时返回错误 / Error on calculation failure
func CalculateDynamicSL(entryPrice, currentPrice, extremePrice, currentSlPrice, initialSlPrice, leverage float64, breakevenReached, isLong bool, cfg *config.DynamicSLConfig) (bool, float64, PhaseType, error) {
	// Validate inputs
	if entryPrice <= 0 {
		return false, 0, "", fmt.Errorf("entry price must be positive, got %.8f", entryPrice)
	}
	if currentPrice <= 0 {
		return false, 0, "", fmt.Errorf("current price must be positive, got %.8f", currentPrice)
	}
	if currentSlPrice <= 0 {
		return false, 0, "", fmt.Errorf("current SL price must be positive, got %.8f", currentSlPrice)
	}
	if initialSlPrice <= 0 {
		return false, 0, "", fmt.Errorf("initial SL price must be positive, got %.8f", initialSlPrice)
	}
	if leverage < 1 {
		return false, 0, "", fmt.Errorf("leverage must be at least 1, got %.2f", leverage)
	}
	if cfg == nil {
		return false, 0, "", fmt.Errorf("config cannot be nil")
	}

	// Calculate breakeven threshold prices
	breakevenLong := entryPrice * 1.001  // Entry + 0.1% for fees
	breakevenShort := entryPrice * 0.999 // Entry - 0.1% for fees

	// Check if we've reached breakeven
	if !breakevenReached {
		if isLong {
			breakevenReached = currentSlPrice >= breakevenLong
		} else {
			breakevenReached = currentSlPrice <= breakevenShort
		}
	}

	// ===== Phase A: Gradual Move to Breakeven (Leverage-Aware) =====
	if !breakevenReached {
		// Calculate price profit percentage
		var priceProfitPct float64
		if isLong {
			// Long: profit = (current - entry) / entry
			priceProfitPct = (currentPrice - entryPrice) / entryPrice
		} else {
			// Short: profit = (entry - current) / entry
			priceProfitPct = (entryPrice - currentPrice) / entryPrice
		}

		// Convert to account profit: accountProfit = priceProfitPct × leverage
		accountProfitPct := priceProfitPct * leverage

		// Check if account profit reaches threshold: profitStepPct × leverage
		threshold := cfg.ProfitStepPct * leverage
		if accountProfitPct >= threshold {
			// Calculate new SL: move by slMoveStepPct × leverage
			moveAmount := cfg.SlMoveStepPct * leverage
			var newSlPrice float64
			if isLong {
				// Long: move SL up
				newSlPrice = currentSlPrice * (1 + moveAmount)
				// Cap at breakeven + small margin
				if newSlPrice > breakevenLong {
					newSlPrice = breakevenLong
				}
			} else {
				// Short: move SL down
				newSlPrice = currentSlPrice * (1 - moveAmount)
				// Cap at breakeven - small margin
				if newSlPrice < breakevenShort {
					newSlPrice = breakevenShort
				}
			}
			return true, newSlPrice, PhaseA, nil
		}

		// Not profitable enough yet for Phase A move
		return false, 0, PhaseA, nil
	}

	// ===== Phase B: Trailing Mode (Only After Breakeven) =====
	if isLong {
		// Long: check if (current - highest) / highest >= trailingStepPct
		if currentPrice > extremePrice && extremePrice > 0 {
			gainFromHighest := (currentPrice - extremePrice) / extremePrice
			if gainFromHighest >= cfg.TrailingStepPct {
				// Move SL up by stopMoveStepPct
				newSlPrice := currentSlPrice * (1 + cfg.StopMoveStepPct)
				return true, newSlPrice, PhaseB, nil
			}
		}
	} else {
		// Short: check if (lowest - current) / lowest >= trailingStepPct
		if currentPrice < extremePrice && extremePrice > 0 {
			gainFromLowest := (extremePrice - currentPrice) / extremePrice
			if gainFromLowest >= cfg.TrailingStepPct {
				// Move SL down by stopMoveStepPct
				newSlPrice := currentSlPrice * (1 - cfg.StopMoveStepPct)
				return true, newSlPrice, PhaseB, nil
			}
		}
	}

	// No adjustment needed
	return false, 0, PhaseB, nil
}

// ShouldAdjustSL 检查是否需要调整止损 / Check if SL adjustment is needed (V5.1)
// 根据追踪器状态和配置，判断是否需要调整止损价格，并计算新的止损价格
// Based on tracker state and configuration, determine if SL adjustment is needed and calculate new SL price
//
// This is a convenience wrapper around CalculateDynamicSL that uses tracker state.
//
// Parameters:
//   - tracker: Dynamic SL tracker with current state
//   - currentPrice: Current market price (from ticker API)
//   - cfg: Dynamic SL configuration with threshold parameters
//
// Returns:
//   - bool: true if SL adjustment is needed
//   - float64: 新的止损价格 / New stop-loss price (only valid if bool is true)
//   - PhaseType: 当前阶段 / Current phase (PhaseA or PhaseB) (V5.1)
//   - error: 计算失败时返回错误 / Error on calculation failure
func ShouldAdjustSL(tracker *models.DynamicSLTracker, currentPrice float64, cfg *config.DynamicSLConfig) (bool, float64, PhaseType, error) {
	// Validate tracker
	if err := tracker.Validate(); err != nil {
		return false, 0, "", fmt.Errorf("invalid tracker: %w", err)
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

	// Use CalculateDynamicSL for the actual calculation (V5.1)
	return CalculateDynamicSL(
		tracker.EntryPrice,
		currentPrice,
		extremePrice,
		tracker.CurrentSlPrice,
		tracker.InitialSlPrice,
		tracker.Leverage,
		tracker.BreakevenReached,
		isLong,
		cfg,
	)
}

// IsBreakevenReached 检查是否已到达保本位 / Check if breakeven has been reached (V5.1)
// 用于判断是否已从 Phase A 进入 Phase B
// Used to determine if we've transitioned from Phase A to Phase B
//
// Parameters:
//   - tracker: Dynamic SL tracker with current state
//
// Returns:
//   - bool: true if breakeven has been reached (SL >= entry for long, SL <= entry for short)
func IsBreakevenReached(tracker *models.DynamicSLTracker) bool {
	isLong := tracker.PosSide == string(models.PositionSideLong) || tracker.PosSide == string(models.PositionSideNet)

	breakevenLong := tracker.EntryPrice * 1.001
	breakevenShort := tracker.EntryPrice * 0.999

	if isLong {
		return tracker.CurrentSlPrice >= breakevenLong
	}
	return tracker.CurrentSlPrice <= breakevenShort
}

// GetDistanceToBreakeven 获取到保本位的距离百分比 / Get distance to breakeven percentage (V5.1)
// 用于日志和调试，显示还需要移动多少才能到达保本位
// Used for logging and debugging, shows how much more SL needs to move to reach breakeven
//
// Parameters:
//   - tracker: Dynamic SL tracker with current state
//
// Returns:
//   - float64: 到保本位的距离百分比 / Distance to breakeven as percentage
//     正值表示还需要移动，负值表示已超过保本位
//     Positive means more movement needed, negative means already past breakeven
func GetDistanceToBreakeven(tracker *models.DynamicSLTracker) float64 {
	isLong := tracker.PosSide == string(models.PositionSideLong) || tracker.PosSide == string(models.PositionSideNet)

	breakevenLong := tracker.EntryPrice * 1.001
	breakevenShort := tracker.EntryPrice * 0.999

	if isLong {
		// For long: distance = (breakeven - currentSL) / currentSL
		return (breakevenLong - tracker.CurrentSlPrice) / tracker.CurrentSlPrice
	}
	// For short: distance = (currentSL - breakeven) / currentSL
	return (tracker.CurrentSlPrice - breakevenShort) / tracker.CurrentSlPrice
}
