package logger

import (
	"io"
	"os"
)

// NewTestLogger creates a logger for testing that discards all output
// 创建一个用于测试的日志记录器，丢弃所有输出
func NewTestLogger() *Logger {
	return &Logger{
		fileWriter: io.Discard,
		level:      DEBUG,
		consoleOut: false,
	}
}

// NewTestLoggerWithOutput creates a logger for testing that writes to stdout
// 创建一个用于测试的日志记录器，输出到标准输出
func NewTestLoggerWithOutput() *Logger {
	return &Logger{
		fileWriter: os.Stdout,
		level:      DEBUG,
		consoleOut: true,
	}
}
