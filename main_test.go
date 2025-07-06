package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	// Create some non-database files
	nonDBFile := filepath.Join(tempDir, "readme.txt")
	err := os.WriteFile(nonDBFile, []byte("This is not a database"), 0644)
	if err != nil {
		t.Fatalf("Failed to create non-DB file: %v", err)
	}

	return tempDir
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
		SourcePath: testDir,
		BatchMode:  true,
	}

	databases, err := discoverDatabases(config)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Should find 4 databases (2 RocksDB + 2 SQLite)
	if len(databases) != 4 {
		t.Errorf("Expected 4 databases, found %d", len(databases))
	}

	// Count by type
	rocksCount := 0
	sqliteCount := 0
	for _, db := range databases {
		switch db.Type {
		case DatabaseTypeRocksDB:
			rocksCount++
		case DatabaseTypeSQLite:
			sqliteCount++
		}
	}

	if rocksCount != 2 {
		t.Errorf("Expected 2 RocksDB databases, found %d", rocksCount)
	}
	if sqliteCount != 2 {
		t.Errorf("Expected 2 SQLite databases, found %d", sqliteCount)
	}
}

// Test file pattern filtering
func TestFilePatternFiltering(t *testing.T) {
	testDir := setupTestDirectory(t)

	// Test include pattern
	config := &Config{
		SourcePath:     testDir,
		BatchMode:      true,
		IncludePattern: "*.db",
	}

	databases, err := discoverDatabases(config)
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
		SourcePath:     testDir,
		BatchMode:      true,
		ExcludePattern: "*.sqlite",
	}

	databases, err = discoverDatabases(config)
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
		SourcePath:   testDir,
		BackupPath:   backupDir,
		BatchMode:    true,
		Method:       "copy",
		Compress:     false,
		RemoveBackup: false,
	}

	// Discover databases
	databases, err := discoverDatabases(config)
	if err != nil {
		t.Fatalf("Failed to discover databases: %v", err)
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each database
	for _, db := range databases {
		dbBackupPath := filepath.Join(backupDir, db.Name)

		switch db.Type {
		case DatabaseTypeRocksDB:
			err = processRocksDB(db.Path, dbBackupPath, config.Method)
		case DatabaseTypeSQLite:
			err = processSQLiteDB(db.Path, dbBackupPath)
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

	err := copyDatabaseData(sourceDB, targetDB)
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
	os.MkdirAll(subDir, 0755)

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
	err := compressDirectory(testDir, archivePath)
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
	err := copyDatabaseData("/non/existent/path", t.TempDir())
	if err == nil {
		t.Error("Expected error for non-existent source database")
	}

	// Test with invalid backup path
	sourceDB := setupTestDB(t)
	err = copyDatabaseData(sourceDB, "/invalid/path/that/cannot/be/created")
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
	err := copyDatabaseData(sourceDB, backupPath)
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
		err := copyDatabaseData(sourceDB, backupPath)
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
		err := copyDatabaseData(sourceDB, targetDB)
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
	err = copyDatabaseData(dbPath, backupPath)
	if err != nil {
		t.Fatalf("Large dataset copy failed: %v", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Large backup directory not created: %s", backupPath)
	}
}
