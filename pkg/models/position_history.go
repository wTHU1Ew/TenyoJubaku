package models

import (
	"fmt"
	"time"
)

// PositionHistory represents a closed position record from OKX
// 历史仓位记录（已平仓的仓位）
type PositionHistory struct {
	ID            int64     `json:"id" db:"id" gorm:"column:id;primaryKey;autoIncrement"`                                                    // Auto-increment ID
	PosId         string    `json:"pos_id" db:"pos_id" gorm:"column:pos_id;type:text;not null;uniqueIndex:idx_positions_history_pos_id"`   // OKX Position ID (unique identifier)
	InstType      string    `json:"inst_type" db:"inst_type" gorm:"column:inst_type;type:text;not null"`                                   // Instrument type: SWAP, FUTURES, etc.
	InstId        string    `json:"inst_id" db:"inst_id" gorm:"column:inst_id;type:text;not null;index:idx_positions_history_inst_id"`     // Instrument ID: BTC-USD-SWAP
	MgnMode       string    `json:"mgn_mode" db:"mgn_mode" gorm:"column:mgn_mode;type:text;not null"`                                      // Margin mode: cross, isolated
	PosSide       string    `json:"pos_side" db:"pos_side" gorm:"column:pos_side;type:text;not null"`                                      // Position side: long, short, net
	Lever         string    `json:"lever" db:"lever" gorm:"column:lever;type:text;not null"`                                               // Leverage
	OpenAvgPx     string    `json:"open_avg_px" db:"open_avg_px" gorm:"column:open_avg_px;type:text;not null"`                             // Average opening price
	CloseAvgPx    string    `json:"close_avg_px" db:"close_avg_px" gorm:"column:close_avg_px;type:text;not null"`                          // Average closing price
	OpenMaxPos    string    `json:"open_max_pos" db:"open_max_pos" gorm:"column:open_max_pos;type:text;not null"`                          // Maximum position size
	CloseTotalPos string    `json:"close_total_pos" db:"close_total_pos" gorm:"column:close_total_pos;type:text;not null"`                 // Total closed position size
	RealizedPnl   string    `json:"realized_pnl" db:"realized_pnl" gorm:"column:realized_pnl;type:text;not null"`                          // Realized profit/loss
	Pnl           string    `json:"pnl" db:"pnl" gorm:"column:pnl;type:text;not null"`                                                     // PnL (excluding fees)
	PnlRatio      string    `json:"pnl_ratio" db:"pnl_ratio" gorm:"column:pnl_ratio;type:text;not null"`                                   // PnL ratio
	Fee           string    `json:"fee" db:"fee" gorm:"column:fee;type:text;not null"`                                                     // Accumulated fee
	FundingFee    string    `json:"funding_fee" db:"funding_fee" gorm:"column:funding_fee;type:text;not null"`                             // Accumulated funding fee
	LiqPenalty    string    `json:"liq_penalty" db:"liq_penalty" gorm:"column:liq_penalty;type:text;not null"`                             // Liquidation penalty
	CloseType     string    `json:"close_type" db:"close_type" gorm:"column:close_type;type:text;not null;index:idx_positions_history_close_type"` // 1=partial, 2=all, 3=liq, 4=partial_liq, 5=adl
	Direction     string    `json:"direction" db:"direction" gorm:"column:direction;type:text;not null"`                                   // long or short
	Ccy           string    `json:"ccy" db:"ccy" gorm:"column:ccy;type:text;not null"`                                                     // Margin currency
	OpenedAt      time.Time `json:"opened_at" db:"opened_at" gorm:"column:opened_at;type:datetime;not null"`                               // Position created time
	ClosedAt      time.Time `json:"closed_at" db:"closed_at" gorm:"column:closed_at;type:datetime;not null;index:idx_positions_history_closed_at"` // Position closed/updated time
	CreatedAt     time.Time `json:"created_at" db:"created_at" gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`  // Local record created time
}

// TableName 指定表名 / Specify table name
func (PositionHistory) TableName() string {
	return "positions_history"
}

// Validate validates the PositionHistory data
func (p *PositionHistory) Validate() error {
	if p.PosId == "" {
		return fmt.Errorf("posId is required")
	}
	if p.InstId == "" {
		return fmt.Errorf("instId is required")
	}
	if p.InstType == "" {
		return fmt.Errorf("instType is required")
	}
	if p.PosSide == "" {
		return fmt.Errorf("posSide is required")
	}
	if p.OpenAvgPx == "" {
		return fmt.Errorf("openAvgPx is required")
	}
	if p.CloseAvgPx == "" {
		return fmt.Errorf("closeAvgPx is required")
	}
	if p.OpenedAt.IsZero() {
		return fmt.Errorf("openedAt is required")
	}
	if p.ClosedAt.IsZero() {
		return fmt.Errorf("closedAt is required")
	}
	return nil
}

// GetCloseTypeDescription returns a human-readable description of the close type
func (p *PositionHistory) GetCloseTypeDescription() string {
	switch p.CloseType {
	case "1":
		return "Partial Close"
	case "2":
		return "Full Close"
	case "3":
		return "Liquidation"
	case "4":
		return "Partial Liquidation"
	case "5", "6":
		return "ADL"
	default:
		return "Unknown"
	}
}
