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
		log.Fatal("Usage: go run generate_mixed_test_dbs.go <base_directory>")
	}

	baseDir := os.Args[1]

	// Remove existing directory if it exists
	if err := os.RemoveAll(baseDir); err != nil {
		log.Printf("Warning: failed to remove existing directory: %v", err)
	}

	// Create base directory
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	// Create RocksDB databases
	rocksDBs := []string{
		"user_data",
		"product_catalog",
		"session_store",
	}

	for _, dbName := range rocksDBs {
		dbPath := filepath.Join(baseDir, dbName)
		if err := createRocksDB(dbPath, dbName); err != nil {
			log.Fatalf("Failed to create RocksDB %s: %v", dbName, err)
		}
		fmt.Printf("Created RocksDB: %s\n", dbPath)
	}

	// Create SQLite databases
	sqliteDBs := []struct {
		name   string
		suffix string
		tables []string
	}{
		{"analytics", ".db", []string{"events", "users", "sessions"}},
		{"logs", ".sqlite", []string{"error_logs", "access_logs"}},
		{"config", ".sqlite3", []string{"settings", "features"}},
	}

	for _, db := range sqliteDBs {
		dbPath := filepath.Join(baseDir, db.name+db.suffix)
		if err := createSQLiteDB(dbPath, db.name, db.tables); err != nil {
			log.Fatalf("Failed to create SQLite DB %s: %v", db.name, err)
		}
		fmt.Printf("Created SQLite DB: %s\n", dbPath)
	}

	// Create some non-database files
	nonDBFiles := []string{
		"README.txt",
		"config.json",
		"backup.log",
	}

	for _, fileName := range nonDBFiles {
		filePath := filepath.Join(baseDir, fileName)
		content := fmt.Sprintf("This is a non-database file: %s\nCreated for testing purposes.\n", fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			log.Printf("Warning: failed to create %s: %v", fileName, err)
		} else {
			fmt.Printf("Created non-DB file: %s\n", filePath)
		}
	}

	// Create a subdirectory with more databases
	subDir := filepath.Join(baseDir, "archived")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		log.Printf("Warning: failed to create subdirectory: %v", err)
	} else {
		// Create a RocksDB in subdirectory
		subRocksPath := filepath.Join(subDir, "old_data")
		if err := createRocksDB(subRocksPath, "old_data"); err != nil {
			log.Printf("Warning: failed to create subdirectory RocksDB: %v", err)
		} else {
			fmt.Printf("Created subdirectory RocksDB: %s\n", subRocksPath)
		}

		// Create a SQLite in subdirectory
		subSQLitePath := filepath.Join(subDir, "archive.db")
		if err := createSQLiteDB(subSQLitePath, "archive", []string{"archived_data"}); err != nil {
			log.Printf("Warning: failed to create subdirectory SQLite: %v", err)
		} else {
			fmt.Printf("Created subdirectory SQLite: %s\n", subSQLitePath)
		}
	}

	fmt.Printf("\nTest database environment created successfully at: %s\n", baseDir)
	fmt.Printf("Total databases created: %d RocksDB + %d SQLite\n", len(rocksDBs)+1, len(sqliteDBs)+1)
}

func createRocksDB(dbPath, dbName string) error {
	// Create RocksDB
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		return fmt.Errorf("failed to create RocksDB: %v", err)
	}
	defer db.Close()

	// Add sample data based on database type
	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	var sampleData map[string]string

	switch dbName {
	case "user_data":
		sampleData = map[string]string{
			"user:1001":    `{"id":1001,"name":"Alice Johnson","email":"alice@example.com","role":"admin","created_at":"2023-01-15T10:30:00Z"}`,
			"user:1002":    `{"id":1002,"name":"Bob Smith","email":"bob@example.com","role":"user","created_at":"2023-02-20T14:45:00Z"}`,
			"user:1003":    `{"id":1003,"name":"Charlie Brown","email":"charlie@example.com","role":"user","created_at":"2023-03-10T09:15:00Z"}`,
			"profile:1001": `{"user_id":1001,"avatar":"avatar1.jpg","bio":"System administrator","preferences":{"theme":"dark","notifications":true}}`,
			"profile:1002": `{"user_id":1002,"avatar":"avatar2.jpg","bio":"Regular user","preferences":{"theme":"light","notifications":false}}`,
		}
	case "product_catalog":
		sampleData = map[string]string{
			"product:p001":         `{"id":"p001","name":"Laptop Pro","category":"electronics","price":1299.99,"stock":45,"description":"High-performance laptop"}`,
			"product:p002":         `{"id":"p002","name":"Wireless Mouse","category":"electronics","price":29.99,"stock":150,"description":"Ergonomic wireless mouse"}`,
			"product:p003":         `{"id":"p003","name":"Coffee Mug","category":"kitchenware","price":12.99,"stock":80,"description":"Ceramic coffee mug"}`,
			"category:electronics": `{"id":"electronics","name":"Electronics","description":"Electronic devices and accessories"}`,
			"category:kitchenware": `{"id":"kitchenware","name":"Kitchenware","description":"Kitchen tools and accessories"}`,
		}
	case "session_store":
		sampleData = map[string]string{
			"session:sess_abc123": `{"user_id":1001,"created_at":"2023-10-01T10:00:00Z","expires_at":"2023-10-01T18:00:00Z","ip":"192.168.1.100"}`,
			"session:sess_def456": `{"user_id":1002,"created_at":"2023-10-01T11:00:00Z","expires_at":"2023-10-01T19:00:00Z","ip":"192.168.1.101"}`,
			"session:sess_ghi789": `{"user_id":1003,"created_at":"2023-10-01T12:00:00Z","expires_at":"2023-10-01T20:00:00Z","ip":"192.168.1.102"}`,
		}
	default:
		sampleData = map[string]string{
			"config:version": "1.0.0",
			"config:debug":   "true",
			"data:sample":    "This is sample data for " + dbName,
		}
	}

	for key, value := range sampleData {
		if err := db.Put(writeOpts, []byte(key), []byte(value)); err != nil {
			return fmt.Errorf("failed to put data %s: %v", key, err)
		}
	}

	return nil
}

func createSQLiteDB(dbPath, dbName string, tables []string) error {
	// Create SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create tables and insert sample data based on database type
	switch dbName {
	case "analytics":
		if err := createAnalyticsSchema(db); err != nil {
			return err
		}
	case "logs":
		if err := createLogsSchema(db); err != nil {
			return err
		}
	case "config":
		if err := createConfigSchema(db); err != nil {
			return err
		}
	case "archive":
		if err := createArchiveSchema(db); err != nil {
			return err
		}
	default:
		// Generic schema
		if err := createGenericSchema(db, tables); err != nil {
			return err
		}
	}

	return nil
}

func createAnalyticsSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			user_id INTEGER,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			data TEXT
		);
		
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		
		INSERT INTO users (id, name, email) VALUES 
			(1, 'Alice Johnson', 'alice@example.com'),
			(2, 'Bob Smith', 'bob@example.com'),
			(3, 'Charlie Brown', 'charlie@example.com');
			
		INSERT INTO events (event_type, user_id, data) VALUES
			('login', 1, '{"ip": "192.168.1.100", "user_agent": "Mozilla/5.0"}'),
			('page_view', 1, '{"page": "/dashboard", "duration": 45}'),
			('login', 2, '{"ip": "192.168.1.101", "user_agent": "Chrome/91.0"}'),
			('logout', 1, '{"session_duration": 1800}');
			
		INSERT INTO sessions (id, user_id, expires_at) VALUES
			('sess_abc123', 1, datetime('now', '+8 hours')),
			('sess_def456', 2, datetime('now', '+8 hours'));
	`

	_, err := db.Exec(schema)
	return err
}

func createLogsSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE error_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			source TEXT,
			stack_trace TEXT
		);
		
		CREATE TABLE access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER,
			response_time INTEGER,
			ip_address TEXT,
			user_agent TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		INSERT INTO error_logs (level, message, source, stack_trace) VALUES
			('ERROR', 'Database connection failed', 'database.go:45', 'stack trace here'),
			('WARN', 'High memory usage detected', 'monitor.go:123', NULL),
			('ERROR', 'Authentication failed', 'auth.go:67', 'auth stack trace');
			
		INSERT INTO access_logs (method, path, status_code, response_time, ip_address, user_agent) VALUES
			('GET', '/api/users', 200, 45, '192.168.1.100', 'Mozilla/5.0'),
			('POST', '/api/login', 401, 12, '192.168.1.101', 'Chrome/91.0'),
			('GET', '/api/products', 200, 67, '192.168.1.102', 'Safari/14.0');
	`

	_, err := db.Exec(schema)
	return err
}

func createConfigSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			description TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE features (
			name TEXT PRIMARY KEY,
			enabled BOOLEAN DEFAULT FALSE,
			config TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		INSERT INTO settings (key, value, description) VALUES
			('app_name', 'Archive Tool', 'Application name'),
			('version', '1.0.0', 'Application version'),
			('debug_mode', 'false', 'Enable debug logging'),
			('max_connections', '100', 'Maximum database connections');
			
		INSERT INTO features (name, enabled, config) VALUES
			('batch_processing', 1, '{"max_batch_size": 1000}'),
			('compression', 1, '{"algorithm": "gzip", "level": 6}'),
			('notifications', 0, '{"email": true, "slack": false}');
	`

	_, err := db.Exec(schema)
	return err
}

func createArchiveSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE archived_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			original_table TEXT NOT NULL,
			data_json TEXT NOT NULL,
			archived_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			checksum TEXT
		);
		
		INSERT INTO archived_data (original_table, data_json, checksum) VALUES
			('users', '{"id": 999, "name": "Archived User", "email": "archived@example.com"}', 'abc123'),
			('products', '{"id": "p999", "name": "Archived Product", "price": 0.00}', 'def456'),
			('sessions', '{"id": "old_session", "user_id": 999, "expired": true}', 'ghi789');
	`

	_, err := db.Exec(schema)
	return err
}

func createGenericSchema(db *sql.DB, tables []string) error {
	for _, table := range tables {
		schema := fmt.Sprintf(`
			CREATE TABLE %s (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			
			INSERT INTO %s (data) VALUES 
				('Sample data for %s table'),
				('Another record in %s'),
				('Third record for testing');
		`, table, table, table, table)

		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create table %s: %v", table, err)
		}
	}

	return nil
}
