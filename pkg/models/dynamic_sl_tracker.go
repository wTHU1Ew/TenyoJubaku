package models

import (
	"fmt"
	"time"
)

// DynamicSLTracker 动态止损追踪器 / Dynamic stop-loss tracker (V5.1 - Leverage-Aware)
// 用于追踪每个持仓的动态止损状态，支持两阶段算法：Phase A (渐进保本) 和 Phase B (追踪止损)
// Track dynamic SL state for each position, supporting two-phase algorithm: Phase A (gradual breakeven) and Phase B (trailing)
type DynamicSLTracker struct {
	// ID 主键 / Primary key
	ID int64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`

	// PositionKey 持仓唯一标识 / Position unique identifier
	// 格式: "{instId}_{posSide}" 例如 "BTC-USDT-SWAP_long"
	// Format: "{instId}_{posSide}" e.g. "BTC-USDT-SWAP_long"
	PositionKey string `json:"position_key" gorm:"column:position_key;type:varchar(100);not null;uniqueIndex:idx_dynamic_sl_position_key"`

	// InstId 合约ID / Instrument ID
	InstId string `json:"inst_id" gorm:"column:inst_id;type:varchar(50);not null;index:idx_dynamic_sl_inst_id"`

	// PosSide 持仓方向 / Position side
	// 可选值: long, short, net
	// Possible values: long, short, net
	PosSide string `json:"pos_side" gorm:"column:pos_side;type:varchar(10);not null"`

	// EntryPrice 入场价格 / Entry price
	EntryPrice float64 `json:"entry_price" gorm:"column:entry_price;type:real;not null"`

	// CurrentSlPrice 当前止损价格 / Current stop-loss price
	CurrentSlPrice float64 `json:"current_sl_price" gorm:"column:current_sl_price;type:real;not null"`

	// InitialSlPrice 初始止损价格 / Initial stop-loss price (V5.1)
	// 用于计算到保本位的距离
	// Used to calculate distance to breakeven
	InitialSlPrice float64 `json:"initial_sl_price" gorm:"column:initial_sl_price;type:real;not null"`

	// Leverage 仓位杠杆倍率 / Position leverage (V5.1)
	// 用于计算杠杆感知的阈值
	// Used to calculate leverage-aware thresholds
	Leverage float64 `json:"leverage" gorm:"column:leverage;type:real;not null;default:1"`

	// HighestPriceReached 达到的最高价格（多头） / Highest price reached (for long positions)
	HighestPriceReached float64 `json:"highest_price_reached" gorm:"column:highest_price_reached;type:real"`

	// LowestPriceReached 达到的最低价格（空头） / Lowest price reached (for short positions)
	LowestPriceReached float64 `json:"lowest_price_reached" gorm:"column:lowest_price_reached;type:real"`

	// BreakevenReached 是否已到达保本位 / Whether breakeven has been reached (V5.1)
	// true 表示进入 Phase B (追踪模式)，false 表示仍在 Phase A (渐进保本)
	// true means entering Phase B (trailing mode), false means still in Phase A (gradual breakeven)
	BreakevenReached bool `json:"breakeven_reached" gorm:"column:breakeven_reached;type:boolean;not null;default:false"`

	// MoveCount 止损移动次数 / Number of SL adjustments made (V5.1)
	// 记录止损价格被调整的次数
	// Track how many times the stop-loss price has been adjusted
	MoveCount int `json:"move_count" gorm:"column:move_count;type:integer;not null;default:0"`

	// LastUpdatedAt 最后更新时间 / Last update timestamp
	LastUpdatedAt time.Time `json:"last_updated_at" gorm:"column:last_updated_at;type:datetime;not null"`

	// CreatedAt 创建时间 / Creation timestamp
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;type:datetime;not null"`
}

// TableName 返回表名 / Return table name
// GORM会使用此方法来确定表名
// GORM uses this method to determine the table name
//
// Returns:
//   - string: 表名"dynamic_sl_tracking" / Table name "dynamic_sl_tracking"
func (DynamicSLTracker) TableName() string {
	return "dynamic_sl_tracking"
}

// Validate 验证tracker字段 / Validate tracker fields (V5.1)
// 检查所有必填字段是否有效，包括新的V5.1字段
// Check if all required fields are valid, including new V5.1 fields
//
// Returns:
//   - error: 如果验证失败返回错误 / Error if validation fails
func (t *DynamicSLTracker) Validate() error {
	if t.PositionKey == "" {
		return fmt.Errorf("position_key cannot be empty")
	}

	if t.InstId == "" {
		return fmt.Errorf("inst_id cannot be empty")
	}

	if t.PosSide == "" {
		return fmt.Errorf("pos_side cannot be empty")
	}

	// Validate pos_side value
	if t.PosSide != "long" && t.PosSide != "short" && t.PosSide != "net" {
		return fmt.Errorf("pos_side must be 'long', 'short', or 'net', got: %s", t.PosSide)
	}

	if t.EntryPrice <= 0 {
		return fmt.Errorf("entry_price must be positive, got: %.8f", t.EntryPrice)
	}

	if t.CurrentSlPrice <= 0 {
		return fmt.Errorf("current_sl_price must be positive, got: %.8f", t.CurrentSlPrice)
	}

	// V5.1: Validate initial_sl_price
	if t.InitialSlPrice <= 0 {
		return fmt.Errorf("initial_sl_price must be positive, got: %.8f", t.InitialSlPrice)
	}

	// V5.1: Validate leverage (must be >= 1)
	if t.Leverage < 1 {
		return fmt.Errorf("leverage must be at least 1, got: %.2f", t.Leverage)
	}

	// V5.1: Validate move_count (must be non-negative)
	if t.MoveCount < 0 {
		return fmt.Errorf("move_count cannot be negative, got: %d", t.MoveCount)
	}

	// For long positions, highest price should be >= entry price
	if (t.PosSide == "long" || t.PosSide == "net") && t.HighestPriceReached > 0 {
		if t.HighestPriceReached < t.EntryPrice {
			return fmt.Errorf("highest_price_reached (%.8f) cannot be less than entry_price (%.8f) for long position",
				t.HighestPriceReached, t.EntryPrice)
		}
	}

	// For short positions, lowest price should be <= entry price
	if t.PosSide == "short" && t.LowestPriceReached > 0 {
		if t.LowestPriceReached > t.EntryPrice {
			return fmt.Errorf("lowest_price_reached (%.8f) cannot be greater than entry_price (%.8f) for short position",
				t.LowestPriceReached, t.EntryPrice)
		}
	}

	return nil
}
