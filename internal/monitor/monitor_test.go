package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Example test demonstrating the use of mock implementations
func TestMonitor_WithMocks(t *testing.T) {
	// Create mock storage
	mockStorage := &storage.MockStorage{
		Balances:  make([]models.AccountBalance, 0),
		Positions: make([]models.Position, 0),
	}

	// Create mock OKX client (using default behavior)
	mockOKX := &okx.MockClient{}

	// Create logger for testing (no file output, console output disabled for tests)
	testLogger, err := logger.New(
		"",      // empty path = no file output
		0,       // Debug level
		10,
		7,
		3,
		false,
		false,   // no console output for tests
	)
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}
	defer testLogger.Close()

	// Create monitor with mocks
	mon := New(mockOKX, mockStorage, testLogger, 60)

	// Test health check
	err = mon.healthCheck()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Verify that health check called both mocks
	if mockOKX.HealthCheckCallCount != 1 {
		t.Errorf("Expected HealthCheck to be called once on OKX client, got %d", mockOKX.HealthCheckCallCount)
	}

	// Note: We don't test fetchAndStore here as it would require more complex setup
	// This is just a demonstration of how to use mocks
}

// Example test showing custom error behavior
func TestMonitor_StorageError(t *testing.T) {
	// Create mock storage that simulates an error
	mockStorage := &storage.MockStorage{
		InsertBalanceFunc: func(balance *models.AccountBalance) error {
			return fmt.Errorf("simulated storage error")
		},
	}

	mockOKX := &okx.MockClient{}

	testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
	defer testLogger.Close()

	mon := New(mockOKX, mockStorage, testLogger, 60)

	// This should fail due to the simulated storage error
	err := mon.fetchAndStoreBalances()
	if err == nil {
		t.Error("Expected error from fetchAndStoreBalances, got nil")
	}
}

// Example benchmark showing performance testing with mocks
func BenchmarkMonitor_FetchAndStore(b *testing.B) {
	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{}
	testLogger, _ := logger.New("", 1, 10, 7, 3, false, false) // Info level
	defer testLogger.Close()

	mon := New(mockOKX, mockStorage, testLogger, 60)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mon.fetchAndStore()
	}
}

// Example table-driven test
func TestMonitor_GetMetrics(t *testing.T) {
	tests := []struct {
		name         string
		successCount int64
		errorCount   int64
	}{
		{"No operations", 0, 0},
		{"Only successes", 5, 0},
		{"Only errors", 0, 3},
		{"Mixed", 10, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &storage.MockStorage{}
			mockOKX := &okx.MockClient{}
			testLogger, _ := logger.New("", 1, 10, 7, 3, false, false)
			defer testLogger.Close()

			mon := New(mockOKX, mockStorage, testLogger, 60)
			mon.successCount = tt.successCount
			mon.errorCount = tt.errorCount
			mon.lastSuccess = time.Now()

			metrics := mon.GetMetrics()

			if metrics["success_count"] != tt.successCount {
				t.Errorf("Expected success_count=%d, got %d", tt.successCount, metrics["success_count"])
			}
			if metrics["error_count"] != tt.errorCount {
				t.Errorf("Expected error_count=%d, got %d", tt.errorCount, metrics["error_count"])
			}
		})
	}
}
