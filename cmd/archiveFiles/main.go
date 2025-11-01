package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"archiveFiles/internal/backup"
	"archiveFiles/internal/compress"
	"archiveFiles/internal/config"
	"archiveFiles/internal/constants"
	"archiveFiles/internal/discovery"
	"archiveFiles/internal/logger"
	"archiveFiles/internal/progress"
	"archiveFiles/internal/restore"
	"archiveFiles/internal/types"
	"archiveFiles/internal/utils"
	"archiveFiles/internal/verify"
)

func main() {
	// Handle lock subcommand
	if len(os.Args) > 1 && os.Args[1] == "lock" {
		lockCmd := flag.NewFlagSet("lock", flag.ExitOnError)
		dbPath := lockCmd.String("db", "", "RocksDB database path")
		duration := lockCmd.String("duration", "", "Lock duration (e.g., 30s, 5m, 1h)")
		lockCmd.Parse(os.Args[2:])

		if *dbPath == "" {
			fmt.Println("Usage: archiveFiles lock -db=database_path [-duration=duration]")
			fmt.Println("Examples:")
			fmt.Println("  archiveFiles lock -db=testdata/dir1/app.db -duration=30s")
			fmt.Println("  archiveFiles lock -db=testdata/dir1/app.db  # Lock indefinitely until Ctrl+C")
			os.Exit(1)
		}

		var lockDuration time.Duration
		if *duration != "" {
			var err error
			lockDuration, err = time.ParseDuration(*duration)
			if err != nil {
				fmt.Printf("Invalid duration format: %v\n", err)
				fmt.Println("Supported formats: 30s, 5m, 1h, etc.")
				os.Exit(1)
			}
		}

		fmt.Printf("Locking RocksDB database: %s\n", *dbPath)
		if lockDuration > 0 {
			fmt.Printf("Lock duration: %v\n", lockDuration)
		} else {
			fmt.Println("Lock indefinitely, press Ctrl+C to release")
		}

		err := utils.LockRocksDB(*dbPath, lockDuration)
		if err != nil {
			fmt.Printf("Lock failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle restore subcommand
	if len(os.Args) > 1 && os.Args[1] == "restore" {
		restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
		backupDir := restoreCmd.String("backup", "", "BackupEngine format backup directory")
		restoreDir := restoreCmd.String("restore", "", "Target directory to restore as original RocksDB structure")
		restoreCmd.Parse(os.Args[2:])

		if *backupDir == "" || *restoreDir == "" {
			fmt.Println("Usage: archiveFiles restore -backup=backup_directory -restore=restore_directory")
			os.Exit(1)
		}

		fmt.Printf("Restoring backup from %s to %s...\n", *backupDir, *restoreDir)
		err := restore.RestoreBackupToPlain(*backupDir, *restoreDir)
		if err != nil {
			fmt.Printf("Restore failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Restore to plain RocksDB directory successful: %s\n", *restoreDir)
		os.Exit(0)
	}

	// Parse configuration
	cfg := parseFlags()

	// Initialize logger with config settings
	initLogger(cfg)

	// Set up context with cancellation support
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Warning("\nReceived signal %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// Log operational mode
	if cfg.DryRun {
		logger.Warning("DRY RUN MODE: No actual changes will be made")
	}
	if cfg.Strict {
		logger.Warning("STRICT MODE: Will fail immediately on any error")
	}

	logger.Info("Starting database archival process...")
	logger.Info("Sources: %v", cfg.SourcePaths)
	logger.Info("Method: %s", cfg.Method)
	logger.Debug("Batch mode: %t", cfg.BatchMode)

	// Create progress tracker
	progressTracker := progress.NewProgressTracker(cfg.ShowProgress)

	// Discover databases from all source directories
	allDatabases := []types.DatabaseInfo{}
	for _, sourcePath := range cfg.SourcePaths {
		logger.Info("Scanning source: %s", sourcePath)

		// Create a temporary config for each source
		sourceConfig := &types.Config{
			SourcePaths:    []string{sourcePath},
			BatchMode:      cfg.BatchMode,
			IncludePattern: cfg.IncludePattern,
			ExcludePattern: cfg.ExcludePattern,
		}

		databases, err := discovery.DiscoverDatabases(sourceConfig, sourcePath)
		if err != nil {
			logger.Warning("Failed to discover databases in %s: %v", sourcePath, err)
			continue
		}

		// Add source root information (size is already calculated during discovery)
		for i := range databases {
			databases[i].SourceRoot = sourcePath
		}

		allDatabases = append(allDatabases, databases...)
	}

	if len(allDatabases) == 0 {
		logger.Fatal("No databases or files found to archive")
	}

	logger.Info("Found %d item(s) to archive:", len(allDatabases))
	var totalSize int64
	for _, db := range allDatabases {
		logger.Info("  - %s (%s) from %s [%s]", db.Name, db.Type.String(), db.SourceRoot, utils.FormatBytes(db.Size))
		totalSize += db.Size
	}

	// Initialize progress tracking
	progressTracker.Init(len(allDatabases), totalSize)

	// Create backup directory
	backupPath := utils.ReplaceDateVars(cfg.BackupPath)
	if backupPath == "" {
		backupPath = utils.ReplaceDateVars(fmt.Sprintf("backup_%d", time.Now().Unix()))
	}

	if cfg.DryRun {
		logger.Info("[DRY RUN] Would create backup directory: %s", backupPath)
	} else {
		if err := os.MkdirAll(backupPath, constants.DirPermission); err != nil {
			logger.Fatal("Failed to create backup directory: %v", err)
		}
	}

	// Determine number of workers
	workers := cfg.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Cap workers at reasonable maximum
	if workers > len(allDatabases) {
		workers = len(allDatabases)
	}

	if workers > 1 {
		logger.Info("Using %d concurrent workers for backup", workers)
	}

	// Process databases with worker pool
	processDatabasesConcurrently(ctx, allDatabases, backupPath, cfg, progressTracker, workers)

	// Check if context was cancelled
	if ctx.Err() != nil {
		logger.Warning("Backup was cancelled: %v", ctx.Err())
		logger.Warning("Partial backup may exist at: %s", backupPath)
		os.Exit(130) // Exit code 130 for Ctrl+C
	}

	// Finish progress tracking
	if cfg.ShowProgress {
		progressTracker.Finish()
	}

	logger.Info("Backup created successfully at: %s", backupPath)

	// Compress backup if requested
	if cfg.Compress {
		archivePath := utils.ReplaceDateVars(cfg.ArchivePath)
		if archivePath == "" {
			archivePath = utils.ReplaceDateVars(fmt.Sprintf("%s.tar.gz", backupPath))
		}

		if cfg.DryRun {
			logger.Info("[DRY RUN] Would create compressed archive: %s", archivePath)
			if cfg.RemoveBackup {
				logger.Info("[DRY RUN] Would remove backup directory: %s", backupPath)
			}
		} else {
			if cfg.ShowProgress {
				logger.Info("Creating compressed archive...")
			}

			err := compress.CompressDirectory(backupPath, archivePath)
			if err != nil {
				logger.Fatal("Failed to compress backup: %v", err)
			}

			logger.Info("Archive created successfully at: %s", archivePath)

			// Remove original backup directory
			if cfg.RemoveBackup {
				err = os.RemoveAll(backupPath)
				if err != nil {
					logger.Warning("Failed to remove backup directory: %v", err)
				} else {
					logger.Info("Backup directory removed: %s", backupPath)
				}
			}
		}
	}

	logger.Info("Archival process completed successfully!")
}

func parseFlags() *types.Config {
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
	cfg := config.GetDefaultConfig()

	flag.StringVar(&cfg.BackupPath, "backup", "", "Backup path (default: backup_timestamp)")
	flag.StringVar(&cfg.ArchivePath, "archive", "", "Archive path (default: backup_path.tar.gz)")
	flag.StringVar(&cfg.Method, "method", "checkpoint", "RocksDB backup method: checkpoint (fast, hard-links), backup (native backup engine), copy (record-by-record)")
	flag.BoolVar(&cfg.Compress, "compress", true, "Compress archived files")
	flag.BoolVar(&cfg.RemoveBackup, "remove-backup", true, "Remove backup directory after compression")
	flag.BoolVar(&cfg.BatchMode, "batch", false, "Process directory containing multiple databases")
	flag.StringVar(&cfg.IncludePattern, "include", "", "Include file patterns (comma-separated, e.g., '*.db,*.sqlite,*.log')")
	flag.StringVar(&cfg.ExcludePattern, "exclude", "", "Exclude file patterns (comma-separated)")
	flag.StringVar(&cfg.Filter, "filter", "", "Filter pattern for source paths (e.g., '*.db' or 'cache*')")
	flag.StringVar(&cfg.CompressionFormat, "compression", "gzip", "Compression format: gzip, zstd, lz4")
	flag.BoolVar(&cfg.ShowProgress, "progress", true, "Show progress bar during archival")
	flag.BoolVar(&cfg.Verify, "verify", false, "Verify backup data integrity against source")
	flag.IntVar(&cfg.Workers, "workers", 0, "Number of concurrent backup workers (0 = auto, based on CPU cores)")
	flag.BoolVar(&cfg.Strict, "strict", false, "Strict mode: fail immediately on any error instead of continuing")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Dry run mode: simulate actions without actually executing them")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level: debug, info, warning, error")
	flag.BoolVar(&cfg.ColorLog, "color-log", true, "Enable colored log output")

	// Parse flags
	flag.Parse()

	// Handle special commands
	if initConfig {
		err := config.GenerateDefaultConfigFile()
		if err != nil {
			logger.Fatal("Failed to generate default config: %v", err)
		}
		fmt.Println("Default configuration file 'archiveFiles.conf' created successfully!")
		os.Exit(0)
	}

	if generateConfig != "" {
		err := config.SaveConfigToJSON(cfg, generateConfig)
		if err != nil {
			logger.Fatal("Failed to generate config file: %v", err)
		}
		fmt.Printf("Configuration file generated: %s\n", generateConfig)
		os.Exit(0)
	}

	// Load configuration from file if specified
	var finalConfig *types.Config
	if configFile != "" {
		loadedConfig, err := config.LoadConfigFromJSON(configFile)
		if err != nil {
			logger.Fatal("Failed to load config file: %v", err)
		}
		finalConfig = config.MergeConfigs(loadedConfig, cfg)
	} else {
		finalConfig = cfg
	}

	// Handle source paths
	if sourceFlag != "" {
		finalConfig.SourcePaths = []string{sourceFlag}
	} else if sourcesFlag != "" {
		finalConfig.SourcePaths = strings.Split(sourcesFlag, ",")
		for i, path := range finalConfig.SourcePaths {
			finalConfig.SourcePaths[i] = strings.TrimSpace(path)
		}
	}

	// Validate configuration
	if len(finalConfig.SourcePaths) == 0 {
		logger.Fatal("No source paths specified. Use -source or -sources flag, or specify in config file.")
	}

	// Auto-detect batch mode if any source is a directory
	for _, sourcePath := range finalConfig.SourcePaths {
		if info, err := os.Stat(sourcePath); err == nil && info.IsDir() {
			finalConfig.BatchMode = true
			break
		}
	}

	// Validate configuration
	if err := finalConfig.Validate(); err != nil {
		logger.Fatal("Configuration validation failed: %v", err)
	}

	return finalConfig
}

// processDatabasesConcurrently processes databases using a worker pool for concurrent backup
func processDatabasesConcurrently(ctx context.Context, databases []types.DatabaseInfo, backupPath string, cfg *types.Config, progressTracker *progress.ProgressTracker, workers int) {
	// Create job channel and error collection
	jobs := make(chan types.DatabaseInfo, len(databases))
	var wg sync.WaitGroup
	var errorsMu sync.Mutex
	errors := make(map[string]error)

	// Start worker pool
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for db := range jobs {
				// Check if context was cancelled before processing
				select {
				case <-ctx.Done():
					logger.Debug("Worker %d stopping due to cancellation", workerID)
					return
				default:
					processDatabase(ctx, db, backupPath, cfg, progressTracker, &errorsMu, errors)
				}
			}
		}(w)
	}

	// Send jobs to workers, respecting context cancellation
	go func() {
		for _, db := range databases {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- db:
			}
		}
		close(jobs)
	}()

	// Wait for all workers to complete
	wg.Wait()

	// Report errors if any
	if len(errors) > 0 {
		logger.Warning("%d database(s) failed to backup:", len(errors))
		for name, err := range errors {
			logger.Error("  - %s: %v", name, err)
		}

		// In strict mode, exit with error if any database failed
		if cfg.Strict {
			logger.Fatal("Backup failed in strict mode due to errors")
		}
	}
}

// processDatabase processes a single database backup
func processDatabase(ctx context.Context, db types.DatabaseInfo, backupPath string, cfg *types.Config, progressTracker *progress.ProgressTracker, errorsMu *sync.Mutex, errors map[string]error) {
	// Check if context was cancelled before starting
	select {
	case <-ctx.Done():
		errorsMu.Lock()
		errors[db.Name] = fmt.Errorf("cancelled: %v", ctx.Err())
		errorsMu.Unlock()
		return
	default:
	}

	// Update progress
	if cfg.ShowProgress {
		progressTracker.SetCurrentFile(db.Name)
	} else {
		if cfg.DryRun {
			logger.Info("[DRY RUN] Would process %s (%s)...", db.Name, db.Type.String())
		} else {
			logger.Info("Processing %s (%s)...", db.Name, db.Type.String())
		}
	}

	// Create a subdirectory structure to avoid name collisions
	sourceBaseName := filepath.Base(db.SourceRoot)
	if sourceBaseName == "." || sourceBaseName == "" {
		sourceBaseName = "root"
	}

	dbBackupPath := filepath.Join(backupPath, sourceBaseName, db.Name)

	// In dry-run mode, simulate the operation
	if cfg.DryRun {
		if !cfg.ShowProgress {
			logger.Info("[DRY RUN] Would backup %s to %s using method: %s", db.Name, dbBackupPath, cfg.Method)
		}
		progressTracker.CompleteItem(db.Size)
		return
	}

	// Ensure the parent directory exists
	parentDir := filepath.Dir(dbBackupPath)
	if err := os.MkdirAll(parentDir, constants.DirPermission); err != nil {
		errorsMu.Lock()
		errors[db.Name] = fmt.Errorf("failed to create parent directory: %v", err)
		errorsMu.Unlock()
		progressTracker.CompleteItem(0)
		return
	}

	// Use safe backup method that handles locked databases
	err := backup.SafeBackupDatabase(db, dbBackupPath, cfg.Method, progressTracker)

	if err != nil {
		if !cfg.ShowProgress {
			logger.Error("Failed to process %s: %v", db.Name, err)
		}
		errorsMu.Lock()
		errors[db.Name] = err
		errorsMu.Unlock()
		progressTracker.CompleteItem(0) // Still count as processed for progress
		return
	}

	// Verify backup if requested
	if cfg.Verify {
		err = verify.VerifyBackup(db, dbBackupPath, progressTracker)
		if err != nil {
			if !cfg.ShowProgress {
				logger.Error("Verification failed for %s: %v", db.Name, err)
			}
			errorsMu.Lock()
			errors[db.Name] = fmt.Errorf("verification failed: %v", err)
			errorsMu.Unlock()
			progressTracker.CompleteItem(db.Size)
			return
		} else {
			if !cfg.ShowProgress {
				logger.Info("Verification passed for %s", db.Name)
			}
		}
	}

	progressTracker.CompleteItem(db.Size)
	if !cfg.ShowProgress {
		logger.Info("Successfully processed %s", db.Name)
	}
}

// initLogger initializes the logger with configuration settings
func initLogger(cfg *types.Config) {
	// Set log level
	logLevel := logger.INFO
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		logLevel = logger.DEBUG
	case "info":
		logLevel = logger.INFO
	case "warning", "warn":
		logLevel = logger.WARNING
	case "error":
		logLevel = logger.ERROR
	default:
		logLevel = logger.INFO
	}

	logger.SetLevel(logLevel)
	logger.SetColorOutput(cfg.ColorLog)

	// Log the logger initialization at debug level
	logger.Debug("Logger initialized: level=%s, color=%t", cfg.LogLevel, cfg.ColorLog)
}
