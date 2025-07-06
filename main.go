package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/linxGnu/grocksdb"
)

type Config struct {
	SourceDBPath string
	BackupPath   string
	ArchivePath  string
	Method       string // backup, checkpoint, copy
	Compress     bool
	RemoveBackup bool
}

func main() {
	config := parseFlags()

	log.Printf("Starting RocksDB archival process...")
	log.Printf("Source DB: %s", config.SourceDBPath)
	log.Printf("Method: %s", config.Method)

	// 创建备份
	backupPath := config.BackupPath
	if backupPath == "" {
		backupPath = fmt.Sprintf("backup_%d", time.Now().Unix())
	}

	var err error
	switch config.Method {
	case "backup":
		err = backupDatabase(config.SourceDBPath, backupPath)
	case "checkpoint":
		err = checkpointDatabase(config.SourceDBPath, backupPath)
	case "copy":
		err = copyDatabaseData(config.SourceDBPath, backupPath)
	default:
		log.Fatalf("Unknown method: %s", config.Method)
	}

	if err != nil {
		log.Fatalf("Failed to create backup: %v", err)
	}

	log.Printf("Backup created successfully at: %s", backupPath)

	// 压缩备份
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

		// 删除原始备份目录
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

	flag.StringVar(&config.SourceDBPath, "source", "", "Source RocksDB path (required)")
	flag.StringVar(&config.BackupPath, "backup", "", "Backup path (default: backup_timestamp)")
	flag.StringVar(&config.ArchivePath, "archive", "", "Archive path (default: backup_path.tar.gz)")
	flag.StringVar(&config.Method, "method", "backup", "Backup method: backup, checkpoint, copy")
	flag.BoolVar(&config.Compress, "compress", true, "Compress the backup")
	flag.BoolVar(&config.RemoveBackup, "remove-backup", true, "Remove backup directory after compression")

	flag.Parse()

	if config.SourceDBPath == "" {
		log.Fatal("Source DB path is required")
	}

	return config
}

// 使用RocksDB内置备份功能
func backupDatabase(sourceDBPath, backupPath string) error {
	// 以只读模式打开源数据库
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDbForReadOnly(opts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer db.Close()

	// 创建备份引擎
	backupOpts := grocksdb.NewDefaultBackupOptions()
	defer backupOpts.Destroy()

	backupEngine, err := grocksdb.OpenBackupEngine(backupOpts, backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup engine: %v", err)
	}
	defer backupEngine.Close()

	// 创建备份
	err = backupEngine.CreateNewBackup(db)
	if err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	return nil
}

// 使用Checkpoint功能
func checkpointDatabase(sourceDBPath, checkpointPath string) error {
	// 以只读模式打开源数据库
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDbForReadOnly(opts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer db.Close()

	// 创建checkpoint
	checkpoint, err := grocksdb.NewCheckpoint(db)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %v", err)
	}
	defer checkpoint.Destroy()

	// 创建checkpoint到指定路径
	err = checkpoint.CreateCheckpoint(checkpointPath, 0)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint: %v", err)
	}

	return nil
}

// 手动遍历复制数据
func copyDatabaseData(sourceDBPath, targetDBPath string) error {
	// 打开源数据库（只读）
	sourceOpts := grocksdb.NewDefaultOptions()
	sourceOpts.SetCreateIfMissing(false)
	defer sourceOpts.Destroy()

	sourceDB, err := grocksdb.OpenDbForReadOnly(sourceOpts, sourceDBPath, false)
	if err != nil {
		return fmt.Errorf("failed to open source db: %v", err)
	}
	defer sourceDB.Close()

	// 创建目标数据库
	targetOpts := grocksdb.NewDefaultOptions()
	targetOpts.SetCreateIfMissing(true)
	defer targetOpts.Destroy()

	targetDB, err := grocksdb.OpenDb(targetOpts, targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to create target db: %v", err)
	}
	defer targetDB.Close()

	// 创建迭代器
	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	iter := sourceDB.NewIterator(readOpts)
	defer iter.Close()

	// 创建写批次
	writeBatch := grocksdb.NewWriteBatch()
	defer writeBatch.Destroy()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	batchSize := 1000
	count := 0

	// 遍历所有数据
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		writeBatch.Put(key.Data(), value.Data())
		count++

		// 批量写入
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

	// 写入剩余数据
	if writeBatch.Count() > 0 {
		err = targetDB.Write(writeOpts, writeBatch)
		if err != nil {
			return fmt.Errorf("failed to write final batch: %v", err)
		}
	}

	return iter.Err()
}

// 压缩目录为tar.gz
func compressDirectory(srcDir, destFile string) error {
	// 创建目标文件
	file, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// 创建tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// 遍历目录
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 创建tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// 设置相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// 写入header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 如果是文件，写入内容
		if info.Mode().IsRegular() {
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
