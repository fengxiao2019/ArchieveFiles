package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

// Test helper functions
func setupTestDB(t *testing.T) string {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_db")

	// Create test database
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Add some test data
	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	testData := map[string]string{
		"key1":           "value1",
		"key2":           "value2",
		"key3":           "value3",
		"config:version": "1.0.0",
		"config:debug":   "true",
		"user:1001":      `{"id":1001,"name":"Alice","email":"alice@example.com"}`,
		"user:1002":      `{"id":1002,"name":"Bob","email":"bob@example.com"}`,
		"product:p001":   `{"id":"p001","name":"Laptop","price":999.99}`,
		"log:2023-10-01": `{"timestamp":"2023-10-01T10:00:00Z","level":"INFO","message":"Test log"}`,
	}

	for key, value := range testData {
		err := db.Put(writeOpts, []byte(key), []byte(value))
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}

	return dbPath
}

func setupTestSQLiteDB(t *testing.T) string {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create table and insert test data
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			email TEXT
		);
		INSERT INTO users (name, email) VALUES 
			('Alice', 'alice@example.com'),
			('Bob', 'bob@example.com'),
			('Charlie', 'charlie@example.com');
	`)
	if err != nil {
		t.Fatalf("Failed to create SQLite test data: %v", err)
	}

	return dbPath
}

func setupTestLogFiles(t *testing.T) []string {
	tempDir := t.TempDir()

	logFiles := []string{
		filepath.Join(tempDir, "application.log"),
		filepath.Join(tempDir, "error.log"),
		filepath.Join(tempDir, "access.log"),
		filepath.Join(tempDir, "debug.txt"),
		filepath.Join(tempDir, "audit_2023.log"),
		filepath.Join(tempDir, "database_queries.log"),
	}

	logContents := []string{
		"2023-10-01 10:00:00 INFO Application started\n2023-10-01 10:01:00 INFO User logged in",
		"2023-10-01 10:05:00 ERROR Database connection failed\n2023-10-01 10:06:00 ERROR Retry failed",
		"192.168.1.1 - - [01/Oct/2023:10:00:00] \"GET /api/users\" 200 1234",
		"DEBUG: Variable x = 42\nDEBUG: Function called with params: {id: 123}",
		"AUDIT: User admin accessed sensitive data at 2023-10-01T10:00:00Z",
		"2023-10-01 10:00:00 SELECT * FROM users WHERE id = 1\n2023-10-01 10:01:00 UPDATE users SET name = 'John' WHERE id = 1",
	}

	for i, logFile := range logFiles {
		err := os.WriteFile(logFile, []byte(logContents[i]), 0644)
		if err != nil {
			t.Fatalf("Failed to create log file %s: %v", logFile, err)
		}
	}

	return logFiles
}

func setupTestDirectory(t *testing.T) string {
	tempDir := t.TempDir()

	// Create multiple databases
	rocksDB1 := filepath.Join(tempDir, "rocks1")
	rocksDB2 := filepath.Join(tempDir, "rocks2")
	sqliteDB1 := filepath.Join(tempDir, "app1.db")
	sqliteDB2 := filepath.Join(tempDir, "app2.sqlite")

	// Create RocksDB databases
	for _, dbPath := range []string{rocksDB1, rocksDB2} {
		opts := grocksdb.NewDefaultOptions()
		opts.SetCreateIfMissing(true)
		defer opts.Destroy()

		db, err := grocksdb.OpenDb(opts, dbPath)
		if err != nil {
			t.Fatalf("Failed to create test RocksDB: %v", err)
		}

		writeOpts := grocksdb.NewDefaultWriteOptions()
		defer writeOpts.Destroy()

		err = db.Put(writeOpts, []byte("test"), []byte("data"))
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}

		db.Close()
	}

	// Create SQLite databases
	for _, dbPath := range []string{sqliteDB1, sqliteDB2} {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("Failed to create SQLite database: %v", err)
		}

		_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, data TEXT); INSERT INTO test (data) VALUES ('test');")
		if err != nil {
			t.Fatalf("Failed to create SQLite test data: %v", err)
		}

		db.Close()
	}

	// Create log files
	logFiles := []string{
		filepath.Join(tempDir, "application.log"),
		filepath.Join(tempDir, "error.txt"),
		filepath.Join(tempDir, "access_log"),
	}

	for _, logFile := range logFiles {
		err := os.WriteFile(logFile, []byte("Test log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create log file: %v", err)
		}
	}

	// Create some non-database files
	nonDBFile := filepath.Join(tempDir, "readme.txt")
	err := os.WriteFile(nonDBFile, []byte("This is not a database"), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-DB file: %v", err)
	}

	return tempDir
}

func setupMultipleTestDirectories(t *testing.T) []string {
	// Create multiple source directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	dir3 := t.TempDir()

	// Directory 1: RocksDB + logs
	rocksDB1 := filepath.Join(dir1, "primary_db")
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, rocksDB1)
	if err != nil {
		t.Fatalf("Failed to create RocksDB: %v", err)
	}

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	err = db.Put(writeOpts, []byte("dir1"), []byte("data1"))
	if err != nil {
		t.Fatalf("Failed to put test data: %v", err)
	}

	db.Close()

	err = os.WriteFile(filepath.Join(dir1, "application.log"), []byte("App log from dir1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	err = os.WriteFile(filepath.Join(dir1, "error.log"), []byte("Error log from dir1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Directory 2: SQLite databases
	sqliteDB1 := filepath.Join(dir2, "users.db")
	sqliteDB2 := filepath.Join(dir2, "products.sqlite")

	for _, dbPath := range []string{sqliteDB1, sqliteDB2} {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("Failed to create SQLite database: %v", err)
		}

		_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, data TEXT); INSERT INTO test (data) VALUES ('dir2_data');")
		if err != nil {
			t.Fatalf("Failed to create SQLite test data: %v", err)
		}

		db.Close()
	}

	// Directory 3: Mixed content
	rocksDB2 := filepath.Join(dir3, "cache_db")
	db, err = grocksdb.OpenDb(opts, rocksDB2)
	if err != nil {
		t.Fatalf("Failed to create RocksDB: %v", err)
	}
	err = db.Put(writeOpts, []byte("dir3"), []byte("cache_data"))
	if err != nil {
		t.Fatalf("Failed to put test data: %v", err)
	}
	db.Close()

	// Create a log file in dir3
	err = os.WriteFile(filepath.Join(dir3, "debug.txt"), []byte("Debug log from dir3"), 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	return []string{dir1, dir2, dir3}
}

func verifyDatabaseContents(t *testing.T, dbPath string, expectedCount int) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDbForReadOnly(opts, dbPath, false)
	if err != nil {
		t.Fatalf("Failed to open database for verification: %v", err)
	}
	defer db.Close()

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	iter := db.NewIterator(readOpts)
	defer iter.Close()

	count := 0
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		if len(key.Data()) == 0 || len(value.Data()) == 0 {
			t.Errorf("Empty key or value found")
		}

		key.Free()
		value.Free()
		count++
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("Iterator error: %v", err)
	}

	if count != expectedCount {
		t.Errorf("Expected %d records, got %d", expectedCount, count)
	}
}

func verifyArchiveContents(t *testing.T, archivePath string) {
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			fileCount++
		}
	}

	if fileCount == 0 {
		t.Error("Archive contains no files")
	}
}

// Test database type detection
func TestDatabaseTypeDetection(t *testing.T) {
	// Test RocksDB detection
	rocksDB := setupTestDB(t)
	if detectDatabaseType(rocksDB) != DatabaseTypeRocksDB {
		t.Error("Failed to detect RocksDB")
	}

	// Test SQLite detection
	sqliteDB := setupTestSQLiteDB(t)
	if detectDatabaseType(sqliteDB) != DatabaseTypeSQLite {
		t.Error("Failed to detect SQLite database")
	}

	// Test unknown type
	tempFile := filepath.Join(t.TempDir(), "unknown.txt")
	err := os.WriteFile(tempFile, []byte("not a database"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if detectDatabaseType(tempFile) != DatabaseTypeUnknown {
		t.Error("Should detect unknown database type")
	}
}

// Test database discovery
func TestDatabaseDiscovery(t *testing.T) {
	testDir := setupTestDirectory(t)

	config := &Config{
		SourcePaths: []string{testDir},
		BatchMode:   true,
	}

	databases, err := discoverDatabases(config, testDir)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Should find items (2 RocksDB + 2 SQLite + 3 log files, excluding readme.txt)
	if len(databases) < 4 {
		t.Errorf("Expected at least 4 items, found %d", len(databases))
	}

	// Count by type
	rocksCount := 0
	sqliteCount := 0
	logCount := 0
	for _, db := range databases {
		switch db.Type {
		case DatabaseTypeRocksDB:
			rocksCount++
		case DatabaseTypeSQLite:
			sqliteCount++
		case DatabaseTypeLogFile:
			logCount++
		}
	}

	if rocksCount != 2 {
		t.Errorf("Expected 2 RocksDB databases, found %d", rocksCount)
	}
	if sqliteCount != 2 {
		t.Errorf("Expected 2 SQLite databases, found %d", sqliteCount)
	}
	if logCount != 3 {
		t.Errorf("Expected 3 log files, found %d", logCount)
	}
}

// Test file pattern filtering
func TestFilePatternFiltering(t *testing.T) {
	testDir := setupTestDirectory(t)

	// Test include pattern
	config := &Config{
		SourcePaths:    []string{testDir},
		BatchMode:      true,
		IncludePattern: "*.db",
	}

	databases, err := discoverDatabases(config, testDir)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Should find .db files (SQLite) and all RocksDB directories (not affected by file patterns)
	foundSQLite := 0
	foundRocksDB := 0
	for _, db := range databases {
		switch db.Type {
		case DatabaseTypeSQLite:
			if !strings.HasSuffix(db.Path, ".db") {
				t.Errorf("Include pattern failed for SQLite, found: %s", db.Path)
			}
			foundSQLite++
		case DatabaseTypeRocksDB:
			foundRocksDB++
		}
	}

	if foundSQLite != 1 {
		t.Errorf("Expected 1 SQLite .db file, found %d", foundSQLite)
	}
	if foundRocksDB != 2 {
		t.Errorf("Expected 2 RocksDB directories (unaffected by file patterns), found %d", foundRocksDB)
	}

	// Test exclude pattern
	config = &Config{
		SourcePaths:    []string{testDir},
		BatchMode:      true,
		ExcludePattern: "*.sqlite",
	}

	databases, err = discoverDatabases(config, testDir)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Should not find .sqlite files
	for _, db := range databases {
		if strings.HasSuffix(db.Path, ".sqlite") {
			t.Errorf("Exclude pattern failed, found: %s", db.Path)
		}
	}
}

// Test SQLite database processing
func TestProcessSQLiteDB(t *testing.T) {
	sourceDB := setupTestSQLiteDB(t)
	targetPath := filepath.Join(t.TempDir(), "sqlite_backup")

	err := processSQLiteDB(sourceDB, targetPath)
	if err != nil {
		t.Fatalf("Failed to process SQLite database: %v", err)
	}

	// Verify backup file exists
	backupFile := filepath.Join(targetPath, filepath.Base(sourceDB))
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Errorf("SQLite backup file not created: %s", backupFile)
	}

	// Verify backup file is a valid SQLite database
	db, err := sql.Open("sqlite3", backupFile)
	if err != nil {
		t.Fatalf("Failed to open backup SQLite database: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query backup database: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 users in backup, got %d", count)
	}
}

// Test batch processing
func TestBatchProcessing(t *testing.T) {
	testDir := setupTestDirectory(t)
	backupDir := filepath.Join(t.TempDir(), "batch_backup")

	config := &Config{
		SourcePaths:  []string{testDir},
		BackupPath:   backupDir,
		BatchMode:    true,
		Method:       "copy",
		Compress:     false,
		RemoveBackup: false,
	}

	// Discover databases
	databases, err := discoverDatabases(config, testDir)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each database/file
	for _, db := range databases {
		dbBackupPath := filepath.Join(backupDir, db.Name)

		switch db.Type {
		case DatabaseTypeRocksDB:
			err = processRocksDB(db.Path, dbBackupPath, config.Method, NewProgressTracker(false))
		case DatabaseTypeSQLite:
			err = processSQLiteDB(db.Path, dbBackupPath)
		case DatabaseTypeLogFile:
			err = processLogFile(db.Path, dbBackupPath)
		}

		if err != nil {
			t.Fatalf("Failed to process %s: %v", db.Name, err)
		}
	}

	// Verify all databases were backed up
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(entries) != len(databases) {
		t.Errorf("Expected %d backup entries, got %d", len(databases), len(entries))
	}
}

// Note: Backup and checkpoint methods use copy implementation

// Test copy method
func TestCopyDatabaseData(t *testing.T) {
	sourceDB := setupTestDB(t)
	targetDB := filepath.Join(t.TempDir(), "copy")

	err := copyDatabaseData(sourceDB, targetDB, NewProgressTracker(false))
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Verify target database exists and has correct data
	verifyDatabaseContents(t, targetDB, 9) // We added 9 test records
}

// Test compression
func TestCompressDirectory(t *testing.T) {
	// Create a test directory with some files
	testDir := t.TempDir()
	subDir := filepath.Join(testDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		filepath.Join(testDir, "file1.txt"):  "content1",
		filepath.Join(testDir, "file2.txt"):  "content2",
		filepath.Join(subDir, "subfile.txt"): "subcontent",
	}

	for path, content := range testFiles {
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Compress the directory
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	err = compressDirectory(testDir, archivePath)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Verify archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive not created: %s", archivePath)
	}

	// Verify archive contents
	verifyArchiveContents(t, archivePath)
}

// Test readonly database access
func TestReadOnlyAccess(t *testing.T) {
	sourceDB := setupTestDB(t)

	// Open database in readonly mode
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDbForReadOnly(opts, sourceDB, false)
	if err != nil {
		t.Fatalf("Failed to open database in readonly mode: %v", err)
	}
	defer db.Close()

	// Try to read data
	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	value, err := db.Get(readOpts, []byte("key1"))
	if err != nil {
		t.Fatalf("Failed to read from readonly database: %v", err)
	}
	defer value.Free()

	if string(value.Data()) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(value.Data()))
	}
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	// Test with non-existent source database
	err := copyDatabaseData("/non/existent/path", t.TempDir(), NewProgressTracker(false))
	if err == nil {
		t.Error("Expected error for non-existent source database")
	}

	// Test with invalid backup path
	sourceDB := setupTestDB(t)
	err = copyDatabaseData(sourceDB, "/invalid/path/that/cannot/be/created", NewProgressTracker(false))
	if err == nil {
		t.Error("Expected error for invalid backup path")
	}
}

// Test full end-to-end workflow
func TestFullWorkflow(t *testing.T) {
	sourceDB := setupTestDB(t)
	tempDir := t.TempDir()
	backupPath := filepath.Join(tempDir, "backup")
	archivePath := filepath.Join(tempDir, "archive.tar.gz")

	// Test copy method
	err := copyDatabaseData(sourceDB, backupPath, NewProgressTracker(false))
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Test compression
	err = compressDirectory(backupPath, archivePath)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Verify archive
	verifyArchiveContents(t, archivePath)

	// Test cleanup
	err = os.RemoveAll(backupPath)
	if err != nil {
		t.Fatalf("Failed to remove backup directory: %v", err)
	}

	// Verify backup directory is gone
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup directory was not removed")
	}

	// Verify archive still exists
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Archive was removed unexpectedly")
	}
}

// Benchmark tests
func BenchmarkCopyDatabaseAsBackup(b *testing.B) {
	sourceDB := setupTestDB(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backupPath := filepath.Join(b.TempDir(), "backup")
		err := copyDatabaseData(sourceDB, backupPath, NewProgressTracker(false))
		if err != nil {
			b.Fatalf("Copy failed: %v", err)
		}
	}
}

func BenchmarkCopyDatabase(b *testing.B) {
	sourceDB := setupTestDB(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetDB := filepath.Join(b.TempDir(), "copy")
		err := copyDatabaseData(sourceDB, targetDB, NewProgressTracker(false))
		if err != nil {
			b.Fatalf("Copy failed: %v", err)
		}
	}
}

// Test with different data sizes
func TestLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "large_db")

	// Create database with more data
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		t.Fatalf("Failed to create large test database: %v", err)
	}
	defer db.Close()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	// Add 1000 records
	for i := 0; i < 1000; i++ {
		key := []byte(strings.Repeat("k", 100) + string(rune(i)))
		value := []byte(strings.Repeat("v", 1000) + string(rune(i)))

		err := db.Put(writeOpts, key, value)
		if err != nil {
			t.Fatalf("Failed to put large data: %v", err)
		}
	}

	// Test copy with large dataset
	backupPath := filepath.Join(tempDir, "large_backup")
	err = copyDatabaseData(dbPath, backupPath, NewProgressTracker(false))
	if err != nil {
		t.Fatalf("Large dataset copy failed: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Large backup directory not created: %s", backupPath)
	}
}

// Test log file processing
func TestProcessLogFile(t *testing.T) {
	logFiles := setupTestLogFiles(t)

	for _, sourceLog := range logFiles {
		targetPath := filepath.Join(t.TempDir(), "log_backup")

		err := processLogFile(sourceLog, targetPath)
		if err != nil {
			t.Fatalf("Failed to process log file %s: %v", sourceLog, err)
		}

		// Verify backup file exists
		backupFile := filepath.Join(targetPath, filepath.Base(sourceLog))
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			t.Errorf("Log backup file not created: %s", backupFile)
		}

		// Verify content is preserved
		originalContent, err := os.ReadFile(sourceLog)
		if err != nil {
			t.Fatalf("Failed to read original log file: %v", err)
		}

		backupContent, err := os.ReadFile(backupFile)
		if err != nil {
			t.Fatalf("Failed to read backup log file: %v", err)
		}

		if string(originalContent) != string(backupContent) {
			t.Errorf("Log file content mismatch for %s", filepath.Base(sourceLog))
		}
	}
}

// Test log file detection
func TestLogFileDetection(t *testing.T) {
	logFiles := setupTestLogFiles(t)

	for _, logFile := range logFiles {
		if !isLogFile(logFile) {
			t.Errorf("Failed to detect log file: %s", logFile)
		}

		if detectDatabaseType(logFile) != DatabaseTypeLogFile {
			t.Errorf("Database type detection failed for log file: %s", logFile)
		}
	}

	// Test non-log files
	tempDir := t.TempDir()
	nonLogFiles := []string{
		filepath.Join(tempDir, "config.json"),
		filepath.Join(tempDir, "readme.md"),
		filepath.Join(tempDir, "script.sh"),
	}

	for _, file := range nonLogFiles {
		err := os.WriteFile(file, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if isLogFile(file) {
			t.Errorf("Incorrectly detected non-log file as log: %s", file)
		}
	}
}

// Test multiple source directories
func TestMultipleSourceDirectories(t *testing.T) {
	sourceDirs := setupMultipleTestDirectories(t)
	backupDir := filepath.Join(t.TempDir(), "multi_backup")

	config := &Config{
		SourcePaths:  sourceDirs,
		BackupPath:   backupDir,
		BatchMode:    true,
		Method:       "copy",
		Compress:     false,
		RemoveBackup: false,
	}

	// Discover all databases from all source directories
	allDatabases := []DatabaseInfo{}
	for _, sourcePath := range sourceDirs {
		databases, err := discoverDatabases(config, sourcePath)
		if err != nil {
			t.Fatalf("Failed to discover databases in %s: %v", sourcePath, err)
		}

		// Add source root information
		for i := range databases {
			databases[i].SourceRoot = sourcePath
		}

		allDatabases = append(allDatabases, databases...)
	}

	// Should find databases and log files from all 3 directories
	if len(allDatabases) < 6 {
		t.Errorf("Expected at least 6 items from multiple directories, found %d", len(allDatabases))
	}

	// Verify we have items from all directories
	sourceRoots := make(map[string]bool)
	for _, db := range allDatabases {
		sourceRoots[db.SourceRoot] = true
	}

	if len(sourceRoots) != 3 {
		t.Errorf("Expected items from 3 source directories, found %d", len(sourceRoots))
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each item
	for _, db := range allDatabases {
		sourceBaseName := filepath.Base(db.SourceRoot)
		dbBackupPath := filepath.Join(backupDir, sourceBaseName, db.Name)

		switch db.Type {
		case DatabaseTypeRocksDB:
			err := processRocksDB(db.Path, dbBackupPath, config.Method, NewProgressTracker(false))
			if err != nil {
				t.Fatalf("Failed to process RocksDB %s: %v", db.Name, err)
			}
		case DatabaseTypeSQLite:
			err := processSQLiteDB(db.Path, dbBackupPath)
			if err != nil {
				t.Fatalf("Failed to process SQLite %s: %v", db.Name, err)
			}
		case DatabaseTypeLogFile:
			err := processLogFile(db.Path, dbBackupPath)
			if err != nil {
				t.Fatalf("Failed to process log file %s: %v", db.Name, err)
			}
		}
	}

	// Verify backup structure
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	// Should have subdirectories for each source
	if len(entries) != 3 {
		t.Errorf("Expected 3 source subdirectories in backup, found %d", len(entries))
	}
}

// Test include patterns for log files
func TestLogFileIncludePatterns(t *testing.T) {
	tempDir := t.TempDir()

	// Create various log files
	logFiles := []string{
		filepath.Join(tempDir, "app.log"),
		filepath.Join(tempDir, "error.log"),
		filepath.Join(tempDir, "debug.txt"),
		filepath.Join(tempDir, "access.log"),
		filepath.Join(tempDir, "info.out"),
	}

	for _, logFile := range logFiles {
		err := os.WriteFile(logFile, []byte("test log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create log file: %v", err)
		}
	}

	// Test include only .log files
	config := &Config{
		SourcePaths:    []string{tempDir},
		BatchMode:      true,
		IncludePattern: "*.log",
	}

	databases, err := discoverDatabases(config, tempDir)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Should find only .log files
	logCount := 0
	for _, db := range databases {
		if db.Type == DatabaseTypeLogFile {
			if !strings.HasSuffix(db.Path, ".log") {
				t.Errorf("Include pattern failed, found non-.log file: %s", db.Path)
			}
			logCount++
		}
	}

	if logCount != 3 {
		t.Errorf("Expected 3 .log files, found %d", logCount)
	}
}

// Test full end-to-end workflow with mixed content
func TestFullWorkflowMixedContent(t *testing.T) {
	sourceDirs := setupMultipleTestDirectories(t)
	backupDir := filepath.Join(t.TempDir(), "full_backup")
	archivePath := filepath.Join(t.TempDir(), "full_archive.tar.gz")

	config := &Config{
		SourcePaths:  sourceDirs,
		BackupPath:   backupDir,
		ArchivePath:  archivePath,
		Method:       "copy",
		Compress:     true,
		RemoveBackup: true,
		BatchMode:    true,
	}

	// Simulate main function workflow
	allDatabases := []DatabaseInfo{}
	for _, sourcePath := range config.SourcePaths {
		databases, err := discoverDatabases(config, sourcePath)
		if err != nil {
			t.Fatalf("Failed to discover databases in %s: %v", sourcePath, err)
		}

		for i := range databases {
			databases[i].SourceRoot = sourcePath
		}

		allDatabases = append(allDatabases, databases...)
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each item
	for _, db := range allDatabases {
		sourceBaseName := filepath.Base(db.SourceRoot)
		dbBackupPath := filepath.Join(backupDir, sourceBaseName, db.Name)

		switch db.Type {
		case DatabaseTypeRocksDB:
			err := processRocksDB(db.Path, dbBackupPath, config.Method, NewProgressTracker(false))
			if err != nil {
				t.Fatalf("Failed to process RocksDB %s: %v", db.Name, err)
			}
		case DatabaseTypeSQLite:
			err := processSQLiteDB(db.Path, dbBackupPath)
			if err != nil {
				t.Fatalf("Failed to process SQLite %s: %v", db.Name, err)
			}
		case DatabaseTypeLogFile:
			err := processLogFile(db.Path, dbBackupPath)
			if err != nil {
				t.Fatalf("Failed to process log file %s: %v", db.Name, err)
			}
		}
	}

	// Compress backup
	err := compressDirectory(backupDir, archivePath)
	if err != nil {
		t.Fatalf("Failed to compress backup: %v", err)
	}

	// Verify archive exists
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatal("Archive file not created")
	}

	// Verify archive contents
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar header: %v", err)
		}

		if !header.FileInfo().IsDir() {
			fileCount++
		}
	}

	if fileCount == 0 {
		t.Error("Archive appears to be empty")
	}

	// Remove backup directory (simulate config.RemoveBackup)
	err = os.RemoveAll(backupDir)
	if err != nil {
		t.Fatalf("Failed to remove backup directory: %v", err)
	}

	// Verify backup directory is gone
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Error("Backup directory should have been removed")
	}
}

// Test different RocksDB backup methods
func TestRocksDBBackupMethods(t *testing.T) {
	// Use existing test database
	testDB := "testdata/dir1/app.db"

	// Verify the test database exists
	if _, err := os.Stat(testDB); os.IsNotExist(err) {
		t.Skipf("Test database %s does not exist, skipping test", testDB)
	}

	methods := []string{"checkpoint", "backup", "copy"}

	for _, method := range methods {
		t.Run(fmt.Sprintf("method_%s", method), func(t *testing.T) {
			// Create target directory for this method
			targetDir := filepath.Join(os.TempDir(), fmt.Sprintf("test_backup_%s_%d", method, time.Now().Unix()))
			defer os.RemoveAll(targetDir)

			// Create progress tracker (disabled for testing)
			progress := NewProgressTracker(false)

			// Test the backup method
			err := processRocksDB(testDB, targetDir, method, progress)
			if err != nil {
				t.Fatalf("Failed to backup using %s method: %v", method, err)
			}

			// Verify backup exists
			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				t.Fatalf("Backup directory not created for method %s", method)
			}

			// Verify backup contains files (all methods should create some files)
			files, err := os.ReadDir(targetDir)
			if err != nil {
				t.Fatalf("Failed to read backup directory for method %s: %v", method, err)
			}
			if len(files) == 0 {
				t.Errorf("Method %s created empty backup directory", method)
			}

			t.Logf("Method %s completed successfully, created %d files/directories", method, len(files))
		})
	}
}

// Test JSON configuration file functionality
func TestJSONConfiguration(t *testing.T) {
	// Create a temporary config file
	configFile := filepath.Join(os.TempDir(), "test_config.json")
	defer os.Remove(configFile)

	// Test configuration
	testConfig := &Config{
		SourcePaths:       []string{"testdata/dir1", "testdata/dir2"},
		BackupPath:        "test-backup",
		ArchivePath:       "test-archive.tar.gz",
		Method:            "checkpoint",
		Compress:          true,
		RemoveBackup:      false,
		BatchMode:         true,
		IncludePattern:    "*.db,*.log",
		ExcludePattern:    "*temp*",
		ShowProgress:      false,
		Filter:            "",
		CompressionFormat: "gzip",
	}

	// Test saving config to JSON
	err := SaveConfigToJSON(testConfig, configFile)
	if err != nil {
		t.Fatalf("Failed to save config to JSON: %v", err)
	}

	// Test loading config from JSON
	loadedConfig, err := LoadConfigFromJSON(configFile)
	if err != nil {
		t.Fatalf("Failed to load config from JSON: %v", err)
	}

	// Verify loaded config matches original
	if !reflect.DeepEqual(testConfig.SourcePaths, loadedConfig.SourcePaths) {
		t.Errorf("SourcePaths mismatch: expected %v, got %v", testConfig.SourcePaths, loadedConfig.SourcePaths)
	}
	if testConfig.Method != loadedConfig.Method {
		t.Errorf("Method mismatch: expected %s, got %s", testConfig.Method, loadedConfig.Method)
	}
	if testConfig.CompressionFormat != loadedConfig.CompressionFormat {
		t.Errorf("CompressionFormat mismatch: expected %s, got %s", testConfig.CompressionFormat, loadedConfig.CompressionFormat)
	}
	if testConfig.ShowProgress != loadedConfig.ShowProgress {
		t.Errorf("ShowProgress mismatch: expected %t, got %t", testConfig.ShowProgress, loadedConfig.ShowProgress)
	}
}

// Test configuration merging (JSON config + command line flags)
func TestConfigMerging(t *testing.T) {
	// JSON config
	jsonConfig := &Config{
		SourcePaths:       []string{"json-source1", "json-source2"},
		Method:            "backup",
		Compress:          false,
		ShowProgress:      false,
		CompressionFormat: "zstd",
	}

	// Flag config (command line overrides)
	flagConfig := &Config{
		SourcePaths:       []string{"flag-source1"},
		Method:            "checkpoint", // Override
		Compress:          true,         // Override
		ShowProgress:      true,         // Override
		CompressionFormat: "gzip",       // Override
		IncludePattern:    "*.db",       // New value
	}

	// Merge configs
	merged := MergeConfigs(jsonConfig, flagConfig)

	// Verify flags override JSON config
	if !reflect.DeepEqual(merged.SourcePaths, flagConfig.SourcePaths) {
		t.Errorf("SourcePaths not overridden: expected %v, got %v", flagConfig.SourcePaths, merged.SourcePaths)
	}
	if merged.Method != flagConfig.Method {
		t.Errorf("Method not overridden: expected %s, got %s", flagConfig.Method, merged.Method)
	}
	if merged.Compress != flagConfig.Compress {
		t.Errorf("Compress not overridden: expected %t, got %t", flagConfig.Compress, merged.Compress)
	}
	if merged.ShowProgress != flagConfig.ShowProgress {
		t.Errorf("ShowProgress not overridden: expected %t, got %t", flagConfig.ShowProgress, merged.ShowProgress)
	}
	if merged.CompressionFormat != flagConfig.CompressionFormat {
		t.Errorf("CompressionFormat not overridden: expected %s, got %s", flagConfig.CompressionFormat, merged.CompressionFormat)
	}
	if merged.IncludePattern != flagConfig.IncludePattern {
		t.Errorf("IncludePattern not set: expected %s, got %s", flagConfig.IncludePattern, merged.IncludePattern)
	}
}

// Test default configuration
func TestDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config.Method != "checkpoint" {
		t.Errorf("Default method should be 'checkpoint', got %s", config.Method)
	}
	if !config.Compress {
		t.Error("Default compress should be true")
	}
	if !config.RemoveBackup {
		t.Error("Default remove_backup should be true")
	}
	if !config.ShowProgress {
		t.Error("Default show_progress should be true")
	}
	if config.CompressionFormat != "gzip" {
		t.Errorf("Default compression format should be 'gzip', got %s", config.CompressionFormat)
	}
}

// Test backup verification functionality
func TestBackupVerification(t *testing.T) {
	tempDir := t.TempDir()

	// Create test databases
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backup")

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create a simple RocksDB database
	rocksDBPath := filepath.Join(sourceDir, "testdb")
	createTestRocksDB(t, rocksDBPath)

	// Create a SQLite database
	sqlitePath := filepath.Join(sourceDir, "test.sqlite")
	createTestSQLiteDB(t, sqlitePath)

	// Create a log file
	logPath := filepath.Join(sourceDir, "test.log")
	createTestLogFile(t, logPath)

	// Test RocksDB verification
	t.Run("RocksDB Verification", func(t *testing.T) {
		progress := NewProgressTracker(false)

		// First backup the database using copy method (which supports verification)
		rocksDBBackupPath := filepath.Join(backupDir, "testdb")
		err := processRocksDB(rocksDBPath, rocksDBBackupPath, "copy", progress)
		if err != nil {
			t.Fatalf("Failed to backup RocksDB: %v", err)
		}

		// Verify the backup
		dbInfo := DatabaseInfo{
			Path: rocksDBPath,
			Type: DatabaseTypeRocksDB,
			Name: "testdb",
		}

		err = VerifyBackup(dbInfo, rocksDBBackupPath, progress)
		if err != nil {
			t.Fatalf("RocksDB verification failed: %v", err)
		}
	})

	// Test SQLite verification
	t.Run("SQLite Verification", func(t *testing.T) {
		progress := NewProgressTracker(false)

		// Backup the database (creates a directory with the file inside)
		sqliteBackupDir := filepath.Join(backupDir, "test.sqlite")
		err := processSQLiteDB(sqlitePath, sqliteBackupDir)
		if err != nil {
			t.Fatalf("Failed to backup SQLite: %v", err)
		}

		// The actual backup file is inside the backup directory
		sqliteBackupPath := filepath.Join(sqliteBackupDir, filepath.Base(sqlitePath))

		// Verify the backup
		dbInfo := DatabaseInfo{
			Path: sqlitePath,
			Type: DatabaseTypeSQLite,
			Name: "test.sqlite",
		}

		err = VerifyBackup(dbInfo, sqliteBackupPath, progress)
		if err != nil {
			t.Fatalf("SQLite verification failed: %v", err)
		}
	})

	// Test log file verification
	t.Run("Log File Verification", func(t *testing.T) {
		progress := NewProgressTracker(false)

		// Backup the log file (creates a directory with the file inside)
		logBackupDir := filepath.Join(backupDir, "test.log")
		err := processLogFile(logPath, logBackupDir)
		if err != nil {
			t.Fatalf("Failed to backup log file: %v", err)
		}

		// The actual backup file is inside the backup directory
		logBackupPath := filepath.Join(logBackupDir, filepath.Base(logPath))

		// Verify the backup
		dbInfo := DatabaseInfo{
			Path: logPath,
			Type: DatabaseTypeLogFile,
			Name: "test.log",
		}

		err = VerifyBackup(dbInfo, logBackupPath, progress)
		if err != nil {
			t.Fatalf("Log file verification failed: %v", err)
		}
	})
}

// createTestRocksDB creates a test RocksDB with some sample data
func createTestRocksDB(t *testing.T, dbPath string) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		t.Fatalf("Failed to create test RocksDB: %v", err)
	}
	defer db.Close()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	// Add test data
	testData := map[string]string{
		"key1":     "value1",
		"key2":     "value2",
		"key3":     "value3",
		"config":   "test_config",
		"metadata": "test_metadata",
	}

	for key, value := range testData {
		err := db.Put(writeOpts, []byte(key), []byte(value))
		if err != nil {
			t.Fatalf("Failed to put test data: %v", err)
		}
	}
}

// Test verification with corrupted data
func TestVerificationWithCorruptedData(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backup")

	// Ensure directories exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create and backup a log file
	logPath := filepath.Join(sourceDir, "test.log")
	createTestLogFile(t, logPath)

	logBackupDir := filepath.Join(backupDir, "test.log")
	err := processLogFile(logPath, logBackupDir)
	if err != nil {
		t.Fatalf("Failed to backup log file: %v", err)
	}

	// The actual backup file is inside the backup directory
	logBackupPath := filepath.Join(logBackupDir, filepath.Base(logPath))

	// Corrupt the backup
	corruptedData := []byte("This is corrupted data")
	err = os.WriteFile(logBackupPath, corruptedData, 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt backup file: %v", err)
	}

	// Verify should fail
	progress := NewProgressTracker(false)
	dbInfo := DatabaseInfo{
		Path: logPath,
		Type: DatabaseTypeLogFile,
		Name: "test.log",
	}

	err = VerifyBackup(dbInfo, logBackupPath, progress)
	if err == nil {
		t.Fatal("Expected verification to fail with corrupted data, but it passed")
	}

	if !strings.Contains(err.Error(), "contents do not match") {
		t.Fatalf("Expected 'contents do not match' error, got: %v", err)
	}
}

// Test JSON configuration with verification
func TestJSONConfigurationWithVerification(t *testing.T) {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "config.json")

	// Create config with verification enabled
	config := &Config{
		SourcePaths:       []string{"/test/path"},
		BackupPath:        "backup",
		ArchivePath:       "archive.tar.gz",
		Method:            "checkpoint",
		Compress:          true,
		RemoveBackup:      true,
		BatchMode:         false,
		IncludePattern:    "*.db",
		ExcludePattern:    "*temp*",
		ShowProgress:      true,
		Filter:            "",
		CompressionFormat: "gzip",
		Verify:            true,
	}

	// Save config
	err := SaveConfigToJSON(config, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := LoadConfigFromJSON(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all fields including Verify
	if !loadedConfig.Verify {
		t.Error("Expected Verify to be true")
	}

	if loadedConfig.Method != "checkpoint" {
		t.Errorf("Expected method 'checkpoint', got '%s'", loadedConfig.Method)
	}

	if loadedConfig.CompressionFormat != "gzip" {
		t.Errorf("Expected compression format 'gzip', got '%s'", loadedConfig.CompressionFormat)
	}
}

// Helper function to create a test SQLite database
func createTestSQLiteDB(t *testing.T, dbPath string) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create a test table and insert some data
	_, err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	for i := 1; i <= 10; i++ {
		_, err = db.Exec("INSERT INTO test_table (name, value) VALUES (?, ?)",
			fmt.Sprintf("test_%d", i), i*100)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}
}

// Helper function to create a test log file
func createTestLogFile(t *testing.T, logPath string) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	logContent := `2024-01-01 00:00:01 INFO Application started
2024-01-01 00:00:02 DEBUG Initializing database connection
2024-01-01 00:00:03 INFO Database connection established
2024-01-01 00:00:04 WARN Cache miss for key: user_123
2024-01-01 00:00:05 ERROR Failed to process request: timeout
2024-01-01 00:00:06 INFO Request processed successfully
`

	err := os.WriteFile(logPath, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
}

// Test default configuration file discovery
func TestDefaultConfigDiscovery(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: Failed to restore original directory: %v", err)
		}
	}()

	// Change to temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("No Default Config Found", func(t *testing.T) {
		configPath := FindDefaultConfig()
		if configPath != "" {
			t.Errorf("Expected no config found, but got: %s", configPath)
		}
	})

	t.Run("Find archiveFiles.conf in Current Directory", func(t *testing.T) {
		// Create a config file in current directory
		testConfig := &Config{
			SourcePaths: []string{"/test/path"},
			Method:      "checkpoint",
			Verify:      true,
		}

		configPath := "archiveFiles.conf"
		err := SaveConfigToJSON(testConfig, configPath)
		if err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		foundPath := FindDefaultConfig()
		expectedPath := "archiveFiles.conf"
		if foundPath != expectedPath {
			t.Errorf("Expected config path '%s', got '%s'", expectedPath, foundPath)
		}

		// Verify the config can be loaded
		loadedConfig, err := LoadConfigFromJSON(foundPath)
		if err != nil {
			t.Fatalf("Failed to load found config: %v", err)
		}

		if !loadedConfig.Verify {
			t.Error("Expected Verify to be true in loaded config")
		}

		os.Remove(configPath)
	})

	t.Run("Find Config in config/ Subdirectory", func(t *testing.T) {
		// Create config subdirectory
		configDir := filepath.Join(tempDir, "config")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		// Create config file in subdirectory
		testConfig := &Config{
			SourcePaths:       []string{"/test/path"},
			Method:            "backup",
			CompressionFormat: "zstd",
		}

		configPath := filepath.Join(configDir, "archiveFiles.json")
		err = SaveConfigToJSON(testConfig, configPath)
		if err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		foundPath := FindDefaultConfig()
		expectedPath := "config/archiveFiles.json"
		if foundPath != expectedPath {
			t.Errorf("Expected config path '%s', got '%s'", expectedPath, foundPath)
		}

		// Verify the config can be loaded
		loadedConfig, err := LoadConfigFromJSON(foundPath)
		if err != nil {
			t.Fatalf("Failed to load found config: %v", err)
		}

		if loadedConfig.Method != "backup" {
			t.Errorf("Expected method 'backup', got '%s'", loadedConfig.Method)
		}

		if loadedConfig.CompressionFormat != "zstd" {
			t.Errorf("Expected compression format 'zstd', got '%s'", loadedConfig.CompressionFormat)
		}

		os.RemoveAll(configDir)
	})

	t.Run("Precedence Order - Current Directory Wins", func(t *testing.T) {
		// Create config in subdirectory first
		configDir := filepath.Join(tempDir, "config")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create config directory: %v", err)
		}

		subdirConfig := &Config{
			SourcePaths: []string{"/subdir/path"},
			Method:      "backup",
		}

		subdirConfigPath := filepath.Join(configDir, "archiveFiles.conf")
		err = SaveConfigToJSON(subdirConfig, subdirConfigPath)
		if err != nil {
			t.Fatalf("Failed to create subdir config: %v", err)
		}

		// Create config in current directory (should take precedence)
		currentDirConfig := &Config{
			SourcePaths: []string{"/current/path"},
			Method:      "checkpoint",
		}

		currentConfigPath := "archiveFiles.conf"
		err = SaveConfigToJSON(currentDirConfig, currentConfigPath)
		if err != nil {
			t.Fatalf("Failed to create current dir config: %v", err)
		}

		// Should find the current directory config first
		foundPath := FindDefaultConfig()
		expectedPath := "archiveFiles.conf"
		if foundPath != expectedPath {
			t.Errorf("Expected current dir config '%s', got '%s'", expectedPath, foundPath)
		}

		// Verify it's the current directory config
		loadedConfig, err := LoadConfigFromJSON(foundPath)
		if err != nil {
			t.Fatalf("Failed to load found config: %v", err)
		}

		if loadedConfig.Method != "checkpoint" {
			t.Errorf("Expected method 'checkpoint' from current dir, got '%s'", loadedConfig.Method)
		}

		// Cleanup
		os.Remove(currentConfigPath)
		os.RemoveAll(configDir)
	})

	t.Run("Invalid JSON File Skipped", func(t *testing.T) {
		// Create an invalid JSON file
		invalidConfigPath := "archiveFiles.conf"
		err := os.WriteFile(invalidConfigPath, []byte("invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid config: %v", err)
		}

		// Should not find any config (invalid one is skipped)
		foundPath := FindDefaultConfig()
		if foundPath != "" {
			t.Errorf("Expected no config found due to invalid JSON, but got: %s", foundPath)
		}

		os.Remove(invalidConfigPath)
	})
}

// Test default configuration file generation
func TestGenerateDefaultConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: Failed to restore original directory: %v", err)
		}
	}()

	// Change to temp directory for testing
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Run("Generate Default Config Successfully", func(t *testing.T) {
		err := GenerateDefaultConfigFile()
		if err != nil {
			t.Fatalf("Failed to generate default config: %v", err)
		}

		// Check if file was created
		configPath := "archiveFiles.conf"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Default config file was not created")
		}

		// Verify the config content
		loadedConfig, err := LoadConfigFromJSON(configPath)
		if err != nil {
			t.Fatalf("Failed to load generated config: %v", err)
		}

		// Check some default values
		if loadedConfig.Method != "checkpoint" {
			t.Errorf("Expected default method 'checkpoint', got '%s'", loadedConfig.Method)
		}

		if !loadedConfig.Compress {
			t.Error("Expected default Compress to be true")
		}

		if loadedConfig.CompressionFormat != "gzip" {
			t.Errorf("Expected default compression format 'gzip', got '%s'", loadedConfig.CompressionFormat)
		}

		if loadedConfig.Verify {
			t.Error("Expected default Verify to be false")
		}

		if loadedConfig.IncludePattern != "*.db,*.sqlite,*.sqlite3,*.log" {
			t.Errorf("Expected default include pattern, got '%s'", loadedConfig.IncludePattern)
		}

		os.Remove(configPath)
	})

	t.Run("Fail When Config Already Exists", func(t *testing.T) {
		configPath := "archiveFiles.conf"

		// Create existing file
		err := os.WriteFile(configPath, []byte("existing content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		// Should fail to generate
		err = GenerateDefaultConfigFile()
		if err == nil {
			t.Fatal("Expected error when config file already exists")
		}

		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}

		os.Remove(configPath)
	})
}
