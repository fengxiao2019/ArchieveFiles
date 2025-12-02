package main

import (
	"archiveFiles/internal/logger"
)

func main() {
	// Set to DEBUG level to show all log levels
	logger.SetLevel(logger.DEBUG)
	logger.SetColorOutput(true)

	// Demonstrate all log levels with colors
	logger.Info("=== Log Level Demo ===")
	logger.Info("This demo shows all available log levels with their colors")
	logger.Info("")

	logger.Debug("This is a DEBUG message (gray color) - for detailed debugging information")
	logger.Info("This is an INFO message (cyan color) - for general information")
	logger.Warning("This is a WARNING message (yellow color) - for warning conditions")
	logger.Error("This is an ERROR message (red color) - for error conditions")

	logger.Info("")
	logger.Info("=== Testing Different Scenarios ===")
	logger.Debug("Initializing database connection...")
	logger.Info("Database connection established successfully")
	logger.Warning("Database connection pool is running low")
	logger.Error("Failed to execute query: table not found")

	logger.Info("")
	logger.Info("=== Formatted Logging ===")
	logger.Debug("Processing %d items from %s", 100, "source_directory")
	logger.Info("Backup completed: %d files processed, total size: %s", 42, "1.5 GB")
	logger.Warning("Disk space remaining: only %d%% available", 15)
	logger.Error("Backup failed for %s: %v", "database.db", "permission denied")

	logger.Info("")
	logger.Info("=== Demo Complete ===")
	logger.Info("To disable colors, use: -color-log=false")
	logger.Info("To change log level, use: -log-level=<debug|info|warning|error>")
}
