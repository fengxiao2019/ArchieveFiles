package discovery

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"archiveFiles/internal/types"
	"archiveFiles/internal/utils"

	_ "github.com/mattn/go-sqlite3"
)

// DiscoverDatabases discovers databases in the source path
func DiscoverDatabases(config *types.Config, sourcePath string) ([]types.DatabaseInfo, error) {
	var databases []types.DatabaseInfo

	// Check if source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	// Check if it's a single file or directory
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source path: %v", err)
	}

	if !info.IsDir() {
		// Single file mode
		dbType := DetectDatabaseType(sourcePath)
		if dbType == types.DatabaseTypeUnknown {
			return nil, fmt.Errorf("unknown file type: %s", sourcePath)
		}

		databases = append(databases, types.DatabaseInfo{
			Path: sourcePath,
			Type: dbType,
			Name: filepath.Base(sourcePath),
		})

		return databases, nil
	}

	// Check if the directory itself is a database (like RocksDB)
	dbType := DetectDatabaseType(sourcePath)
	if dbType != types.DatabaseTypeUnknown {
		// The entire directory is a database, treat it as a single unit
		databases = append(databases, types.DatabaseInfo{
			Path: sourcePath,
			Type: dbType,
			Name: filepath.Base(sourcePath),
		})

		return databases, nil
	}

	// Directory mode - scan directory for multiple databases
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Detect database/file type (works for both files and directories)
		dbType := DetectDatabaseType(path)
		if dbType == types.DatabaseTypeUnknown {
			return nil
		}

		// Check include/exclude patterns (only for files, not directories like RocksDB)
		if !info.IsDir() && !utils.ShouldIncludeFile(path, config.IncludePattern, config.ExcludePattern) {
			return nil
		}

		// Create relative name for backup
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		databases = append(databases, types.DatabaseInfo{
			Path: path,
			Type: dbType,
			Name: strings.ReplaceAll(relPath, string(filepath.Separator), "_"),
		})

		// If this is a RocksDB directory, don't walk into it
		if info.IsDir() && dbType == types.DatabaseTypeRocksDB {
			return filepath.SkipDir
		}

		return nil
	})

	return databases, err
}

// DetectDatabaseType detects database type based on file characteristics
func DetectDatabaseType(path string) types.DatabaseType {
	// Check if it's a RocksDB directory
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		// Look for RocksDB files
		if hasRocksDBFiles(path) {
			return types.DatabaseTypeRocksDB
		}
	}

	// Check if it's a SQLite file
	if isSQLiteFile(path) {
		return types.DatabaseTypeSQLite
	}

	// Check if it's a log file
	if isLogFile(path) {
		return types.DatabaseTypeLogFile
	}

	return types.DatabaseTypeUnknown
}

// hasRocksDBFiles checks if directory contains RocksDB files
func hasRocksDBFiles(dirPath string) bool {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, "CURRENT") ||
			strings.HasPrefix(name, "MANIFEST") ||
			strings.HasPrefix(name, "LOG") ||
			strings.HasSuffix(name, ".sst") ||
			strings.HasSuffix(name, ".log") {
			return true
		}
	}

	return false
}

// isSQLiteFile checks if file is a SQLite database
func isSQLiteFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" {
		// Verify it's actually a SQLite file by checking header
		return hasValidSQLiteHeader(filePath)
	}

	return false
}

// isLogFile checks if file is a log file
func isLogFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := strings.ToLower(filepath.Base(filePath))

	// Check by extension (.log files are always logs)
	if ext == ".log" || ext == ".logx" {
		return true
	}

	// For .txt files, check if they have log-related names
	if ext == ".txt" {
		logPatterns := []string{
			"access", "error", "debug", "info", "warn", "trace",
			"audit", "security", "application", "system", "server",
			"database", "sql", "query", "transaction", "backup",
		}

		for _, pattern := range logPatterns {
			if strings.Contains(filename, pattern) {
				return true
			}
		}
	}

	// Check by filename patterns (for files without extensions or other extensions)
	logPatterns := []string{
		"access", "error", "debug", "info", "warn", "trace",
		"audit", "security", "application", "system", "server",
		"database", "sql", "query", "transaction", "backup",
	}

	// Only consider files with log-like names and no extension or specific extensions
	if ext == "" || ext == ".out" {
		for _, pattern := range logPatterns {
			if strings.Contains(filename, pattern) {
				return true
			}
		}
	}

	return false
}

// hasValidSQLiteHeader checks SQLite header more reliably
func hasValidSQLiteHeader(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 16)
	if n, err := file.Read(header); err != nil || n < 16 {
		return false
	}

	return string(header[:15]) == "SQLite format 3"
}

// CheckDatabaseLock checks if a database is locked by another process
func CheckDatabaseLock(dbPath string, dbType types.DatabaseType) (*types.DatabaseLockInfo, error) {
	switch dbType {
	case types.DatabaseTypeRocksDB:
		return checkRocksDBLock(dbPath)
	case types.DatabaseTypeSQLite:
		return checkSQLiteLock(dbPath)
	default:
		// For log files and other types, check if file is open
		return checkFileLock(dbPath)
	}
}

// checkRocksDBLock checks if RocksDB is locked by another process
func checkRocksDBLock(dbPath string) (*types.DatabaseLockInfo, error) {
	info := &types.DatabaseLockInfo{}

	// Check for LOCK file
	lockFile := filepath.Join(dbPath, "LOCK")
	if _, err := os.Stat(lockFile); err == nil {
		info.IsLocked = true
		info.LockType = "RocksDB LOCK file"
		info.ProcessInfo = "Database is locked by another RocksDB process"
		return info, nil
	}

	return info, nil
}

// checkSQLiteLock checks if SQLite database is locked
func checkSQLiteLock(dbPath string) (*types.DatabaseLockInfo, error) {
	info := &types.DatabaseLockInfo{}

	// Check for SQLite lock files
	lockFiles := []string{
		dbPath + "-wal",
		dbPath + "-shm",
		dbPath + "-journal",
	}

	hasLockFiles := false
	for _, lockFile := range lockFiles {
		if _, err := os.Stat(lockFile); err == nil {
			hasLockFiles = true
			break
		}
	}

	// Try to open the database with a write lock to test if it's locked
	db, err := sql.Open("sqlite3", dbPath+"?mode=rw&_txlock=exclusive&_timeout=100")
	if err != nil {
		if strings.Contains(err.Error(), "locked") || strings.Contains(err.Error(), "busy") {
			info.IsLocked = true
			info.LockType = "SQLite database lock"
			info.ProcessInfo = "Database is locked by another SQLite process"
			return info, nil
		}
	} else {
		db.Close()
	}

	if hasLockFiles {
		info.LockType = "SQLite WAL/journal files present"
		info.ProcessInfo = "Database may be in use (WAL/journal files exist)"
	}

	return info, nil
}

// checkFileLock checks if a file is locked by another process
func checkFileLock(filePath string) (*types.DatabaseLockInfo, error) {
	info := &types.DatabaseLockInfo{}

	// Try to open the file with exclusive access
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsPermission(err) {
			info.IsLocked = true
			info.LockType = "File permission lock"
			info.ProcessInfo = "File is locked by another process"
			return info, nil
		}
		return info, err
	}
	defer file.Close()

	// Try to get an exclusive lock (Unix-specific)
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if err == syscall.EWOULDBLOCK {
			info.IsLocked = true
			info.LockType = "File lock (flock)"
			info.ProcessInfo = "File is locked by another process"
			return info, nil
		}
	} else {
		// Release the lock
		if unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); unlockErr != nil {
			// Log error but continue
		}
	}

	return info, nil
}
