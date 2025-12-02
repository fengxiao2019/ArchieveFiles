package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"archiveFiles/internal/constants"
	"archiveFiles/internal/types"
)

// LoadConfigFromJSON loads configuration from a JSON file
func LoadConfigFromJSON(filename string) (*types.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	config := &types.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %v", err)
	}

	return config, nil
}

// SaveConfigToJSON saves configuration to a JSON file
func SaveConfigToJSON(config *types.Config, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %v", err)
	}

	err = os.WriteFile(filename, data, constants.FilePermission)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %v", filename, err)
	}

	return nil
}

// GetDefaultConfig returns a configuration with sensible defaults
func GetDefaultConfig() *types.Config {
	return &types.Config{
		Method:    constants.MethodCheckpoint,
		Compress:  true,
		BatchMode: false,
		Verify:    false,
		LogLevel:  "info",
		ColorLog:  true,
	}
}

// MergeConfigs merges command line flags into JSON config (flags override JSON)
func MergeConfigs(jsonConfig *types.Config, flagConfig *types.Config) *types.Config {
	// Handle nil cases
	if jsonConfig == nil && flagConfig == nil {
		return nil
	}
	if jsonConfig == nil {
		return flagConfig
	}
	if flagConfig == nil {
		return jsonConfig
	}

	// Start with JSON config as base
	merged := *jsonConfig

	// Override with command line flags (only if they're not default values)
	if len(flagConfig.SourcePaths) > 0 {
		merged.SourcePaths = flagConfig.SourcePaths
	}
	if flagConfig.BackupPath != "" {
		merged.BackupPath = flagConfig.BackupPath
	}
	if flagConfig.ArchivePath != "" {
		merged.ArchivePath = flagConfig.ArchivePath
	}
	// Always override method (even if it's the default) since it's explicitly set
	merged.Method = flagConfig.Method

	// For boolean flags, we need special handling since false might be intentional
	// Override if we detect flags were explicitly set
	if flagConfig.BackupPath != "" || flagConfig.ArchivePath != "" {
		// If other flag fields are set, assume boolean flags were also explicitly set
		merged.Compress = flagConfig.Compress
		merged.Verify = flagConfig.Verify
		merged.DryRun = flagConfig.DryRun
	}

	return &merged
}

// FindDefaultConfig searches for default configuration files in standard locations
func FindDefaultConfig() string {
	// Standard configuration file names to search for
	configNames := []string{
		"archiveFiles.conf",
		"archiveFiles.json",
		".archiveFiles.conf",
		".archiveFiles.json",
	}

	// Standard search paths (in order of precedence)
	searchPaths := []string{
		".", // Current directory (highest precedence)
		"./config",
		"./configs",
		os.Getenv("HOME") + "/.config/archiveFiles", // User config directory
		os.Getenv("HOME") + "/.archiveFiles",        // User home directory
	}

	// Search each path for each config name
	for _, searchPath := range searchPaths {
		for _, configName := range configNames {
			configPath := filepath.Join(searchPath, configName)

			// Check if file exists and is readable
			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				// Verify it's a valid JSON config file
				if _, err := LoadConfigFromJSON(configPath); err == nil {
					return configPath
				}
			}
		}
	}

	return "" // No default config found
}

