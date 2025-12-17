package notifier

import (
	"context"
	"time"
)

// Interface defines the contract for sending order confirmation notifications.
// This interface abstracts the notification mechanism, enabling:
// - Easy testing through mock implementations
// - Flexibility to switch notification channels (log, email, Telegram, etc.)
// - Adherence to Dependency Inversion Principle (DIP)
//
// All implementations should be non-blocking and handle errors gracefully.
type Interface interface {
	// SendConfirmationRequest sends a notification requesting user confirmation for an order.
	// Used when an order needs user confirmation to prevent size reduction.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - info: Confirmation information including order details and timeout deadline
	//
	// Returns error on notification delivery failure.
	// Implementations should not panic on transient failures.
	SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error
}

// ConfirmationInfo contains the information needed for a confirmation notification.
type ConfirmationInfo struct {
	// OrderID is the unique identifier for the order
	OrderID string

	// Instrument is the trading pair (e.g., "BTC-USDT-SWAP")
	Instrument string

	// Side is the order side ("buy" or "sell")
	Side string

	// Size is the current order size
	Size string

	// Price is the order price (empty for market orders)
	Price string

	// PlacedAt is when the order was originally placed
	PlacedAt time.Time

	// TimeoutIn is the duration until the next timeout
	// After this duration, the order size may be reduced if not confirmed
	TimeoutIn time.Duration

	// ConfirmationCount is the number of confirmations received so far
	ConfirmationCount int

	// TimeoutCount is the number of timeouts that have occurred
	TimeoutCount int

	// MaxTimeouts is the maximum allowed timeouts before cancellation
	MaxTimeouts int
}
