# Database Archive Tool

A Go tool for backing up and archiving RocksDB and SQLite databases, especially useful when the databases are in use by other processes.

## Features

- **Read-only Access**: Open databases in read-only mode, safe for concurrent use
- **Multi-database Support**: Supports both RocksDB (directory) and SQLite (file) databases
- **Batch Processing**: Scan a directory and archive all databases found
- **Smart Detection**: Automatically detects database type (RocksDB directory or SQLite file)
- **File Filtering**: Supports include/exclude file patterns for flexible selection
- **Multiple Backup Methods**: For RocksDB, supports three backup methods:
  - `backup`: Use RocksDB's built-in backup engine (recommended)
  - `checkpoint`: Use RocksDB checkpoint feature
  - `copy`: Manually copy all key-value pairs
- **Automatic Compression**: Optionally compresses the backup into a tar.gz archive
- **Cleanup**: Optionally removes the backup directory after compression

## Installation

### Prerequisites

You need to have the RocksDB C++ library installed:

```bash
# macOS (Homebrew)
brew install rocksdb

# Ubuntu/Debian
sudo apt-get install librocksdb-dev

# CentOS/RHEL
sudo yum install rocksdb-devel
```

### Build

```bash
git clone <your-repo>
cd archiveFiles
go mod tidy
go build -o rocksdb-archive
```

## Usage

### Basic Usage

```bash
# Backup a single RocksDB database
./rocksdb-archive -source /path/to/rocksdb

# Backup a single SQLite database
./rocksdb-archive -source /path/to/database.db

# Batch process all databases in a directory
./rocksdb-archive -source /path/to/database/directory -batch=true

# Automatically detect batch mode (if source is a directory)
./rocksdb-archive -source /path/to/database/directory

# Use file filtering to only process certain database types
./rocksdb-archive -source /path/to/directory -include="*.db,*.sqlite"
```

### Command-line Options

- `-source`: Source database path or directory (required)
- `-backup`: Backup path (default: backup_timestamp)
- `-archive`: Archive file path (default: backup_path.tar.gz)
- `-method`: RocksDB backup method: backup, checkpoint, copy (default: backup)
- `-batch`: Force batch mode for directory processing (default: auto-detect)
- `-include`: Include file patterns, comma-separated (e.g., "*.db,*.sqlite")
- `-exclude`: Exclude file patterns, comma-separated
- `-compress`: Compress the backup (default: true)
- `-remove-backup`: Remove backup directory after compression (default: true)

### Examples

```bash
# Full backup and archive workflow for a single RocksDB database
./rocksdb-archive \
  -source /var/lib/myapp/rocksdb \
  -backup /tmp/myapp-backup \
  -archive /backup/myapp-$(date +%Y%m%d).tar.gz \
  -method backup \
  -compress=true \
  -remove-backup=true

# Batch process all databases in a directory
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/batch-backup \
  -archive /backup/all-databases-$(date +%Y%m%d).tar.gz \
  -batch=true \
  -compress=true \
  -remove-backup=true

# Only backup SQLite databases
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/sqlite-backup \
  -archive /backup/sqlite-$(date +%Y%m%d).tar.gz \
  -include="*.db,*.sqlite,*.sqlite3" \
  -compress=true \
  -remove-backup=true

# Exclude certain files
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/filtered-backup \
  -exclude="*temp*,*cache*" \
  -compress=true \
  -remove-backup=true
```

## Notes

- **Read-only mode**: The tool opens databases in read-only mode and does not interfere with running applications.
- **Disk space**: Ensure you have enough disk space for backups and archives.
- **Permissions**: Make sure you have read access to the source databases and write access to the backup/archive locations.
- **Dependencies**: The RocksDB C++ library must be installed on your system.

## Troubleshooting

- **"failed to open source db"**: Check the source path and permissions.
- **"no required module provides package"**: Run `go mod tidy` to download dependencies.
- **Link errors**: Ensure the RocksDB library is installed and available to the linker.

## License

MIT License 