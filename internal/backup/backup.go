package backup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"archiveFiles/internal/discovery"
	"archiveFiles/internal/progress"
	"archiveFiles/internal/types"
	"archiveFiles/internal/utils"
)

// SafeBackupDatabase performs a safe backup of a database, handling locked databases appropriately
func SafeBackupDatabase(sourceInfo types.DatabaseInfo, targetPath string, method string, progressTracker *progress.ProgressTracker) error {
	// Check if database is locked
	lockInfo, err := discovery.CheckDatabaseLock(sourceInfo.Path, sourceInfo.Type)
	if err != nil {
		log.Printf("Warning: Could not check database lock status for %s: %v", sourceInfo.Path, err)
		// Continue with normal backup if we can't check lock status
	}

	if lockInfo != nil && lockInfo.IsLocked {
		log.Printf("Warning: Database %s is locked (%s: %s)", sourceInfo.Path, lockInfo.LockType, lockInfo.ProcessInfo)

		// For locked databases, we need to use safe methods
		switch sourceInfo.Type {
		case types.DatabaseTypeRocksDB:
			return safeBackupLockedRocksDB(sourceInfo.Path, targetPath, progressTracker)
		case types.DatabaseTypeSQLite:
			return safeBackupLockedSQLite(sourceInfo.Path, targetPath, progressTracker)
		default:
			return fmt.Errorf("cannot safely backup locked file: %s (%s)", sourceInfo.Path, lockInfo.ProcessInfo)
		}
	}

	// Database is not locked, proceed with normal backup
	switch sourceInfo.Type {
	case types.DatabaseTypeRocksDB:
		return ProcessRocksDB(sourceInfo.Path, targetPath, method, progressTracker)
	case types.DatabaseTypeSQLite:
		return ProcessSQLiteDB(sourceInfo.Path, targetPath)
	case types.DatabaseTypeLogFile:
		return ProcessLogFile(sourceInfo.Path, targetPath)
	default:
		return fmt.Errorf("unknown database type: %s", sourceInfo.Path)
	}
}

// ProcessRocksDB processes a RocksDB database using the specified method
func ProcessRocksDB(sourceDBPath, targetDBPath, method string, progressTracker *progress.ProgressTracker) error {
	switch method {
	case "backup":
		return BackupRocksDB(sourceDBPath, targetDBPath, progressTracker)
	case "checkpoint":
		return CheckpointRocksDB(sourceDBPath, targetDBPath, progressTracker)
	case "copy":
		return CopyDatabaseData(sourceDBPath, targetDBPath, progressTracker)
	case "copy-files":
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	default:
		return fmt.Errorf("unknown method: %s. Available methods: backup, checkpoint, copy, copy-files", method)
	}
}

// ProcessSQLiteDB processes a SQLite database
func ProcessSQLiteDB(sourceDBPath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Copy SQLite file
	targetFile := filepath.Join(targetPath, filepath.Base(sourceDBPath))
	return CopySQLiteDatabase(sourceDBPath, targetFile)
}

// ProcessLogFile processes a log file by copying it to the target path
func ProcessLogFile(sourceLogPath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Copy log file
	targetFile := filepath.Join(targetPath, filepath.Base(sourceLogPath))
	return utils.CopyFile(sourceLogPath, targetFile)
}

// CopySQLiteDatabase copies a SQLite database file
func CopySQLiteDatabase(sourcePath, targetPath string) error {
	return utils.CopyFile(sourcePath, targetPath)
}

// SafeCopySQLiteDatabase performs a safe copy of a SQLite database
func SafeCopySQLiteDatabase(sourcePath, targetPath string) error {
	return fmt.Errorf("SafeCopySQLiteDatabase not implemented yet - will be in internal/backup/sqlite.go")
}

// safeBackupLockedRocksDB performs a safe backup of a locked RocksDB
func safeBackupLockedRocksDB(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	log.Printf("Attempting safe backup of locked RocksDB: %s", sourceDBPath)
	progressTracker.SetCurrentFile(fmt.Sprintf("Safe backup of locked RocksDB: %s", sourceDBPath))

	// For locked RocksDB, we try checkpoint method first, then backup engine
	err := safeBackupUsingCheckpoint(sourceDBPath, targetDBPath, progressTracker)
	if err != nil {
		log.Printf("Checkpoint method failed for locked RocksDB, trying backup engine: %v", err)
		return safeBackupUsingBackupEngine(sourceDBPath, targetDBPath, progressTracker)
	}

	return nil
}

// safeBackupUsingCheckpoint uses checkpoint API for locked databases
func safeBackupUsingCheckpoint(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	log.Printf("Using checkpoint method for locked RocksDB: %s", sourceDBPath)
	progressTracker.SetCurrentFile(fmt.Sprintf("Creating checkpoint for locked RocksDB: %s", sourceDBPath))

	// Use the checkpoint functionality from rocksdb package
	err := CheckpointRocksDB(sourceDBPath, targetDBPath, progressTracker)
	if err != nil {
		return fmt.Errorf("checkpoint creation failed for locked RocksDB: %v", err)
	}

	log.Printf("✅ Successfully created checkpoint backup of locked RocksDB")
	return nil
}

// safeBackupUsingBackupEngine uses backup engine for locked databases
func safeBackupUsingBackupEngine(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	log.Printf("Using backup engine for locked RocksDB: %s", sourceDBPath)
	progressTracker.SetCurrentFile(fmt.Sprintf("Creating backup engine backup for locked RocksDB: %s", sourceDBPath))

	// Use the backup engine functionality from rocksdb package
	err := BackupRocksDB(sourceDBPath, targetDBPath, progressTracker)
	if err != nil {
		return fmt.Errorf("backup engine failed for locked RocksDB: %v", err)
	}

	log.Printf("✅ Successfully created backup engine backup of locked RocksDB")
	return nil
}

// safeBackupLockedSQLite performs a safe backup of a locked SQLite database
func safeBackupLockedSQLite(sourceDBPath, targetPath string, progressTracker *progress.ProgressTracker) error {
	log.Printf("Attempting safe backup of locked SQLite: %s", sourceDBPath)
	progressTracker.SetCurrentFile(fmt.Sprintf("Safe backup of locked SQLite: %s", sourceDBPath))

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	targetFile := filepath.Join(targetPath, filepath.Base(sourceDBPath))

	// Use SQLite's online backup API which is safe for live databases
	err := SafeCopySQLiteDatabase(sourceDBPath, targetFile)
	if err != nil {
		return fmt.Errorf("safe SQLite backup failed: %v", err)
	}

	log.Printf("✅ Successfully created safe backup of locked SQLite")
	return nil
}
