package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath, INFO, 10, 7, 3, false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Error("expected logger instance, got nil")
	}

	// Verify log directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("log directory was not created")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input       string
		expected    Level
		expectError bool
	}{
		{"DEBUG", DEBUG, false},
		{"INFO", INFO, false},
		{"WARN", WARN, false},
		{"ERROR", ERROR, false},
		{"debug", DEBUG, false},
		{"info", INFO, false},
		{"INVALID", INFO, true},
		{"", INFO, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLevel(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if level != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, level)
				}
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Create logger with INFO level
	logger, err := New(logPath, INFO, 10, 7, 3, false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write logs at different levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// Give time for writes to complete
	time.Sleep(100 * time.Millisecond)

	// Read log file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)

	// DEBUG should not be logged (level is INFO)
	if strings.Contains(contentStr, "debug message") {
		t.Error("DEBUG message should not be logged when level is INFO")
	}

	// INFO, WARN, ERROR should be logged
	if !strings.Contains(contentStr, "info message") {
		t.Error("INFO message should be logged")
	}
	if !strings.Contains(contentStr, "warn message") {
		t.Error("WARN message should be logged")
	}
	if !strings.Contains(contentStr, "error message") {
		t.Error("ERROR message should be logged")
	}

	// Check log format
	if !strings.Contains(contentStr, "[INFO]") {
		t.Error("log should contain [INFO] level indicator")
	}
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mask api_key",
			input:    "api_key=abcd1234567890",
			expected: "api_key=abcd****",
		},
		{
			name:     "mask api_secret",
			input:    "api_secret:secret123456",
			expected: "api_secret:secr****",
		},
		{
			name:     "mask passphrase",
			input:    "passphrase: mypassphrase123",
			expected: "passphrase: mypa****",
		},
		{
			name:     "mask short value",
			input:    "api_key=abc",
			expected: "api_key=****",
		},
		{
			name:     "multiple sensitive fields",
			input:    "api_key=key123 api_secret=secret456",
			expected: "api_key=key1**** api_secret=secr****",
		},
		{
			name:     "case insensitive",
			input:    "API_KEY=key123",
			expected: "API_KEY=key1****",
		},
		{
			name:     "no sensitive data",
			input:    "normal log message",
			expected: "normal log message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcd1234", "abcd****"},
		{"abc", "****"},
		{"", "****"},
		{"12345678", "1234****"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskValue(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLogFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath, DEBUG, 10, 7, 3, false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write formatted log
	logger.Info("User %s logged in at %d", "testuser", 12345)

	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Check formatting worked
	if !strings.Contains(contentStr, "User testuser logged in at 12345") {
		t.Errorf("log formatting failed: %s", contentStr)
	}

	// Check timestamp format
	if !strings.Contains(contentStr, "[") || !strings.Contains(contentStr, "]") {
		t.Error("log should contain timestamp in brackets")
	}
}

func TestConnectionStatusLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(logPath, INFO, 10, 7, 3, false, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	// Log connection success
	logger.Info("OKX API connection successful")

	// Log connection failure
	logger.Error("OKX API connection failed: timeout")

	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "OKX API connection successful") {
		t.Error("connection success should be logged")
	}

	if !strings.Contains(contentStr, "[INFO]") {
		t.Error("connection success should be INFO level")
	}

	if !strings.Contains(contentStr, "connection failed") {
		t.Error("connection failure should be logged")
	}

	if !strings.Contains(contentStr, "[ERROR]") {
		t.Error("connection failure should be ERROR level")
	}
}
