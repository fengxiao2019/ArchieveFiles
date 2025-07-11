package utils

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linxGnu/grocksdb"
)

// LockRocksDB 锁定一个 RocksDB 数据库，用于测试目的
func LockRocksDB(dbPath string, duration time.Duration) error {
	log.Printf("正在锁定 RocksDB 数据库: %s", dbPath)

	// 打开数据库（这会创建锁文件）
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return fmt.Errorf("无法打开数据库进行锁定: %v", err)
	}
	defer db.Close()

	log.Printf("✅ 数据库已锁定: %s", dbPath)
	log.Printf("锁定时间: %v", duration)

	// 设置信号处理，允许优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 如果指定了持续时间，使用定时器
	if duration > 0 {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			log.Printf("锁定时间到期，释放锁定: %s", dbPath)
		case sig := <-sigChan:
			log.Printf("收到信号 %v，释放锁定: %s", sig, dbPath)
		}
	} else {
		// 如果没有指定持续时间，等待信号
		log.Printf("数据库已锁定，按 Ctrl+C 释放锁定...")
		sig := <-sigChan
		log.Printf("收到信号 %v，释放锁定: %s", sig, dbPath)
	}

	log.Printf("✅ 数据库锁定已释放: %s", dbPath)
	return nil
}

// IsRocksDBLocked 检查 RocksDB 是否被锁定
func IsRocksDBLocked(dbPath string) bool {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		// 如果无法打开，可能是被锁定了
		return true
	}
	defer db.Close()

	// 如果能成功打开，说明没有被锁定
	return false
}
