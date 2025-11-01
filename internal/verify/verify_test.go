package verify

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"archiveFiles/internal/types"

	_ "github.com/mattn/go-sqlite3"
)

func TestVerifyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	sourcePath := filepath.Join(tempDir, "source.log")
	content := []byte("test log content\n")
	if err := os.WriteFile(sourcePath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create backup directory and file
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	backupPath := filepath.Join(backupDir, "source.log")
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Test verification
	err := verifyFile(sourcePath, backupDir)
	if err != nil {
		t.Errorf("Verification should pass for identical files, got error: %v", err)
	}
}

func TestVerifyFile_SizeMismatch(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	sourcePath := filepath.Join(tempDir, "source.log")
	if err := os.WriteFile(sourcePath, []byte("test log content\n"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create backup with different content
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	backupPath := filepath.Join(backupDir, "source.log")
	if err := os.WriteFile(backupPath, []byte("different\n"), 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Test verification
	err := verifyFile(sourcePath, backupDir)
	if err == nil {
		t.Error("Verification should fail for files with different sizes")
	}
}

func TestVerifyFile_MissingBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	sourcePath := filepath.Join(tempDir, "source.log")
	if err := os.WriteFile(sourcePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create empty backup directory
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// Test verification
	err := verifyFile(sourcePath, backupDir)
	if err == nil {
		t.Error("Verification should fail when backup file is missing")
	}
}

func TestVerifySQLite(t *testing.T) {
	tempDir := t.TempDir()

	// Create source SQLite database
	sourcePath := filepath.Join(tempDir, "source.db")
	db, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}

	// Create a simple table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec("INSERT INTO test (value) VALUES ('test1'), ('test2')")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to insert data: %v", err)
	}

	db.Close()

	// Create backup directory and copy database
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	backupPath := filepath.Join(backupDir, "source.db")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("Failed to read source: %v", err)
	}

	if err := os.WriteFile(backupPath, sourceData, 0644); err != nil {
		t.Fatalf("Failed to write backup: %v", err)
	}

	// Test verification
	err = verifySQLite(sourcePath, backupDir)
	if err != nil {
		t.Errorf("Verification should pass for valid SQLite backup, got error: %v", err)
	}
}

func TestVerifySQLite_CorruptedBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create source SQLite database
	sourcePath := filepath.Join(tempDir, "source.db")
	db, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}

	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create table: %v", err)
	}

	db.Close()

	// Create corrupted backup
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	backupPath := filepath.Join(backupDir, "source.db")
	if err := os.WriteFile(backupPath, []byte("corrupted data"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted backup: %v", err)
	}

	// Test verification
	err = verifySQLite(sourcePath, backupDir)
	if err == nil {
		t.Error("Verification should fail for corrupted SQLite backup")
	}
}

func TestCalculateFileHash(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content for hashing")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate hash
	hash1, err := calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Calculate hash again
	hash2, err := calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash second time: %v", err)
	}

	// Hashes should be identical
	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic, got %s and %s", hash1, hash2)
	}

	// Hash should be 64 characters (SHA256 in hex)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

func TestVerifyBackup(t *testing.T) {
	tempDir := t.TempDir()

	// Create a log file for testing
	sourcePath := filepath.Join(tempDir, "test.log")
	content := []byte("test log content\n")
	if err := os.WriteFile(sourcePath, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create backup
	backupDir := filepath.Join(tempDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	backupPath := filepath.Join(backupDir, "test.log")
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Create database info
	dbInfo := types.DatabaseInfo{
		Path: sourcePath,
		Type: types.DatabaseTypeLogFile,
		Name: "test.log",
	}

	// Test verification
	err := VerifyBackup(dbInfo, backupDir, nil)
	if err != nil {
		t.Errorf("VerifyBackup should succeed, got error: %v", err)
	}
}

func TestVerifyBackup_UnsupportedType(t *testing.T) {
	dbInfo := types.DatabaseInfo{
		Path: "/tmp/test",
		Type: types.DatabaseTypeUnknown,
		Name: "test",
	}

	err := VerifyBackup(dbInfo, "/tmp/backup", nil)
	if err == nil {
		t.Error("VerifyBackup should fail for unsupported database type")
	}
}
