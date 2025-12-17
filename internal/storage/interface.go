package storage

import (
	"context"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Interface defines the contract for database storage operations.
// This interface abstracts the underlying storage implementation, enabling:
// - Easy testing through mock implementations
// - Flexibility to switch database backends (SQLite, PostgreSQL, etc.)
// - Adherence to Dependency Inversion Principle (DIP)
//
// All implementations must be thread-safe and handle errors gracefully.
type Interface interface {
	// InsertAccountBalance inserts an account balance record into storage.
	// The balance parameter must contain valid Currency, Balance, and Available fields.
	// On success, the generated ID is written back to balance.ID.
	//
	// Returns error on validation failure or database write failure.
	InsertAccountBalance(balance *models.AccountBalance) error

	// InsertPosition inserts a position record into storage.
	// The position parameter must contain valid Instrument, PositionSide, and PositionSize fields.
	// On success, the generated ID is written back to position.ID.
	//
	// Returns error on validation failure or database write failure.
	InsertPosition(position *models.Position) error

	// GetLatestAccountBalances retrieves the most recent account balance records.
	// Returns all currency balances with the latest timestamp, sorted by currency.
	// Returns an empty slice if no records exist.
	//
	// Returns error on database query failure.
	GetLatestAccountBalances() ([]models.AccountBalance, error)

	// GetLatestPositions retrieves the most recent position records.
	// Returns all positions from the latest snapshot, sorted by instrument.
	// Returns an empty slice if no records exist.
	//
	// Note: This is mainly used for historical data analysis.
	// TPSL service should fetch positions directly from exchange API for real-time accuracy.
	//
	// Returns error on database query failure.
	GetLatestPositions() ([]models.Position, error)

	// GetAccountBalancesByTimeRange queries account balances within a time range.
	// Returns balances for the specified currency between startTime and endTime (inclusive).
	// Results are ordered by timestamp in ascending order.
	//
	// Returns error on database query failure.
	GetAccountBalancesByTimeRange(currency string, startTime, endTime time.Time) ([]models.AccountBalance, error)

	// HealthCheck verifies database connectivity.
	// Returns nil if database is reachable and responsive.
	//
	// Returns error on connection failure.
	HealthCheck() error

	// Close closes the database connection and releases resources.
	// Should be called during graceful shutdown.
	//
	// Returns error on closure failure.
	Close() error

	// === Order History Operations ===

	// InsertOrderHistory inserts an order history record into storage.
	// Used for tracking order frequency within rolling weekly windows.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - order: Order history record with OrderID, InstId, Side, Size, etc.
	//
	// Returns error on validation failure or database write failure.
	InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error

	// GetOrderCountForWeek returns the count of orders placed in a given week.
	// The week is identified by weekStart (Monday 00:00:00 UTC).
	// Use models.GetWeekStart() to calculate weekStart from any time.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - weekStart: Monday 00:00:00 UTC of the target week
	//
	// Returns the total number of orders placed during that week.
	// Returns error on database query failure.
	GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error)

	// GetOrdersForWeek retrieves all orders placed in a given week.
	// Returns detailed order information for frequency analysis and auditing.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - weekStart: Monday 00:00:00 UTC of the target week
	//
	// Returns slice of order history records, ordered by PlacedAt timestamp.
	// Returns empty slice if no orders exist for that week.
	// Returns error on database query failure.
	GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error)

	// GetWeeklyOrderStats retrieves aggregated order statistics for a week.
	// Returns counts broken down by status, order type, and reduce-only flag.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - weekStart: Monday 00:00:00 UTC of the target week
	//
	// Returns WeeklyOrderCount with aggregated statistics.
	// Returns error on database query failure.
	GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error)

	// === Pending Confirmation Operations ===

	// InsertPendingConfirmation inserts a pending confirmation record.
	// Used when an order requires user confirmation to avoid size reduction.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - conf: Pending confirmation record with OrderID, NextConfirmationDue, etc.
	//
	// Returns error on validation failure or database write failure.
	InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error

	// GetPendingConfirmationsDue retrieves confirmations that need checking.
	// Returns all pending confirmations where NextConfirmationDue <= now.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - now: Current time for comparison
	//
	// Returns slice of pending confirmations that are due for processing.
	// Returns empty slice if no confirmations are due.
	// Returns error on database query failure.
	GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error)

	// GetPendingConfirmation retrieves a specific pending confirmation by order ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - orderID: The order ID to query
	//
	// Returns the pending confirmation record.
	// Returns error if not found or on database query failure.
	GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error)

	// UpdatePendingConfirmation updates a pending confirmation record.
	// Used to update status, confirmation count, timeout count, size, etc.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - orderID: The order ID to update
	//   - update: Struct containing the fields to update
	//
	// Returns error if order not found or on database update failure.
	UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error

	// DeletePendingConfirmation removes a pending confirmation record.
	// Used when an order is confirmed, canceled, or reaches terminal state.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - orderID: The order ID to delete
	//
	// Returns error if order not found or on database delete failure.
	DeletePendingConfirmation(ctx context.Context, orderID string) error
}

// Ensure Storage implements Interface at compile time
var _ Interface = (*Storage)(nil)
