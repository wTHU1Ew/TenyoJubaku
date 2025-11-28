package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Level 日志级别 / Log level
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String converts log level to string
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel 解析日志级别 / Parse log level from string
func ParseLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	default:
		return INFO, fmt.Errorf("invalid log level: %s", s)
	}
}

// Logger 日志记录器 / Logger instance
type Logger struct {
	level      Level
	fileWriter io.Writer
	consoleOut bool
}

// New 创建新的日志记录器 / Create new logger instance
// 初始化日志记录器，配置文件输出、日志级别和轮转策略
// Initialize logger with file output, log level, and rotation strategy
//
// Parameters:
//   - filePath: Log file path (e.g., "./logs/app.log"), directory will be created if not exists
//   - level: Minimum log level (DEBUG, INFO, WARN, ERROR), logs below this level won't be recorded
//   - maxSize: Maximum size of single log file in MB, auto-rotate when exceeded
//   - maxAge: Days to retain log files
//   - maxBackups: Number of old log files to keep
//   - compress: Whether to compress rotated log files
//   - console: Whether to output to console as well
//
// Returns:
//   - *Logger: 已配置的日志记录器实例 / Configured logger instance
//   - error: 日志目录创建失败时返回错误 / Error on log directory creation failure
func New(filePath string, level Level, maxSize, maxAge, maxBackups int, compress, console bool) (*Logger, error) {
	// Ensure log directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create file writer with rotation
	fileWriter := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSize,    // megabytes
		MaxAge:     maxAge,     // days
		MaxBackups: maxBackups, // number of backups
		Compress:   compress,   // compress rotated files
		LocalTime:  true,       // use local time for filenames
	}

	return &Logger{
		level:      level,
		fileWriter: fileWriter,
		consoleOut: console,
	}, nil
}

// log 写入日志 / Write log entry
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)

	// Mask sensitive data
	message = maskSensitiveData(message)

	logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level.String(), message)

	// Write to file
	if l.fileWriter != nil {
		l.fileWriter.Write([]byte(logEntry))
	}

	// Write to console if enabled
	if l.consoleOut {
		fmt.Print(logEntry)
	}
}

// Debug 调试日志 / Debug log
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 信息日志 / Info log
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 警告日志 / Warning log
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 错误日志 / Error log
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Close 关闭日志记录器 / Close logger and flush buffers
func (l *Logger) Close() error {
	if closer, ok := l.fileWriter.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// maskSensitiveData 屏蔽敏感数据（复杂字符串处理算法）/ Mask sensitive data in log messages (complex string processing)
// 自动检测并屏蔽日志消息中的敏感数据（API密钥、密码等）
// Automatically detect and mask sensitive data in log messages (API keys, passwords, etc.)
//
// 算法详解 / Algorithm Details:
// 1. 定义敏感关键词列表（api_key, secret, passphrase等）
//    Define sensitive keyword list (api_key, secret, passphrase, etc.)
// 2. 对每个关键词，查找"key=value"或"key: value"模式
//    For each keyword, find "key=value" or "key: value" patterns
// 3. 定位值的起始和结束位置（以空格、逗号、换行符为分隔）
//    Locate value start and end positions (delimited by space, comma, newline)
// 4. 用maskValue函数屏蔽值（保留前4个字符）
//    Mask value using maskValue function (keep first 4 characters)
// 5. 替换原始消息中的值 / Replace value in original message
//
// Parameters:
//   - message: Original log message
//
// Returns:
//   - string: 屏蔽敏感数据后的消息 / Message with sensitive data masked
//     例如: "api_key=abcd1234567890" → "api_key=abcd****"
//     Example: "api_key=abcd1234567890" → "api_key=abcd****"
func maskSensitiveData(message string) string {
	// List of sensitive keywords to mask
	sensitiveKeys := []string{
		"api_key", "apikey", "api-key",
		"api_secret", "apisecret", "api-secret", "secret",
		"passphrase", "password", "pwd",
		"token", "auth",
		"OK-ACCESS-KEY", "OK-ACCESS-SECRET", "OK-ACCESS-SIGN", "OK-ACCESS-PASSPHRASE",
	}

	result := message
	for _, key := range sensitiveKeys {
		// Pattern: key=value or key:value or key: value
		// Replace with key=****
		patterns := []string{
			key + "=",
			key + ":",
			key + ": ",
		}

		for _, pattern := range patterns {
			if idx := strings.Index(strings.ToLower(result), strings.ToLower(pattern)); idx != -1 {
				// Find the start of the value
				valueStart := idx + len(pattern)
				if valueStart < len(result) {
					// Find the end of the value (space, comma, newline, or end of string)
					valueEnd := valueStart
					for valueEnd < len(result) && result[valueEnd] != ' ' && result[valueEnd] != ',' && result[valueEnd] != '\n' && result[valueEnd] != '"' && result[valueEnd] != '}' {
						valueEnd++
					}

					if valueEnd > valueStart {
						// Mask the value, showing only first 4 characters
						value := result[valueStart:valueEnd]
						masked := maskValue(value)
						result = result[:valueStart] + masked + result[valueEnd:]
					}
				}
			}
		}
	}

	return result
}

// maskValue 屏蔽值 / Mask a value, showing only first 4 characters
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:4] + "****"
}
