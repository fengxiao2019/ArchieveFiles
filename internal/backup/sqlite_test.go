package backup

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// createTestSQLiteDB creates a test SQLite database with sample data
func createTestSQLiteDB(t *testing.T, path string) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create a test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert sample data
	_, err = db.Exec(`
		INSERT INTO users (name, email) VALUES
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('Charlie', 'charlie@example.com')
	`)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Create an index
	_, err = db.Exec(`
		CREATE INDEX idx_email ON users(email)
	`)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
}

// verifyTestSQLiteDB verifies the test database has the expected data
func verifyTestSQLiteDB(t *testing.T, path string) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Check row count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 rows, got %d", count)
	}

	// Verify specific data
	var name, email string
	err = db.QueryRow("SELECT name, email FROM users WHERE id = 1").Scan(&name, &email)
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}

	if name != "Alice" || email != "alice@example.com" {
		t.Errorf("Expected Alice/alice@example.com, got %s/%s", name, email)
	}

	// Verify index exists
	rows, err := db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='index' AND name='idx_email'
	`)
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("Index idx_email not found")
	}
}

func TestSafeCopySQLiteDatabase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sqlite_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Create test database
	createTestSQLiteDB(t, sourcePath)

	// Test safe copy
	err = SafeCopySQLiteDatabase(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("SafeCopySQLiteDatabase failed: %v", err)
	}

	// Verify target database
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Fatal("Target database was not created")
	}

	// Verify data integrity
	verifyTestSQLiteDB(t, targetPath)
}

func TestVacuumIntoBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vacuum_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Create test database
	createTestSQLiteDB(t, sourcePath)

	// Test VACUUM INTO
	err = vacuumIntoBackup(sourcePath, targetPath)
	if err != nil {
		// VACUUM INTO might not be available on older SQLite versions
		t.Skipf("VACUUM INTO not available: %v", err)
	}

	// Verify target database
	verifyTestSQLiteDB(t, targetPath)
}

func TestCopyDatabaseTableByTable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "table_copy_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Create test database
	createTestSQLiteDB(t, sourcePath)

	// Test table-by-table copy
	err = copyDatabaseTableByTable(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("copyDatabaseTableByTable failed: %v", err)
	}

	// Verify target database
	verifyTestSQLiteDB(t, targetPath)
}

func TestSafeCopySQLiteDatabaseWithComplexSchema(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "complex_schema_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "source.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Create database with complex schema
	db, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create multiple tables
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			price REAL
		);

		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			product_id INTEGER,
			quantity INTEGER,
			FOREIGN KEY (product_id) REFERENCES products(id)
		);

		CREATE VIEW expensive_products AS
			SELECT * FROM products WHERE price > 100;

		CREATE TRIGGER update_timestamp
			AFTER UPDATE ON products
			BEGIN
				SELECT 1; -- Dummy trigger
			END;
	`)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert data
	_, err = db.Exec(`
		INSERT INTO products (name, price) VALUES
			('Laptop', 999.99),
			('Mouse', 29.99),
			('Keyboard', 79.99);

		INSERT INTO orders (product_id, quantity) VALUES
			(1, 2),
			(2, 5);
	`)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}
	db.Close()

	// Test safe copy
	err = SafeCopySQLiteDatabase(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("SafeCopySQLiteDatabase failed: %v", err)
	}

	// Verify target database has all schema objects
	targetDB, err := sql.Open("sqlite3", targetPath)
	if err != nil {
		t.Fatalf("Failed to open target database: %v", err)
	}
	defer targetDB.Close()

	// Check tables
	var tableCount int
	err = targetDB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("Failed to count tables: %v", err)
	}
	if tableCount != 2 {
		t.Errorf("Expected 2 tables, got %d", tableCount)
	}

	// Check views
	var viewCount int
	err = targetDB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='view'
	`).Scan(&viewCount)
	if err != nil {
		t.Fatalf("Failed to count views: %v", err)
	}
	if viewCount != 1 {
		t.Errorf("Expected 1 view, got %d", viewCount)
	}

	// Check triggers
	var triggerCount int
	err = targetDB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='trigger'
	`).Scan(&triggerCount)
	if err != nil {
		t.Fatalf("Failed to count triggers: %v", err)
	}
	if triggerCount != 1 {
		t.Errorf("Expected 1 trigger, got %d", triggerCount)
	}

	// Verify data
	var productCount, orderCount int
	targetDB.QueryRow("SELECT COUNT(*) FROM products").Scan(&productCount)
	targetDB.QueryRow("SELECT COUNT(*) FROM orders").Scan(&orderCount)

	if productCount != 3 {
		t.Errorf("Expected 3 products, got %d", productCount)
	}
	if orderCount != 2 {
		t.Errorf("Expected 2 orders, got %d", orderCount)
	}
}

func TestSafeCopySQLiteDatabaseEmpty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "empty_db_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "empty.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Create empty database with just a table
	db, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	_, err = db.Exec("CREATE TABLE empty_table (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	db.Close()

	// Test safe copy
	err = SafeCopySQLiteDatabase(sourcePath, targetPath)
	if err != nil {
		t.Fatalf("SafeCopySQLiteDatabase failed: %v", err)
	}

	// Verify target database
	targetDB, err := sql.Open("sqlite3", targetPath)
	if err != nil {
		t.Fatalf("Failed to open target database: %v", err)
	}
	defer targetDB.Close()

	var count int
	err = targetDB.QueryRow("SELECT COUNT(*) FROM empty_table").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query empty table: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 rows, got %d", count)
	}
}

func TestSafeCopySQLiteDatabaseNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "nonexistent_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "nonexistent.db")
	targetPath := filepath.Join(tempDir, "target.db")

	// Test with non-existent source
	err = SafeCopySQLiteDatabase(sourcePath, targetPath)
	if err == nil {
		t.Error("Expected error for non-existent source database")
	}
}
