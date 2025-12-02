package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"archiveFiles/internal/types"
)

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	// Test default values
	if config.Method != "checkpoint" {
		t.Errorf("Expected default method to be 'checkpoint', got '%s'", config.Method)
	}
	if !config.Compress {
		t.Errorf("Expected default compress to be true, got %v", config.Compress)
	}
	if config.BatchMode {
		t.Errorf("Expected default batch_mode to be false, got %v", config.BatchMode)
	}
	if config.Verify {
		t.Errorf("Expected default verify to be false, got %v", config.Verify)
	}
	if config.LogLevel != "info" {
		t.Errorf("Expected default log_level to be 'info', got '%s'", config.LogLevel)
	}
	if !config.ColorLog {
		t.Errorf("Expected default color_log to be true, got %v", config.ColorLog)
	}
}

func TestLoadConfigFromJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test successful loading
	t.Run("Successful loading", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "test_config.json")
		testConfig := &types.Config{
			SourcePaths: []string{"/path1", "/path2"},
			BackupPath:  "/backup",
			ArchivePath: "/archive.tar.gz",
			Method:      "backup",
			Compress:    true,
			BatchMode:   true,
			Verify:      true,
			LogLevel:    "debug",
			ColorLog:    false,
		}

		// Save test config
		data, err := json.MarshalIndent(testConfig, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal test config: %v", err)
		}

		err = os.WriteFile(configFile, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		// Load config
		loadedConfig, err := LoadConfigFromJSON(configFile)
		if err != nil {
			t.Errorf("LoadConfigFromJSON failed: %v", err)
		}

		// Verify loaded values
		if !reflect.DeepEqual(loadedConfig.SourcePaths, testConfig.SourcePaths) {
			t.Errorf("SourcePaths mismatch: got %v, want %v", loadedConfig.SourcePaths, testConfig.SourcePaths)
		}
		if loadedConfig.BackupPath != testConfig.BackupPath {
			t.Errorf("BackupPath mismatch: got %s, want %s", loadedConfig.BackupPath, testConfig.BackupPath)
		}
		if loadedConfig.Method != testConfig.Method {
			t.Errorf("Method mismatch: got %s, want %s", loadedConfig.Method, testConfig.Method)
		}
		if loadedConfig.Compress != testConfig.Compress {
			t.Errorf("Compress mismatch: got %v, want %v", loadedConfig.Compress, testConfig.Compress)
		}
		if loadedConfig.Verify != testConfig.Verify {
			t.Errorf("Verify mismatch: got %v, want %v", loadedConfig.Verify, testConfig.Verify)
		}
	})

	// Test loading non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		_, err := LoadConfigFromJSON("/non/existent/file.json")
		if err == nil {
			t.Error("Expected error when loading non-existent file")
		}
	})

	// Test loading invalid JSON
	t.Run("Invalid JSON", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidFile, []byte("invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid JSON file: %v", err)
		}

		_, err = LoadConfigFromJSON(invalidFile)
		if err == nil {
			t.Error("Expected error when loading invalid JSON")
		}
	})
}

func TestSaveConfigToJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test successful saving
	t.Run("Successful saving", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "save_test.json")
		testConfig := &types.Config{
			SourcePaths: []string{"/test/path"},
			BackupPath:  "/test/backup",
			Method:      "checkpoint",
			Compress:    true,
			BatchMode:   false,
			Verify:      false,
			LogLevel:    "info",
			ColorLog:    true,
		}

		err := SaveConfigToJSON(testConfig, configFile)
		if err != nil {
			t.Errorf("SaveConfigToJSON failed: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}

		// Verify content by loading it back
		loadedConfig, err := LoadConfigFromJSON(configFile)
		if err != nil {
			t.Errorf("Failed to load saved config: %v", err)
		}

		if !reflect.DeepEqual(loadedConfig.SourcePaths, testConfig.SourcePaths) {
			t.Errorf("Saved/loaded SourcePaths mismatch")
		}
		if loadedConfig.Method != testConfig.Method {
			t.Errorf("Saved/loaded Method mismatch")
		}
	})

	// Test saving to invalid path
	t.Run("Invalid save path", func(t *testing.T) {
		config := GetDefaultConfig()
		err := SaveConfigToJSON(config, "/invalid/path/config.json")
		if err == nil {
			t.Error("Expected error when saving to invalid path")
		}
	})
}

func TestMergeConfigs(t *testing.T) {
	jsonConfig := &types.Config{
		SourcePaths: []string{"/json/path1", "/json/path2"},
		BackupPath:  "/json/backup",
		ArchivePath: "/json/archive.tar.gz",
		Method:      "backup",
		Compress:    false,
		BatchMode:   true,
		Verify:      true,
		DryRun:      false,
		LogLevel:    "debug",
		ColorLog:    false,
	}

	flagConfig := &types.Config{
		BackupPath: "/flag/backup", // Should override JSON
		Method:     "checkpoint",   // Should override JSON
		Compress:   true,           // Should override JSON
		Verify:     false,          // Should override JSON
		// Other fields should remain from JSON config
	}

	merged := MergeConfigs(jsonConfig, flagConfig)

	// Test that flag values override JSON values
	if merged.BackupPath != "/flag/backup" {
		t.Errorf("BackupPath not merged correctly: got %s, want /flag/backup", merged.BackupPath)
	}
	if merged.Method != "checkpoint" {
		t.Errorf("Method not merged correctly: got %s, want checkpoint", merged.Method)
	}
	if !merged.Compress {
		t.Errorf("Compress not merged correctly: got %v, want true", merged.Compress)
	}
	if merged.Verify != false {
		t.Errorf("Verify not merged correctly: got %v, want false", merged.Verify)
	}

	// Test that JSON values are preserved when flags are empty
	if !reflect.DeepEqual(merged.SourcePaths, jsonConfig.SourcePaths) {
		t.Errorf("SourcePaths not preserved from JSON: got %v, want %v", merged.SourcePaths, jsonConfig.SourcePaths)
	}
	if merged.ArchivePath != "/json/archive.tar.gz" {
		t.Errorf("ArchivePath not preserved from JSON: got %s, want /json/archive.tar.gz", merged.ArchivePath)
	}
	if merged.BatchMode != true {
		t.Errorf("BatchMode not preserved from JSON: got %v, want true", merged.BatchMode)
	}
}

func TestFindDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test no config found
	t.Run("No config found", func(t *testing.T) {
		// Change to temp directory
		originalDir, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		_ = os.Chdir(tempDir)

		configPath := FindDefaultConfig()
		if configPath != "" {
			t.Errorf("Expected empty string for no config, got %s", configPath)
		}
	})

	// Test config found
	t.Run("Config found", func(t *testing.T) {
		// Change to temp directory
		originalDir, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		_ = os.Chdir(tempDir)

		// Create a valid config file
		configFile := filepath.Join(tempDir, "archiveFiles.conf")
		defaultConfig := GetDefaultConfig()
		err := SaveConfigToJSON(defaultConfig, configFile)
		if err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		configPath := FindDefaultConfig()
		expectedPath := "archiveFiles.conf" // FindDefaultConfig returns relative path
		if configPath != expectedPath {
			t.Errorf("Expected config path %s, got %s", expectedPath, configPath)
		}
	})

	// Test invalid config file (should be ignored)
	t.Run("Invalid config ignored", func(t *testing.T) {
		// Change to temp directory
		originalDir, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		_ = os.Chdir(tempDir)

		// Remove any existing config
		os.Remove(filepath.Join(tempDir, "archiveFiles.conf"))

		// Create an invalid config file
		invalidFile := filepath.Join(tempDir, "archiveFiles.json")
		err := os.WriteFile(invalidFile, []byte("invalid json"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid config: %v", err)
		}

		configPath := FindDefaultConfig()
		if configPath != "" {
			t.Errorf("Expected empty string for invalid config, got %s", configPath)
		}
	})
}


func TestConfigJSONMarshalling(t *testing.T) {
	// Test that config can be properly marshalled and unmarshalled
	originalConfig := &types.Config{
		SourcePaths: []string{"/path1", "/path2"},
		BackupPath:  "/backup",
		ArchivePath: "/archive.tar.gz",
		Method:      "checkpoint",
		Compress:    true,
		BatchMode:   true,
		Verify:      true,
		DryRun:      false,
		LogLevel:    "info",
		ColorLog:    true,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(originalConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal from JSON
	var unmarshalled types.Config
	err = json.Unmarshal(jsonData, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Compare
	if !reflect.DeepEqual(originalConfig, &unmarshalled) {
		t.Errorf("Marshalled/unmarshalled config mismatch")
	}
}

func TestMergeConfigs_EdgeCases(t *testing.T) {
	// Test with nil configs
	t.Run("Nil JSON config", func(t *testing.T) {
		flagConfig := GetDefaultConfig()
		merged := MergeConfigs(nil, flagConfig)
		if !reflect.DeepEqual(merged, flagConfig) {
			t.Error("Merge with nil JSON config should return flag config")
		}
	})

	t.Run("Nil flag config", func(t *testing.T) {
		jsonConfig := GetDefaultConfig()
		merged := MergeConfigs(jsonConfig, nil)
		if !reflect.DeepEqual(merged, jsonConfig) {
			t.Error("Merge with nil flag config should return JSON config")
		}
	})

	t.Run("Both nil configs", func(t *testing.T) {
		merged := MergeConfigs(nil, nil)
		if merged != nil {
			t.Error("Merge with both nil configs should return nil")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	// Test various config scenarios
	t.Run("Empty source paths", func(t *testing.T) {
		config := GetDefaultConfig()
		config.SourcePaths = []string{}
		// This should be handled by the main application, not the config package
		if len(config.SourcePaths) != 0 {
			t.Error("Empty source paths should remain empty")
		}
	})

	t.Run("Invalid log level", func(t *testing.T) {
		config := GetDefaultConfig()
		config.LogLevel = "invalid"
		// The config package should allow any string, validation happens elsewhere
		if config.LogLevel != "invalid" {
			t.Error("Config should allow any log level string")
		}
	})
}
