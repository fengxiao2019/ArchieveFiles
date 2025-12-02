package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  LogLevel
		logLevel  LogLevel
		message   string
		shouldLog bool
	}{
		{"Debug logged at DEBUG level", DEBUG, DEBUG, "debug message", true},
		{"Info logged at DEBUG level", DEBUG, INFO, "info message", true},
		{"Debug not logged at INFO level", INFO, DEBUG, "debug message", false},
		{"Info logged at INFO level", INFO, INFO, "info message", true},
		{"Warning logged at INFO level", INFO, WARNING, "warning message", true},
		{"Error logged at WARNING level", WARNING, ERROR, "error message", true},
		{"Warning not logged at ERROR level", ERROR, WARNING, "warning message", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, tt.minLevel, false)

			switch tt.logLevel {
			case DEBUG:
				logger.Debug(tt.message)
			case INFO:
				logger.Info(tt.message)
			case WARNING:
				logger.Warning(tt.message)
			case ERROR:
				logger.Error(tt.message)
			}

			output := buf.String()
			if tt.shouldLog {
				if !strings.Contains(output, tt.message) {
					t.Errorf("Expected log to contain %q, got %q", tt.message, output)
				}
				if !strings.Contains(output, levelNames[tt.logLevel]) {
					t.Errorf("Expected log to contain level %q, got %q", levelNames[tt.logLevel], output)
				}
			} else {
				if output != "" {
					t.Errorf("Expected no log output, got %q", output)
				}
			}
		})
	}
}

func TestColorOutput(t *testing.T) {
	tests := []struct {
		name        string
		colorOutput bool
		level       LogLevel
		message     string
		expectColor bool
	}{
		{"Color enabled for INFO", true, INFO, "test message", true},
		{"Color disabled for INFO", false, INFO, "test message", false},
		{"Color enabled for ERROR", true, ERROR, "error message", true},
		{"Color disabled for ERROR", false, ERROR, "error message", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, DEBUG, tt.colorOutput)

			switch tt.level {
			case INFO:
				logger.Info(tt.message)
			case ERROR:
				logger.Error(tt.message)
			}

			output := buf.String()
			hasColor := strings.Contains(output, "\033[")

			if tt.expectColor && !hasColor {
				t.Errorf("Expected color codes in output, got %q", output)
			}
			if !tt.expectColor && hasColor {
				t.Errorf("Expected no color codes in output, got %q", output)
			}

			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected output to contain %q, got %q", tt.message, output)
			}
		})
	}
}

func TestLoggerPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO, false)
	logger.SetPrefix("[TEST]")

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[TEST]") {
		t.Errorf("Expected output to contain prefix [TEST], got %q", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain message, got %q", output)
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO, false)

	// Should not log debug at INFO level
	logger.Debug("debug message 1")
	if buf.String() != "" {
		t.Errorf("Expected no output at INFO level for debug message")
	}

	// Change to DEBUG level
	logger.SetLevel(DEBUG)
	logger.Debug("debug message 2")
	if !strings.Contains(buf.String(), "debug message 2") {
		t.Errorf("Expected debug message after changing level to DEBUG")
	}

	// Change to ERROR level
	buf.Reset()
	logger.SetLevel(ERROR)
	logger.Info("info message")
	if buf.String() != "" {
		t.Errorf("Expected no output at ERROR level for info message")
	}

	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("Expected error message at ERROR level")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Test that package-level functions work
	var buf bytes.Buffer
	defaultLogger = New(&buf, INFO, false)

	Info("test info message")
	output := buf.String()
	if !strings.Contains(output, "test info message") {
		t.Errorf("Expected output to contain message, got %q", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain INFO level, got %q", output)
	}
}

func TestPrintfCompatibility(t *testing.T) {
	var buf bytes.Buffer
	defaultLogger = New(&buf, INFO, false)

	Printf("test %s %d", "message", 123)
	output := buf.String()
	if !strings.Contains(output, "test message 123") {
		t.Errorf("Expected formatted message, got %q", output)
	}
}

func TestLevelNames(t *testing.T) {
	expected := map[LogLevel]string{
		DEBUG:   "DEBUG",
		INFO:    "INFO",
		WARNING: "WARN",
		ERROR:   "ERROR",
		FATAL:   "FATAL",
	}

	for level, expectedName := range expected {
		if levelNames[level] != expectedName {
			t.Errorf("Expected level name %q for level %d, got %q", expectedName, level, levelNames[level])
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, INFO, false)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("Message from goroutine %d", id)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	// Should have 10 log messages
	count := strings.Count(output, "[INFO]")
	if count != 10 {
		t.Errorf("Expected 10 log messages, got %d", count)
	}
}
