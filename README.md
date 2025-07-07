# archiveFiles

A powerful tool for archiving RocksDB and SQLite databases with comprehensive progress tracking, multiple backup methods, and JSON configuration support.

## ✨ Features

- **Multiple Database Support**: Archive RocksDB databases, SQLite files, and log files
- **Three Backup Methods**: 
  - `checkpoint`: Fast hard-link based backup (default)
  - `backup`: Native RocksDB backup engine 
  - `copy`: Record-by-record copy with progress tracking
- **Multiple Source Support**: Process multiple source directories in one command
- **Progress Tracking**: Real-time progress bars with ETA and throughput information
- **Flexible Compression**: Support for gzip, zstd, and lz4 compression formats
- **Pattern Filtering**: Include/exclude files based on patterns
- **JSON Configuration**: Configuration file support for complex setups
- **Batch Processing**: Automatically discover and process multiple databases
- **Data Verification**: Verify backup integrity against source data (optional)

## 🚀 Quick Start

### Basic Usage

```bash
# Simple backup with progress
./archiveFiles database1 database2 archive.tar.gz

# High-performance checkpoint backup
./archiveFiles -method=checkpoint /data/databases /backup/db-backup.tar.zst
```

### Using JSON Configuration

```bash
# Generate a sample configuration file
./archiveFiles -generate-config=my-config.json

# Run with configuration file
./archiveFiles -config=my-config.json

# Override config with command line flags
./archiveFiles -config=my-config.json -method=backup -archive=override.tar.gz
```

## 📋 Configuration

### Command Line Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-config` | string | | JSON configuration file path |
| `-generate-config` | string | | Generate sample config file and exit |
| `-source` | string | | Source database path or directory |
| `-sources` | string | | Multiple source paths (comma-separated) |
| `-backup` | string | `backup_timestamp` | Backup directory path |
| `-archive` | string | `backup_path.tar.gz` | Final archive path |
| `-method` | string | `checkpoint` | RocksDB backup method |
| `-compress` | bool | `true` | Compress archived files |
| `-remove-backup` | bool | `true` | Remove backup directory after compression |
| `-batch` | bool | `false` | Process directory containing multiple databases |
| `-include` | string | | Include file patterns (e.g., "*.db,*.log") |
| `-exclude` | string | | Exclude file patterns (e.g., "*temp*,*cache*") |
| `-filter` | string | | Filter pattern for source paths |
| `-compression` | string | `gzip` | Compression format: `gzip`, `zstd`, `lz4` |
| `-progress` | bool | `true` | Show progress bar during archival |
| `-verify` | bool | `false` | Verify backup data integrity against source |

### JSON Configuration

JSON configuration files provide a convenient way to save and reuse complex backup settings. Command line flags always override JSON configuration values.

#### Sample Configuration

```json
{
  "source_paths": [
    "/data/databases/production",
    "/data/logs/application"
  ],
  "backup_path": "/backup/daily/backup_$(date +%Y%m%d)",
  "archive_path": "/backup/archives/production_$(date +%Y%m%d).tar.zst",
  "method": "checkpoint",
  "compress": true,
  "remove_backup": true,
  "batch_mode": true,
  "include_pattern": "*.db,*.sqlite,*.log",
  "exclude_pattern": "*temp*,*cache*,*.tmp",
  "show_progress": false,
  "filter": "",
  "compression_format": "zstd",
  "verify": false
}
```

#### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `source_paths` | array | List of source directories or files |
| `backup_path` | string | Temporary backup directory |
| `archive_path` | string | Final compressed archive path |
| `method` | string | RocksDB backup method |
| `compress` | boolean | Enable compression |
| `remove_backup` | boolean | Remove backup directory after archiving |
| `batch_mode` | boolean | Process multiple databases in directories |
| `include_pattern` | string | File patterns to include |
| `exclude_pattern` | string | File patterns to exclude |
| `show_progress` | boolean | Display progress bar |
| `filter` | string | Filter pattern for source paths |
| `compression_format` | string | Compression format: gzip, zstd, lz4 |
| `verify` | bool | Verify backup data integrity against source |

## 🏎️ RocksDB Backup Methods

Choose the optimal backup method for your use case:

### 1. Checkpoint (Default - Recommended)
- **Speed**: ⚡ Fastest (uses hard-links)
- **Use Case**: Development, frequent backups
- **Pros**: Nearly instantaneous, minimal disk usage
- **Cons**: Requires same filesystem, not suitable for remote backup

```bash
./archiveFiles -method=checkpoint /data/rocksdb /backup/
```

### 2. Backup (Native Engine)
- **Speed**: 🚀 Fast (native RocksDB backup)
- **Use Case**: Production environments, incremental backups
- **Pros**: Supports incremental backups, cross-filesystem
- **Cons**: Slightly slower than checkpoint

```bash
./archiveFiles -method=backup /data/rocksdb /backup/
```

### 3. Copy (Record-by-Record)
- **Speed**: 🐌 Slowest (compatibility mode)
- **Use Case**: Maximum compatibility, corrupted databases
- **Pros**: Works with any RocksDB version, thorough verification
- **Cons**: Much slower, higher resource usage

```bash
./archiveFiles -method=copy /data/rocksdb /backup/
```

## 📊 Performance Comparison

Based on a 71.3 KB test dataset:

| Method | Time | Archive Size | Speed |
|--------|------|-------------|--------|
| Checkpoint | 0.064s | 3.5 KB | Fastest |
| Backup | 0.078s | 3.7 KB | Fast |
| Copy | 0.114s | 14.9 KB | Slowest |

## 💼 Use Cases & Examples

### Development Workflow

```bash
# Quick daily backup
./archiveFiles -config=configs/development-backup.json

# Override for specific test
./archiveFiles -config=dev-config.json -archive=test-backup-$(date +%H%M).tar.gz
```

### Production Automation

```bash
# Automated daily backup with timestamp
./archiveFiles -config=configs/production-backup.json -progress=false

# Weekly full backup with maximum compression
./archiveFiles -config=prod-config.json -compression=zstd -archive=/backup/weekly/$(date +%Y%m%d).tar.zst
```

### Selective Backup

```bash
# Only database files, exclude logs
./archiveFiles -include="*.db,*.sqlite" -exclude="*.log" /data/mixed /backup/db-only.tar.gz

# Only log files for troubleshooting
./archiveFiles -include="*.log,*.txt" /data/logs /backup/logs-$(date +%Y%m%d).tar.gz
```

## 🛠️ Installation & Building

```bash
# Build from source
go build -o archiveFiles main.go

# Run tests
go test -v

# Build with all optimizations
make build

# Run performance benchmarks
make benchmark
```

## 📝 Configuration Examples

The tool includes several pre-configured examples:

```bash
# Generate sample configurations
make generate-configs

# Test with development config
make test-config

# Test config override behavior
make test-config-override
```

## 🧪 Testing

Comprehensive test suite with 20+ test cases:

```bash
# Run all tests
go test -v

# Test specific functionality
go test -run TestJSON
go test -run TestRocksDBBackupMethods
go test -run TestConfigMerging
```

## 🔧 Advanced Features

- **Progress Tracking**: Real-time progress with speed and ETA calculations
- **Multi-source Support**: Process multiple directories in one command
- **Smart Detection**: Automatic database type detection
- **Flexible Patterns**: Include/exclude with glob patterns
- **Compression Options**: Multiple algorithms with optimal defaults
- **Error Handling**: Graceful handling of permission issues and corrupted files
- **Logging**: Comprehensive logging for troubleshooting

## 📋 Requirements

- Go 1.22+
- RocksDB C++ library
- SQLite3

## 🤝 Contributing

1. Write tests for new features
2. Follow TDD workflow
3. Ensure all tests pass
4. Update documentation

## 📄 License

MIT License - see LICENSE file for details.

## 🔍 Data Verification

The verification feature ensures that your backup data is identical to the source data. When enabled with `-verify` or `"verify": true` in JSON config, the tool performs comprehensive integrity checks:

### Verification Methods

- **RocksDB**: Compares record counts, then iterates through all key-value pairs to ensure exact matches
- **SQLite**: Validates schema integrity and compares table data using checksums
- **Log Files**: Performs byte-by-byte comparison of file contents

### Usage Examples

```bash
# Verify backup during creation
./archiveFiles -verify -method=checkpoint /data/rocksdb verified-backup.tar.gz

# JSON configuration with verification
{
  "source_paths": ["/data/db1", "/data/db2"],
  "archive_path": "verified-backup.tar.zst",
  "method": "checkpoint",
  "compression_format": "zstd",
  "verify": true,
  "show_progress": false
}
```

### Verification Output

```
2024-01-01 10:00:01 Processing testdb (RocksDB)...
2024-01-01 10:00:02 Verifying testdb...
2024-01-01 10:00:03 ✅ Verification passed for testdb
2024-01-01 10:00:04 Successfully processed testdb
```

**Note**: Verification adds time to the backup process as it reads and compares all data. For large databases, consider running verification periodically rather than on every backup. 