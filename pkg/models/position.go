package models

import (
	"fmt"
	"time"
)

// Position 持仓信息 / Position data model
type Position struct {
	ID            int64        `json:"id" db:"id"`
	Timestamp     time.Time    `json:"timestamp" db:"timestamp"`
	Instrument    string       `json:"instrument" db:"instrument"`
	PositionSide  PositionSide `json:"position_side" db:"position_side"`
	PositionSize  float64      `json:"position_size" db:"position_size"`
	AveragePrice  float64      `json:"average_price" db:"average_price"`
	UnrealizedPnL float64      `json:"unrealized_pnl" db:"unrealized_pnl"`
	Margin        float64      `json:"margin" db:"margin"`
	Leverage      float64      `json:"leverage" db:"leverage"`
	MarginMode    MarginMode   `json:"margin_mode" db:"margin_mode"`
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
