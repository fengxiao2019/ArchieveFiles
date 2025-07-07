# RocksDB Archive Tool Makefile

# Variables
BINARY_NAME=rocksdb-archive
TEST_DB_PATH=./testdata/test_db
COVERAGE_FILE=coverage.out

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Build with race detection
.PHONY: build-race
build-race:
	@echo "Building $(BINARY_NAME) with race detection..."
	go build -race -o $(BINARY_NAME) .

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run short tests (excluding large dataset tests)
.PHONY: test-short
test-short:
	@echo "Running short tests..."
	go test -short -v ./...

# Generate test database
.PHONY: generate-testdb
generate-testdb:
	@echo "Generating test database..."
	@mkdir -p testdata
	go run testdata/generate_test_db.go $(TEST_DB_PATH)

# Generate mixed test databases for batch testing
.PHONY: generate-mixed-testdbs
generate-mixed-testdbs:
	@echo "Generating mixed test databases..."
	@mkdir -p testdata
	go run testdata/generate_mixed_test_dbs.go ./testdata/mixed_dbs

# Clean generated files
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	rm -f coverage.html
	rm -rf testdata/test_db*
	rm -rf testdata/mixed_dbs*
	rm -rf testdata/backup*
	rm -rf testdata/*.tar.gz
	rm -rf *.tar.gz
	rm -rf *.tar.zst
	m -f bench-* archiveFiles

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Lint the code
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	elif [ -f "$(HOME)/go/bin/golangci-lint" ]; then \
		$(HOME)/go/bin/golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(HOME)/go/bin/golangci-lint run; \
	fi

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet the code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Run all quality checks
.PHONY: check
check: fmt vet test-short

# Install the binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install .

# Example usage targets
.PHONY: example-backup
example-backup: generate-testdb build
	@echo "Example: Backup method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/backup-example -method backup

.PHONY: example-checkpoint
example-checkpoint: generate-testdb build
	@echo "Example: Checkpoint method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/checkpoint-example -method checkpoint

.PHONY: example-copy
example-copy: generate-testdb build
	@echo "Example: Copy method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/copy-example -method copy

.PHONY: example-compress
example-compress: generate-testdb build
	@echo "Example: Full workflow with compression"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/compress-example -archive ./testdata/archive-example.tar.gz -method backup -compress=true -remove-backup=true

# Batch processing examples
.PHONY: example-batch
example-batch: generate-mixed-testdbs build
	@echo "Example: Batch processing multiple databases"
	./$(BINARY_NAME) -source ./testdata/mixed_dbs -backup ./testdata/batch-backup -archive ./testdata/batch-archive.tar.gz -batch=true -compress=true -remove-backup=true

.PHONY: example-batch-filter
example-batch-filter: generate-mixed-testdbs build
	@echo "Example: Batch processing with file filtering"
	./$(BINARY_NAME) -source ./testdata/mixed_dbs -backup ./testdata/filtered-backup -include="*.db,*.sqlite" -batch=true -compress=true -remove-backup=true

.PHONY: example-sqlite-only
example-sqlite-only: generate-mixed-testdbs build
	@echo "Example: Process only SQLite databases"
	./$(BINARY_NAME) -source ./testdata/mixed_dbs -backup ./testdata/sqlite-backup -include="*.db,*.sqlite,*.sqlite3" -batch=true -compress=true -remove-backup=true

# New examples
.PHONY: example-multi
example-multi: build
	@echo "=== Multiple Source Directories Example ==="
	@mkdir -p testdata/dir1 testdata/dir2 testdata/dir3
	@# Create test databases in different directories
	@go run testdata/generate_test_db.go testdata/dir1/app.db sqlite
	@go run testdata/generate_test_db.go testdata/dir2/cache rocksdb
	@echo "App log from dir1" > testdata/dir1/application.log
	@echo "Error log from dir2" > testdata/dir2/error.log
	@echo "Debug info from dir3" > testdata/dir3/debug.txt
	./$(BINARY_NAME) -sources="testdata/dir1,testdata/dir2,testdata/dir3" -backup=backup_multi -archive=multi_sources.tar.gz -compress=true
	@echo "Multiple source directories archived to multi_sources.tar.gz"

.PHONY: example-logs
example-logs: build
	@echo "=== Log Files Only Example ==="
	@mkdir -p testdata/logs
	@echo "Application started at $(shell date)" > testdata/logs/application.log
	@echo "ERROR: Database connection failed" > testdata/logs/error.log
	@echo "DEBUG: Processing user request" > testdata/logs/debug.txt
	@echo "ACCESS: 192.168.1.1 GET /api/users" > testdata/logs/access_log
	@echo "AUDIT: Admin user logged in" > testdata/logs/audit_trail.log
	./$(BINARY_NAME) -source=testdata/logs -include="*.log,*.txt" -backup=backup_logs -archive=logs_only.tar.gz -compress=true
	@echo "Log files archived to logs_only.tar.gz"

.PHONY: example-filter
example-filter: build
	@echo "=== Filtering Example ==="
	./$(BINARY_NAME) -source=testdata/mixed_dbs -include="*.db,*.log" -exclude="*temp*,*cache*" -backup=backup_filtered -archive=filtered.tar.gz -compress=true
	@echo "Filtered databases archived to filtered.tar.gz"

.PHONY: example-uncompressed
example-uncompressed: build
	@echo "=== Uncompressed Backup Example ==="
	./$(BINARY_NAME) -source=testdata/mixed_dbs -backup=backup_uncompressed -compress=false -remove-backup=false
	@echo "Uncompressed backup created in backup_uncompressed/"

.PHONY: example-methods
example-methods: build
	@echo "=== RocksDB Backup Methods Comparison ==="
	@echo "Testing backup method:"
	./$(BINARY_NAME) -source=testdata/mixed_dbs/rocks1 -method=backup -backup=backup_method_backup -archive=method_backup.tar.gz
	@echo "Testing checkpoint method:"
	./$(BINARY_NAME) -source=testdata/mixed_dbs/rocks1 -method=checkpoint -backup=backup_method_checkpoint -archive=method_checkpoint.tar.gz
	@echo "Testing copy method:"
	./$(BINARY_NAME) -source=testdata/mixed_dbs/rocks1 -method=copy -backup=backup_method_copy -archive=method_copy.tar.gz

.PHONY: example-progress
example-progress: build
	@echo "=== Progress Bar Example ==="
	@echo "Creating test data..."
	@mkdir -p testdata/dir1 testdata/dir2
	@go run testdata/generate_test_db.go testdata/dir1/app.db rocksdb
	@go run testdata/generate_test_db.go testdata/dir2/cache rocksdb
	@echo "Application started at $(shell date)" > testdata/dir1/application.log
	@echo "ERROR: Database connection failed" > testdata/dir2/error.log
	@echo "Running with progress bar (default):"
	./$(BINARY_NAME) -sources="testdata/dir1,testdata/dir2" -archive="progress_demo.tar.gz" -progress=true

.PHONY: example-no-progress
example-no-progress: build
	@echo "=== No Progress Bar Example (for automation) ==="
	@mkdir -p testdata/dir3
	@echo "Debug info from dir3" > testdata/dir3/debug.txt
	@echo "Running without progress bar:"
	./$(BINARY_NAME) -source="testdata/dir3" -archive="no_progress.tar.gz" -progress=false
	@echo "All methods tested, archives created"

# View archive contents
.PHONY: view-archive
view-archive:
	@echo "=== Archive Contents ==="
	@if [ -f batch_all.tar.gz ]; then \
		echo "Contents of batch_all.tar.gz:"; \
		tar -tzf batch_all.tar.gz | head -20; \
	else \
		echo "No archive found. Run 'make example-batch' first."; \
	fi

# Full demo - run all examples
.PHONY: demo
demo: generate-mixed-testdbs build example example-batch example-multi example-logs example-filter view-archive
	@echo ""
	@echo "=== Demo Complete ==="
	@echo "Created archives:"
	@ls -lh *.tar.gz 2>/dev/null || echo "No archives found"
	@echo ""
	@echo "Created backup directories:"
	@ls -d backup_* 2>/dev/null || echo "No backup directories found"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  build-race         - Build with race detection"
	@echo "  test               - Run all tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-short         - Run short tests only"
	@echo "  bench              - Run benchmarks"
	@echo "  generate-testdb    - Generate test database"
	@echo "  clean              - Clean generated files"
	@echo "  deps               - Install dependencies"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Vet code"
	@echo "  check              - Run quality checks"
	@echo "  install            - Install binary"
	@echo "  example-backup     - Run backup example"
	@echo "  example-checkpoint - Run checkpoint example"
	@echo "  example-copy       - Run copy example"
	@echo "  example-compress   - Run compression example"
	@echo "  example-batch      - Run batch processing example"
	@echo "  example-multi      - Run multiple source directories example"
	@echo "  example-logs       - Run log files only example"
	@echo "  example-filter     - Run filtering example"
	@echo "  example-uncompressed - Run uncompressed backup example"
	@echo "  example-methods    - Run RocksDB backup methods comparison"
	@echo "  view-archive       - View archive contents"
	@echo "  demo               - Run all examples"
	@echo "  help               - Show this help"

# Test with multiple sources
test: build
	./$(BINARY_NAME) testdata/dir1 testdata/dir2 output

# Test with progress disabled (automation mode)
test-no-progress: build
	./$(BINARY_NAME) -progress=false testdata/dir1 testdata/dir2 output

# Test different RocksDB methods
test-checkpoint: build
	./$(BINARY_NAME) -method=checkpoint testdata/dir1 output-checkpoint

test-backup: build
	./$(BINARY_NAME) -method=backup testdata/dir1 output-backup

test-copy: build
	./$(BINARY_NAME) -method=copy testdata/dir1 output-copy

# Performance comparison
benchmark: build test-checkpoint test-backup test-copy
	@echo "All backup methods completed. Compare the times above."

# Clean up test outputs
clean:
	rm -rf output* $(BINARY_NAME) backup_*

# Test JSON configuration support
test-config: build
	./$(BINARY_NAME) -config=test-config.json

test-config-override: build
	./$(BINARY_NAME) -config=test-config.json -method=backup -archive=config-override.tar.gz

# Generate configuration examples
generate-configs: build
	@mkdir -p configs
	./$(BINARY_NAME) -generate-config=configs/sample.json
	@echo "Configuration examples created in configs/ directory"

# Test with different configuration files
test-dev-config: build generate-configs
	@echo "Testing development configuration..."
	./$(BINARY_NAME) -config=configs/development-backup.json || echo "Dev config test (expected to fail - paths don't exist)"

# Clean configuration files
clean-configs:
	rm -rf configs/ *.json config-* override-*

# Test with verification enabled
test-verify:
	./$(BINARY_NAME) -verify -method=checkpoint testdata/dir1 verified-backup.tar.gz

# Test verification with JSON config
test-verify-config:
	echo '{"source_paths":["testdata/dir1","testdata/dir2"],"archive_path":"verified-backup.tar.zst","method":"checkpoint","compression_format":"zstd","verify":true,"show_progress":false}' > verify-config.json
	./$(BINARY_NAME) -config=verify-config.json

# Clean verification files
clean-verify:
	rm -f verified-backup.tar.gz verified-backup.tar.zst verify-config.json

# Initialize default configuration file
init-config:
	./archiveFiles -init

# Test with default configuration (after creating it)
test-default-config: init-config
	@echo "Testing with default configuration..."
	./archiveFiles testdata default-config-test.tar.gz

# Test default config discovery
test-config-discovery:
	@echo "Creating test config in current directory..."
	echo '{"source_paths":["testdata"],"archive_path":"discovery-test.tar.gz","method":"checkpoint","verify":false,"show_progress":false}' > archiveFiles.conf
	@echo "Running without -config flag (should auto-discover)..."
	./archiveFiles
	@echo "Cleaning up..."
	rm -f archiveFiles.conf discovery-test.tar.gz

# Clean default configuration files
clean-default-config:
	rm -f archiveFiles.conf archiveFiles.json config.json .archiveFiles.conf .archiveFiles.json
	rm -f default-config-test.tar.gz discovery-test.tar.gz

.PHONY: build test test-no-progress test-checkpoint test-backup test-copy benchmark clean test-config test-config-override generate-configs test-dev-config clean-configs test-verify test-verify-config clean-verify init-config test-default-config test-config-discovery clean-default-config 