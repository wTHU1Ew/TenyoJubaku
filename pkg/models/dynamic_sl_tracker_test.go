package models

import (
	"strings"
	"testing"
	"time"
)

func TestDynamicSLTracker_TableName(t *testing.T) {
	tracker := DynamicSLTracker{}
	expected := "dynamic_sl_tracking"
	if got := tracker.TableName(); got != expected {
		t.Errorf("TableName() = %v, want %v", got, expected)
	}
}

func TestDynamicSLTracker_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tracker *DynamicSLTracker
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid long position tracker",
			tracker: &DynamicSLTracker{
				PositionKey:         "BTC-USDT-SWAP_long",
				InstId:              "BTC-USDT-SWAP",
				PosSide:             "long",
				EntryPrice:          40000,
				CurrentSlPrice:      38000,
				HighestPriceReached: 40500,
				FirstMoveTriggered:  false,
				LastUpdatedAt:       time.Now(),
				CreatedAt:           time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid short position tracker",
			tracker: &DynamicSLTracker{
				PositionKey:        "ETH-USDT-SWAP_short",
				InstId:             "ETH-USDT-SWAP",
				PosSide:            "short",
				EntryPrice:         3000,
				CurrentSlPrice:     3100,
				LowestPriceReached: 2950,
				FirstMoveTriggered: true,
				LastUpdatedAt:      time.Now(),
				CreatedAt:          time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty position_key",
			tracker: &DynamicSLTracker{
				PositionKey:    "",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "long",
				EntryPrice:     40000,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "position_key cannot be empty",
		},
		{
			name: "empty inst_id",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "",
				PosSide:        "long",
				EntryPrice:     40000,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "inst_id cannot be empty",
		},
		{
			name: "empty pos_side",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "",
				EntryPrice:     40000,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "pos_side cannot be empty",
		},
		{
			name: "invalid pos_side",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "invalid",
				EntryPrice:     40000,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "pos_side must be 'long', 'short', or 'net'",
		},
		{
			name: "zero entry_price",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "long",
				EntryPrice:     0,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "entry_price must be positive",
		},
		{
			name: "negative entry_price",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "long",
				EntryPrice:     -1000,
				CurrentSlPrice: 38000,
			},
			wantErr: true,
			errMsg:  "entry_price must be positive",
		},
		{
			name: "zero current_sl_price",
			tracker: &DynamicSLTracker{
				PositionKey:    "BTC-USDT-SWAP_long",
				InstId:         "BTC-USDT-SWAP",
				PosSide:        "long",
				EntryPrice:     40000,
				CurrentSlPrice: 0,
			},
			wantErr: true,
			errMsg:  "current_sl_price must be positive",
		},
		{
			name: "long position: highest price < entry price",
			tracker: &DynamicSLTracker{
				PositionKey:         "BTC-USDT-SWAP_long",
				InstId:              "BTC-USDT-SWAP",
				PosSide:             "long",
				EntryPrice:          40000,
				CurrentSlPrice:      38000,
				HighestPriceReached: 39000, // Less than entry price
			},
			wantErr: true,
			errMsg:  "highest_price_reached",
		},
		{
			name: "short position: lowest price > entry price",
			tracker: &DynamicSLTracker{
				PositionKey:        "ETH-USDT-SWAP_short",
				InstId:             "ETH-USDT-SWAP",
				PosSide:            "short",
				EntryPrice:         3000,
				CurrentSlPrice:     3100,
				LowestPriceReached: 3050, // Greater than entry price
			},
			wantErr: true,
			errMsg:  "lowest_price_reached",
		},
		{
			name: "net position with highest price",
			tracker: &DynamicSLTracker{
				PositionKey:         "BTC-USDT-SWAP_net",
				InstId:              "BTC-USDT-SWAP",
				PosSide:             "net",
				EntryPrice:          40000,
				CurrentSlPrice:      38000,
				HighestPriceReached: 40500,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tracker.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}
