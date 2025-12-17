package ordercontrol

import (
	"context"
	"testing"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/notifier"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
)

// TestCheckMakerOnly_Disabled tests that maker-only check is skipped when disabled
func TestCheckMakerOnly_Disabled(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             false, // Disabled
			MinPriceDistancePct: 0.05,  // Value doesn't matter when disabled
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - market order (would normally fail)
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "market", // Market order
		Sz:      "0.01",
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error when maker-only disabled, got: %v", err)
	}
}

// TestCheckMakerOnly_MarketOrder tests rejection of market orders
func TestCheckMakerOnly_MarketOrder(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.05,
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - market order
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "market",
		Sz:      "0.01",
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err == nil {
		t.Error("Expected error for market order when maker-only enabled, got nil")
	}
}

// TestCheckMakerOnly_PostOnly tests that post_only orders always pass
func TestCheckMakerOnly_PostOnly(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.05,
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - post_only order
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "post_only",
		Sz:      "0.01",
		Px:      "50000",
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for post_only order, got: %v", err)
	}
}

// TestCheckMakerOnly_LimitOrder_SufficientDistance tests limit order with sufficient price distance
func TestCheckMakerOnly_LimitOrder_SufficientDistance(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.1, // 0.1% minimum distance
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{
		TickerResponse: &okx.TickerResponse{
			Data: []okx.TickerData{
				{
					InstId: "BTC-USDT-SWAP",
					BidPx:  "50000", // Best bid: 50000
					AskPx:  "50100", // Best ask: 50100
				},
			},
		},
	}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - buy order at 49900 (0.4% below ask of 50100)
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "limit",
		Sz:      "0.01",
		Px:      "49900", // 0.4% below ask (sufficient)
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for buy order with sufficient distance, got: %v", err)
	}
}

// TestCheckMakerOnly_LimitOrder_InsufficientDistance tests limit order with insufficient price distance
func TestCheckMakerOnly_LimitOrder_InsufficientDistance(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.5, // 0.5% minimum distance
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{
		TickerResponse: &okx.TickerResponse{
			Data: []okx.TickerData{
				{
					InstId: "BTC-USDT-SWAP",
					BidPx:  "50000",
					AskPx:  "50100",
				},
			},
		},
	}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - buy order at 50050 (0.1% below ask, insufficient)
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "limit",
		Sz:      "0.01",
		Px:      "50050", // 0.1% below ask (insufficient, needs 0.5%)
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err == nil {
		t.Error("Expected error for buy order with insufficient distance, got nil")
	}
}

// TestCheckMakerOnly_SellOrder_SufficientDistance tests sell order with sufficient price distance
func TestCheckMakerOnly_SellOrder_SufficientDistance(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.1, // 0.1% minimum distance
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{
		TickerResponse: &okx.TickerResponse{
			Data: []okx.TickerData{
				{
					InstId: "BTC-USDT-SWAP",
					BidPx:  "50000", // Best bid: 50000
					AskPx:  "50100",
				},
			},
		},
	}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - sell order at 50200 (0.4% above bid of 50000)
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "sell",
		OrdType: "limit",
		Sz:      "0.01",
		Px:      "50200", // 0.4% above bid (sufficient)
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err != nil {
		t.Errorf("Expected no error for sell order with sufficient distance, got: %v", err)
	}
}

// TestCheckMakerOnly_BuyOrder_AboveMarket tests that buy orders above market are rejected
func TestCheckMakerOnly_BuyOrder_AboveMarket(t *testing.T) {
	// Setup
	cfg := &config.OrderControlConfig{
		Enabled: true,
		MakerOnly: config.MakerOnlyConfig{
			Enabled:             true,
			MinPriceDistancePct: 0.1,
		},
	}

	mockStorage := &storage.MockStorage{}
	mockOKX := &okx.MockClient{
		TickerResponse: &okx.TickerResponse{
			Data: []okx.TickerData{
				{
					InstId: "BTC-USDT-SWAP",
					BidPx:  "50000",
					AskPx:  "50100", // Market ask: 50100
				},
			},
		},
	}
	mockNotifier := notifier.NewLogNotifier(logger.NewTestLogger())
	log := logger.NewTestLogger()

	service := New(cfg, mockOKX, mockStorage, mockNotifier, log)

	// Test - buy order at 50200 (above market ask, would be taker)
	req := &okx.OrderRequest{
		InstId:  "BTC-USDT-SWAP",
		Side:    "buy",
		OrdType: "limit",
		Sz:      "0.01",
		Px:      "50200", // Above ask (would be taker)
	}

	err := service.checkMakerOnly(context.Background(), req)

	// Assert
	if err == nil {
		t.Error("Expected error for buy order above market, got nil")
	}
}
