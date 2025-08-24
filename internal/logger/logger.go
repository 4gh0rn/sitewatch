package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger is a structured logger wrapper
type Logger struct {
	*slog.Logger
}

// LogLevel represents log levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// LogFormat represents log output formats
type LogFormat string

const (
	FormatText LogFormat = "text"
	FormatJSON LogFormat = "json"
)

// Config holds logger configuration
type Config struct {
	Level  LogLevel
	Format LogFormat
	Output io.Writer
}

// Default logger instance
var defaultLogger *Logger

// NewLogger creates a new structured logger
func NewLogger(config Config) *Logger {
	// Set default output to stdout
	if config.Output == nil {
		config.Output = os.Stdout
	}

	// Parse log level
	level := parseLogLevel(config.Level)
	
	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: level == slog.LevelDebug, // Add source info for debug level
	}
	
	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(config.Output, opts)
	default:
		handler = slog.NewTextHandler(config.Output, opts)
	}

	logger := slog.New(handler)
	return &Logger{Logger: logger}
}

// InitDefault initializes the default logger from environment variables
func InitDefault() {
	config := Config{
		Level:  getLevelFromEnv(),
		Format: getFormatFromEnv(),
		Output: os.Stdout,
	}
	
	defaultLogger = NewLogger(config)
	
	// Replace standard log output with structured logger
	slog.SetDefault(defaultLogger.Logger)
}

// Default returns the default logger instance
func Default() *Logger {
	if defaultLogger == nil {
		InitDefault()
	}
	return defaultLogger
}

// Component-specific loggers
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With("component", component),
	}
}

// Request-specific logger
func (l *Logger) WithRequest(method, path string) *Logger {
	return &Logger{
		Logger: l.With(
			"method", method,
			"path", path,
		),
	}
}

// Site-specific logger
func (l *Logger) WithSite(siteID, siteName string) *Logger {
	return &Logger{
		Logger: l.With(
			"site_id", siteID,
			"site_name", siteName,
		),
	}
}

// Ping-specific logger
func (l *Logger) WithPing(siteID, ip, lineType string) *Logger {
	return &Logger{
		Logger: l.With(
			"site_id", siteID,
			"ip", ip,
			"line_type", lineType,
		),
	}
}

// Auth-specific logger
func (l *Logger) WithAuth(tokenName, authType string) *Logger {
	return &Logger{
		Logger: l.With(
			"token_name", tokenName,
			"auth_type", authType,
		),
	}
}

// Convenience methods for different log levels
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// Helper functions
func parseLogLevel(level LogLevel) slog.Level {
	switch strings.ToLower(string(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getLevelFromEnv() LogLevel {
	if level := os.Getenv("SITEWATCH_LOG_LEVEL"); level != "" {
		return LogLevel(strings.ToLower(level))
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		return LogLevel(strings.ToLower(level))
	}
	return LevelInfo
}

func getFormatFromEnv() LogFormat {
	if format := os.Getenv("SITEWATCH_LOG_FORMAT"); format != "" {
		return LogFormat(strings.ToLower(format))
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		return LogFormat(strings.ToLower(format))
	}
	return FormatText
}