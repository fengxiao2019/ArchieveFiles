package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"archiveFiles/internal/types"
)

func TestDetectDatabaseType(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "discovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test RocksDB detection
	t.Run("RocksDB detection", func(t *testing.T) {
		// Create RocksDB directory structure
		rocksdbDir := filepath.Join(tempDir, "test_rocksdb")
		err := os.MkdirAll(rocksdbDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create RocksDB directory: %v", err)
		}

		// Create typical RocksDB files
		rocksdbFiles := []string{
			"CURRENT",
			"MANIFEST-000001",
			"000001.log",
			"000002.sst",
			"LOCK",
			"OPTIONS-000001",
		}

		for _, file := range rocksdbFiles {
			filePath := filepath.Join(rocksdbDir, file)
			err = os.WriteFile(filePath, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create RocksDB file %s: %v", file, err)
			}
		}

		dbType := DetectDatabaseType(rocksdbDir)
		if dbType != types.DatabaseTypeRocksDB {
			t.Errorf("Expected RocksDB type, got %s", dbType.String())
		}
	})

	// Test SQLite detection
	t.Run("SQLite detection", func(t *testing.T) {
		// Create SQLite database file
		sqliteFile := filepath.Join(tempDir, "test.db")
		err := os.WriteFile(sqliteFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file: %v", err)
		}

		dbType := DetectDatabaseType(sqliteFile)
		if dbType != types.DatabaseTypeSQLite {
			t.Errorf("Expected SQLite type, got %s", dbType.String())
		}
	})

	// Test SQLite detection with different extensions
	t.Run("SQLite different extensions", func(t *testing.T) {
		sqliteExtensions := []string{".sqlite", ".sqlite3", ".db3"}

		for _, ext := range sqliteExtensions {
			sqliteFile := filepath.Join(tempDir, "test"+ext)
			err := os.WriteFile(sqliteFile, []byte("SQLite format 3\x00"), 0644)
			if err != nil {
				t.Fatalf("Failed to create SQLite file with extension %s: %v", ext, err)
			}

			dbType := DetectDatabaseType(sqliteFile)
			if dbType != types.DatabaseTypeSQLite {
				t.Errorf("Expected SQLite type for extension %s, got %s", ext, dbType.String())
			}
		}
	})

	// Test log file detection
	t.Run("Log file detection", func(t *testing.T) {
		logExtensions := []string{".log", ".txt"}

		for _, ext := range logExtensions {
			logFile := filepath.Join(tempDir, "test"+ext)
			err := os.WriteFile(logFile, []byte("Log file content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create log file with extension %s: %v", ext, err)
			}

			dbType := DetectDatabaseType(logFile)
			if dbType != types.DatabaseTypeLogFile {
				t.Errorf("Expected LogFile type for extension %s, got %s", ext, dbType.String())
			}
		}
	})

	// Test unknown file type
	t.Run("Unknown file type", func(t *testing.T) {
		unknownFile := filepath.Join(tempDir, "test.unknown")
		err := os.WriteFile(unknownFile, []byte("Unknown content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create unknown file: %v", err)
		}

		dbType := DetectDatabaseType(unknownFile)
		if dbType != types.DatabaseTypeUnknown {
			t.Errorf("Expected Unknown type, got %s", dbType.String())
		}
	})

	// Test non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "non_existent.db")

		dbType := DetectDatabaseType(nonExistentFile)
		if dbType != types.DatabaseTypeUnknown {
			t.Errorf("Expected Unknown type for non-existent file, got %s", dbType.String())
		}
	})

	// Test empty directory
	t.Run("Empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		dbType := DetectDatabaseType(emptyDir)
		if dbType != types.DatabaseTypeUnknown {
			t.Errorf("Expected Unknown type for empty directory, got %s", dbType.String())
		}
	})

	// Test partial RocksDB structure
	t.Run("Partial RocksDB structure", func(t *testing.T) {
		partialDir := filepath.Join(tempDir, "partial_rocksdb")
		err := os.MkdirAll(partialDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create partial RocksDB directory: %v", err)
		}

		// Create only CURRENT file (insufficient for RocksDB detection)
		currentFile := filepath.Join(partialDir, "CURRENT")
		err = os.WriteFile(currentFile, []byte("MANIFEST-000001"), 0644)
		if err != nil {
			t.Fatalf("Failed to create CURRENT file: %v", err)
		}

		dbType := DetectDatabaseType(partialDir)
		if dbType != types.DatabaseTypeUnknown {
			t.Errorf("Expected Unknown type for partial RocksDB, got %s", dbType.String())
		}
	})
}

func TestDiscoverDatabases(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "discovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test single file discovery
	t.Run("Single file discovery", func(t *testing.T) {
		// Create a single SQLite file
		sqliteFile := filepath.Join(tempDir, "single.db")
		err := os.WriteFile(sqliteFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{sqliteFile},
			BatchMode:   false,
		}

		databases, err := DiscoverDatabases(config, sqliteFile)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		if len(databases) != 1 {
			t.Errorf("Expected 1 database, got %d", len(databases))
		}

		if databases[0].Type != types.DatabaseTypeSQLite {
			t.Errorf("Expected SQLite type, got %s", databases[0].Type.String())
		}

		if databases[0].Name != "single.db" {
			t.Errorf("Expected name 'single.db', got %s", databases[0].Name)
		}
	})

	// Test RocksDB directory discovery
	t.Run("RocksDB directory discovery", func(t *testing.T) {
		// Create RocksDB directory
		rocksdbDir := filepath.Join(tempDir, "rocksdb_test")
		err := os.MkdirAll(rocksdbDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create RocksDB directory: %v", err)
		}

		// Create RocksDB files
		rocksdbFiles := []string{
			"CURRENT",
			"MANIFEST-000001",
			"000001.log",
			"000002.sst",
			"LOCK",
		}

		for _, file := range rocksdbFiles {
			filePath := filepath.Join(rocksdbDir, file)
			err = os.WriteFile(filePath, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create RocksDB file %s: %v", file, err)
			}
		}

		config := &types.Config{
			SourcePaths: []string{rocksdbDir},
			BatchMode:   false,
		}

		databases, err := DiscoverDatabases(config, rocksdbDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		if len(databases) != 1 {
			t.Errorf("Expected 1 database, got %d", len(databases))
		}

		if databases[0].Type != types.DatabaseTypeRocksDB {
			t.Errorf("Expected RocksDB type, got %s", databases[0].Type.String())
		}

		if databases[0].Name != "rocksdb_test" {
			t.Errorf("Expected name 'rocksdb_test', got %s", databases[0].Name)
		}
	})

	// Test batch mode discovery
	t.Run("Batch mode discovery", func(t *testing.T) {
		// Create batch directory structure
		batchDir := filepath.Join(tempDir, "batch")
		err := os.MkdirAll(batchDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create batch directory: %v", err)
		}

		// Create multiple databases
		sqliteFile1 := filepath.Join(batchDir, "db1.sqlite")
		sqliteFile2 := filepath.Join(batchDir, "db2.db")
		logFile := filepath.Join(batchDir, "app.log")

		err = os.WriteFile(sqliteFile1, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file 1: %v", err)
		}

		err = os.WriteFile(sqliteFile2, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file 2: %v", err)
		}

		err = os.WriteFile(logFile, []byte("Log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create log file: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{batchDir},
			BatchMode:   true,
		}

		databases, err := DiscoverDatabases(config, batchDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		if len(databases) != 3 {
			t.Errorf("Expected 3 databases, got %d", len(databases))
		}

		// Verify all types are found
		typeCount := make(map[types.DatabaseType]int)
		for _, db := range databases {
			typeCount[db.Type]++
		}

		if typeCount[types.DatabaseTypeSQLite] != 2 {
			t.Errorf("Expected 2 SQLite databases, got %d", typeCount[types.DatabaseTypeSQLite])
		}

		if typeCount[types.DatabaseTypeLogFile] != 1 {
			t.Errorf("Expected 1 log file, got %d", typeCount[types.DatabaseTypeLogFile])
		}
	})

	// Test with include patterns
	t.Run("Include patterns", func(t *testing.T) {
		// Create pattern test directory
		patternDir := filepath.Join(tempDir, "pattern")
		err := os.MkdirAll(patternDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create pattern directory: %v", err)
		}

		// Create various files
		files := []string{
			"app.db",
			"cache.db",
			"debug.log",
			"config.txt",
			"readme.md",
		}

		for _, file := range files {
			filePath := filepath.Join(patternDir, file)
			content := "test content"
			if filepath.Ext(file) == ".db" {
				content = "SQLite format 3\x00"
			}
			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
		}

		config := &types.Config{
			SourcePaths:    []string{patternDir},
			BatchMode:      true,
			IncludePattern: "*.db",
		}

		databases, err := DiscoverDatabases(config, patternDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		if len(databases) != 2 {
			t.Errorf("Expected 2 databases with include pattern, got %d", len(databases))
		}

		// Verify all found files have .db extension
		for _, db := range databases {
			if filepath.Ext(db.Name) != ".db" {
				t.Errorf("Found file %s doesn't match include pattern", db.Name)
			}
		}
	})

	// Test with exclude patterns
	t.Run("Exclude patterns", func(t *testing.T) {
		// Create exclude test directory
		excludeDir := filepath.Join(tempDir, "exclude")
		err := os.MkdirAll(excludeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create exclude directory: %v", err)
		}

		// Create various files
		files := []string{
			"app.db",
			"cache.db",
			"temp.db",
			"debug.log",
		}

		for _, file := range files {
			filePath := filepath.Join(excludeDir, file)
			content := "test content"
			if filepath.Ext(file) == ".db" {
				content = "SQLite format 3\x00"
			}
			err = os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", file, err)
			}
		}

		config := &types.Config{
			SourcePaths:    []string{excludeDir},
			BatchMode:      true,
			ExcludePattern: "temp*",
		}

		databases, err := DiscoverDatabases(config, excludeDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		// Should find 3 files (excluding temp.db)
		if len(databases) != 3 {
			t.Errorf("Expected 3 databases with exclude pattern, got %d", len(databases))
		}

		// Verify temp.db is not included
		for _, db := range databases {
			if db.Name == "temp.db" {
				t.Errorf("Found excluded file temp.db")
			}
		}
	})

	// Test non-existent source path
	t.Run("Non-existent source", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "non_existent")
		config := &types.Config{
			SourcePaths: []string{nonExistentPath},
			BatchMode:   false,
		}

		_, err := DiscoverDatabases(config, nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent source path")
		}
	})

	// Test unknown single file
	t.Run("Unknown single file", func(t *testing.T) {
		unknownFile := filepath.Join(tempDir, "unknown.xyz")
		err := os.WriteFile(unknownFile, []byte("unknown content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create unknown file: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{unknownFile},
			BatchMode:   false,
		}

		_, err = DiscoverDatabases(config, unknownFile)
		if err == nil {
			t.Error("Expected error for unknown file type")
		}
	})
}

func TestCheckDatabaseLock(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "lock_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test RocksDB lock detection
	t.Run("RocksDB lock detection", func(t *testing.T) {
		rocksdbDir := filepath.Join(tempDir, "rocksdb_lock")
		err := os.MkdirAll(rocksdbDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create RocksDB directory: %v", err)
		}

		// Create RocksDB files including LOCK file
		rocksdbFiles := []string{
			"CURRENT",
			"MANIFEST-000001",
			"000001.log",
			"LOCK",
		}

		for _, file := range rocksdbFiles {
			filePath := filepath.Join(rocksdbDir, file)
			err = os.WriteFile(filePath, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create RocksDB file %s: %v", file, err)
			}
		}

		lockInfo, err := CheckDatabaseLock(rocksdbDir, types.DatabaseTypeRocksDB)
		if err != nil {
			t.Errorf("CheckDatabaseLock failed: %v", err)
		}

		// Should detect lock file
		if lockInfo == nil {
			t.Error("Expected lock info, got nil")
		} else if !lockInfo.IsLocked {
			t.Error("Expected database to be detected as locked")
		}
	})

	// Test SQLite lock detection
	t.Run("SQLite lock detection", func(t *testing.T) {
		sqliteFile := filepath.Join(tempDir, "test.db")
		err := os.WriteFile(sqliteFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file: %v", err)
		}

		// Create WAL file to simulate active SQLite
		walFile := sqliteFile + "-wal"
		err = os.WriteFile(walFile, []byte("WAL content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create WAL file: %v", err)
		}

		lockInfo, err := CheckDatabaseLock(sqliteFile, types.DatabaseTypeSQLite)
		if err != nil {
			t.Errorf("CheckDatabaseLock failed: %v", err)
		}

		// Should detect WAL file
		if lockInfo == nil {
			t.Error("Expected lock info, got nil")
		} else if !lockInfo.IsLocked {
			t.Error("Expected database to be detected as locked")
		}
	})

	// Test unlocked database
	t.Run("Unlocked database", func(t *testing.T) {
		sqliteFile := filepath.Join(tempDir, "unlocked.db")
		err := os.WriteFile(sqliteFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create SQLite file: %v", err)
		}

		lockInfo, err := CheckDatabaseLock(sqliteFile, types.DatabaseTypeSQLite)
		if err != nil {
			t.Errorf("CheckDatabaseLock failed: %v", err)
		}

		// Should not detect lock
		if lockInfo != nil && lockInfo.IsLocked {
			t.Error("Expected database to be unlocked")
		}
	})

	// Test unknown database type
	t.Run("Unknown database type", func(t *testing.T) {
		unknownFile := filepath.Join(tempDir, "unknown.xyz")
		err := os.WriteFile(unknownFile, []byte("unknown content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create unknown file: %v", err)
		}

		lockInfo, err := CheckDatabaseLock(unknownFile, types.DatabaseTypeUnknown)
		if err != nil {
			t.Errorf("CheckDatabaseLock failed: %v", err)
		}

		// Should return nil for unknown types
		if lockInfo != nil {
			t.Error("Expected nil lock info for unknown database type")
		}
	})

	// Test non-existent database
	t.Run("Non-existent database", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "non_existent.db")

		lockInfo, err := CheckDatabaseLock(nonExistentFile, types.DatabaseTypeSQLite)
		if err != nil {
			t.Errorf("CheckDatabaseLock failed: %v", err)
		}

		// Should return nil for non-existent files
		if lockInfo != nil {
			t.Error("Expected nil lock info for non-existent database")
		}
	})
}

func TestDiscoverDatabases_EdgeCases(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "discovery_edge_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with symlinks
	t.Run("Symlinks", func(t *testing.T) {
		// Create a real database file
		realFile := filepath.Join(tempDir, "real.db")
		err := os.WriteFile(realFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create real database file: %v", err)
		}

		// Create a symlink to the database
		symlinkFile := filepath.Join(tempDir, "symlink.db")
		err = os.Symlink(realFile, symlinkFile)
		if err != nil {
			t.Skipf("Skipping symlink test: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{tempDir},
			BatchMode:   true,
		}

		databases, err := DiscoverDatabases(config, tempDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		// Should find both the real file and the symlink
		if len(databases) < 1 {
			t.Errorf("Expected at least 1 database, got %d", len(databases))
		}
	})

	// Test with deeply nested directories
	t.Run("Deeply nested directories", func(t *testing.T) {
		// Create deeply nested structure
		nestedPath := filepath.Join(tempDir, "level1", "level2", "level3", "level4")
		err := os.MkdirAll(nestedPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create nested directories: %v", err)
		}

		// Create a database in the deep directory
		deepFile := filepath.Join(nestedPath, "deep.db")
		err = os.WriteFile(deepFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			t.Fatalf("Failed to create deep database file: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{tempDir},
			BatchMode:   true,
		}

		databases, err := DiscoverDatabases(config, tempDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		// Should find the deeply nested file
		// The name will be constructed from the relative path with separators replaced by underscores
		expectedName := "level1_level2_level3_level4_deep.db"
		found := false
		for _, db := range databases {
			if db.Name == expectedName {
				found = true
				break
			}
		}

		if !found {
			t.Error("Failed to find deeply nested database")
		}
	})

	// Test with very long file names
	t.Run("Long file names", func(t *testing.T) {
		// Create a file with a very long name
		longName := "very_long_database_name_" + string(make([]byte, 100))
		for i := range longName[25:] {
			longName = longName[:25+i] + "x" + longName[25+i+1:]
		}
		longName += ".db"

		longFile := filepath.Join(tempDir, longName)
		err := os.WriteFile(longFile, []byte("SQLite format 3\x00"), 0644)
		if err != nil {
			// Skip this test if the system can't create files with long names
			t.Skipf("System doesn't support long file names: %v", err)
		}

		config := &types.Config{
			SourcePaths: []string{tempDir},
			BatchMode:   true,
		}

		databases, err := DiscoverDatabases(config, tempDir)
		if err != nil {
			t.Errorf("DiscoverDatabases failed: %v", err)
		}

		// Should handle long file names
		found := false
		for _, db := range databases {
			if db.Name == longName {
				found = true
				break
			}
		}

		if !found {
			t.Error("Failed to find long-named database")
		}
	})
}
