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

// LockRocksDB locks a RocksDB database for testing purposes
func LockRocksDB(dbPath string, duration time.Duration) error {
	log.Printf("Locking RocksDB database: %s", dbPath)

	// Open database (this creates the lock file)
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database for locking: %v", err)
	}
	defer db.Close()

	log.Printf("Database locked: %s", dbPath)
	log.Printf("Lock duration: %v", duration)

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// If duration is specified, use a timer
	if duration > 0 {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			log.Printf("Lock duration expired, releasing lock: %s", dbPath)
		case sig := <-sigChan:
			log.Printf("Received signal %v, releasing lock: %s", sig, dbPath)
		}
	} else {
		// If no duration specified, wait for signal
		log.Printf("Database locked, press Ctrl+C to release lock...")
		sig := <-sigChan
		log.Printf("Received signal %v, releasing lock: %s", sig, dbPath)
	}

	log.Printf("Database lock released: %s", dbPath)
	return nil
}

// IsRocksDBLocked checks if a RocksDB database is locked
func IsRocksDBLocked(dbPath string) bool {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		// If unable to open, it may be locked
		return true
	}
	defer db.Close()

	// If successfully opened, it's not locked
	return false
}
