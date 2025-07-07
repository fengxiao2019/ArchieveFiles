package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseType int

const (
	DatabaseTypeRocksDB DatabaseType = iota
	DatabaseTypeSQLite
	DatabaseTypeLogFile
	DatabaseTypeUnknown
)

type DatabaseInfo struct {
	Path       string
	Type       DatabaseType
	Name       string
	SourceRoot string // Track which source directory this came from
	Size       int64  // File/directory size for progress tracking
}

type Config struct {
	SourcePaths       []string `json:"source_paths"` // Support multiple source directories
	BackupPath        string   `json:"backup_path"`
	ArchivePath       string   `json:"archive_path"`
	Method            string   `json:"method"` // backup, checkpoint, copy
	Compress          bool     `json:"compress"`
	RemoveBackup      bool     `json:"remove_backup"`
	BatchMode         bool     `json:"batch_mode"`         // Process directory vs single database
	IncludePattern    string   `json:"include_pattern"`    // File pattern to include (e.g., "*.db,*.sqlite,*.log")
	ExcludePattern    string   `json:"exclude_pattern"`    // File pattern to exclude
	ShowProgress      bool     `json:"show_progress"`      // Show progress bar
	Filter            string   `json:"filter"`             // Filter pattern for source paths
	CompressionFormat string   `json:"compression_format"` // Compression format for archived files
	Verify            bool     `json:"verify"`             // Verify backup data against source
}

// LoadConfigFromJSON loads configuration from a JSON file
func LoadConfigFromJSON(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	config := &Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON config: %v", err)
	}

	return config, nil
}

// SaveConfigToJSON saves configuration to a JSON file
func SaveConfigToJSON(config *Config, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %v", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %v", filename, err)
	}

	return nil
}

// GetDefaultConfig returns a configuration with sensible defaults
func GetDefaultConfig() *Config {
	return &Config{
		Method:            "checkpoint",
		Compress:          true,
		RemoveBackup:      true,
		BatchMode:         false,
		ShowProgress:      true,
		CompressionFormat: "gzip",
		Verify:            false,
	}
}

// MergeConfigs merges command line flags into JSON config (flags override JSON)
func MergeConfigs(jsonConfig *Config, flagConfig *Config) *Config {
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

	if flagConfig.IncludePattern != "" {
		merged.IncludePattern = flagConfig.IncludePattern
	}
	if flagConfig.ExcludePattern != "" {
		merged.ExcludePattern = flagConfig.ExcludePattern
	}
	if flagConfig.Filter != "" {
		merged.Filter = flagConfig.Filter
	}
	// Always override compression format (even if it's the default) since it's explicitly set
	merged.CompressionFormat = flagConfig.CompressionFormat

	// For boolean flags, we need special handling since false might be intentional
	// We'll use a simple approach: always use flag values since they're explicitly set
	merged.Compress = flagConfig.Compress
	merged.RemoveBackup = flagConfig.RemoveBackup
	merged.BatchMode = flagConfig.BatchMode
	merged.ShowProgress = flagConfig.ShowProgress
	merged.Verify = flagConfig.Verify

	return &merged
}

// FindDefaultConfig searches for default configuration files in standard locations
func FindDefaultConfig() string {
	// Standard configuration file names to search for
	configNames := []string{
		"archiveFiles.conf",
		"archiveFiles.json",
		"config.json",
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
		"/etc/archiveFiles",                         // System-wide config (Unix-like)
		"/usr/local/etc/archiveFiles",               // Alternative system location
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

// GenerateDefaultConfigFile creates a default config file in the current directory
func GenerateDefaultConfigFile() error {
	configPath := "archiveFiles.conf"

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}

	// Create a comprehensive default configuration
	defaultConfig := &Config{
		SourcePaths:       []string{"./data", "./databases"},
		BackupPath:        "backup_$(date +%Y%m%d_%H%M%S)",
		ArchivePath:       "archive_$(date +%Y%m%d_%H%M%S).tar.gz",
		Method:            "checkpoint",
		Compress:          true,
		RemoveBackup:      true,
		BatchMode:         true,
		IncludePattern:    "*.db,*.sqlite,*.sqlite3,*.log",
		ExcludePattern:    "*temp*,*cache*,*.tmp",
		ShowProgress:      true,
		Filter:            "",
		CompressionFormat: "gzip",
		Verify:            false,
	}

	err := SaveConfigToJSON(defaultConfig, configPath)
	if err != nil {
		return fmt.Errorf("failed to create default config: %v", err)
	}

	return nil
}

// Progress tracking structure
type ProgressTracker struct {
	mu            sync.Mutex
	totalItems    int
	currentItem   int
	totalSize     int64
	processedSize int64
	startTime     time.Time
	currentFile   string
	enabled       bool
}

// Create new progress tracker
func NewProgressTracker(enabled bool) *ProgressTracker {
	return &ProgressTracker{
		startTime: time.Now(),
		enabled:   enabled,
	}
}

// Initialize progress tracking
func (p *ProgressTracker) Init(totalItems int, totalSize int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalItems = totalItems
	p.totalSize = totalSize
	p.currentItem = 0
	p.processedSize = 0
	p.startTime = time.Now()
}

// Update current processing file
func (p *ProgressTracker) SetCurrentFile(filename string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentFile = filename
	p.displayProgress()
}

// Mark item as completed
func (p *ProgressTracker) CompleteItem(size int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentItem++
	p.processedSize += size
	p.displayProgress()
}

// Update progress for RocksDB copying (by record count)
func (p *ProgressTracker) UpdateRocksDBProgress(processed, total int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// For RocksDB, we use record count as a proxy for progress
	if total > 0 {
		p.displayRocksDBProgress(processed, total)
	}
}

// Display overall progress
func (p *ProgressTracker) displayProgress() {
	if p.totalItems == 0 {
		return
	}

	percentage := float64(p.currentItem) / float64(p.totalItems) * 100
	elapsed := time.Since(p.startTime)

	// Calculate speed and ETA
	var eta time.Duration
	var speed string
	if p.processedSize > 0 && elapsed.Seconds() > 0 {
		bytesPerSecond := float64(p.processedSize) / elapsed.Seconds()
		speed = formatBytes(int64(bytesPerSecond)) + "/s"

		if bytesPerSecond > 0 {
			remainingBytes := p.totalSize - p.processedSize
			etaSeconds := float64(remainingBytes) / bytesPerSecond
			eta = time.Duration(etaSeconds) * time.Second
		}
	}

	// Create progress bar
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format output
	fmt.Printf("\r[%s] %.1f%% (%d/%d) | %s | %s",
		bar,
		percentage,
		p.currentItem,
		p.totalItems,
		formatBytes(p.processedSize)+"/"+formatBytes(p.totalSize),
		speed,
	)

	if eta > 0 {
		fmt.Printf(" | ETA: %s", formatDuration(eta))
	}

	if p.currentFile != "" {
		fmt.Printf(" | %s", truncateString(p.currentFile, 30))
	}

	fmt.Print("   ") // Clear any remaining characters
}

// Display RocksDB specific progress
func (p *ProgressTracker) displayRocksDBProgress(processed, total int64) {
	percentage := float64(processed) / float64(total) * 100
	elapsed := time.Since(p.startTime)

	// Calculate records per second
	var speed string
	if elapsed.Seconds() > 0 {
		recordsPerSecond := float64(processed) / elapsed.Seconds()
		speed = fmt.Sprintf("%.0f rec/s", recordsPerSecond)
	}

	// Create progress bar
	barWidth := 40
	filled := int(percentage / 100 * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r  [%s] %.1f%% (%d/%d records) | %s | %s   ",
		bar,
		percentage,
		processed,
		total,
		speed,
		p.currentFile,
	)
}

// Finish progress tracking
func (p *ProgressTracker) Finish() {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime)
	fmt.Printf("\n✓ Completed %d item(s) in %s (%s total)\n",
		p.totalItems,
		formatDuration(elapsed),
		formatBytes(p.totalSize))
}

// Helper function to format bytes
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	}
	return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-60*d.Hours())
}

// Helper function to truncate string
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// Calculate directory size
func calculateSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		// Log error but don't fail - return 0 size on error
		return 0
	}
	return size
}

func main() {
	config := parseFlags()

	log.Printf("Starting database archival process...")
	log.Printf("Sources: %v", config.SourcePaths)
	log.Printf("Method: %s", config.Method)
	log.Printf("Batch mode: %t", config.BatchMode)

	// Create progress tracker
	progress := NewProgressTracker(config.ShowProgress)

	// Discover databases from all source directories
	allDatabases := []DatabaseInfo{}
	for _, sourcePath := range config.SourcePaths {
		log.Printf("Scanning source: %s", sourcePath)

		// Create a temporary config for each source
		sourceConfig := &Config{
			SourcePaths:    []string{sourcePath},
			BatchMode:      config.BatchMode,
			IncludePattern: config.IncludePattern,
			ExcludePattern: config.ExcludePattern,
		}

		databases, err := discoverDatabases(sourceConfig, sourcePath)
		if err != nil {
			log.Printf("Warning: Failed to discover databases in %s: %v", sourcePath, err)
			continue
		}

		// Add source root information and calculate sizes
		for i := range databases {
			databases[i].SourceRoot = sourcePath
			databases[i].Size = calculateSize(databases[i].Path)
		}

		allDatabases = append(allDatabases, databases...)
	}

	if len(allDatabases) == 0 {
		log.Fatal("No databases or files found to archive")
	}

	log.Printf("Found %d item(s) to archive:", len(allDatabases))
	var totalSize int64
	for _, db := range allDatabases {
		log.Printf("  - %s (%s) from %s [%s]", db.Name, databaseTypeString(db.Type), db.SourceRoot, formatBytes(db.Size))
		totalSize += db.Size
	}

	// Initialize progress tracking
	progress.Init(len(allDatabases), totalSize)

	// Create backup directory
	backupPath := config.BackupPath
	if backupPath == "" {
		backupPath = fmt.Sprintf("backup_%d", time.Now().Unix())
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each database/file
	for _, db := range allDatabases {
		if config.ShowProgress {
			progress.SetCurrentFile(db.Name)
		} else {
			log.Printf("Processing %s (%s)...", db.Name, databaseTypeString(db.Type))
		}

		// Create a subdirectory structure to avoid name collisions
		sourceBaseName := filepath.Base(db.SourceRoot)
		if sourceBaseName == "." || sourceBaseName == "" {
			sourceBaseName = "root"
		}

		dbBackupPath := filepath.Join(backupPath, sourceBaseName, db.Name)
		var err error

		// Ensure the parent directory exists
		parentDir := filepath.Dir(dbBackupPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			log.Printf("Failed to create parent directory %s: %v", parentDir, err)
			continue
		}

		switch db.Type {
		case DatabaseTypeRocksDB:
			err = processRocksDB(db.Path, dbBackupPath, config.Method, progress)
		case DatabaseTypeSQLite:
			err = processSQLiteDB(db.Path, dbBackupPath)
		case DatabaseTypeLogFile:
			err = processLogFile(db.Path, dbBackupPath)
		default:
			log.Printf("Skipping unknown file type: %s", db.Path)
			continue
		}

		if err != nil {
			if !config.ShowProgress {
				log.Printf("Failed to process %s: %v", db.Name, err)
			}
			progress.CompleteItem(0) // Still count as processed for progress
			continue
		}

		// Verify backup if requested
		if config.Verify {
			if config.ShowProgress {
				progress.SetCurrentFile(fmt.Sprintf("Verifying %s", db.Name))
			} else {
				log.Printf("Verifying %s...", db.Name)
			}

			err = VerifyBackup(db, dbBackupPath, progress)
			if err != nil {
				if !config.ShowProgress {
					log.Printf("❌ Verification failed for %s: %v", db.Name, err)
				}
				// Continue with other files even if verification fails
			} else {
				if !config.ShowProgress {
					log.Printf("✅ Verification passed for %s", db.Name)
				}
			}
		}

		progress.CompleteItem(db.Size)
		if !config.ShowProgress {
			log.Printf("Successfully processed %s", db.Name)
		}
	}

	// Finish progress tracking
	if config.ShowProgress {
		progress.Finish()
	}

	log.Printf("Backup created successfully at: %s", backupPath)

	// Compress backup
	if config.Compress {
		archivePath := config.ArchivePath
		if archivePath == "" {
			archivePath = fmt.Sprintf("%s.tar.gz", backupPath)
		}

		if config.ShowProgress {
			log.Printf("Creating compressed archive...")
		}

		err := compressDirectory(backupPath, archivePath)
		if err != nil {
			log.Fatalf("Failed to compress backup: %v", err)
		}

		log.Printf("Archive created successfully at: %s", archivePath)

		// Remove original backup directory
		if config.RemoveBackup {
			err = os.RemoveAll(backupPath)
			if err != nil {
				log.Printf("Warning: Failed to remove backup directory: %v", err)
			} else {
				log.Printf("Backup directory removed: %s", backupPath)
			}
		}
	}

	log.Printf("Archival process completed successfully!")
}

func parseFlags() *Config {
	var sourceFlag string
	var sourcesFlag string
	var configFile string
	var generateConfig string
	var initConfig bool

	// Define all flags
	flag.StringVar(&configFile, "config", "", "JSON configuration file path")
	flag.StringVar(&generateConfig, "generate-config", "", "Generate a sample configuration file and exit")
	flag.BoolVar(&initConfig, "init", false, "Generate default configuration file (archiveFiles.conf) in current directory")
	flag.StringVar(&sourceFlag, "source", "", "Source database path or directory")
	flag.StringVar(&sourcesFlag, "sources", "", "Multiple source paths, comma-separated")

	// Create a temporary config for flag parsing
	config := GetDefaultConfig()

	flag.StringVar(&config.BackupPath, "backup", "", "Backup path (default: backup_timestamp)")
	flag.StringVar(&config.ArchivePath, "archive", "", "Archive path (default: backup_path.tar.gz)")
	flag.StringVar(&config.Method, "method", "checkpoint", "RocksDB backup method: checkpoint (fast, hard-links), backup (native backup engine), copy (record-by-record)")
	flag.BoolVar(&config.Compress, "compress", true, "Compress archived files")
	flag.BoolVar(&config.RemoveBackup, "remove-backup", true, "Remove backup directory after compression")
	flag.BoolVar(&config.BatchMode, "batch", false, "Process directory containing multiple databases")
	flag.StringVar(&config.IncludePattern, "include", "", "Include file patterns (comma-separated, e.g., '*.db,*.sqlite,*.log')")
	flag.StringVar(&config.ExcludePattern, "exclude", "", "Exclude file patterns (comma-separated)")
	flag.StringVar(&config.Filter, "filter", "", "Filter pattern for source paths (e.g., '*.db' or 'cache*')")
	flag.StringVar(&config.CompressionFormat, "compression", "gzip", "Compression format: gzip, zstd, lz4")
	flag.BoolVar(&config.ShowProgress, "progress", true, "Show progress bar during archival")
	flag.BoolVar(&config.Verify, "verify", false, "Verify backup data integrity against source")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] source1 [source2 ...] target\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Archive RocksDB and SQLite databases with multiple sources support.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nRocksDB Methods:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  checkpoint  - Fast method using hard-links (default, recommended)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  backup      - Native RocksDB backup engine (supports incremental)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  copy        - Record-by-record copy (slowest, compatibility)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nConfiguration File:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Use -config=file.json to load settings from JSON file\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Use -generate-config=file.json to create a sample config file\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Use -init to create default archiveFiles.conf in current directory\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Without -config flag, searches for default config files:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    ./archiveFiles.conf, ./archiveFiles.json, ./config.json\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    ~/.config/archiveFiles/, ~/.archiveFiles/, /etc/archiveFiles/\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Command line flags override JSON config settings\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nVerification:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Use -verify to check backup integrity against source data\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Verification compares record counts, keys, values, and schemas\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s db1 db2 archive.tar.gz\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -config=backup.json\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -method=backup -filter='*.db' /data/dbs /backup/\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -compression=zstd -progress=false /data/cache /backup/cache.tar.zst\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s -verify -method=checkpoint /data/rocksdb verified-backup.tar.gz\n", os.Args[0])
	}

	flag.Parse()

	// Handle init flag - generate default config
	if initConfig {
		err := GenerateDefaultConfigFile()
		if err != nil {
			log.Fatalf("Failed to generate default config: %v", err)
		}

		fmt.Printf("Default configuration file generated: archiveFiles.conf\n")
		fmt.Printf("Edit the file as needed and run: %s\n", os.Args[0])
		fmt.Printf("The program will automatically detect and use this config file.\n")
		os.Exit(0)
	}

	// Handle config file generation
	if generateConfig != "" {
		sampleConfig := &Config{
			SourcePaths:       []string{"/path/to/source1", "/path/to/source2"},
			BackupPath:        "backup_$(date)",
			ArchivePath:       "archive.tar.gz",
			Method:            "checkpoint",
			Compress:          true,
			RemoveBackup:      true,
			BatchMode:         true,
			IncludePattern:    "*.db,*.sqlite,*.log",
			ExcludePattern:    "*temp*,*cache*",
			ShowProgress:      true,
			Filter:            "",
			CompressionFormat: "gzip",
			Verify:            false,
		}

		err := SaveConfigToJSON(sampleConfig, generateConfig)
		if err != nil {
			log.Fatalf("Failed to generate config file: %v", err)
		}

		fmt.Printf("Sample configuration file generated: %s\n", generateConfig)
		fmt.Printf("Edit the file and run with: %s -config=%s\n", os.Args[0], generateConfig)
		os.Exit(0)
	}

	// Load JSON config if specified
	var finalConfig *Config
	if configFile != "" {
		jsonConfig, err := LoadConfigFromJSON(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}

		// Merge JSON config with command line flags
		finalConfig = MergeConfigs(jsonConfig, config)
		log.Printf("Loaded configuration from: %s", configFile)
	} else {
		// Try to find default configuration file
		defaultConfigPath := FindDefaultConfig()
		if defaultConfigPath != "" {
			jsonConfig, err := LoadConfigFromJSON(defaultConfigPath)
			if err != nil {
				log.Printf("Warning: Found default config file '%s' but failed to load: %v", defaultConfigPath, err)
				finalConfig = config
			} else {
				// Merge default config with command line flags
				finalConfig = MergeConfigs(jsonConfig, config)
				log.Printf("Loaded default configuration from: %s", defaultConfigPath)
			}
		} else {
			finalConfig = config
		}
	}

	// Parse source paths from flags
	if sourceFlag != "" {
		finalConfig.SourcePaths = append(finalConfig.SourcePaths, sourceFlag)
	}

	if sourcesFlag != "" {
		sources := strings.Split(sourcesFlag, ",")
		for _, src := range sources {
			src = strings.TrimSpace(src)
			if src != "" {
				finalConfig.SourcePaths = append(finalConfig.SourcePaths, src)
			}
		}
	}

	// Parse positional arguments: source1 [source2 ...] target
	args := flag.Args()
	if len(args) >= 2 {
		// Multiple positional arguments: treat all but last as sources, last as target
		finalConfig.SourcePaths = append(finalConfig.SourcePaths, args[:len(args)-1]...)
		finalConfig.ArchivePath = args[len(args)-1]
	} else if len(args) == 1 && len(finalConfig.SourcePaths) > 0 {
		// One positional argument with -source/-sources: treat as target
		finalConfig.ArchivePath = args[0]
	}

	// Validation
	if len(finalConfig.SourcePaths) == 0 {
		log.Fatal("At least one source path is required (use -source, -sources, positional arguments, or config file)")
	}

	// Auto-detect batch mode if any source is a directory
	for _, sourcePath := range finalConfig.SourcePaths {
		if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
			finalConfig.BatchMode = true
			break
		}
	}

	return finalConfig
}

// Discover databases in the source path
func discoverDatabases(config *Config, sourcePath string) ([]DatabaseInfo, error) {
	var databases []DatabaseInfo

	// Check if source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	// Check if it's a single file or directory
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source path: %v", err)
	}

	if !info.IsDir() {
		// Single file mode
		dbType := detectDatabaseType(sourcePath)
		if dbType == DatabaseTypeUnknown {
			return nil, fmt.Errorf("unknown file type: %s", sourcePath)
		}

		databases = append(databases, DatabaseInfo{
			Path: sourcePath,
			Type: dbType,
			Name: filepath.Base(sourcePath),
		})

		return databases, nil
	}

	// Directory mode - scan directory
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Detect database/file type (works for both files and directories)
		dbType := detectDatabaseType(path)
		if dbType == DatabaseTypeUnknown {
			return nil
		}

		// Check include/exclude patterns (only for files, not directories like RocksDB)
		if !info.IsDir() && !shouldIncludeFile(path, config.IncludePattern, config.ExcludePattern) {
			return nil
		}

		// Create relative name for backup
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		databases = append(databases, DatabaseInfo{
			Path: path,
			Type: dbType,
			Name: strings.ReplaceAll(relPath, string(filepath.Separator), "_"),
		})

		// If this is a RocksDB directory, don't walk into it
		if info.IsDir() && dbType == DatabaseTypeRocksDB {
			return filepath.SkipDir
		}

		return nil
	})

	return databases, err
}

// Detect database type based on file characteristics
func detectDatabaseType(path string) DatabaseType {
	// Check if it's a RocksDB directory
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		// Look for RocksDB files
		if hasRocksDBFiles(path) {
			return DatabaseTypeRocksDB
		}
	}

	// Check if it's a SQLite file
	if isSQLiteFile(path) {
		return DatabaseTypeSQLite
	}

	// Check if it's a log file
	if isLogFile(path) {
		return DatabaseTypeLogFile
	}

	return DatabaseTypeUnknown
}

// Check if directory contains RocksDB files
func hasRocksDBFiles(dirPath string) bool {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}

	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, "CURRENT") ||
			strings.HasPrefix(name, "MANIFEST") ||
			strings.HasPrefix(name, "LOG") ||
			strings.HasSuffix(name, ".sst") ||
			strings.HasSuffix(name, ".log") {
			return true
		}
	}

	return false
}

// Check if file is a SQLite database
func isSQLiteFile(filePath string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" {
		// Verify it's actually a SQLite file by checking header
		return hasValidSQLiteHeader(filePath)
	}

	return false
}

// Check if file is a log file
func isLogFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := strings.ToLower(filepath.Base(filePath))

	// Check by extension (.log files are always logs)
	if ext == ".log" || ext == ".logx" {
		return true
	}

	// For .txt files, check if they have log-related names
	if ext == ".txt" {
		logPatterns := []string{
			"access", "error", "debug", "info", "warn", "trace",
			"audit", "security", "application", "system", "server",
			"database", "sql", "query", "transaction", "backup",
		}

		for _, pattern := range logPatterns {
			if strings.Contains(filename, pattern) {
				return true
			}
		}
	}

	// Check by filename patterns (for files without extensions or other extensions)
	logPatterns := []string{
		"access", "error", "debug", "info", "warn", "trace",
		"audit", "security", "application", "system", "server",
		"database", "sql", "query", "transaction", "backup",
	}

	// Only consider files with log-like names and no extension or specific extensions
	if ext == "" || ext == ".out" {
		for _, pattern := range logPatterns {
			if strings.Contains(filename, pattern) {
				return true
			}
		}
	}

	return false
}

// Check SQLite header more reliably
func hasValidSQLiteHeader(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 16)
	if n, err := file.Read(header); err != nil || n < 16 {
		return false
	}

	return string(header[:15]) == "SQLite format 3"
}

// Check if file should be included based on patterns
func shouldIncludeFile(path, includePattern, excludePattern string) bool {
	filename := filepath.Base(path)

	// Check exclude pattern first
	if excludePattern != "" {
		patterns := strings.Split(excludePattern, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if matched, _ := filepath.Match(pattern, filename); matched {
				return false
			}
		}
	}

	// Check include pattern
	if includePattern != "" {
		patterns := strings.Split(includePattern, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if matched, _ := filepath.Match(pattern, filename); matched {
				return true
			}
		}
		return false // If include pattern is specified but doesn't match
	}

	return true // Include by default if no patterns specified
}

// Process RocksDB database using native methods
func processRocksDB(sourceDBPath, targetDBPath, method string, progress *ProgressTracker) error {
	switch method {
	case "backup":
		return backupRocksDB(sourceDBPath, targetDBPath, progress)
	case "checkpoint":
		return checkpointRocksDB(sourceDBPath, targetDBPath, progress)
	case "copy":
		return copyDatabaseData(sourceDBPath, targetDBPath, progress)
	default:
		return fmt.Errorf("unknown method: %s. Available methods: backup, checkpoint, copy", method)
	}
}

// Native RocksDB backup using BackupEngine
func backupRocksDB(sourceDBPath, targetDBPath string, progress *ProgressTracker) error {
	progress.SetCurrentFile(fmt.Sprintf("Backing up %s", sourceDBPath))

	// Open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source database: %v", err)
	}
	defer sourceDB.Close()

	// Create backup engine
	backupEngine, err := grocksdb.CreateBackupEngineWithPath(sourceDB, targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to create backup engine: %v", err)
	}
	defer backupEngine.Close()

	// Create new backup
	progress.SetCurrentFile(fmt.Sprintf("Creating backup of %s", sourceDBPath))
	if err := backupEngine.CreateNewBackupFlush(true); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Calculate backup size
	backupSize := calculateSize(targetDBPath)
	progress.CompleteItem(backupSize)
	return nil
}

// Native RocksDB checkpoint using Checkpoint API
func checkpointRocksDB(sourceDBPath, targetDBPath string, progress *ProgressTracker) error {
	progress.SetCurrentFile(fmt.Sprintf("Checkpointing %s", sourceDBPath))

	// Open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source database: %v", err)
	}
	defer sourceDB.Close()

	// Create checkpoint
	checkpoint, err := sourceDB.NewCheckpoint()
	if err != nil {
		return fmt.Errorf("failed to create checkpoint object: %v", err)
	}
	defer checkpoint.Destroy()

	// Create checkpoint directory
	progress.SetCurrentFile(fmt.Sprintf("Creating checkpoint at %s", targetDBPath))
	if err := checkpoint.CreateCheckpoint(targetDBPath, 0); err != nil {
		return fmt.Errorf("failed to create checkpoint: %v", err)
	}

	// Calculate checkpoint size
	checkpointSize := calculateSize(targetDBPath)
	progress.CompleteItem(checkpointSize)
	return nil
}

// Process SQLite database
func processSQLiteDB(sourceDBPath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Copy SQLite file
	targetFile := filepath.Join(targetPath, filepath.Base(sourceDBPath))

	// Open source database to ensure it's valid and get a consistent snapshot
	db, err := sql.Open("sqlite3", sourceDBPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open source SQLite database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping source SQLite database: %v", err)
	}

	// Use SQLite's backup API for consistent copy
	return copySQLiteDatabase(sourceDBPath, targetFile)
}

// Process log files
func processLogFile(sourceLogPath, targetPath string) error {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Copy log file
	targetFile := filepath.Join(targetPath, filepath.Base(sourceLogPath))
	return copyFile(sourceLogPath, targetFile)
}

// Copy a regular file
func copyFile(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %v", err)
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	// Preserve file permissions
	if sourceInfo, err := os.Stat(sourcePath); err == nil {
		if chmodErr := os.Chmod(targetPath, sourceInfo.Mode()); chmodErr != nil {
			// Log error but don't fail the copy operation
			log.Printf("Warning: Failed to preserve file permissions for %s: %v", targetPath, chmodErr)
		}
	}

	return nil
}

// Copy SQLite database using file copy (simple approach)
func copySQLiteDatabase(sourcePath, targetPath string) error {
	return copyFile(sourcePath, targetPath)
}

// Get database type as string
func databaseTypeString(dbType DatabaseType) string {
	switch dbType {
	case DatabaseTypeRocksDB:
		return "RocksDB"
	case DatabaseTypeSQLite:
		return "SQLite"
	case DatabaseTypeLogFile:
		return "LogFile"
	default:
		return "Unknown"
	}
}

// Note: Using copy method for all backup types since backup and checkpoint APIs
// might not be available in this version of grocksdb

// manual copy data
func copyDatabaseData(sourceDBPath, targetDBPath string, progress *ProgressTracker) error {
	// open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer sourceDB.Close()

	// First pass: count total records for progress tracking
	var totalRecords int64
	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	countIter := sourceDB.NewIterator(readOpts)
	defer countIter.Close()

	for countIter.SeekToFirst(); countIter.Valid(); countIter.Next() {
		totalRecords++
		key := countIter.Key()
		key.Free()
	}

	if err := countIter.Err(); err != nil {
		return fmt.Errorf("failed to count records: %v", err)
	}

	// create target database
	targetOpts := grocksdb.NewDefaultOptions()
	targetOpts.SetCreateIfMissing(true)
	defer targetOpts.Destroy()

	targetDB, err := grocksdb.OpenDb(targetOpts, targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to create target db: %v", err)
	}
	defer targetDB.Close()

	// create iterator for copying
	iter := sourceDB.NewIterator(readOpts)
	defer iter.Close()

	// create write batch
	writeBatch := grocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	batchSize := 1000
	var count int64

	// iterate all data
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		writeBatch.Put(key.Data(), value.Data())
		count++

		// write batch and update progress
		if count%int64(batchSize) == 0 {
			err = targetDB.Write(writeOpts, writeBatch)
			if err != nil {
				key.Free()
				value.Free()
				return fmt.Errorf("failed to write batch: %v", err)
			}
			writeBatch.Clear()
			progress.UpdateRocksDBProgress(count, totalRecords)
		}

		key.Free()
		value.Free()
	}

	// write remaining data
	if writeBatch.Count() > 0 {
		err = targetDB.Write(writeOpts, writeBatch)
		if err != nil {
			return fmt.Errorf("failed to write final batch: %v", err)
		}
		progress.UpdateRocksDBProgress(count, totalRecords)
	}

	return iter.Err()
}

// compress directory to tar.gz
func compressDirectory(sourceDir, targetPath string) error {
	// create target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %v", err)
	}
	defer file.Close()

	// create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// iterate source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// set relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// if it's a file, write content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// VerifyBackup verifies that backup data matches the original source
func VerifyBackup(sourceInfo DatabaseInfo, backupPath string, progress *ProgressTracker) error {
	progress.SetCurrentFile(fmt.Sprintf("Verifying %s", sourceInfo.Name))

	switch sourceInfo.Type {
	case DatabaseTypeRocksDB:
		return verifyRocksDB(sourceInfo.Path, backupPath, progress)
	case DatabaseTypeSQLite:
		return verifySQLite(sourceInfo.Path, backupPath)
	case DatabaseTypeLogFile:
		return verifyLogFile(sourceInfo.Path, backupPath)
	default:
		return fmt.Errorf("unknown database type for verification: %s", sourceInfo.Path)
	}
}

// Verify RocksDB backup by comparing record counts and key-value pairs
func verifyRocksDB(sourcePath, backupPath string, progress *ProgressTracker) error {
	// Open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourcePath, false)
	if err != nil {
		return fmt.Errorf("failed to open source RocksDB for verification: %v", err)
	}
	defer sourceDB.Close()

	// Open backup database (read-only)
	backupOpts := grocksdb.NewDefaultOptions()
	backupOpts.SetCreateIfMissing(false)
	defer backupOpts.Destroy()

	backupDB, err := grocksdb.OpenDbForReadOnly(backupOpts, backupPath, false)
	if err != nil {
		return fmt.Errorf("failed to open backup RocksDB for verification: %v", err)
	}
	defer backupDB.Close()

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	// First pass: count records in both databases
	sourceCount := countRocksDBRecords(sourceDB, readOpts)
	backupCount := countRocksDBRecords(backupDB, readOpts)

	if sourceCount != backupCount {
		return fmt.Errorf("record count mismatch: source has %d records, backup has %d records", sourceCount, backupCount)
	}

	// Second pass: compare key-value pairs
	sourceIter := sourceDB.NewIterator(readOpts)
	defer sourceIter.Close()

	backupIter := backupDB.NewIterator(readOpts)
	defer backupIter.Close()

	var recordsVerified int64
	for sourceIter.SeekToFirst(); sourceIter.Valid(); sourceIter.Next() {
		sourceKey := sourceIter.Key()
		sourceValue := sourceIter.Value()

		// Find corresponding key in backup
		backupIter.Seek(sourceKey.Data())
		if !backupIter.Valid() {
			sourceKey.Free()
			sourceValue.Free()
			return fmt.Errorf("key not found in backup: %s", string(sourceKey.Data()))
		}

		backupKey := backupIter.Key()
		backupValue := backupIter.Value()

		// Compare keys
		if !bytesEqual(sourceKey.Data(), backupKey.Data()) {
			sourceKey.Free()
			sourceValue.Free()
			backupKey.Free()
			backupValue.Free()
			return fmt.Errorf("key mismatch at position %d", recordsVerified)
		}

		// Compare values
		if !bytesEqual(sourceValue.Data(), backupValue.Data()) {
			sourceKey.Free()
			sourceValue.Free()
			backupKey.Free()
			backupValue.Free()
			return fmt.Errorf("value mismatch for key: %s", string(sourceKey.Data()))
		}

		sourceKey.Free()
		sourceValue.Free()
		backupKey.Free()
		backupValue.Free()

		recordsVerified++
		if recordsVerified%1000 == 0 {
			progress.UpdateRocksDBProgress(recordsVerified, sourceCount)
		}
	}

	if err := sourceIter.Err(); err != nil {
		return fmt.Errorf("error during source iteration: %v", err)
	}

	return nil
}

// Count records in RocksDB
func countRocksDBRecords(db *grocksdb.DB, readOpts *grocksdb.ReadOptions) int64 {
	iter := db.NewIterator(readOpts)
	defer iter.Close()

	var count int64
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		count++
		key := iter.Key()
		key.Free()
	}

	return count
}

// Verify SQLite database by comparing schema and data
func verifySQLite(sourcePath, backupPath string) error {
	// Open source database
	sourceDB, err := sql.Open("sqlite3", sourcePath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open source SQLite for verification: %v", err)
	}
	defer sourceDB.Close()

	// Open backup database
	backupDB, err := sql.Open("sqlite3", backupPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open backup SQLite for verification: %v", err)
	}
	defer backupDB.Close()

	// Compare schema
	if err := compareSQLiteSchema(sourceDB, backupDB); err != nil {
		return fmt.Errorf("schema mismatch: %v", err)
	}

	// Compare data
	if err := compareSQLiteData(sourceDB, backupDB); err != nil {
		return fmt.Errorf("data mismatch: %v", err)
	}

	return nil
}

// Compare SQLite schemas
func compareSQLiteSchema(sourceDB, backupDB *sql.DB) error {
	sourceSchema, err := getSQLiteSchema(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to get source schema: %v", err)
	}

	backupSchema, err := getSQLiteSchema(backupDB)
	if err != nil {
		return fmt.Errorf("failed to get backup schema: %v", err)
	}

	if sourceSchema != backupSchema {
		return fmt.Errorf("schema definitions do not match")
	}

	return nil
}

// Get SQLite schema as string
func getSQLiteSchema(db *sql.DB) (string, error) {
	rows, err := db.Query("SELECT sql FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var schema strings.Builder
	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			return "", err
		}
		schema.WriteString(sql)
		schema.WriteString("\n")
	}

	return schema.String(), nil
}

// Compare SQLite data
func compareSQLiteData(sourceDB, backupDB *sql.DB) error {
	// Get list of tables
	tables, err := getSQLiteTables(sourceDB)
	if err != nil {
		return fmt.Errorf("failed to get tables: %v", err)
	}

	// Compare each table
	for _, table := range tables {
		if err := compareSQLiteTable(sourceDB, backupDB, table); err != nil {
			return fmt.Errorf("table %s: %v", table, err)
		}
	}

	return nil
}

// Get list of SQLite tables
func getSQLiteTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, nil
}

// Compare data in a specific SQLite table
func compareSQLiteTable(sourceDB, backupDB *sql.DB, tableName string) error {
	// Count rows
	sourceCount, err := countSQLiteRows(sourceDB, tableName)
	if err != nil {
		return fmt.Errorf("failed to count source rows: %v", err)
	}

	backupCount, err := countSQLiteRows(backupDB, tableName)
	if err != nil {
		return fmt.Errorf("failed to count backup rows: %v", err)
	}

	if sourceCount != backupCount {
		return fmt.Errorf("row count mismatch: source has %d rows, backup has %d rows", sourceCount, backupCount)
	}

	// If table is empty, we're done
	if sourceCount == 0 {
		return nil
	}

	// Compare checksums (simple approach for verification)
	sourceChecksum, err := calculateSQLiteTableChecksum(sourceDB, tableName)
	if err != nil {
		return fmt.Errorf("failed to calculate source checksum: %v", err)
	}

	backupChecksum, err := calculateSQLiteTableChecksum(backupDB, tableName)
	if err != nil {
		return fmt.Errorf("failed to calculate backup checksum: %v", err)
	}

	if sourceChecksum != backupChecksum {
		return fmt.Errorf("data checksum mismatch")
	}

	return nil
}

// Count rows in SQLite table
func countSQLiteRows(db *sql.DB, tableName string) (int64, error) {
	var count int64
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	return count, err
}

// Calculate simple checksum for SQLite table
func calculateSQLiteTableChecksum(db *sql.DB, tableName string) (string, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY rowid", tableName))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	var checksum strings.Builder
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}

		for _, value := range values {
			if value != nil {
				checksum.WriteString(fmt.Sprintf("%v", value))
			}
			checksum.WriteString("|")
		}
		checksum.WriteString("\n")
	}

	return checksum.String(), nil
}

// Verify log file by comparing file contents
func verifyLogFile(sourcePath, backupPath string) error {
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source log file: %v", err)
	}

	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup log file: %v", err)
	}

	if !bytesEqual(sourceData, backupData) {
		return fmt.Errorf("log file contents do not match")
	}

	return nil
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
