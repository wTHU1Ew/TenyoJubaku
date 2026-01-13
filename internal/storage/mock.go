package storage

import (
	"context"
	"sync"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// MockStorage is a mock implementation of the Storage interface for testing.
// It stores data in memory and provides configurable behavior for testing various scenarios.
//
// Example usage:
//
//	mock := &MockStorage{
//	    Balances:  make([]models.AccountBalance, 0),
//	    Positions: make([]models.Position, 0),
//	}
//	monitor := monitor.New(okxClient, mock, logger, 60)
type MockStorage struct {
	mu        sync.Mutex
	Balances  []models.AccountBalance
	Positions []models.Position

	// Order Control test data
	OrderCountForWeek int
	WeeklyStats       *models.WeeklyOrderCount
	OrderHistory      []models.OrderHistory

	// Optional: Configure custom behavior
	InsertBalanceFunc func(*models.AccountBalance) error
	InsertPositionFunc func(*models.Position) error
	GetLatestBalancesFunc func() ([]models.AccountBalance, error)
	GetLatestPositionsFunc func() ([]models.Position, error)
	GetBalancesByTimeRangeFunc func(string, time.Time, time.Time) ([]models.AccountBalance, error)
	HealthCheckFunc func() error
	CloseFunc func() error
}

// InsertAccountBalance mocks inserting an account balance.
func (m *MockStorage) InsertAccountBalance(balance *models.AccountBalance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is provided, use it
	if m.InsertBalanceFunc != nil {
		return m.InsertBalanceFunc(balance)
	}

	// Default behavior: append to slice
	balance.ID = int64(len(m.Balances) + 1)
	m.Balances = append(m.Balances, *balance)
	return nil
}

// InsertPosition mocks inserting a position.
func (m *MockStorage) InsertPosition(position *models.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is provided, use it
	if m.InsertPositionFunc != nil {
		return m.InsertPositionFunc(position)
	}

	// Default behavior: append to slice
	position.ID = int64(len(m.Positions) + 1)
	m.Positions = append(m.Positions, *position)
	return nil
}

// GetLatestAccountBalances mocks retrieving latest balances.
func (m *MockStorage) GetLatestAccountBalances() ([]models.AccountBalance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is provided, use it
	if m.GetLatestBalancesFunc != nil {
		return m.GetLatestBalancesFunc()
	}

	// Default behavior: return all balances (for simplicity)
	return m.Balances, nil
}

// GetLatestPositions mocks retrieving latest positions.
func (m *MockStorage) GetLatestPositions() ([]models.Position, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is provided, use it
	if m.GetLatestPositionsFunc != nil {
		return m.GetLatestPositionsFunc()
	}

	// Default behavior: return all positions (for simplicity)
	return m.Positions, nil
}

// GetAccountBalancesByTimeRange mocks querying balances by time range.
func (m *MockStorage) GetAccountBalancesByTimeRange(currency string, startTime, endTime time.Time) ([]models.AccountBalance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If custom function is provided, use it
	if m.GetBalancesByTimeRangeFunc != nil {
		return m.GetBalancesByTimeRangeFunc(currency, startTime, endTime)
	}

	// Default behavior: filter by currency and time range
	result := make([]models.AccountBalance, 0)
	for _, balance := range m.Balances {
		if balance.Currency == currency &&
			!balance.Timestamp.Before(startTime) &&
			!balance.Timestamp.After(endTime) {
			result = append(result, balance)
		}
	}
	return result, nil
}

// HealthCheck mocks health check.
func (m *MockStorage) HealthCheck() error {
	// If custom function is provided, use it
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc()
	}

	// Default behavior: always healthy
	return nil
}

// Close mocks closing the storage.
func (m *MockStorage) Close() error {
	// If custom function is provided, use it
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}

	// Default behavior: no-op
	return nil
}

// Reset clears all stored data (useful for test cleanup).
func (m *MockStorage) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Balances = make([]models.AccountBalance, 0)
	m.Positions = make([]models.Position, 0)
}

// Order history mock methods
func (m *MockStorage) InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	order.ID = int64(len(m.OrderHistory) + 1)
	m.OrderHistory = append(m.OrderHistory, *order)
	return nil
}

func (m *MockStorage) GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.OrderCountForWeek, nil
}

func (m *MockStorage) GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.OrderHistory, nil
}

func (m *MockStorage) GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.WeeklyStats, nil
}

// Pending confirmation mock methods (stubs)
func (m *MockStorage) InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error {
	return nil
}

func (m *MockStorage) GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error) {
	return nil, nil
}

func (m *MockStorage) GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error) {
	return nil, nil
}

func (m *MockStorage) UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error {
	return nil
}

func (m *MockStorage) DeletePendingConfirmation(ctx context.Context, orderID string) error {
	return nil
}

// Position History mock methods (stubs)
func (m *MockStorage) InsertPositionHistory(ctx context.Context, position *models.PositionHistory) error {
	return nil
}

func (m *MockStorage) GetPositionsHistory(ctx context.Context, instId string, limit int) ([]models.PositionHistory, error) {
	return nil, nil
}

// Dynamic SL Tracker mock methods (stubs)
func (m *MockStorage) InsertDynamicSLTracker(ctx context.Context, tracker *models.DynamicSLTracker) error {
	return nil
}

func (m *MockStorage) GetDynamicSLTracker(ctx context.Context, positionKey string) (*models.DynamicSLTracker, error) {
	return nil, nil
}

func (m *MockStorage) UpdateDynamicSLTracker(ctx context.Context, tracker *models.DynamicSLTracker) error {
	return nil
}

func (m *MockStorage) DeleteDynamicSLTracker(ctx context.Context, positionKey string) error {
	return nil
}

func (m *MockStorage) GetAllDynamicSLTrackers(ctx context.Context) ([]models.DynamicSLTracker, error) {
	return nil, nil
}

func (m *MockStorage) CleanupOrphanedTrackers(ctx context.Context, openPositionKeys []string) (int, error) {
	return 0, nil
}
