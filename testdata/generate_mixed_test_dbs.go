package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_mixed_test_dbs.go <base_path>")
		fmt.Println("  base_path: Base directory where test databases will be created")
		fmt.Println("")
		fmt.Println("This will create:")
		fmt.Println("  <base_path>/dir1/app_db (RocksDB)")
		fmt.Println("  <base_path>/dir1/application.log")
		fmt.Println("  <base_path>/dir2/users.db (SQLite)")
		fmt.Println("  <base_path>/dir2/error.log")
		fmt.Println("  <base_path>/dir3/cache_db (RocksDB)")
		fmt.Println("  <base_path>/dir3/debug.txt")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  go run generate_mixed_test_dbs.go ./mixed_dbs")
		os.Exit(1)
	}

	basePath := os.Args[1]

	// Create directory structure
	dirs := []string{
		filepath.Join(basePath, "dir1"),
		filepath.Join(basePath, "dir2"),
		filepath.Join(basePath, "dir3"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create databases and files
	createDir1(filepath.Join(basePath, "dir1"))
	createDir2(filepath.Join(basePath, "dir2"))
	createDir3(filepath.Join(basePath, "dir3"))

	fmt.Printf("Mixed test databases created successfully in %s\n", basePath)
}

func createDir1(dir string) {
	fmt.Printf("Creating dir1 content in: %s\n", dir)

	// Create RocksDB
	dbPath := filepath.Join(dir, "app_db")
	createRocksDB(dbPath, map[string]string{
		"app:config":     `{"theme":"dark","lang":"en"}`,
		"app:version":    "1.0.0",
		"user:session:1": `{"user_id":1,"token":"abc123"}`,
		"user:session:2": `{"user_id":2,"token":"def456"}`,
		"cache:stats":    `{"visits":1000,"users":50}`,
	})

	// Create log file
	logPath := filepath.Join(dir, "application.log")
	logContent := `2024-01-01 10:00:00 INFO  Application started
2024-01-01 10:00:01 INFO  Database connected
2024-01-01 10:00:02 INFO  Server listening on :8080
2024-01-01 10:05:00 INFO  User 1 logged in
2024-01-01 10:10:00 INFO  User 2 logged in
2024-01-01 10:15:00 WARN  High memory usage detected
2024-01-01 10:20:00 INFO  Cleanup completed
`
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}
}

func createDir2(dir string) {
	fmt.Printf("Creating dir2 content in: %s\n", dir)

	// Create SQLite database
	dbPath := filepath.Join(dir, "users.db")
	createSQLiteDB(dbPath)

	// Create error log file
	logPath := filepath.Join(dir, "error.log")
	logContent := `2024-01-01 10:00:00 ERROR Database connection timeout
2024-01-01 10:05:00 ERROR Failed to authenticate user: invalid credentials
2024-01-01 10:10:00 ERROR Network timeout on external API call
2024-01-01 10:15:00 WARN  Rate limit exceeded for user 123
2024-01-01 10:20:00 ERROR Failed to process payment: card declined
2024-01-01 10:25:00 ERROR Internal server error: null pointer exception
`
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		log.Fatalf("Failed to create error log file: %v", err)
	}
}

func createDir3(dir string) {
	fmt.Printf("Creating dir3 content in: %s\n", dir)

	// Create RocksDB for cache
	dbPath := filepath.Join(dir, "cache_db")
	createRocksDB(dbPath, map[string]string{
		"cache:user:1":      `{"name":"Alice","email":"alice@example.com"}`,
		"cache:user:2":      `{"name":"Bob","email":"bob@example.com"}`,
		"cache:product:100": `{"name":"Laptop","price":999.99}`,
		"cache:product:101": `{"name":"Mouse","price":29.99}`,
		"cache:session:abc": `{"expires":"2024-12-31T23:59:59Z"}`,
		"cache:config":      `{"ttl":3600,"max_size":1000}`,
	})

	// Create debug file
	debugPath := filepath.Join(dir, "debug.txt")
	debugContent := `Debug Session Started: 2024-01-01 10:00:00
Memory Usage: 512MB
Active Connections: 25
Cache Hit Rate: 85%
Query Performance:
  - SELECT users: 2ms avg
  - SELECT products: 1ms avg
  - INSERT orders: 5ms avg
Cache Statistics:
  - Total Keys: 1000
  - Expired Keys: 50
  - Memory Used: 128MB
System Health: OK
Debug Session Ended: 2024-01-01 10:30:00
`
	if err := os.WriteFile(debugPath, []byte(debugContent), 0644); err != nil {
		log.Fatalf("Failed to create debug file: %v", err)
	}
}

func createRocksDB(dbPath string, testData map[string]string) {
	// Remove existing database if it exists
	if err := os.RemoveAll(dbPath); err != nil {
		log.Fatalf("Failed to remove existing database: %v", err)
	}

	// Create RocksDB options
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	// Open database
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		log.Fatalf("Failed to create RocksDB: %v", err)
	}
	defer db.Close()

	// Create write options
	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	// Add test data
	for key, value := range testData {
		err := db.Put(writeOpts, []byte(key), []byte(value))
		if err != nil {
			log.Fatalf("Failed to put data (%s): %v", key, err)
		}
	}

	fmt.Printf("  Created RocksDB with %d records\n", len(testData))
}

func createSQLiteDB(dbPath string) {
	// Remove existing database if it exists
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to remove existing database: %v", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create tables and insert test data
	schema := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login DATETIME
	);

	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test users
	users := [][]interface{}{
		{1, "alice", "alice@example.com", "2024-01-01 09:00:00"},
		{2, "bob", "bob@example.com", "2024-01-01 09:30:00"},
		{3, "charlie", "charlie@example.com", "2024-01-01 10:00:00"},
	}

	for _, user := range users {
		_, err := db.Exec("INSERT INTO users (id, username, email, last_login) VALUES (?, ?, ?, ?)", user...)
		if err != nil {
			log.Fatalf("Failed to insert user: %v", err)
		}
	}

	// Insert test sessions
	sessions := [][]interface{}{
		{"sess_abc123", 1, "2024-01-01 10:00:00", "2024-01-01 18:00:00"},
		{"sess_def456", 2, "2024-01-01 10:30:00", "2024-01-01 18:30:00"},
		{"sess_ghi789", 3, "2024-01-01 11:00:00", "2024-01-01 19:00:00"},
	}

	for _, session := range sessions {
		_, err := db.Exec("INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)", session...)
		if err != nil {
			log.Fatalf("Failed to insert session: %v", err)
		}
	}

	fmt.Printf("  Created SQLite database with %d users and %d sessions\n", len(users), len(sessions))
}
