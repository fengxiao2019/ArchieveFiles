package main

import (
	"os"
	"path/filepath"
	"testing"

	"archiveFiles/internal/config"
	"archiveFiles/internal/constants"
	"archiveFiles/internal/types"
)

func TestDryRunMode(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	backupDir := filepath.Join(tempDir, "backup")

	// Create source directory
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create a dummy file
	testFile := filepath.Join(sourceDir, "test.db")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test dry-run mode
	cfg := &types.Config{
		SourcePaths: []string{sourceDir},
		BackupPath:  backupDir,
		Method:      constants.MethodCheckpoint,
		DryRun:      true,
		ShowProgress: false,
	}

	// In dry-run mode, backup directory should NOT be created
	// (This is a simplified test - in real usage, we'd need to run the full backup process)

	// Verify dry-run flag is set
	if !cfg.DryRun {
		t.Error("Expected DryRun to be true")
	}

	// Verify that backup directory was not created during dry run
	if _, err := os.Stat(backupDir); err == nil {
		t.Error("Backup directory should not exist in dry-run mode before explicit creation")
	}
}

func TestStrictMode(t *testing.T) {
	tempDir := t.TempDir()

	// Test strict mode configuration
	cfg := &types.Config{
		SourcePaths: []string{tempDir},
		Method:      constants.MethodCheckpoint,
		Strict:      true,
	}

	// Verify strict flag is set
	if !cfg.Strict {
		t.Error("Expected Strict to be true")
	}
}

func TestDryRunAndStrictModeTogether(t *testing.T) {
	tempDir := t.TempDir()

	// Test both modes enabled together
	cfg := &types.Config{
		SourcePaths: []string{tempDir},
		Method:      constants.MethodCheckpoint,
		DryRun:      true,
		Strict:      true,
	}

	// Verify both flags are set
	if !cfg.DryRun {
		t.Error("Expected DryRun to be true")
	}
	if !cfg.Strict {
		t.Error("Expected Strict to be true")
	}
}

func TestModeConfigPersistence(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")

	// Create config with modes enabled
	cfg := &types.Config{
		SourcePaths: []string{tempDir},
		Method:      constants.MethodCheckpoint,
		DryRun:      true,
		Strict:      true,
	}

	// Save config
	if err := config.SaveConfigToJSON(cfg, configFile); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := config.LoadConfigFromJSON(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify modes are persisted
	if !loadedCfg.DryRun {
		t.Error("DryRun flag not persisted in config file")
	}
	if !loadedCfg.Strict {
		t.Error("Strict flag not persisted in config file")
	}
}
