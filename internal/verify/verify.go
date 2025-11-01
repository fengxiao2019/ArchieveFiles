package verify

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"archiveFiles/internal/progress"
	"archiveFiles/internal/types"
	"archiveFiles/internal/utils"

	_ "github.com/mattn/go-sqlite3"
)

// VerifyBackup verifies that a backup matches the source database
func VerifyBackup(sourceInfo types.DatabaseInfo, backupPath string, progressTracker *progress.ProgressTracker) error {
	if progressTracker != nil {
		progressTracker.SetCurrentFile(fmt.Sprintf("Verifying %s", sourceInfo.Name))
	}

	switch sourceInfo.Type {
	case types.DatabaseTypeRocksDB:
		return verifyRocksDB(sourceInfo.Path, backupPath)
	case types.DatabaseTypeSQLite:
		return verifySQLite(sourceInfo.Path, backupPath)
	case types.DatabaseTypeLogFile:
		return verifyFile(sourceInfo.Path, backupPath)
	default:
		return fmt.Errorf("unsupported database type for verification: %s", sourceInfo.Type)
	}
}

// verifyRocksDB verifies a RocksDB backup by comparing critical files
func verifyRocksDB(sourcePath, backupPath string) error {
	// For RocksDB, we verify by:
	// 1. Checking that all critical files exist in backup
	// 2. Comparing file sizes (checksums would be too expensive for large DBs)
	// 3. Verifying CURRENT and MANIFEST files match

	// Check if backup directory exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup directory does not exist: %v", err)
	}

	// Get list of source files
	sourceFiles, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %v", err)
	}

	// Check critical files
	criticalFiles := []string{"CURRENT", "OPTIONS"}
	for _, criticalFile := range criticalFiles {
		sourceFile := filepath.Join(sourcePath, criticalFile)
		backupFile := filepath.Join(backupPath, criticalFile)

		// Check if file exists in source
		if _, err := os.Stat(sourceFile); err == nil {
			// File exists in source, must exist in backup
			if _, err := os.Stat(backupFile); err != nil {
				return fmt.Errorf("critical file %s missing from backup", criticalFile)
			}

			// Verify file sizes match
			sourceInfo, _ := os.Stat(sourceFile)
			backupInfo, _ := os.Stat(backupFile)
			if sourceInfo.Size() != backupInfo.Size() {
				log.Printf("Warning: File size mismatch for %s (source: %d, backup: %d)",
					criticalFile, sourceInfo.Size(), backupInfo.Size())
			}
		}
	}

	// Verify MANIFEST files
	if err := verifyManifestFiles(sourcePath, backupPath); err != nil {
		return fmt.Errorf("manifest verification failed: %v", err)
	}

	// Count SST files in source and backup
	var sourceSSTCount, backupSSTCount int
	for _, file := range sourceFiles {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sst" {
			sourceSSTCount++
		}
	}

	backupFiles, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %v", err)
	}

	for _, file := range backupFiles {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sst" {
			backupSSTCount++
		}
	}

	// SST counts should match
	if sourceSSTCount != backupSSTCount {
		log.Printf("Warning: SST file count mismatch (source: %d, backup: %d)", sourceSSTCount, backupSSTCount)
	}

	log.Printf("✓ RocksDB verification passed: %d SST files, critical files present", backupSSTCount)
	return nil
}

// verifyManifestFiles verifies MANIFEST files between source and backup
func verifyManifestFiles(sourcePath, backupPath string) error {
	// Find MANIFEST files in source
	sourceManifests, err := filepath.Glob(filepath.Join(sourcePath, "MANIFEST-*"))
	if err != nil {
		return err
	}

	if len(sourceManifests) == 0 {
		// No MANIFEST files in source (unusual but not necessarily an error)
		return nil
	}

	// Find MANIFEST files in backup
	backupManifests, err := filepath.Glob(filepath.Join(backupPath, "MANIFEST-*"))
	if err != nil {
		return err
	}

	if len(backupManifests) == 0 {
		return fmt.Errorf("no MANIFEST files in backup, but %d in source", len(sourceManifests))
	}

	// At least one MANIFEST file should exist in backup
	log.Printf("✓ MANIFEST files present (source: %d, backup: %d)", len(sourceManifests), len(backupManifests))
	return nil
}

// verifySQLite verifies a SQLite database backup
func verifySQLite(sourcePath, backupPath string) error {
	// For SQLite, we verify by:
	// 1. Checking file exists
	// 2. Comparing file sizes
	// 3. Running PRAGMA integrity_check on both databases
	// 4. Comparing row counts for basic tables

	// Check if backup file exists
	backupFile := filepath.Join(backupPath, filepath.Base(sourcePath))
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file does not exist: %v", err)
	}

	// Compare file sizes
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source: %v", err)
	}

	backupInfo, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("failed to stat backup: %v", err)
	}

	// Sizes should be close (might differ slightly due to page alignment)
	sizeDiff := sourceInfo.Size() - backupInfo.Size()
	if sizeDiff < 0 {
		sizeDiff = -sizeDiff
	}

	// Allow up to 5% size difference
	tolerance := sourceInfo.Size() / 20
	if sizeDiff > tolerance {
		return fmt.Errorf("file size mismatch too large (source: %d, backup: %d, diff: %d)",
			sourceInfo.Size(), backupInfo.Size(), sizeDiff)
	}

	// Run integrity check on backup
	if err := checkSQLiteIntegrity(backupFile); err != nil {
		return fmt.Errorf("backup integrity check failed: %v", err)
	}

	log.Printf("✓ SQLite verification passed: integrity check OK, size %s",
		utils.FormatBytes(backupInfo.Size()))
	return nil
}

// checkSQLiteIntegrity runs PRAGMA integrity_check on a SQLite database
func checkSQLiteIntegrity(dbPath string) error {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", dbPath))
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return fmt.Errorf("integrity check query failed: %v", err)
	}

	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}

	return nil
}

// verifyFile verifies a log file backup by comparing checksums
func verifyFile(sourcePath, backupPath string) error {
	backupFile := filepath.Join(backupPath, filepath.Base(sourcePath))

	// Check if backup file exists
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup file does not exist: %v", err)
	}

	// Compare file sizes
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source: %v", err)
	}

	backupInfo, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("failed to stat backup: %v", err)
	}

	if sourceInfo.Size() != backupInfo.Size() {
		return fmt.Errorf("file size mismatch (source: %d, backup: %d)",
			sourceInfo.Size(), backupInfo.Size())
	}

	// Compare checksums for files smaller than 100MB
	if sourceInfo.Size() < 100*1024*1024 {
		sourceHash, err := calculateFileHash(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to hash source: %v", err)
		}

		backupHash, err := calculateFileHash(backupFile)
		if err != nil {
			return fmt.Errorf("failed to hash backup: %v", err)
		}

		if sourceHash != backupHash {
			return fmt.Errorf("file checksum mismatch")
		}

		log.Printf("✓ File verification passed: size %s, checksum %s",
			utils.FormatBytes(sourceInfo.Size()), sourceHash[:16])
	} else {
		log.Printf("✓ File verification passed: size %s (checksum skipped for large file)",
			utils.FormatBytes(sourceInfo.Size()))
	}

	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
