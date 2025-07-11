package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archiveFiles/internal/backup"
	"archiveFiles/internal/compress"
	"archiveFiles/internal/config"
	"archiveFiles/internal/discovery"
	"archiveFiles/internal/progress"
	"archiveFiles/internal/restore"
	"archiveFiles/internal/types"
	"archiveFiles/internal/utils"
)

func main() {
	// Handle lock subcommand
	if len(os.Args) > 1 && os.Args[1] == "lock" {
		lockCmd := flag.NewFlagSet("lock", flag.ExitOnError)
		dbPath := lockCmd.String("db", "", "RocksDB 数据库路径")
		duration := lockCmd.String("duration", "", "锁定持续时间 (例如: 30s, 5m, 1h)")
		lockCmd.Parse(os.Args[2:])

		if *dbPath == "" {
			fmt.Println("用法: archiveFiles lock -db=数据库路径 [-duration=持续时间]")
			fmt.Println("示例:")
			fmt.Println("  archiveFiles lock -db=testdata/dir1/app.db -duration=30s")
			fmt.Println("  archiveFiles lock -db=testdata/dir1/app.db  # 无限期锁定，直到按 Ctrl+C")
			os.Exit(1)
		}

		var lockDuration time.Duration
		if *duration != "" {
			var err error
			lockDuration, err = time.ParseDuration(*duration)
			if err != nil {
				fmt.Printf("❌ 无效的持续时间格式: %v\n", err)
				fmt.Println("支持的格式: 30s, 5m, 1h, 等")
				os.Exit(1)
			}
		}

		fmt.Printf("锁定 RocksDB 数据库: %s\n", *dbPath)
		if lockDuration > 0 {
			fmt.Printf("锁定持续时间: %v\n", lockDuration)
		} else {
			fmt.Println("无限期锁定，按 Ctrl+C 释放")
		}

		err := utils.LockRocksDB(*dbPath, lockDuration)
		if err != nil {
			fmt.Printf("❌ 锁定失败: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle restore subcommand
	if len(os.Args) > 1 && os.Args[1] == "restore" {
		restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
		backupDir := restoreCmd.String("backup", "", "BackupEngine 格式的备份目录")
		restoreDir := restoreCmd.String("restore", "", "还原为原始 RocksDB 结构的目标目录")
		restoreCmd.Parse(os.Args[2:])

		if *backupDir == "" || *restoreDir == "" {
			fmt.Println("用法: archiveFiles restore -backup=备份目录 -restore=还原目录")
			os.Exit(1)
		}

		fmt.Printf("Restoring backup from %s to %s...\n", *backupDir, *restoreDir)
		err := restore.RestoreBackupToPlain(*backupDir, *restoreDir)
		if err != nil {
			fmt.Printf("❌ Restore failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Restore to plain RocksDB directory successful: %s\n", *restoreDir)
		os.Exit(0)
	}

	// Parse configuration
	cfg := parseFlags()

	log.Printf("Starting database archival process...")
	log.Printf("Sources: %v", cfg.SourcePaths)
	log.Printf("Method: %s", cfg.Method)
	log.Printf("Batch mode: %t", cfg.BatchMode)

	// Create progress tracker
	progressTracker := progress.NewProgressTracker(cfg.ShowProgress)

	// Discover databases from all source directories
	allDatabases := []types.DatabaseInfo{}
	for _, sourcePath := range cfg.SourcePaths {
		log.Printf("Scanning source: %s", sourcePath)

		// Create a temporary config for each source
		sourceConfig := &types.Config{
			SourcePaths:    []string{sourcePath},
			BatchMode:      cfg.BatchMode,
			IncludePattern: cfg.IncludePattern,
			ExcludePattern: cfg.ExcludePattern,
		}

		databases, err := discovery.DiscoverDatabases(sourceConfig, sourcePath)
		if err != nil {
			log.Printf("Warning: Failed to discover databases in %s: %v", sourcePath, err)
			continue
		}

		// Add source root information and calculate sizes
		for i := range databases {
			databases[i].SourceRoot = sourcePath
			databases[i].Size = utils.CalculateSize(databases[i].Path)
		}

		allDatabases = append(allDatabases, databases...)
	}

	if len(allDatabases) == 0 {
		log.Fatal("No databases or files found to archive")
	}

	log.Printf("Found %d item(s) to archive:", len(allDatabases))
	var totalSize int64
	for _, db := range allDatabases {
		log.Printf("  - %s (%s) from %s [%s]", db.Name, db.Type.String(), db.SourceRoot, utils.FormatBytes(db.Size))
		totalSize += db.Size
	}

	// Initialize progress tracking
	progressTracker.Init(len(allDatabases), totalSize)

	// Create backup directory
	backupPath := cfg.BackupPath
	if backupPath == "" {
		backupPath = fmt.Sprintf("backup_%d", time.Now().Unix())
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each database/file
	for _, db := range allDatabases {
		if cfg.ShowProgress {
			progressTracker.SetCurrentFile(db.Name)
		} else {
			log.Printf("Processing %s (%s)...", db.Name, db.Type.String())
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

		// Use safe backup method that handles locked databases
		err = backup.SafeBackupDatabase(db, dbBackupPath, cfg.Method, progressTracker)

		if err != nil {
			if !cfg.ShowProgress {
				log.Printf("Failed to process %s: %v", db.Name, err)
			}
			progressTracker.CompleteItem(0) // Still count as processed for progress
			continue
		}

		// TODO: Implement verification if requested
		// if cfg.Verify {
		//     err = verifyBackup(db, dbBackupPath, progressTracker)
		//     if err != nil {
		//         log.Printf("❌ Verification failed for %s: %v", db.Name, err)
		//     }
		// }

		progressTracker.CompleteItem(db.Size)
		if !cfg.ShowProgress {
			log.Printf("Successfully processed %s", db.Name)
		}
	}

	// Finish progress tracking
	if cfg.ShowProgress {
		progressTracker.Finish()
	}

	log.Printf("Backup created successfully at: %s", backupPath)

	// Compress backup if requested
	if cfg.Compress {
		archivePath := cfg.ArchivePath
		if archivePath == "" {
			archivePath = fmt.Sprintf("%s.tar.gz", backupPath)
		}

		if cfg.ShowProgress {
			log.Printf("Creating compressed archive...")
		}

		err := compress.CompressDirectory(backupPath, archivePath)
		if err != nil {
			log.Fatalf("Failed to compress backup: %v", err)
		}

		log.Printf("Archive created successfully at: %s", archivePath)

		// Remove original backup directory
		if cfg.RemoveBackup {
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

	// Parse flags
	flag.Parse()

	// Handle special commands
	if initConfig {
		err := config.GenerateDefaultConfigFile()
		if err != nil {
			log.Fatalf("Failed to generate default config: %v", err)
		}
		fmt.Println("Default configuration file 'archiveFiles.conf' created successfully!")
		os.Exit(0)
	}

	if generateConfig != "" {
		err := config.SaveConfigToJSON(cfg, generateConfig)
		if err != nil {
			log.Fatalf("Failed to generate config file: %v", err)
		}
		fmt.Printf("Configuration file generated: %s\n", generateConfig)
		os.Exit(0)
	}

	// Load configuration from file if specified
	var finalConfig *types.Config
	if configFile != "" {
		loadedConfig, err := config.LoadConfigFromJSON(configFile)
		if err != nil {
			log.Fatalf("Failed to load config file: %v", err)
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
		log.Fatal("No source paths specified. Use -source or -sources flag, or specify in config file.")
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
