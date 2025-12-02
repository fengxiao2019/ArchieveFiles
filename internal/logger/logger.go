package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"
)

var (
	levelNames = map[LogLevel]string{
		DEBUG:   "DEBUG",
		INFO:    "INFO",
		WARNING: "WARN",
		ERROR:   "ERROR",
		FATAL:   "FATAL",
	}

	levelColors = map[LogLevel]string{
		DEBUG:   colorGray,
		INFO:    colorCyan,
		WARNING: colorYellow,
		ERROR:   colorRed,
		FATAL:   colorPurple,
	}
)

// Logger is a leveled logger with color support
type Logger struct {
	mu          sync.Mutex
	output      io.Writer
	minLevel    LogLevel
	colorOutput bool
	prefix      string
	stdLogger   *log.Logger
}

// Global default logger
var defaultLogger *Logger

func init() {
	defaultLogger = New(os.Stderr, INFO, true)
}

// New creates a new Logger
func New(output io.Writer, minLevel LogLevel, colorOutput bool) *Logger {
	return &Logger{
		output:      output,
		minLevel:    minLevel,
		colorOutput: colorOutput,
		stdLogger:   log.New(output, "", log.LstdFlags),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// SetColorOutput enables or disables color output
func (l *Logger) SetColorOutput(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colorOutput = enabled
}

// SetPrefix sets the logger prefix
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.minLevel {
		return
	}

	message := fmt.Sprintf(format, v...)
	levelName := levelNames[level]

	var output string
	if l.colorOutput {
		color := levelColors[level]
		output = fmt.Sprintf("%s[%s]%s %s", color, levelName, colorReset, message)
	} else {
		output = fmt.Sprintf("[%s] %s", levelName, message)
	}

	if l.prefix != "" {
		output = l.prefix + " " + output
	}

	// Use the standard logger to get timestamp
	l.stdLogger.SetPrefix("")
	_ = l.stdLogger.Output(3, output)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
	l.log(WARNING, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
	os.Exit(1)
}

// Package-level functions using the default logger

// SetLevel sets the minimum log level for the default logger
func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetColorOutput enables or disables color output for the default logger
func SetColorOutput(enabled bool) {
	defaultLogger.SetColorOutput(enabled)
}

// SetPrefix sets the prefix for the default logger
func SetPrefix(prefix string) {
	defaultLogger.SetPrefix(prefix)
}

// Debug logs a debug message using the default logger
func Debug(format string, v ...interface{}) {
	defaultLogger.Debug(format, v...)
}

// Info logs an info message using the default logger
func Info(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

// Warning logs a warning message using the default logger
func Warning(format string, v ...interface{}) {
	defaultLogger.Warning(format, v...)
}

// Error logs an error message using the default logger
func Error(format string, v ...interface{}) {
	defaultLogger.Error(format, v...)
}

// Fatal logs a fatal message and exits using the default logger
func Fatal(format string, v ...interface{}) {
	defaultLogger.Fatal(format, v...)
}

// Printf logs an info message (for compatibility with standard log)
func Printf(format string, v ...interface{}) {
	defaultLogger.Info(format, v...)
}

// Print logs an info message (for compatibility with standard log)
func Print(v ...interface{}) {
	defaultLogger.Info(fmt.Sprint(v...))
}

// Println logs an info message (for compatibility with standard log)
func Println(v ...interface{}) {
	defaultLogger.Info(fmt.Sprint(v...))
}

// Fatalf logs a fatal message and exits (for compatibility with standard log)
func Fatalf(format string, v ...interface{}) {
	defaultLogger.Fatal(format, v...)
}

// GetDefaultLogger returns the default logger instance
func GetDefaultLogger() *Logger {
	return defaultLogger
}
