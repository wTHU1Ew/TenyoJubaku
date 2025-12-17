package models

import (
	"fmt"
	"time"
)

// OrderHistory represents a placed order for frequency tracking
type OrderHistory struct {
	ID         int64
	OrderID    string
	InstId     string
	Side       string // buy, sell
	OrdType    string // limit, market, post_only, fok, ioc
	Size       string
	Price      string
	ReduceOnly bool
	PlacedAt   time.Time
	WeekStart  time.Time // Monday 00:00:00 UTC of the week
	Status     string    // placed, filled, canceled, failed
	CreatedAt  time.Time
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
type PendingConfirmation struct {
	ID                  int64
	OrderID             string
	InstId              string
	Side                string
	OrdType             string
	OriginalSize        string
	CurrentSize         string
	Price               string
	PlacedAt            time.Time
	LastConfirmationAt  *time.Time
	NextConfirmationDue time.Time
	ConfirmationCount   int
	TimeoutCount        int
	Status              string // pending, confirmed, timeout, canceled
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
