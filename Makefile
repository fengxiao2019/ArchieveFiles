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
	golangci-lint run

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
	@echo "  help               - Show this help" 