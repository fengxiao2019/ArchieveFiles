package constants

// File and directory permissions
const (
	DirPermission  = 0755 // Standard directory permission
	FilePermission = 0644 // Standard file permission
)

// RocksDB backup constants
const (
	RocksDBWriteBatchSize       = 1000 // Number of records per write batch
	RocksDBProgressUpdateInterval = 5000 // Update progress every N records
)

// Progress display constants
const (
	ProgressBarWidth          = 40  // Width of progress bar in characters
	ProgressFileNameMaxLength = 30  // Maximum length of displayed file name
	DefaultProgressEnabled    = true // Show progress by default
)

// Worker pool constants
const (
	DefaultWorkersAuto = 0 // 0 means auto-detect based on CPU cores
)

// Compression constants
const (
	DefaultCompressionFormat = "gzip"
	CompressionBufferSize    = 32 * 1024 // 32KB buffer for compression
)

// Backup method constants
const (
	MethodCheckpoint = "checkpoint" // Recommended method
	MethodBackup     = "backup"     // BackupEngine method
	MethodCopy       = "copy"       // Record-by-record copy
	MethodCopyFiles  = "copy-files" // File-level copy
)

// Default paths and patterns
const (
	DefaultBackupPathFormat  = "backup_%d"                // Using Unix timestamp
	DefaultArchivePathFormat = "%s.tar.gz"                // Archive format
	DefaultIncludePattern    = "*.db,*.sqlite,*.sqlite3,*.log"
	DefaultExcludePattern    = "*temp*,*cache*,*.tmp"
)

// Database detection constants
const (
	MinRocksDBFilesRequired = 2 // Minimum RocksDB marker files needed
	SQLiteHeaderSize        = 16 // Size of SQLite header to read
)
