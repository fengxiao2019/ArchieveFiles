package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/linxGnu/grocksdb"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run generate_test_db.go <db_path>")
	}

	dbPath := os.Args[1]

	// Remove existing database if it exists
	if err := os.RemoveAll(dbPath); err != nil {
		log.Printf("Warning: failed to remove existing db: %v", err)
	}

	// Create database directory
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	// Create database with simple approach - just use default column family
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	defer opts.Destroy()

	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	writeOpts := grocksdb.NewDefaultWriteOptions()
	defer writeOpts.Destroy()

	// Populate database with all test data (using default column family)
	allData := map[string]string{
		// Configuration data
		"config:version":     "1.0.0",
		"config:environment": "test",
		"config:debug":       "true",
		"counter:total":      "12345",
		"counter:active":     "987",
		"status:health":      "ok",
		"status:uptime":      "3600",

		// User data
		"user:1001": `{"id":1001,"name":"Alice Johnson","email":"alice@example.com","age":28,"role":"admin","created_at":"2023-01-15T10:30:00Z"}`,
		"user:1002": `{"id":1002,"name":"Bob Smith","email":"bob@example.com","age":34,"role":"user","created_at":"2023-02-20T14:45:00Z"}`,
		"user:1003": `{"id":1003,"name":"Charlie Brown","email":"charlie@example.com","age":22,"role":"user","created_at":"2023-03-10T09:15:00Z"}`,
		"user:1004": `{"id":1004,"name":"Diana Prince","email":"diana@example.com","age":31,"role":"moderator","created_at":"2023-04-05T16:20:00Z"}`,
		"user:1005": `{"id":1005,"name":"Eve Wilson","email":"eve@example.com","age":26,"role":"user","created_at":"2023-05-12T11:00:00Z"}`,
		"user:1006": `{"id":1006,"name":"Frank Miller","email":"frank@example.com","age":39,"role":"admin","created_at":"2023-06-18T13:30:00Z"}`,
		"user:1007": `{"id":1007,"name":"Grace Lee","email":"grace@example.com","age":29,"role":"user","created_at":"2023-07-22T08:45:00Z"}`,
		"user:1008": `{"id":1008,"name":"Henry Davis","email":"henry@example.com","age":35,"role":"user","created_at":"2023-08-14T15:10:00Z"}`,
		"user:1009": `{"id":1009,"name":"Ivy Chen","email":"ivy@example.com","age":27,"role":"moderator","created_at":"2023-09-03T12:25:00Z"}`,
		"user:1010": `{"id":1010,"name":"Jack Thompson","email":"jack@example.com","age":33,"role":"user","created_at":"2023-10-08T17:40:00Z"}`,

		// Product data
		"product:p001": `{"id":"p001","name":"Laptop Pro","category":"electronics","price":1299.99,"stock":45,"description":"High-performance laptop"}`,
		"product:p002": `{"id":"p002","name":"Wireless Mouse","category":"electronics","price":29.99,"stock":150,"description":"Ergonomic wireless mouse"}`,
		"product:p003": `{"id":"p003","name":"Coffee Mug","category":"kitchenware","price":12.99,"stock":80,"description":"Ceramic coffee mug"}`,
		"product:p004": `{"id":"p004","name":"Notebook","category":"stationery","price":5.99,"stock":200,"description":"Lined notebook"}`,
		"product:p005": `{"id":"p005","name":"Desk Lamp","category":"furniture","price":89.99,"stock":25,"description":"LED desk lamp"}`,
		"product:p006": `{"id":"p006","name":"Keyboard","category":"electronics","price":79.99,"stock":60,"description":"Mechanical keyboard"}`,
		"product:p007": `{"id":"p007","name":"Water Bottle","category":"sports","price":19.99,"stock":120,"description":"Stainless steel water bottle"}`,
		"product:p008": `{"id":"p008","name":"Backpack","category":"bags","price":49.99,"stock":35,"description":"Travel backpack"}`,

		// Log data
		"log:2023-10-01T10:00:00Z": `{"timestamp":"2023-10-01T10:00:00Z","level":"INFO","message":"Application started","service":"web-server"}`,
		"log:2023-10-01T10:15:00Z": `{"timestamp":"2023-10-01T10:15:00Z","level":"INFO","message":"User login successful","service":"auth","user_id":1001}`,
		"log:2023-10-01T10:30:00Z": `{"timestamp":"2023-10-01T10:30:00Z","level":"WARN","message":"High memory usage detected","service":"monitor","memory_usage":"85%"}`,
		"log:2023-10-01T10:45:00Z": `{"timestamp":"2023-10-01T10:45:00Z","level":"ERROR","message":"Database connection failed","service":"api","error":"connection timeout"}`,
		"log:2023-10-01T11:00:00Z": `{"timestamp":"2023-10-01T11:00:00Z","level":"INFO","message":"Database connection restored","service":"api"}`,
		"log:2023-10-01T11:15:00Z": `{"timestamp":"2023-10-01T11:15:00Z","level":"DEBUG","message":"Cache miss for key user:1005","service":"cache"}`,
		"log:2023-10-01T11:30:00Z": `{"timestamp":"2023-10-01T11:30:00Z","level":"INFO","message":"Backup completed successfully","service":"backup"}`,
		"log:2023-10-01T11:45:00Z": `{"timestamp":"2023-10-01T11:45:00Z","level":"WARN","message":"Disk space low","service":"monitor","disk_usage":"92%"}`,
		"log:2023-10-01T12:00:00Z": `{"timestamp":"2023-10-01T12:00:00Z","level":"ERROR","message":"Payment processing failed","service":"payment","order_id":"ord_12345"}`,
		"log:2023-10-01T12:15:00Z": `{"timestamp":"2023-10-01T12:15:00Z","level":"INFO","message":"System maintenance scheduled","service":"admin","maintenance_time":"2023-10-02T02:00:00Z"}`,
	}

	for key, value := range allData {
		err := db.Put(writeOpts, []byte(key), []byte(value))
		if err != nil {
			log.Printf("Failed to put %s: %v", key, err)
		}
	}

	fmt.Printf("Test database created successfully at: %s\n", dbPath)
	fmt.Printf("Total records: %d\n", len(allData))
}
