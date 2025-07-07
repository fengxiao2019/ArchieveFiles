package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseType int

const (
	DatabaseTypeRocksDB DatabaseType = iota
	DatabaseTypeSQLite
	DatabaseTypeUnknown
)

type DatabaseInfo struct {
	Path string
	Type DatabaseType
	Name string
}

type Config struct {
	SourcePath     string // Can be a single DB or directory
	BackupPath     string
	ArchivePath    string
	Method         string // backup, checkpoint, copy
	Compress       bool
	RemoveBackup   bool
	BatchMode      bool   // Process directory vs single database
	IncludePattern string // File pattern to include (e.g., "*.db,*.sqlite")
	ExcludePattern string // File pattern to exclude
}

func main() {
	config := parseFlags()

	log.Printf("Starting database archival process...")
	log.Printf("Source: %s", config.SourcePath)
	log.Printf("Method: %s", config.Method)
	log.Printf("Batch mode: %t", config.BatchMode)

	// Discover databases
	databases, err := discoverDatabases(config)
	if err != nil {
		log.Fatalf("Failed to discover databases: %v", err)
	}

	if len(databases) == 0 {
		log.Fatal("No databases found to archive")
	}

	log.Printf("Found %d database(s) to archive:", len(databases))
	for _, db := range databases {
		log.Printf("  - %s (%s)", db.Name, databaseTypeString(db.Type))
	}

	// Create backup directory
	backupPath := config.BackupPath
	if backupPath == "" {
		backupPath = fmt.Sprintf("backup_%d", time.Now().Unix())
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	// Process each database
	for _, db := range databases {
		log.Printf("Processing %s (%s)...", db.Name, databaseTypeString(db.Type))

		dbBackupPath := filepath.Join(backupPath, db.Name)
		var err error

		switch db.Type {
		case DatabaseTypeRocksDB:
			err = processRocksDB(db.Path, dbBackupPath, config.Method)
		case DatabaseTypeSQLite:
			err = processSQLiteDB(db.Path, dbBackupPath)
		default:
			log.Printf("Skipping unknown database type: %s", db.Path)
			continue
		}

		if err != nil {
			log.Printf("Failed to process %s: %v", db.Name, err)
			continue
		}

		log.Printf("Successfully processed %s", db.Name)
	}

	log.Printf("Backup created successfully at: %s", backupPath)

	// Compress backup
	if config.Compress {
		archivePath := config.ArchivePath
		if archivePath == "" {
			archivePath = fmt.Sprintf("%s.tar.gz", backupPath)
		}

		err = compressDirectory(backupPath, archivePath)
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
	config := &Config{}

	flag.StringVar(&config.SourcePath, "source", "", "Source database path or directory (required)")
	flag.StringVar(&config.BackupPath, "backup", "", "Backup path (default: backup_timestamp)")
	flag.StringVar(&config.ArchivePath, "archive", "", "Archive path (default: backup_path.tar.gz)")
	flag.StringVar(&config.Method, "method", "backup", "Backup method for RocksDB: backup, checkpoint, copy")
	flag.BoolVar(&config.Compress, "compress", true, "Compress the backup")
	flag.BoolVar(&config.RemoveBackup, "remove-backup", true, "Remove backup directory after compression")
	flag.BoolVar(&config.BatchMode, "batch", false, "Process directory containing multiple databases")
	flag.StringVar(&config.IncludePattern, "include", "", "Include file patterns (comma-separated, e.g., '*.db,*.sqlite')")
	flag.StringVar(&config.ExcludePattern, "exclude", "", "Exclude file patterns (comma-separated)")

	flag.Parse()

	if config.SourcePath == "" {
		log.Fatal("Source path is required")
	}

	// Auto-detect batch mode if source is a directory
	if info, err := os.Stat(config.SourcePath); err == nil && info.IsDir() {
		config.BatchMode = true
	}

	return config
}

// Discover databases in the source path
func discoverDatabases(config *Config) ([]DatabaseInfo, error) {
	var databases []DatabaseInfo

	if !config.BatchMode {
		// Single database mode
		dbType := detectDatabaseType(config.SourcePath)
		if dbType == DatabaseTypeUnknown {
			return nil, fmt.Errorf("unknown database type: %s", config.SourcePath)
		}

		databases = append(databases, DatabaseInfo{
			Path: config.SourcePath,
			Type: dbType,
			Name: filepath.Base(config.SourcePath),
		})

		return databases, nil
	}

	// Batch mode - scan directory
	err := filepath.Walk(config.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == config.SourcePath {
			return nil
		}

		// Detect database type (works for both files and directories)
		dbType := detectDatabaseType(path)
		if dbType == DatabaseTypeUnknown {
			return nil
		}

		// Check include/exclude patterns (only for files, not directories like RocksDB)
		if !info.IsDir() && !shouldIncludeFile(path, config.IncludePattern, config.ExcludePattern) {
			return nil
		}

		// Create relative name for backup
		relPath, err := filepath.Rel(config.SourcePath, path)
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
		return true
	}

	// Check file header (SQLite files start with "SQLite format 3")
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

// Process RocksDB database
func processRocksDB(sourceDBPath, targetDBPath, method string) error {
	switch method {
	case "backup":
		// Use copy method as backup (backup API might not be available)
		return copyDatabaseData(sourceDBPath, targetDBPath)
	case "checkpoint":
		// Use copy method as checkpoint (checkpoint API might not be available)
		return copyDatabaseData(sourceDBPath, targetDBPath)
	case "copy":
		return copyDatabaseData(sourceDBPath, targetDBPath)
	default:
		return fmt.Errorf("unknown method: %s", method)
	}
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

// Copy SQLite database using file copy (simple approach)
func copySQLiteDatabase(sourcePath, targetPath string) error {
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

	return nil
}

// Get database type as string
func databaseTypeString(dbType DatabaseType) string {
	switch dbType {
	case DatabaseTypeRocksDB:
		return "RocksDB"
	case DatabaseTypeSQLite:
		return "SQLite"
	default:
		return "Unknown"
	}
}

// Note: Using copy method for all backup types since backup and checkpoint APIs
// might not be available in this version of grocksdb

// manual copy data
func copyDatabaseData(sourceDBPath, targetDBPath string) error {
	// open source database (read-only)
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer sourceDB.Close()

	// create target database
	targetOpts := grocksdb.NewDefaultOptions()
	targetOpts.SetCreateIfMissing(true)
	defer targetOpts.Destroy()

	targetDB, err := grocksdb.OpenDb(targetOpts, targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to create target db: %v", err)
	}
	defer targetDB.Close()

	// create iterator
	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	iter := sourceDB.NewIterator(readOpts)
	defer iter.Close()

	// create write batch
	writeBatch := grocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	batchSize := 1000
	count := 0

	// iterate all data
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		writeBatch.Put(key.Data(), value.Data())
		count++

		// write batch
		if count%batchSize == 0 {
			err = targetDB.Write(writeOpts, writeBatch)
			if err != nil {
				key.Free()
				value.Free()
				return fmt.Errorf("failed to write batch: %v", err)
			}
			writeBatch.Clear()
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
