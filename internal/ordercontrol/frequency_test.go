package ordercontrol

import (
	"context"
	"testing"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/notifier"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// TestCheckFrequencyLimit_Disabled tests that frequency limit check is skipped when disabled
func TestCheckFrequencyLimit_Disabled(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		FrequencyLimit: config.FrequencyLimitConfig{
			Enabled:           false, // Disabled
			WeeklyMaxOrders:   20,    // Value doesn't matter when disabled
			ExcludeReduceOnly: false,
		},
	}

	mockStorage := &storage.MockStorage{
		OrderCountForWeek: 100, // Way over limit, but should be ignored
	}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test
	req := &okx.OrderRequest{
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Sz:         "0.01",
		Px:         "50000",
		ReduceOnly: false,
	}

	err := service.checkFrequencyLimit(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error when frequency limit disabled, got: %v", err)
	}
}

// TestCheckFrequencyLimit_UnderLimit tests successful check when under limit
func TestCheckFrequencyLimit_UnderLimit(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		FrequencyLimit: config.FrequencyLimitConfig{
			Enabled:           true,
			WeeklyMaxOrders:   20,
			ExcludeReduceOnly: false,
		},
	}

	mockStorage := &storage.MockStorage{
		OrderCountForWeek: 10, // Under limit
	}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test
	req := &okx.OrderRequest{
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Sz:         "0.01",
		Px:         "50000",
		ReduceOnly: false,
	}

	err := service.checkFrequencyLimit(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error when under limit (10/20), got: %v", err)
	}
}

// TestCheckFrequencyLimit_AtLimit tests rejection when at limit
func TestCheckFrequencyLimit_AtLimit(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		FrequencyLimit: config.FrequencyLimitConfig{
			Enabled:           true,
			WeeklyMaxOrders:   20,
			ExcludeReduceOnly: false,
		},
	}

	mockStorage := &storage.MockStorage{
		OrderCountForWeek: 20, // At limit
	}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test
	req := &okx.OrderRequest{
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Sz:         "0.01",
		Px:         "50000",
		ReduceOnly: false,
	}

	err := service.checkFrequencyLimit(context.Background(), req)

	// Assert
	if err == nil {
		t.Error("Expected error when at limit (20/20), got nil")
	}
}

// TestCheckFrequencyLimit_ExcludeReduceOnly tests that reduce-only orders are excluded
func TestCheckFrequencyLimit_ExcludeReduceOnly(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		FrequencyLimit: config.FrequencyLimitConfig{
			Enabled:           true,
			WeeklyMaxOrders:   20,
			ExcludeReduceOnly: true, // Exclude reduce-only orders
		},
	}

	mockStorage := &storage.MockStorage{
		OrderCountForWeek: 25, // Over limit, but this is a reduce-only order
	}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - reduce-only order
	req := &okx.OrderRequest{
		InstId:     "BTC-USDT-SWAP",
		Side:       "sell",
		OrdType:    "limit",
		Sz:         "0.01",
		Px:         "51000",
		ReduceOnly: true, // This is a reduce-only order
	}

	err := service.checkFrequencyLimit(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for reduce-only order when ExcludeReduceOnly=true, got: %v", err)
	}
}

// TestCheckFrequencyLimit_WithStats tests frequency check with detailed stats
func TestCheckFrequencyLimit_WithStats(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		FrequencyLimit: config.FrequencyLimitConfig{
			Enabled:           true,
			WeeklyMaxOrders:   20,
			ExcludeReduceOnly: true,
		},
	}

	weekStart := models.GetWeekStart(time.Now())
	mockStorage := &storage.MockStorage{
		OrderCountForWeek: 25, // Total 25 orders
		WeeklyStats: &models.WeeklyOrderCount{
			WeekStart:        weekStart,
			TotalOrders:      25,
			ReduceOnlyOrders: 10,
			RegularOrders:    15, // Only 15 regular orders (under 20 limit)
		},
	}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - regular order (not reduce-only)
	req := &okx.OrderRequest{
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Sz:         "0.01",
		Px:         "50000",
		ReduceOnly: false,
	}

	err := service.checkFrequencyLimit(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error (15 regular orders < 20 limit), got: %v", err)
	}
}
