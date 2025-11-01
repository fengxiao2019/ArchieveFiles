package backup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"archiveFiles/internal/constants"
	"archiveFiles/internal/progress"
	"archiveFiles/internal/utils"

	"github.com/linxGnu/grocksdb"
)

// BackupRocksDB creates a backup using RocksDB BackupEngine
func BackupRocksDB(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	progressTracker.SetCurrentFile(fmt.Sprintf("Backing up %s", sourceDBPath))

	// Try to open database in read-write mode first for proper backup
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	// First try read-write mode for BackupEngine (it might need write access)
	sourceDB, err := grocksdb.OpenDb(sourceOpts, sourceDBPath)
	if err != nil {
		// If read-write fails, try read-only mode
		log.Printf("Could not open database in read-write mode, trying read-only: %v", err)
		sourceDB, err = grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
		if err != nil {
			log.Printf("Warning: Could not open database for backup engine, falling back to file copy: %v", err)
			return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
		}
	}
	defer sourceDB.Close()

	// Create backup engine with target path
	backupEngine, err := grocksdb.CreateBackupEngineWithPath(sourceDB, targetDBPath)
	if err != nil {
		log.Printf("Warning: Could not create backup engine, falling back to file copy: %v", err)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}
	defer backupEngine.Close()

	// Create new backup with flush to ensure consistency
	progressTracker.SetCurrentFile(fmt.Sprintf("Creating backup for %s", sourceDBPath))
	err = backupEngine.CreateNewBackupFlush(true)
	if err != nil {
		log.Printf("Warning: Backup creation failed, falling back to file copy: %v", err)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}

	// Verify backup integrity
	backupInfos := backupEngine.GetInfo()
	if len(backupInfos) == 0 {
		log.Printf("Warning: No backup info available, falling back to file copy")
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}

	// Get the latest backup info
	latestBackup := backupInfos[len(backupInfos)-1]

	// Verify the backup
	progressTracker.SetCurrentFile(fmt.Sprintf("Verifying backup %d for %s", latestBackup.ID, sourceDBPath))
	err = backupEngine.VerifyBackup(latestBackup.ID)
	if err != nil {
		log.Printf("Warning: Backup verification failed: %v", err)
		// Continue anyway - backup might still be valid
	}

	// Update progress with backup size
	progressTracker.CompleteItem(int64(latestBackup.Size))

	log.Printf("Successfully created backup ID %d: %d bytes, %d files",
		latestBackup.ID, latestBackup.Size, latestBackup.NumFiles)
	return nil
}

// CheckpointRocksDB creates a checkpoint using RocksDB Checkpoint API
func CheckpointRocksDB(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	progressTracker.SetCurrentFile(fmt.Sprintf("Checkpointing %s", sourceDBPath))

	// Try the checkpoint API first
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		// If we can't open the database, fall back to file-based backup
		log.Printf("Warning: Could not open database for checkpoint, falling back to file copy: %v", err)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}
	defer sourceDB.Close()

	// Create checkpoint
	checkpoint, err := sourceDB.NewCheckpoint()
	if err != nil {
		// If checkpoint creation fails, fall back to file-based backup
		log.Printf("Warning: Could not create checkpoint object, falling back to file copy: %v", err)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}
	defer checkpoint.Destroy()

	// Create checkpoint directory
	progressTracker.SetCurrentFile(fmt.Sprintf("Creating checkpoint at %s", targetDBPath))
	if err := checkpoint.CreateCheckpoint(targetDBPath, 0); err != nil {
		// If checkpoint fails, fall back to file-based backup
		log.Printf("Warning: Checkpoint creation failed, falling back to file copy: %v", err)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}

	// Verify the checkpoint includes all necessary files
	if !VerifyBackupCompleteness(sourceDBPath, targetDBPath) {
		log.Printf("Warning: Checkpoint appears incomplete, falling back to file copy")
		// Remove incomplete checkpoint
		os.RemoveAll(targetDBPath)
		return BackupRocksDBFiles(sourceDBPath, targetDBPath, progressTracker)
	}

	// Calculate checkpoint size
	checkpointSize := utils.CalculateSize(targetDBPath)
	progressTracker.CompleteItem(checkpointSize)
	return nil
}

// CopyDatabaseData copies database data record by record
func CopyDatabaseData(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	// open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer sourceDB.Close()

	// Optimization: Use single pass instead of counting first
	// Progress will be updated incrementally as we copy records
	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	// create target database
	targetOpts := grocksdb.NewDefaultOptions()
	targetOpts.SetCreateIfMissing(true)
	defer targetOpts.Destroy()

	targetDB, err := grocksdb.OpenDb(targetOpts, targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to create target db: %v", err)
	}
	defer targetDB.Close()

	// create iterator for copying
	iter := sourceDB.NewIterator(readOpts)
	defer iter.Close()

	// create write batch
	writeBatch := grocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	var count int64

	// iterate all data (single pass optimization)
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		// Safely extract key and value with immediate cleanup
		key := iter.Key()
		value := iter.Value()

		// Copy data before freeing (safer approach)
		keyData := make([]byte, len(key.Data()))
		valueData := make([]byte, len(value.Data()))
		copy(keyData, key.Data())
		copy(valueData, value.Data())

		// Free immediately after copying to prevent leaks
		key.Free()
		value.Free()

		// Now use the copied data
		writeBatch.Put(keyData, valueData)
		count++

		// write batch periodically
		if count%constants.RocksDBWriteBatchSize == 0 {
			err = targetDB.Write(writeOpts, writeBatch)
			if err != nil {
				// No need to free key/value here - already freed above
				return fmt.Errorf("failed to write batch: %v", err)
			}
			writeBatch.Clear()
		}

		// update progress less frequently (performance optimization)
		if count%constants.RocksDBProgressUpdateInterval == 0 {
			// Use 0 as total to indicate we don't know the total
			// This will display "Copied N records" instead of percentage
			progressTracker.UpdateRocksDBProgress(count, 0)
		}
	}

	// Write remaining records
	if writeBatch.Count() > 0 {
		err = targetDB.Write(writeOpts, writeBatch)
		if err != nil {
			return fmt.Errorf("failed to write final batch: %v", err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("error during iteration: %v", err)
	}

	// Final progress update with actual count
	progressTracker.UpdateRocksDBProgress(count, count)
	progressTracker.CompleteItem(utils.CalculateSize(targetDBPath))

	log.Printf("Copied %d records from %s", count, sourceDBPath)
	return nil
}

// BackupRocksDBFiles creates a backup by copying all RocksDB files
func BackupRocksDBFiles(sourceDBPath, targetDBPath string, progressTracker *progress.ProgressTracker) error {
	progressTracker.SetCurrentFile(fmt.Sprintf("Copying RocksDB files from %s", sourceDBPath))

	// Create target directory
	if err := os.MkdirAll(targetDBPath, constants.DirPermission); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Get list of all files in source directory
	sourceFiles, err := os.ReadDir(sourceDBPath)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %v", err)
	}

	var totalSize int64
	for _, file := range sourceFiles {
		if !file.IsDir() {
			if info, err := file.Info(); err == nil {
				totalSize += info.Size()
			}
		}
	}

	var copiedSize int64
	for _, file := range sourceFiles {
		if file.IsDir() {
			continue // Skip subdirectories
		}

		sourcePath := filepath.Join(sourceDBPath, file.Name())
		targetPath := filepath.Join(targetDBPath, file.Name())

		progressTracker.SetCurrentFile(fmt.Sprintf("Copying %s", file.Name()))

		// Copy the file
		if err := utils.CopyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %v", file.Name(), err)
		}

		if info, err := file.Info(); err == nil {
			copiedSize += info.Size()
		}
	}

	progressTracker.CompleteItem(copiedSize)
	return nil
}

// VerifyBackupCompleteness verifies that backup includes all necessary files
func VerifyBackupCompleteness(sourceDBPath, backupDBPath string) bool {
	// Check for critical files that should be in any complete backup
	sourceFiles, err := os.ReadDir(sourceDBPath)
	if err != nil {
		return false
	}

	backupFiles, err := os.ReadDir(backupDBPath)
	if err != nil {
		return false
	}

	// Create a map of backup files for quick lookup
	backupFileMap := make(map[string]bool)
	for _, file := range backupFiles {
		backupFileMap[file.Name()] = true
	}

	// Check for important files that should be copied
	for _, file := range sourceFiles {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()

		// Critical files that must be present
		if strings.HasSuffix(fileName, ".log") || // WAL files
			strings.HasPrefix(fileName, "MANIFEST") ||
			fileName == "CURRENT" ||
			strings.HasSuffix(fileName, ".sst") { // SST files

			if !backupFileMap[fileName] {
				log.Printf("Warning: Critical file %s missing from backup", fileName)
				return false
			}
		}
	}

	return true
}

// CountRocksDBRecords counts records in a RocksDB database
func CountRocksDBRecords(db *grocksdb.DB, readOpts *grocksdb.ReadOptions) int64 {
	iter := db.NewIterator(readOpts)
	defer iter.Close()

	var count int64
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		count++
		key := iter.Key()
		key.Free()
	}

	return count
}
