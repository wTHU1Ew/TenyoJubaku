package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 配置结构 / Configuration structure
type Config struct {
	OKX          OKXConfig          `yaml:"okx"`
	Monitoring   MonitoringConfig   `yaml:"monitoring"`
	Database     DatabaseConfig     `yaml:"database"`
	Logging      LoggingConfig      `yaml:"logging"`
	TPSL         TPSLConfig         `yaml:"tpsl"`
	DynamicSL    DynamicSLConfig    `yaml:"dynamic_sl"`
	OrderControl OrderControlConfig `yaml:"order_control"`
}

// OKXConfig OKX API配置 / OKX API configuration
type OKXConfig struct {
	APIURL      string `yaml:"api_url"`
	APIKey      string `yaml:"api_key"`
	APISecret   string `yaml:"api_secret"`
	Passphrase  string `yaml:"passphrase"`
	Timeout     int    `yaml:"timeout"`
	MaxRetries  int    `yaml:"max_retries"`
	DebugEnable bool   `yaml:"debug_enable"`
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

// TPSLConfig TPSL管理配置 / TPSL management configuration
type TPSLConfig struct {
	Enabled         bool    `yaml:"enabled"`
	CheckInterval   int     `yaml:"check_interval"`
	VolatilityPct   float64 `yaml:"volatility_pct"`
	ProfitLossRatio float64 `yaml:"profit_loss_ratio"`
}

// DynamicSLConfig 动态追踪止损配置 / Dynamic trailing stop-loss configuration (Feature 5)
type DynamicSLConfig struct {
	// Enabled 是否启用动态追踪止损 / Whether to enable dynamic trailing stop-loss
	Enabled bool `yaml:"enabled"`

	// FirstMovePct 首次移动止损阈值百分比 / First move threshold percentage
	// 当盈利达到此百分比时，将止损移动到入场价（保本）
	// When profit reaches this percentage, move stop-loss to entry price (breakeven)
	// 例如 0.01 表示 1% 盈利时触发
	// E.g., 0.01 means trigger at 1% profit
	FirstMovePct float64 `yaml:"first_move_pct"`

	// TrailingStepPct 追踪步长百分比 / Trailing step percentage
	// 价格每上涨此百分比，就计算是否需要调整止损
	// Calculate SL adjustment when price increases by this percentage
	// 例如 0.005 表示价格每上涨 0.5% 检查一次
	// E.g., 0.005 means check every 0.5% price increase
	TrailingStepPct float64 `yaml:"trailing_step_pct"`

	// StopMoveStepPct 止损移动步长百分比 / Stop-loss move step percentage
	// 每次调整止损时，止损价格移动的百分比
	// Percentage to move stop-loss price on each adjustment
	// 例如 0.001 表示每次移动 0.1%
	// E.g., 0.001 means move 0.1% each time
	StopMoveStepPct float64 `yaml:"stop_move_step_pct"`
}

// OrderControlConfig Order Control配置 / Order Control configuration (Feature 3)
type OrderControlConfig struct {
	// Enabled 是否启用订单控制 / Whether to enable order control
	Enabled bool `yaml:"enabled"`

	// FrequencyLimit 订单频率限制配置 / Order frequency limit configuration
	FrequencyLimit FrequencyLimitConfig `yaml:"frequency_limit"`

	// MakerOnly Maker-only订单配置 / Maker-only order configuration
	MakerOnly MakerOnlyConfig `yaml:"maker_only"`

	// Confirmation 订单确认配置 / Order confirmation configuration
	Confirmation ConfirmationConfig `yaml:"confirmation"`
}

// FrequencyLimitConfig 订单频率限制配置 / Order frequency limit configuration
type FrequencyLimitConfig struct {
	// Enabled 是否启用频率限制 / Whether to enable frequency limit
	Enabled bool `yaml:"enabled"`

	// WeeklyMaxOrders 每周最大订单数 / Maximum orders per week
	WeeklyMaxOrders int `yaml:"weekly_max_orders"`

	// ExcludeReduceOnly 是否排除reduce-only订单 / Whether to exclude reduce-only orders
	// 如果为true，reduce-only订单不计入频率限制
	// If true, reduce-only orders do not count towards frequency limit
	ExcludeReduceOnly bool `yaml:"exclude_reduce_only"`
}

// MakerOnlyConfig Maker-only订单配置 / Maker-only order configuration
type MakerOnlyConfig struct {
	// Enabled 是否启用maker-only限制 / Whether to enable maker-only restriction
	Enabled bool `yaml:"enabled"`

	// MinPriceDistancePct 最小价格距离百分比 / Minimum price distance percentage
	// 订单价格与市场价格的最小距离，例如0.001表示0.1%
	// Minimum distance between order price and market price, e.g., 0.001 for 0.1%
	MinPriceDistancePct float64 `yaml:"min_price_distance_pct"`

	// AllowTakerForReduceOnly 是否允许reduce-only订单使用taker / Allow taker for reduce-only orders
	// 如果为true，reduce-only订单可以使用market订单类型
	// If true, reduce-only orders can use market order type
	AllowTakerForReduceOnly bool `yaml:"allow_taker_for_reduce_only"`

	// MaxTakerPct 允许的最大taker订单百分比 / Maximum taker order percentage
	// reduce-only订单中允许的最大taker订单比例
	// Maximum percentage of taker orders allowed for reduce-only orders
	MaxTakerPct float64 `yaml:"max_taker_pct"`

	// TickerStalenessSeconds 行情过期时间（秒）/ Ticker staleness threshold (seconds)
	// 超过此时间的行情数据被视为过期
	// Ticker data older than this is considered stale
	TickerStalenessSeconds int `yaml:"ticker_staleness_seconds"`
}

// ConfirmationConfig 订单确认配置 / Order confirmation configuration
type ConfirmationConfig struct {
	// Enabled 是否启用确认机制 / Whether to enable confirmation mechanism
	Enabled bool `yaml:"enabled"`

	// CheckIntervalSeconds 确认检查间隔（秒）/ Confirmation check interval (seconds)
	// 检查待确认订单的频率
	// Frequency of checking pending confirmations
	CheckIntervalSeconds int `yaml:"check_interval_seconds"`

	// ConfirmationIntervalHours 确认间隔（小时）/ Confirmation interval (hours)
	// 每隔多少小时需要一次确认
	// How often confirmation is required
	ConfirmationIntervalHours int `yaml:"confirmation_interval_hours"`

	// WaitingPeriodHours 等待期（小时）/ Waiting period (hours)
	// 第一次确认请求之前的等待时间
	// Waiting time before first confirmation request
	WaitingPeriodHours int `yaml:"waiting_period_hours"`

	// TimeoutSizeReductionPct 超时后的规模减少百分比 / Size reduction percentage on timeout
	// 例如0.1表示每次超时减少10%的订单规模
	// E.g., 0.1 means reduce order size by 10% on each timeout
	TimeoutSizeReductionPct float64 `yaml:"timeout_size_reduction_pct"`

	// MaxTimeouts 最大超时次数 / Maximum number of timeouts
	// 超过此次数后撤销订单
	// Cancel order after exceeding this count
	MaxTimeouts int `yaml:"max_timeouts"`

	// NotificationMethod 通知方式 / Notification method
	// 可选值: "log", "email", "telegram" 等
	// Options: "log", "email", "telegram", etc.
	NotificationMethod string `yaml:"notification_method"`
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

	// Validate TPSL configuration
	// Set defaults if not specified
	if c.TPSL.CheckInterval <= 0 {
		c.TPSL.CheckInterval = 300 // Default 5 minutes
	}
	if c.TPSL.VolatilityPct == 0 {
		c.TPSL.VolatilityPct = 0.01 // Default 1%
	}
	if c.TPSL.ProfitLossRatio == 0 {
		c.TPSL.ProfitLossRatio = 5.0 // Default 5:1
	}

	// Validate TPSL parameters
	if c.TPSL.VolatilityPct <= 0 || c.TPSL.VolatilityPct > 1.0 {
		return fmt.Errorf("tpsl.volatility_pct must be between 0 and 1, got %f", c.TPSL.VolatilityPct)
	}
	if c.TPSL.ProfitLossRatio <= 0 {
		return fmt.Errorf("tpsl.profit_loss_ratio must be positive, got %f", c.TPSL.ProfitLossRatio)
	}
	if c.TPSL.CheckInterval <= 0 {
		return fmt.Errorf("tpsl.check_interval must be positive, got %d", c.TPSL.CheckInterval)
	}

	// Validate Dynamic SL configuration (only if enabled)
	if c.DynamicSL.Enabled {
		// Set defaults if not specified
		if c.DynamicSL.FirstMovePct == 0 {
			c.DynamicSL.FirstMovePct = 0.01 // Default 1%
		}
		if c.DynamicSL.TrailingStepPct == 0 {
			c.DynamicSL.TrailingStepPct = 0.005 // Default 0.5%
		}
		if c.DynamicSL.StopMoveStepPct == 0 {
			c.DynamicSL.StopMoveStepPct = 0.001 // Default 0.1%
		}

		// Validate Dynamic SL parameters
		if c.DynamicSL.FirstMovePct <= 0 || c.DynamicSL.FirstMovePct > 1.0 {
			return fmt.Errorf("dynamic_sl.first_move_pct must be between 0 and 1, got %f", c.DynamicSL.FirstMovePct)
		}
		if c.DynamicSL.TrailingStepPct <= 0 || c.DynamicSL.TrailingStepPct > 1.0 {
			return fmt.Errorf("dynamic_sl.trailing_step_pct must be between 0 and 1, got %f", c.DynamicSL.TrailingStepPct)
		}
		if c.DynamicSL.StopMoveStepPct <= 0 || c.DynamicSL.StopMoveStepPct > 1.0 {
			return fmt.Errorf("dynamic_sl.stop_move_step_pct must be between 0 and 1, got %f", c.DynamicSL.StopMoveStepPct)
		}

		// Validate logical relationships
		// StopMoveStepPct should be smaller than TrailingStepPct for meaningful trailing
		if c.DynamicSL.StopMoveStepPct > c.DynamicSL.TrailingStepPct {
			return fmt.Errorf("dynamic_sl.stop_move_step_pct (%f) should not be greater than trailing_step_pct (%f)",
				c.DynamicSL.StopMoveStepPct, c.DynamicSL.TrailingStepPct)
		}

		// FirstMovePct should be reasonable (typically larger than trailing step)
		if c.DynamicSL.FirstMovePct < c.DynamicSL.TrailingStepPct {
			return fmt.Errorf("dynamic_sl.first_move_pct (%f) should not be less than trailing_step_pct (%f)",
				c.DynamicSL.FirstMovePct, c.DynamicSL.TrailingStepPct)
		}
	}

	// Validate Order Control configuration (only if enabled)
	if c.OrderControl.Enabled {
		// Validate Frequency Limit
		if c.OrderControl.FrequencyLimit.Enabled {
			if c.OrderControl.FrequencyLimit.WeeklyMaxOrders <= 0 {
				c.OrderControl.FrequencyLimit.WeeklyMaxOrders = 20 // Default
			}
		}

		// Validate Maker-Only
		if c.OrderControl.MakerOnly.Enabled {
			if c.OrderControl.MakerOnly.MinPriceDistancePct <= 0 {
				c.OrderControl.MakerOnly.MinPriceDistancePct = 0.001 // Default 0.1%
			}
			if c.OrderControl.MakerOnly.MinPriceDistancePct > 1.0 {
				return fmt.Errorf("order_control.maker_only.min_price_distance_pct must be <= 1.0, got %f", c.OrderControl.MakerOnly.MinPriceDistancePct)
			}
			if c.OrderControl.MakerOnly.MaxTakerPct < 0 || c.OrderControl.MakerOnly.MaxTakerPct > 1.0 {
				return fmt.Errorf("order_control.maker_only.max_taker_pct must be between 0 and 1, got %f", c.OrderControl.MakerOnly.MaxTakerPct)
			}
			if c.OrderControl.MakerOnly.TickerStalenessSeconds <= 0 {
				c.OrderControl.MakerOnly.TickerStalenessSeconds = 60 // Default 60 seconds
			}
		}

		// Validate Confirmation
		if c.OrderControl.Confirmation.Enabled {
			if c.OrderControl.Confirmation.CheckIntervalSeconds <= 0 {
				c.OrderControl.Confirmation.CheckIntervalSeconds = 3600 // Default 1 hour
			}
			if c.OrderControl.Confirmation.ConfirmationIntervalHours <= 0 {
				c.OrderControl.Confirmation.ConfirmationIntervalHours = 24 // Default 24 hours
			}
			if c.OrderControl.Confirmation.WaitingPeriodHours < 0 {
				c.OrderControl.Confirmation.WaitingPeriodHours = 48 // Default 48 hours
			}
			if c.OrderControl.Confirmation.TimeoutSizeReductionPct <= 0 || c.OrderControl.Confirmation.TimeoutSizeReductionPct > 1.0 {
				return fmt.Errorf("order_control.confirmation.timeout_size_reduction_pct must be between 0 and 1, got %f", c.OrderControl.Confirmation.TimeoutSizeReductionPct)
			}
			if c.OrderControl.Confirmation.MaxTimeouts <= 0 {
				c.OrderControl.Confirmation.MaxTimeouts = 3 // Default 3 timeouts
			}
			if c.OrderControl.Confirmation.NotificationMethod == "" {
				c.OrderControl.Confirmation.NotificationMethod = "log" // Default to log
			}
			// Validate notification method
			validMethods := map[string]bool{"log": true, "email": true, "telegram": true}
			if !validMethods[c.OrderControl.Confirmation.NotificationMethod] {
				return fmt.Errorf("invalid notification method: %s (must be log, email, or telegram)", c.OrderControl.Confirmation.NotificationMethod)
			}
		}
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
