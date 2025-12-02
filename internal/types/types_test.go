package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archiveFiles/internal/constants"
)

func TestDatabaseType_String(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DatabaseType
		expected string
	}{
		{"RocksDB", DatabaseTypeRocksDB, "RocksDB"},
		{"SQLite", DatabaseTypeSQLite, "SQLite"},
		{"LogFile", DatabaseTypeLogFile, "LogFile"},
		{"Unknown", DatabaseTypeUnknown, "Unknown"},
		{"Invalid", DatabaseType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dbType.String()
			if got != tt.expected {
				t.Errorf("DatabaseType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDatabaseInfo_Fields(t *testing.T) {
	dbInfo := DatabaseInfo{
		Path:       "/path/to/db",
		Type:       DatabaseTypeRocksDB,
		Name:       "test_db",
		SourceRoot: "/source",
		Size:       1024,
	}

	if dbInfo.Path != "/path/to/db" {
		t.Errorf("Expected Path to be '/path/to/db', got %s", dbInfo.Path)
	}
	if dbInfo.Type != DatabaseTypeRocksDB {
		t.Errorf("Expected Type to be DatabaseTypeRocksDB, got %v", dbInfo.Type)
	}
	if dbInfo.Name != "test_db" {
		t.Errorf("Expected Name to be 'test_db', got %s", dbInfo.Name)
	}
	if dbInfo.SourceRoot != "/source" {
		t.Errorf("Expected SourceRoot to be '/source', got %s", dbInfo.SourceRoot)
	}
	if dbInfo.Size != 1024 {
		t.Errorf("Expected Size to be 1024, got %d", dbInfo.Size)
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	config := Config{
		Method:   "checkpoint",
		Compress: true,
		Verify:   false,
	}

	if config.Method != "checkpoint" {
		t.Errorf("Expected Method to be 'checkpoint', got %s", config.Method)
	}
	if !config.Compress {
		t.Errorf("Expected Compress to be true, got %v", config.Compress)
	}
	if config.Verify {
		t.Errorf("Expected Verify to be false, got %v", config.Verify)
	}
}

func TestConfig_SourcePaths(t *testing.T) {
	config := Config{
		SourcePaths: []string{"/path1", "/path2", "/path3"},
	}

	if len(config.SourcePaths) != 3 {
		t.Errorf("Expected 3 source paths, got %d", len(config.SourcePaths))
	}

	expected := []string{"/path1", "/path2", "/path3"}
	for i, path := range config.SourcePaths {
		if path != expected[i] {
			t.Errorf("Expected source path %d to be %s, got %s", i, expected[i], path)
		}
	}
}

func TestDatabaseLockInfo_Fields(t *testing.T) {
	lockInfo := DatabaseLockInfo{
		IsLocked:    true,
		LockType:    "RocksDB LOCK file",
		ProcessInfo: "process 1234",
	}

	if !lockInfo.IsLocked {
		t.Errorf("Expected IsLocked to be true, got %v", lockInfo.IsLocked)
	}
	if lockInfo.LockType != "RocksDB LOCK file" {
		t.Errorf("Expected LockType to be 'RocksDB LOCK file', got %s", lockInfo.LockType)
	}
	if lockInfo.ProcessInfo != "process 1234" {
		t.Errorf("Expected ProcessInfo to be 'process 1234', got %s", lockInfo.ProcessInfo)
	}
}

func TestDatabaseType_Constants(t *testing.T) {
	// Test that constants have expected values
	if DatabaseTypeRocksDB != 0 {
		t.Errorf("Expected DatabaseTypeRocksDB to be 0, got %d", DatabaseTypeRocksDB)
	}
	if DatabaseTypeSQLite != 1 {
		t.Errorf("Expected DatabaseTypeSQLite to be 1, got %d", DatabaseTypeSQLite)
	}
	if DatabaseTypeLogFile != 2 {
		t.Errorf("Expected DatabaseTypeLogFile to be 2, got %d", DatabaseTypeLogFile)
	}
	if DatabaseTypeUnknown != 3 {
		t.Errorf("Expected DatabaseTypeUnknown to be 3, got %d", DatabaseTypeUnknown)
	}
}

func TestConfig_Validate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create temp source dir: %v", err)
	}

	t.Run("Valid configuration", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{sourceDir},
			BackupPath:  filepath.Join(tempDir, "backup"),
			ArchivePath: filepath.Join(tempDir, "archive.tar.gz"),
			Method:      constants.MethodCheckpoint,
			Compress:    true,
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected valid config to pass validation, got error: %v", err)
		}
	})

	t.Run("Empty source paths", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{},
			Method:      constants.MethodCheckpoint,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for empty source paths")
		}
		if !strings.Contains(err.Error(), "no source paths") {
			t.Errorf("Expected error about no source paths, got: %v", err)
		}
	})

	t.Run("Empty string in source paths", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{""},
			Method:      constants.MethodCheckpoint,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for empty source path string")
		}
		if !strings.Contains(err.Error(), "empty source path") {
			t.Errorf("Expected error about empty source path, got: %v", err)
		}
	})

	t.Run("Non-existent source path", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{filepath.Join(tempDir, "nonexistent")},
			Method:      constants.MethodCheckpoint,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for non-existent source path")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("Expected error about path not existing, got: %v", err)
		}
	})

	t.Run("Invalid backup method", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{sourceDir},
			Method:      "invalid-method",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for invalid backup method")
		}
		if !strings.Contains(err.Error(), "invalid backup method") {
			t.Errorf("Expected error about invalid method, got: %v", err)
		}
	})

	t.Run("Invalid log level", func(t *testing.T) {
		cfg := &Config{
			SourcePaths: []string{sourceDir},
			Method:      constants.MethodCheckpoint,
			LogLevel:    "invalid-level",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for invalid log level")
		}
		if !strings.Contains(err.Error(), "invalid log level") {
			t.Errorf("Expected error about invalid log level, got: %v", err)
		}
	})

	t.Run("Valid log levels", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warning", "error"}

		for _, level := range validLevels {
			cfg := &Config{
				SourcePaths: []string{sourceDir},
				Method:      constants.MethodCheckpoint,
				LogLevel:    level,
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("Expected log level=%s to be valid, got error: %v", level, err)
			}
		}
	})
}

func TestValidatePathSecurity(t *testing.T) {
	t.Run("Valid paths", func(t *testing.T) {
		validPaths := []string{
			"./data",
			"../backup",
			"/tmp/backup",
			"/home/user/data",
			"data/databases",
			"./data/../backup", // Resolves to ./backup
		}

		for _, path := range validPaths {
			err := validatePathSecurity(path)
			if err != nil {
				t.Errorf("Expected path %q to be valid, got error: %v", path, err)
			}
		}
	})

	t.Run("Empty path", func(t *testing.T) {
		err := validatePathSecurity("")
		if err == nil {
			t.Error("Expected error for empty path")
		}
		if !strings.Contains(err.Error(), "empty path") {
			t.Errorf("Expected error about empty path, got: %v", err)
		}
	})

	t.Run("Path with null bytes", func(t *testing.T) {
		err := validatePathSecurity("data\x00malicious")
		if err == nil {
			t.Error("Expected error for path with null bytes")
		}
		if !strings.Contains(err.Error(), "null bytes") {
			t.Errorf("Expected error about null bytes, got: %v", err)
		}
	})

	t.Run("Excessive path traversal", func(t *testing.T) {
		err := validatePathSecurity("../../../../../../../../etc/passwd")
		if err == nil {
			t.Error("Expected error for excessive path traversal")
		}
		if !strings.Contains(err.Error(), "excessive path traversal") {
			t.Errorf("Expected error about excessive traversal, got: %v", err)
		}
	})

	t.Run("System directory access - /etc", func(t *testing.T) {
		err := validatePathSecurity("/etc/passwd")
		if err == nil {
			t.Error("Expected error for accessing /etc")
		}
		if !strings.Contains(err.Error(), "system directory") {
			t.Errorf("Expected error about system directory, got: %v", err)
		}
	})

	t.Run("System directory access - /bin", func(t *testing.T) {
		err := validatePathSecurity("/bin/bash")
		if err == nil {
			t.Error("Expected error for accessing /bin")
		}
		if !strings.Contains(err.Error(), "system directory") {
			t.Errorf("Expected error about system directory, got: %v", err)
		}
	})

	t.Run("System directory access - /usr/bin", func(t *testing.T) {
		err := validatePathSecurity("/usr/bin/ls")
		if err == nil {
			t.Error("Expected error for accessing /usr/bin")
		}
		if !strings.Contains(err.Error(), "system directory") {
			t.Errorf("Expected error about system directory, got: %v", err)
		}
	})

	t.Run("Safe /tmp and /home paths", func(t *testing.T) {
		safePaths := []string{
			"/tmp/backup",
			"/home/user/backup",
			"/var/tmp/data",
		}

		for _, path := range safePaths {
			err := validatePathSecurity(path)
			if err != nil {
				t.Errorf("Expected path %q to be safe, got error: %v", path, err)
			}
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("Contains value", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		if !contains(slice, "banana") {
			t.Error("Expected contains to return true for 'banana'")
		}
	})

	t.Run("Does not contain value", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		if contains(slice, "orange") {
			t.Error("Expected contains to return false for 'orange'")
		}
	})

	t.Run("Empty slice", func(t *testing.T) {
		slice := []string{}
		if contains(slice, "apple") {
			t.Error("Expected contains to return false for empty slice")
		}
	})

	t.Run("Case sensitive", func(t *testing.T) {
		slice := []string{"Apple", "Banana", "Cherry"}
		if contains(slice, "apple") {
			t.Error("Expected contains to be case-sensitive")
		}
	})
}
