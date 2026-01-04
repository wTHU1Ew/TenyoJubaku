package models

import (
	"fmt"
	"time"
)

// Position 持仓信息 / Position data model
type Position struct {
	ID            int64        `json:"id" db:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Timestamp     time.Time    `json:"timestamp" db:"timestamp" gorm:"column:timestamp;type:datetime;not null;index:idx_positions_timestamp"`
	Instrument    string       `json:"instrument" db:"instrument" gorm:"column:instrument;type:varchar(50);not null"`
	PositionSide  PositionSide `json:"position_side" db:"position_side" gorm:"column:position_side;type:varchar(10);not null"`
	PositionSize  float64      `json:"position_size" db:"position_size" gorm:"column:position_size;type:real;not null"`
	AveragePrice  float64      `json:"average_price" db:"average_price" gorm:"column:average_price;type:real;not null"`
	UnrealizedPnL float64      `json:"unrealized_pnl" db:"unrealized_pnl" gorm:"column:unrealized_pnl;type:real;not null"`
	Margin        float64      `json:"margin" db:"margin" gorm:"column:margin;type:real;not null"`
	Leverage      float64      `json:"leverage" db:"leverage" gorm:"column:leverage;type:real"`
	MarginMode    MarginMode   `json:"margin_mode" db:"margin_mode" gorm:"column:margin_mode;type:varchar(10);default:cross"`
}

// TableName 指定表名 / Specify table name
func (Position) TableName() string {
	return "positions"
}

// Validate 验证持仓数据 / Validate position data
func (p *Position) Validate() error {
	if p.Instrument == "" {
		return fmt.Errorf("instrument is required")
	}
	if p.PositionSide == "" {
		return fmt.Errorf("position_side is required")
	}
	if !p.PositionSide.IsValid() {
		return fmt.Errorf("invalid position_side: %s (must be 'long', 'short', or 'net')", p.PositionSide)
	}
	if p.PositionSize < 0 {
		return fmt.Errorf("position_size cannot be negative")
	}
	if p.AveragePrice < 0 {
		return fmt.Errorf("average_price cannot be negative")
	}
	if p.Margin < 0 {
		return fmt.Errorf("margin cannot be negative")
	}
	if p.Leverage < 0 {
		return fmt.Errorf("leverage cannot be negative")
	}
	if p.MarginMode != "" && !p.MarginMode.IsValid() {
		return fmt.Errorf("invalid margin_mode: %s (must be 'cross' or 'isolated')", p.MarginMode)
	}
	return nil
}

// String 字符串表示 / String representation
func (p *Position) String() string {
	return fmt.Sprintf("Position{Instrument=%s, Side=%s, Size=%.8f, AvgPrice=%.8f, UnrealizedPnL=%.8f, Margin=%.8f, Leverage=%.2f, Mode=%s, Timestamp=%s}",
		p.Instrument, p.PositionSide, p.PositionSize, p.AveragePrice, p.UnrealizedPnL, p.Margin, p.Leverage, p.MarginMode, p.Timestamp.Format(time.RFC3339))
}
