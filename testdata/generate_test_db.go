package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_test_db.go <output_path> [type]")
		fmt.Println("  output_path: Path where the database will be created")
		fmt.Println("  type: 'rocksdb' or 'sqlite' (default: auto-detect from extension)")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run generate_test_db.go ./test_db rocksdb")
		fmt.Println("  go run generate_test_db.go ./test.db sqlite")
		fmt.Println("  go run generate_test_db.go ./test.sqlite  # auto-detects as SQLite")
		os.Exit(1)
	}

	outputPath := os.Args[1]
	var dbType string

	if len(os.Args) >= 3 {
		dbType = strings.ToLower(os.Args[2])
	} else {
		// Auto-detect based on file extension
		ext := strings.ToLower(filepath.Ext(outputPath))
		switch ext {
		case ".db", ".sqlite", ".sqlite3":
			dbType = "sqlite"
		default:
			dbType = "rocksdb"
		}
	}

	switch dbType {
	case "rocksdb":
		createRocksDB(outputPath)
	case "sqlite":
		createSQLiteDB(outputPath)
	default:
		log.Fatalf("Unknown database type: %s. Use 'rocksdb' or 'sqlite'", dbType)
	}
}

func createRocksDB(dbPath string) {
	fmt.Printf("Creating RocksDB at: %s\n", dbPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

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
	testData := map[string]string{
		"user:1":         `{"id":1,"name":"Alice","email":"alice@example.com"}`,
		"user:2":         `{"id":2,"name":"Bob","email":"bob@example.com"}`,
		"user:3":         `{"id":3,"name":"Charlie","email":"charlie@example.com"}`,
		"config:theme":   "dark",
		"config:lang":    "en",
		"session:abc123": `{"user_id":1,"expires":"2024-12-31T23:59:59Z"}`,
		"session:def456": `{"user_id":2,"expires":"2024-12-31T23:59:59Z"}`,
		"cache:popular":  `["item1","item2","item3"]`,
		"metrics:count":  "42",
	}

	fmt.Printf("Adding %d test records...\n", len(testData))
	for key, value := range testData {
		err := db.Put(writeOpts, []byte(key), []byte(value))
		if err != nil {
			log.Fatalf("Failed to put data (%s): %v", key, err)
		}
	}

	fmt.Printf("✅ RocksDB created successfully with %d records\n", len(testData))
}

func createSQLiteDB(dbPath string) {
	fmt.Printf("Creating SQLite database at: %s\n", dbPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

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
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE products (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price REAL NOT NULL,
		category TEXT,
		in_stock BOOLEAN DEFAULT 1
	);

	CREATE TABLE orders (
		id INTEGER PRIMARY KEY,
		user_id INTEGER,
		product_id INTEGER,
		quantity INTEGER DEFAULT 1,
		order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test users
	users := [][]interface{}{
		{1, "Alice Johnson", "alice@example.com"},
		{2, "Bob Smith", "bob@example.com"},
		{3, "Charlie Brown", "charlie@example.com"},
		{4, "Diana Prince", "diana@example.com"},
		{5, "Eve Wilson", "eve@example.com"},
	}

	for _, user := range users {
		_, err := db.Exec("INSERT INTO users (id, name, email) VALUES (?, ?, ?)", user...)
		if err != nil {
			log.Fatalf("Failed to insert user: %v", err)
		}
	}

	// Insert test products
	products := [][]interface{}{
		{1, "Laptop", 999.99, "Electronics", true},
		{2, "Mouse", 29.99, "Electronics", true},
		{3, "Keyboard", 79.99, "Electronics", true},
		{4, "Monitor", 299.99, "Electronics", false},
		{5, "Desk Chair", 199.99, "Furniture", true},
		{6, "Coffee Mug", 12.99, "Kitchen", true},
		{7, "Notebook", 5.99, "Office", true},
		{8, "Pen Set", 15.99, "Office", true},
	}

	for _, product := range products {
		_, err := db.Exec("INSERT INTO products (id, name, price, category, in_stock) VALUES (?, ?, ?, ?, ?)", product...)
		if err != nil {
			log.Fatalf("Failed to insert product: %v", err)
		}
	}

	// Insert test orders
	orders := [][]interface{}{
		{1, 1, 1, 1}, // Alice bought a Laptop
		{2, 1, 2, 2}, // Alice bought 2 Mice
		{3, 2, 3, 1}, // Bob bought a Keyboard
		{4, 3, 5, 1}, // Charlie bought a Desk Chair
		{5, 4, 6, 3}, // Diana bought 3 Coffee Mugs
		{6, 5, 7, 5}, // Eve bought 5 Notebooks
	}

	for _, order := range orders {
		_, err := db.Exec("INSERT INTO orders (id, user_id, product_id, quantity) VALUES (?, ?, ?, ?)", order...)
		if err != nil {
			log.Fatalf("Failed to insert order: %v", err)
		}
	}

	fmt.Printf("✅ SQLite database created successfully with %d users, %d products, and %d orders\n",
		len(users), len(products), len(orders))
}
