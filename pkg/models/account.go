package models

import (
	"fmt"
	"time"
)

// AccountBalance 账户余额 / Account balance data model
type AccountBalance struct {
	ID        int64     `json:"id" db:"id"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
	Currency  string    `json:"currency" db:"currency"`
	Balance   float64   `json:"balance" db:"balance"`
	Available float64   `json:"available" db:"available"`
	Frozen    float64   `json:"frozen" db:"frozen"`
	Equity    float64   `json:"equity" db:"equity"`
}

// Validate 验证账户余额数据 / Validate account balance data
func (ab *AccountBalance) Validate() error {
	if ab.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if ab.Balance < 0 {
		return fmt.Errorf("balance cannot be negative")
	}
	if ab.Available < 0 {
		return fmt.Errorf("available cannot be negative")
	}
	if ab.Frozen < 0 {
		return fmt.Errorf("frozen cannot be negative")
	}
	return nil
}

// String 字符串表示 / String representation
func (ab *AccountBalance) String() string {
	return fmt.Sprintf("AccountBalance{Currency=%s, Balance=%.8f, Available=%.8f, Frozen=%.8f, Equity=%.8f, Timestamp=%s}",
		ab.Currency, ab.Balance, ab.Available, ab.Frozen, ab.Equity, ab.Timestamp.Format(time.RFC3339))
}
