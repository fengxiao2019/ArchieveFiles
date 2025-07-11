package types

import (
	"testing"
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
