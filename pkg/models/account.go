package models

import (
	"fmt"
	"time"
)

// AccountBalance 账户余额 / Account balance data model
type AccountBalance struct {
	ID        int64     `json:"id" db:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Timestamp time.Time `json:"timestamp" db:"timestamp" gorm:"column:timestamp;type:datetime;not null;index:idx_account_balances_timestamp"`
	Currency  string    `json:"currency" db:"currency" gorm:"column:currency;type:varchar(10);not null;index:idx_account_balances_currency"`
	Balance   float64   `json:"balance" db:"balance" gorm:"column:balance;type:real;not null"`
	Available float64   `json:"available" db:"available" gorm:"column:available;type:real;not null"`
	Frozen    float64   `json:"frozen" db:"frozen" gorm:"column:frozen;type:real;not null"`
	Equity    float64   `json:"equity" db:"equity" gorm:"column:equity;type:real"`
}

// TableName 指定表名 / Specify table name
func (AccountBalance) TableName() string {
	return "account_balances"
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
