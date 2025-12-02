# RocksDB Archive Tool Makefile

# Variables
BINARY_NAME=archiveFiles
COVERAGE_FILE=coverage.out
TEST_DB_PATH=testdata/test_db

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/archiveFiles

# Build with race detection
.PHONY: build-race
build-race:
	@echo "Building $(BINARY_NAME) with race detection..."
	go build -race -o $(BINARY_NAME) ./cmd/archiveFiles

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
	@echo "Generating test RocksDB..."
	cd testdata && go run generate_test_db.go test_db rocksdb

# Generate mixed test databases
.PHONY: generate-mixed-testdbs
generate-mixed-testdbs:
	@echo "Generating mixed test databases..."
	cd testdata && go run generate_mixed_test_dbs.go mixed_dbs

# Clean generated files
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	rm -rf output* backup_*
	rm -f coverage.html
	rm -rf testdata/test_db*
	rm -rf testdata/mixed_dbs*
	rm -rf testdata/backup*
	rm -rf testdata/*.tar.gz
	rm -rf *.tar.gz
	rm -rf *.tar.zst
	rm -f bench-*
	rm -f coverage.html
	rm -rf testdata/testdata

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
example-backup: build generate-testdb
	@echo "Example: Backup method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/backup-example -method backup -compress=false

.PHONY: example-checkpoint
example-checkpoint: build generate-testdb
	@echo "Example: Checkpoint method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/checkpoint-example -method checkpoint -compress=false

.PHONY: example-copy
example-copy: build generate-testdb
	@echo "Example: Copy method"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/copy-example -method copy -compress=false

.PHONY: example-compress
example-compress: build generate-testdb
	@echo "Example: Full workflow with compression"
	./$(BINARY_NAME) -source $(TEST_DB_PATH) -backup ./testdata/compress-example -archive ./testdata/compressed.tar.gz

.PHONY: example-batch
example-batch: build generate-mixed-testdbs
	@echo "Example: Batch processing multiple databases"
	./$(BINARY_NAME) -sources="testdata/mixed_dbs/dir1,testdata/mixed_dbs/dir2,testdata/mixed_dbs/dir3" -backup=backup_batch -archive=batch_result.tar.gz

.PHONY: example-batch-filter
example-batch-filter: build generate-mixed-testdbs
	@echo "Example: Batch processing"
	./$(BINARY_NAME) -sources="testdata/mixed_dbs/dir1,testdata/mixed_dbs/dir2" -backup=backup_filtered -archive=filtered_result.tar.gz

.PHONY: example-sqlite-only
example-sqlite-only: build generate-mixed-testdbs
	@echo "Example: SQLite databases"
	./$(BINARY_NAME) -source="testdata/mixed_dbs/dir2" -backup=backup_sqlite -archive=sqlite_result.tar.gz

# New examples
.PHONY: example-multi
example-multi: build generate-mixed-testdbs
	@echo "=== Multiple Source Directories Example ==="
	./$(BINARY_NAME) -sources="testdata/mixed_dbs/dir1,testdata/mixed_dbs/dir2,testdata/mixed_dbs/dir3" -backup=backup_multi -archive=multi_sources.tar.gz
	@echo "Multiple source directories archived to multi_sources.tar.gz"

.PHONY: example-logs
example-logs: build generate-mixed-testdbs
	@echo "=== Log Files Example ==="
	./$(BINARY_NAME) -source="testdata/mixed_dbs" -backup=backup_logs -archive=logs_only.tar.gz
	@echo "Log files archived to logs_only.tar.gz"

.PHONY: example-progress
example-progress: build generate-mixed-testdbs
	@echo "=== Progress Bar Example ==="
	@echo "Running with progress bar (default):"
	./$(BINARY_NAME) -sources="testdata/mixed_dbs/dir1,testdata/mixed_dbs/dir2" -archive="progress_demo.tar.gz"

.PHONY: example-no-progress
example-no-progress: build
	@echo "=== Automation Mode (no progress bar) ==="
	@mkdir -p testdata/dir3
	@echo "Test data" > testdata/dir3/test.log
	@echo "Running in automation mode (error log level):"
	./$(BINARY_NAME) -source="testdata/dir3" -archive="no_progress.tar.gz" -log-level=error
	@echo "Archive created in automation mode"

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
demo: build example example-batch example-multi example-logs example-filter view-archive
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
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-race          - Run tests with race detection"
	@echo "  test-short         - Run short tests only"
	@echo "  bench              - Run benchmarks"
	@echo "  clean              - Clean generated files"
	@echo "  deps               - Install dependencies"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Vet code"
	@echo "  check              - Run quality checks"
	@echo "  install            - Install binary"
	@echo "  generate-testdb    - Generate test RocksDB database"
	@echo "  generate-mixed-testdbs - Generate mixed test databases"
	@echo "  example-backup     - Run backup example"
	@echo "  example-checkpoint - Run checkpoint example"
	@echo "  example-copy       - Run copy example"
	@echo "  example-compress   - Run compression example"
	@echo "  example-batch      - Run batch processing example"
	@echo "  example-multi      - Run multiple sources example"
	@echo "  example-progress   - Run progress bar example"
	@echo "  init-config        - Generate default configuration file"
	@echo "  test-verify        - Test backup verification"
	@echo "  test-config        - Test configuration loading"
	@echo "  help               - Show this help message"


# Test with automation mode (progress disabled via error log level)
test-no-progress: build
	./$(BINARY_NAME) -log-level=error -source=testdata/dir1

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
	echo '{"source_paths":["testdata/dir1","testdata/dir2"],"archive_path":"verified-backup.tar.gz","method":"checkpoint","verify":true,"log_level":"error"}' > verify-config.json
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
	echo '{"source_paths":["testdata"],"archive_path":"discovery-test.tar.gz","method":"checkpoint","verify":false,"log_level":"error"}' > archiveFiles.conf
	@echo "Running without -config flag (should auto-discover)..."
	./archiveFiles
	@echo "Cleaning up..."
	rm -f archiveFiles.conf discovery-test.tar.gz

# Clean default configuration files
clean-default-config:
	rm -f archiveFiles.conf archiveFiles.json config.json .archiveFiles.conf .archiveFiles.json
	rm -f default-config-test.tar.gz discovery-test.tar.gz

.PHONY: build test test-no-progress test-checkpoint test-backup test-copy benchmark clean test-config test-config-override generate-configs test-dev-config clean-configs test-verify test-verify-config clean-verify init-config test-default-config test-config-discovery clean-default-config 