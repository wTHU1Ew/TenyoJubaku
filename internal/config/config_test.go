package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temporary directory for test configs
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		configData  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			configData: `
okx:
  api_url: "https://www.okx.com"
  api_key: "test-api-key-123"
  api_secret: "test-secret-456"
  passphrase: "test-passphrase"
  timeout: 30
  max_retries: 3
monitoring:
  interval: 60
  enabled: true
database:
  path: "./data/test.db"
  wal_mode: true
  max_open_conns: 1
  max_idle_conns: 1
logging:
  file_path: "./logs/test.log"
  level: "INFO"
  max_size: 100
  max_age: 30
  max_backups: 10
  compress: true
  console: true
`,
			expectError: false,
		},
		{
			name: "missing api_key",
			configData: `
okx:
  api_url: "https://www.okx.com"
  api_secret: "test-secret"
  passphrase: "test-passphrase"
`,
			expectError: true,
			errorMsg:    "api_key is required",
		},
		{
			name: "placeholder api_key",
			configData: `
okx:
  api_url: "https://www.okx.com"
  api_key: "your-api-key-here"
  api_secret: "test-secret"
  passphrase: "test-passphrase"
`,
			expectError: true,
			errorMsg:    "must not be a placeholder",
		},
		{
			name: "invalid log level",
			configData: `
okx:
  api_url: "https://www.okx.com"
  api_key: "test-key"
  api_secret: "test-secret"
  passphrase: "test-passphrase"
logging:
  level: "INVALID"
`,
			expectError: true,
			errorMsg:    "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config to temporary file
			configPath := filepath.Join(tmpDir, "config_"+tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.configData), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Load configuration
			cfg, err := Load(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("expected config but got nil")
				}
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
					Timeout:    30,
					MaxRetries: 3,
				},
				TPSL: TPSLConfig{
					Enabled:         true,
					CheckInterval:   300,
					VolatilityPct:   0.01,
					ProfitLossRatio: 5.0,
				},
			},
			expectError: false,
		},
		{
			name: "missing api_url",
			config: Config{
				OKX: OKXConfig{
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
				},
			},
			expectError: true,
			errorMsg:    "api_url is required",
		},
		{
			name: "defaults applied",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
					// Timeout and MaxRetries not set
				},
			},
			expectError: false,
		},
		{
			name: "invalid volatility_pct too high",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
				},
				TPSL: TPSLConfig{
					VolatilityPct: 1.5, // > 1.0
				},
			},
			expectError: true,
			errorMsg:    "volatility_pct must be between 0 and 1",
		},
		{
			name: "invalid volatility_pct negative",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
				},
				TPSL: TPSLConfig{
					VolatilityPct: -0.01,
				},
			},
			expectError: true,
			errorMsg:    "volatility_pct must be between 0 and 1",
		},
		{
			name: "invalid profit_loss_ratio negative",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
				},
				TPSL: TPSLConfig{
					ProfitLossRatio: -5.0,
				},
			},
			expectError: true,
			errorMsg:    "profit_loss_ratio must be positive",
		},
		{
			name: "TPSL defaults applied",
			config: Config{
				OKX: OKXConfig{
					APIURL:     "https://www.okx.com",
					APIKey:     "valid-key",
					APISecret:  "valid-secret",
					Passphrase: "valid-passphrase",
				},
				// TPSL not set - should use defaults
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Check defaults were applied
				if tt.config.OKX.Timeout == 0 && tt.config.OKX.Timeout != 30 {
					t.Error("default timeout not applied")
				}
			}
		})
	}
}

func TestMaskSensitive(t *testing.T) {
	cfg := Config{
		OKX: OKXConfig{
			APIURL:     "https://www.okx.com",
			APIKey:     "abcd1234567890",
			APISecret:  "secret123",
			Passphrase: "pass123",
			Timeout:    30,
		},
		Monitoring: MonitoringConfig{
			Interval: 60,
			Enabled:  true,
		},
		Database: DatabaseConfig{
			Path: "./data/test.db",
		},
		Logging: LoggingConfig{
			Level: "INFO",
		},
	}

	masked := cfg.MaskSensitive()

	// Check that API key is masked
	if !contains(masked, "abcd****") {
		t.Errorf("API key not properly masked: %s", masked)
	}

	// Check that full API key is not present
	if contains(masked, "abcd1234567890") {
		t.Errorf("Full API key should not be present in masked output: %s", masked)
	}

	// Check that other fields are present
	if !contains(masked, "https://www.okx.com") {
		t.Errorf("API URL should be present: %s", masked)
	}
	if !contains(masked, "INFO") {
		t.Errorf("Log level should be present: %s", masked)
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcd1234", "abcd****"},
		{"abc", "****"},
		{"", "****"},
		{"a", "****"},
		{"12345678", "1234****"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskString(tt.input)
			if result != tt.expected {
				t.Errorf("maskString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
