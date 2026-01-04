package models

import (
	"fmt"
	"time"
)

// OrderHistory represents a placed order for frequency tracking
// 订单历史记录 / Order history record for tracking order frequency
type OrderHistory struct {
	ID         int64     `json:"id" db:"id" gorm:"column:id;primaryKey;autoIncrement"`
	OrderID    string    `json:"order_id" db:"order_id" gorm:"column:order_id;type:text;not null;index:idx_order_history_order_id"`
	InstId     string    `json:"inst_id" db:"inst_id" gorm:"column:inst_id;type:text;not null"`
	Side       string    `json:"side" db:"side" gorm:"column:side;type:text;not null"`                               // buy, sell
	OrdType    string    `json:"ord_type" db:"ord_type" gorm:"column:ord_type;type:text;not null"`                  // limit, market, post_only, fok, ioc
	Size       string    `json:"size" db:"size" gorm:"column:size;type:text;not null"`
	Price      string    `json:"price" db:"price" gorm:"column:price;type:text"`
	ReduceOnly bool      `json:"reduce_only" db:"reduce_only" gorm:"column:reduce_only;type:boolean;not null;default:0"`
	PlacedAt   time.Time `json:"placed_at" db:"placed_at" gorm:"column:placed_at;type:datetime;not null;index:idx_order_history_placed_at"`
	WeekStart  time.Time `json:"week_start" db:"week_start" gorm:"column:week_start;type:date;not null;index:idx_order_history_week_start"` // Monday 00:00:00 UTC of the week
	Status     string    `json:"status" db:"status" gorm:"column:status;type:text;not null"`                                                 // placed, filled, canceled, failed
	CreatedAt  time.Time `json:"created_at" db:"created_at" gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP"`
}

// TableName 指定表名 / Specify table name
func (OrderHistory) TableName() string {
	return "order_history"
}

// Validate validates the order history record
func (o *OrderHistory) Validate() error {
	if o.OrderID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}
	if o.InstId == "" {
		return fmt.Errorf("instrument ID cannot be empty")
	}
	if o.Side != "buy" && o.Side != "sell" {
		return fmt.Errorf("side must be buy or sell")
	}
	if o.Size == "" {
		return fmt.Errorf("size cannot be empty")
	}
	if o.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return nil
}

// PendingConfirmation represents a pending order requiring confirmation
// 待确认订单记录 / Pending confirmation record for order control
type PendingConfirmation struct {
	ID                  int64      `json:"id" db:"id" gorm:"column:id;primaryKey;autoIncrement"`
	OrderID             string     `json:"order_id" db:"order_id" gorm:"column:order_id;type:text;not null;uniqueIndex:idx_pending_confirmation_order_id"`
	InstId              string     `json:"inst_id" db:"inst_id" gorm:"column:inst_id;type:text;not null"`
	Side                string     `json:"side" db:"side" gorm:"column:side;type:text;not null"`
	OrdType             string     `json:"ord_type" db:"ord_type" gorm:"column:ord_type;type:text;not null"`
	OriginalSize        string     `json:"original_size" db:"original_size" gorm:"column:original_size;type:text;not null"`
	CurrentSize         string     `json:"current_size" db:"current_size" gorm:"column:current_size;type:text;not null"`
	Price               string     `json:"price" db:"price" gorm:"column:price;type:text"`
	PlacedAt            time.Time  `json:"placed_at" db:"placed_at" gorm:"column:placed_at;type:datetime;not null"`
	LastConfirmationAt  *time.Time `json:"last_confirmation_at" db:"last_confirmation_at" gorm:"column:last_confirmation_at;type:datetime"`
	NextConfirmationDue time.Time  `json:"next_confirmation_due" db:"next_confirmation_due" gorm:"column:next_confirmation_due;type:datetime;not null;index:idx_pending_confirmation_next_due"`
	ConfirmationCount   int        `json:"confirmation_count" db:"confirmation_count" gorm:"column:confirmation_count;type:integer;default:0"`
	TimeoutCount        int        `json:"timeout_count" db:"timeout_count" gorm:"column:timeout_count;type:integer;default:0"`
	Status              string     `json:"status" db:"status" gorm:"column:status;type:text;not null"` // pending, confirmed, timeout, canceled
	CreatedAt           time.Time  `json:"created_at" db:"created_at" gorm:"column:created_at;type:datetime;not null"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at" gorm:"column:updated_at;type:datetime;not null"`
}

// TableName 指定表名 / Specify table name
func (PendingConfirmation) TableName() string {
	return "pending_confirmations"
}

// Validate validates the pending confirmation record
func (p *PendingConfirmation) Validate() error {
	if p.OrderID == "" {
		return fmt.Errorf("order ID cannot be empty")
	}
	if p.InstId == "" {
		return fmt.Errorf("instrument ID cannot be empty")
	}
	if p.OriginalSize == "" {
		return fmt.Errorf("original size cannot be empty")
	}
	if p.CurrentSize == "" {
		return fmt.Errorf("current size cannot be empty")
	}
	if p.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return nil
}

// GetWeekStart returns the Monday 00:00:00 UTC of the week for a given time
func GetWeekStart(t time.Time) time.Time {
	// Convert to UTC
	t = t.UTC()

	// Get the weekday (0 = Sunday, 1 = Monday, ...)
	weekday := t.Weekday()

	// Calculate days to subtract to get to Monday
	daysToMonday := int(weekday)
	if weekday == time.Sunday {
		daysToMonday = 6 // Sunday is 6 days after Monday
	} else {
		daysToMonday = int(weekday) - 1
	}

	// Subtract days to get to Monday, then truncate to start of day
	monday := t.AddDate(0, 0, -daysToMonday)
	weekStart := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)

	return weekStart
}

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPlaced   OrderStatus = "placed"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusFailed   OrderStatus = "failed"
)

// String returns the string representation of OrderStatus
func (s OrderStatus) String() string {
	return string(s)
}

// ConfirmationStatus represents the status of a pending confirmation
type ConfirmationStatus string

const (
	ConfirmationStatusPending   ConfirmationStatus = "pending"
	ConfirmationStatusConfirmed ConfirmationStatus = "confirmed"
	ConfirmationStatusTimeout   ConfirmationStatus = "timeout"
	ConfirmationStatusCanceled  ConfirmationStatus = "canceled"
)

// String returns the string representation of ConfirmationStatus
func (s ConfirmationStatus) String() string {
	return string(s)
}

// ConfirmationUpdate represents updates to apply to a pending confirmation
type ConfirmationUpdate struct {
	CurrentSize         *string
	LastConfirmationAt  *time.Time
	NextConfirmationDue *time.Time
	ConfirmationCount   *int
	TimeoutCount        *int
	Status              *string
	UpdatedAt           time.Time
}

// WeeklyOrderCount represents order count statistics for a week
type WeeklyOrderCount struct {
	WeekStart         time.Time
	TotalOrders       int
	ReduceOnlyOrders  int
	RegularOrders     int // Total - ReduceOnly
	PlacedOrders      int // Only placed orders
	FilledOrders      int
	CanceledOrders    int
	FailedOrders      int
}

// String returns a formatted string representation
func (w *WeeklyOrderCount) String() string {
	return fmt.Sprintf("Week %s: Total=%d, Regular=%d, ReduceOnly=%d, Placed=%d",
		w.WeekStart.Format("2006-01-02"), w.TotalOrders, w.RegularOrders,
		w.ReduceOnlyOrders, w.PlacedOrders)
}
