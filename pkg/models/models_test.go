package models

import (
	"testing"
	"time"
)

func TestAccountBalanceValidate(t *testing.T) {
	tests := []struct {
		name        string
		balance     AccountBalance
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid balance",
			balance: AccountBalance{
				Currency:  "USDT",
				Balance:   1000.0,
				Available: 800.0,
				Frozen:    200.0,
				Equity:    1000.0,
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing currency",
			balance: AccountBalance{
				Balance:   1000.0,
				Available: 800.0,
				Frozen:    200.0,
			},
			expectError: true,
			errorMsg:    "currency is required",
		},
		{
			name: "negative balance",
			balance: AccountBalance{
				Currency:  "USDT",
				Balance:   -100.0,
				Available: 0,
				Frozen:    0,
			},
			expectError: true,
			errorMsg:    "balance cannot be negative",
		},
		{
			name: "negative available",
			balance: AccountBalance{
				Currency:  "USDT",
				Balance:   1000.0,
				Available: -100.0,
				Frozen:    0,
			},
			expectError: true,
			errorMsg:    "available cannot be negative",
		},
		{
			name: "negative frozen",
			balance: AccountBalance{
				Currency:  "USDT",
				Balance:   1000.0,
				Available: 1000.0,
				Frozen:    -100.0,
			},
			expectError: true,
			errorMsg:    "frozen cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.balance.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAccountBalanceString(t *testing.T) {
	ab := AccountBalance{
		Currency:  "USDT",
		Balance:   1000.12345678,
		Available: 800.0,
		Frozen:    200.0,
		Equity:    1000.0,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	str := ab.String()

	if str == "" {
		t.Error("String() should not be empty")
	}

	// Check that it contains key information
	if !contains(str, "USDT") {
		t.Error("String should contain currency")
	}
	if !contains(str, "1000.12345678") {
		t.Error("String should contain balance with precision")
	}
}

func TestPositionValidate(t *testing.T) {
	tests := []struct {
		name        string
		position    Position
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid long position",
			position: Position{
				Instrument:    "BTC-USDT",
				PositionSide:  "long",
				PositionSize:  1.5,
				AveragePrice:  50000.0,
				UnrealizedPnL: 100.0,
				Margin:        1000.0,
				Leverage:      5.0,
				Timestamp:     time.Now(),
			},
			expectError: false,
		},
		{
			name: "valid short position",
			position: Position{
				Instrument:    "ETH-USDT",
				PositionSide:  "short",
				PositionSize:  2.0,
				AveragePrice:  3000.0,
				UnrealizedPnL: -50.0,
				Margin:        500.0,
				Leverage:      3.0,
				Timestamp:     time.Now(),
			},
			expectError: false,
		},
		{
			name: "missing instrument",
			position: Position{
				PositionSide:  "long",
				PositionSize:  1.0,
				AveragePrice:  50000.0,
			},
			expectError: true,
			errorMsg:    "instrument is required",
		},
		{
			name: "missing position_side",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSize: 1.0,
			},
			expectError: true,
			errorMsg:    "position_side is required",
		},
		{
			name: "invalid position_side",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSide: "invalid",
				PositionSize: 1.0,
			},
			expectError: true,
			errorMsg:    "position_side must be 'long', 'short', or 'net'",
		},
		{
			name: "negative position_size",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSide: "long",
				PositionSize: -1.0,
			},
			expectError: true,
			errorMsg:    "position_size cannot be negative",
		},
		{
			name: "negative average_price",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSide: "long",
				PositionSize: 1.0,
				AveragePrice: -100.0,
			},
			expectError: true,
			errorMsg:    "average_price cannot be negative",
		},
		{
			name: "negative margin",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSide: "long",
				PositionSize: 1.0,
				AveragePrice: 50000.0,
				Margin:       -100.0,
			},
			expectError: true,
			errorMsg:    "margin cannot be negative",
		},
		{
			name: "negative leverage",
			position: Position{
				Instrument:   "BTC-USDT",
				PositionSide: "long",
				PositionSize: 1.0,
				AveragePrice: 50000.0,
				Margin:       1000.0,
				Leverage:     -5.0,
			},
			expectError: true,
			errorMsg:    "leverage cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.position.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPositionString(t *testing.T) {
	p := Position{
		Instrument:    "BTC-USDT",
		PositionSide:  "long",
		PositionSize:  1.5,
		AveragePrice:  50000.0,
		UnrealizedPnL: 100.5,
		Margin:        1000.0,
		Leverage:      5.0,
		Timestamp:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	str := p.String()

	if str == "" {
		t.Error("String() should not be empty")
	}

	// Check that it contains key information
	if !contains(str, "BTC-USDT") {
		t.Error("String should contain instrument")
	}
	if !contains(str, "long") {
		t.Error("String should contain position side")
	}
	if !contains(str, "1.50000000") {
		t.Error("String should contain position size with precision")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
