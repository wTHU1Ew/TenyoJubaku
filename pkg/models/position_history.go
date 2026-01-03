package models

import (
	"fmt"
	"time"
)

// PositionHistory represents a closed position record from OKX
// 历史仓位记录（已平仓的仓位）
type PositionHistory struct {
	ID            int64     `json:"id"`            // Auto-increment ID
	PosId         string    `json:"pos_id"`        // OKX Position ID (unique identifier)
	InstType      string    `json:"inst_type"`     // Instrument type: SWAP, FUTURES, etc.
	InstId        string    `json:"inst_id"`       // Instrument ID: BTC-USD-SWAP
	MgnMode       string    `json:"mgn_mode"`      // Margin mode: cross, isolated
	PosSide       string    `json:"pos_side"`      // Position side: long, short, net
	Lever         string    `json:"lever"`         // Leverage
	OpenAvgPx     string    `json:"open_avg_px"`   // Average opening price
	CloseAvgPx    string    `json:"close_avg_px"`  // Average closing price
	OpenMaxPos    string    `json:"open_max_pos"`  // Maximum position size
	CloseTotalPos string    `json:"close_total_pos"` // Total closed position size
	RealizedPnl   string    `json:"realized_pnl"`  // Realized profit/loss
	Pnl           string    `json:"pnl"`           // PnL (excluding fees)
	PnlRatio      string    `json:"pnl_ratio"`     // PnL ratio
	Fee           string    `json:"fee"`           // Accumulated fee
	FundingFee    string    `json:"funding_fee"`   // Accumulated funding fee
	LiqPenalty    string    `json:"liq_penalty"`   // Liquidation penalty
	CloseType     string    `json:"close_type"`    // 1=partial, 2=all, 3=liq, 4=partial_liq, 5=adl
	Direction     string    `json:"direction"`     // long or short
	Ccy           string    `json:"ccy"`           // Margin currency
	OpenedAt      time.Time `json:"opened_at"`     // Position created time
	ClosedAt      time.Time `json:"closed_at"`     // Position closed/updated time
	CreatedAt     time.Time `json:"created_at"`    // Local record created time
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
