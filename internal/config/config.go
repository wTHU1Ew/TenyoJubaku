package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置结构 / Configuration structure
type Config struct {
	OKX        OKXConfig        `yaml:"okx"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Database   DatabaseConfig   `yaml:"database"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// OKXConfig OKX API配置 / OKX API configuration
type OKXConfig struct {
	APIURL     string `yaml:"api_url"`
	APIKey     string `yaml:"api_key"`
	APISecret  string `yaml:"api_secret"`
	Passphrase string `yaml:"passphrase"`
	Timeout    int    `yaml:"timeout"`
	MaxRetries int    `yaml:"max_retries"`
}

// MonitoringConfig 监控配置 / Monitoring configuration
type MonitoringConfig struct {
	Interval int  `yaml:"interval"`
	Enabled  bool `yaml:"enabled"`
}

// DatabaseConfig 数据库配置 / Database configuration
type DatabaseConfig struct {
	Path         string `yaml:"path"`
	WALMode      bool   `yaml:"wal_mode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// LoggingConfig 日志配置 / Logging configuration
type LoggingConfig struct {
	FilePath   string `yaml:"file_path"`
	Level      string `yaml:"level"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
	Console    bool   `yaml:"console"`
}

// Load 加载配置文件 / Load configuration from file
// 从指定路径加载YAML配置文件，解析并验证配置项
// Load YAML configuration file from specified path, parse and validate all settings
//
// Parameters:
//   - path: Path to the configuration file (e.g., "configs/config.yaml")
//
// Returns:
//   - *Config: 已验证的配置对象 / Validated configuration object with all settings
//   - error: 文件不存在、解析失败或验证失败时返回错误 / Error if file not found, parsing fails, or validation fails
func Load(path string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Check if template exists
		templatePath := "configs/config.template.yaml"
		if _, err := os.Stat(templatePath); err == nil {
			return nil, fmt.Errorf("config file not found at %s. Please copy %s to %s and fill in your credentials", path, templatePath, path)
		}
		return nil, fmt.Errorf("config file not found at %s", path)
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置 / Validate configuration
// 验证所有配置项的有效性，并为未设置的项应用默认值
// Validate all configuration items and apply default values for unset items
//
// Returns:
//   - error: 当必需配置项缺失或无效时返回错误 / Error when required items are missing or invalid
//     例如 / Examples: missing API credentials, invalid log level, etc.
func (c *Config) Validate() error {
	// Validate OKX configuration
	if c.OKX.APIURL == "" {
		return fmt.Errorf("okx.api_url is required")
	}
	if c.OKX.APIKey == "" || strings.Contains(c.OKX.APIKey, "your-api-key") {
		return fmt.Errorf("okx.api_key is required and must not be a placeholder value")
	}
	if c.OKX.APISecret == "" || strings.Contains(c.OKX.APISecret, "your-api-secret") {
		return fmt.Errorf("okx.api_secret is required and must not be a placeholder value")
	}
	if c.OKX.Passphrase == "" || strings.Contains(c.OKX.Passphrase, "your-api-passphrase") {
		return fmt.Errorf("okx.passphrase is required and must not be a placeholder value")
	}
	if c.OKX.Timeout <= 0 {
		c.OKX.Timeout = 30 // Default timeout
	}
	if c.OKX.MaxRetries < 0 {
		c.OKX.MaxRetries = 3 // Default max retries
	}

	// Validate monitoring configuration
	if c.Monitoring.Interval <= 0 {
		c.Monitoring.Interval = 60 // Default 60 seconds
	}

	// Validate database configuration
	if c.Database.Path == "" {
		c.Database.Path = "./data/tenyojubaku.db"
	}
	if c.Database.MaxOpenConns <= 0 {
		c.Database.MaxOpenConns = 1
	}
	if c.Database.MaxIdleConns <= 0 {
		c.Database.MaxIdleConns = 1
	}

	// Validate logging configuration
	if c.Logging.FilePath == "" {
		c.Logging.FilePath = "./logs/app.log"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "INFO"
	}
	// Validate log level
	validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true}
	if !validLevels[strings.ToUpper(c.Logging.Level)] {
		return fmt.Errorf("invalid log level: %s (must be DEBUG, INFO, WARN, or ERROR)", c.Logging.Level)
	}
	c.Logging.Level = strings.ToUpper(c.Logging.Level)

	if c.Logging.MaxSize <= 0 {
		c.Logging.MaxSize = 100
	}
	if c.Logging.MaxAge <= 0 {
		c.Logging.MaxAge = 30
	}
	if c.Logging.MaxBackups < 0 {
		c.Logging.MaxBackups = 10
	}

	return nil
}

// MaskSensitive 屏蔽敏感信息用于日志记录 / Mask sensitive information for logging
// 生成配置的字符串表示，其中敏感数据（API密钥等）被屏蔽
// Generate string representation of configuration with sensitive data (API keys, etc.) masked
//
// Returns:
//   - string: 屏蔽后的配置字符串，敏感字段仅显示前4个字符
//     String with masked configuration, sensitive fields show only first 4 characters
//     例如 / Example: "APIKey=abcd****" instead of full key
func (c *Config) MaskSensitive() string {
	return fmt.Sprintf("Config{OKX{APIURL=%s, APIKey=%s, Timeout=%d}, Monitoring{Interval=%d, Enabled=%t}, Database{Path=%s}, Logging{Level=%s}}",
		c.OKX.APIURL,
		maskString(c.OKX.APIKey),
		c.OKX.Timeout,
		c.Monitoring.Interval,
		c.Monitoring.Enabled,
		c.Database.Path,
		c.Logging.Level,
	)
}

// maskString 屏蔽字符串，只显示前4个字符 / Mask string, show only first 4 characters
// 将敏感字符串转换为安全格式，保留前4个字符用于调试
// Convert sensitive string to safe format, keeping first 4 characters for debugging
//
// Parameters:
//   - s: Original string to be masked
//
// Returns:
//   - string: 屏蔽后的字符串 / Masked string
//     如果长度≤4: 返回"****" / If length ≤ 4: returns "****"
//     如果长度>4: 返回"前4字符****" / If length > 4: returns "first4chars****"
func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}
