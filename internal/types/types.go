package types

import "time"

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
