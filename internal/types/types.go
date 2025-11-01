package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archiveFiles/internal/constants"
)

// DatabaseType represents the type of database
type DatabaseType int

const (
	DatabaseTypeRocksDB DatabaseType = iota
	DatabaseTypeSQLite
	DatabaseTypeLogFile
	DatabaseTypeUnknown
)

// String returns the string representation of DatabaseType
func (dt DatabaseType) String() string {
	switch dt {
	case DatabaseTypeRocksDB:
		return "RocksDB"
	case DatabaseTypeSQLite:
		return "SQLite"
	case DatabaseTypeLogFile:
		return "LogFile"
	default:
		return "Unknown"
	}
}

// DatabaseInfo contains information about a discovered database
type DatabaseInfo struct {
	Path       string       // Path to the database
	Type       DatabaseType // Type of database
	Name       string       // Name for backup
	SourceRoot string       // Track which source directory this came from
	Size       int64        // File/directory size for progress tracking
}

// Config holds all configuration options
type Config struct {
	SourcePaths       []string `json:"source_paths"` // Support multiple source directories
	BackupPath        string   `json:"backup_path"`
	ArchivePath       string   `json:"archive_path"`
	Method            string   `json:"method"` // backup, checkpoint, copy
	Compress          bool     `json:"compress"`
	RemoveBackup      bool     `json:"remove_backup"`
	BatchMode         bool     `json:"batch_mode"`         // Process directory vs single database
	IncludePattern    string   `json:"include_pattern"`    // File pattern to include
	ExcludePattern    string   `json:"exclude_pattern"`    // File pattern to exclude
	ShowProgress      bool     `json:"show_progress"`      // Show progress bar
	Filter            string   `json:"filter"`             // Filter pattern for source paths
	CompressionFormat string   `json:"compression_format"` // Compression format for archived files
	Verify            bool     `json:"verify"`             // Verify backup data against source
	Workers           int      `json:"workers"`            // Number of concurrent backup workers (0 = auto)
	Strict            bool     `json:"strict"`             // Strict mode: fail on any error instead of continuing
	DryRun            bool     `json:"dry_run"`            // Dry run mode: simulate actions without executing them
	LogLevel          string   `json:"log_level"`          // Log level: debug, info, warning, error (default: info)
	ColorLog          bool     `json:"color_log"`          // Enable colored log output (default: true)
}

// DatabaseLockInfo contains information about database locks
type DatabaseLockInfo struct {
	IsLocked    bool
	ProcessInfo string
	LockType    string
}

// BackupProgress represents backup progress information
type BackupProgress struct {
	CurrentFile    string
	ProcessedItems int
	TotalItems     int
	ProcessedSize  int64
	TotalSize      int64
	StartTime      time.Time
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	// Validate source paths
	if len(c.SourcePaths) == 0 {
		return fmt.Errorf("no source paths specified")
	}

	for _, sourcePath := range c.SourcePaths {
		if sourcePath == "" {
			return fmt.Errorf("empty source path not allowed")
		}

		// Security: Check for path traversal attempts
		if err := validatePathSecurity(sourcePath); err != nil {
			return fmt.Errorf("invalid source path %s: %v", sourcePath, err)
		}

		// Check if path exists
		if _, err := os.Stat(sourcePath); err != nil {
			return fmt.Errorf("source path does not exist: %s", sourcePath)
		}
	}

	// Validate backup path
	if c.BackupPath != "" {
		if err := validatePathSecurity(c.BackupPath); err != nil {
			return fmt.Errorf("invalid backup path: %v", err)
		}
	}

	// Validate archive path
	if c.ArchivePath != "" {
		if err := validatePathSecurity(c.ArchivePath); err != nil {
			return fmt.Errorf("invalid archive path: %v", err)
		}
	}

	// Validate backup method
	validMethods := []string{
		constants.MethodCheckpoint,
		constants.MethodBackup,
		constants.MethodCopy,
		constants.MethodCopyFiles,
	}
	if !contains(validMethods, c.Method) {
		return fmt.Errorf("invalid backup method: %s (valid: %s)", c.Method, strings.Join(validMethods, ", "))
	}

	// Validate compression format
	validFormats := []string{"gzip", "zstd", "lz4"}
	if c.Compress && !contains(validFormats, c.CompressionFormat) {
		return fmt.Errorf("invalid compression format: %s (valid: %s)", c.CompressionFormat, strings.Join(validFormats, ", "))
	}

	// Validate workers
	if c.Workers < 0 {
		return fmt.Errorf("workers must be >= 0 (got %d)", c.Workers)
	}
	if c.Workers > 256 {
		return fmt.Errorf("workers must be <= 256 (got %d, unreasonably high)", c.Workers)
	}

	// Validate log level
	if c.LogLevel != "" {
		validLevels := []string{"debug", "info", "warning", "error"}
		if !contains(validLevels, strings.ToLower(c.LogLevel)) {
			return fmt.Errorf("invalid log level: %s (valid: %s)", c.LogLevel, strings.Join(validLevels, ", "))
		}
	}

	return nil
}

// validatePathSecurity checks for path traversal and other security issues
func validatePathSecurity(path string) error {
	if path == "" {
		return fmt.Errorf("empty path not allowed")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null bytes")
	}

	// Clean the path first
	cleaned := filepath.Clean(path)

	// Check for excessive path traversal (suspicious pattern)
	// Multiple ".." components are allowed but excessive ones are suspicious
	components := strings.Split(cleaned, string(filepath.Separator))
	dotDotCount := 0
	for _, comp := range components {
		if comp == ".." {
			dotDotCount++
			if dotDotCount > 3 {
				return fmt.Errorf("excessive path traversal detected (too many '..' components)")
			}
		}
	}

	// Check for absolute paths trying to access system directories
	// This is a warning, not an error, as absolute paths might be legitimate
	dangerousRoots := []string{"/etc", "/bin", "/sbin", "/usr/bin", "/usr/sbin", "/boot", "/sys", "/proc"}
	absPath, err := filepath.Abs(cleaned)
	if err == nil {
		for _, dangerous := range dangerousRoots {
			if strings.HasPrefix(absPath, dangerous) {
				return fmt.Errorf("accessing system directory %s is not allowed", dangerous)
			}
		}
	}

	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
