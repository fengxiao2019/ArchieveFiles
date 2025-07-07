# archiveFiles

A comprehensive tool for archiving RocksDB and SQLite databases with verification capabilities.

## Features

- **Multiple Database Support**: Archive RocksDB, SQLite databases, and log files
- **Flexible Backup Methods**: 
  - `checkpoint`: Uses native database checkpoint APIs (recommended)
  - `backup`: Uses database-specific backup engines
  - `copy`: Copies data record-by-record
- **Database Lock Detection**: Automatically detects and safely handles databases in use by other processes
- **Safe Backup for Live Databases**: Uses atomic operations safe for production databases
- **Batch Processing**: Process multiple databases and directories in one operation
- **Compression**: Optional gzip compression of archives
- **Verification**: Verify backup integrity against source data
- **Progress Tracking**: Real-time progress display for long-running operations
- **Configuration Management**: JSON-based configuration with auto-discovery

## Database Lock Handling

The tool automatically detects when databases are locked or in use by other processes and uses safe backup methods:

### RocksDB Lock Detection
- Detects RocksDB `LOCK` files
- Attempts read-only database access to verify lock status
- For locked databases, uses the checkpoint API which is safe for live databases

### SQLite Lock Detection  
- Detects SQLite WAL files (`-wal`, `-shm`, `-journal`)
- Tests for exclusive database locks
- For locked databases, uses SQLite's online backup API

### Safe Backup Methods
When a database is detected as locked:
- **RocksDB**: Uses the checkpoint API which creates atomic, consistent snapshots
- **SQLite**: Uses SQLite's backup command with table-by-table copying
- **Log Files**: Reports error for locked log files (cannot safely copy)

### Example Output
```
2025/07/08 07:46:20 Warning: Database /path/to/db is locked (SQLite database lock: Database is locked by another SQLite process)
2025/07/08 07:46:20 Attempting safe backup of locked SQLite: /path/to/db
2025/07/08 07:46:20 ✅ Successfully created safe backup of locked SQLite
```

## Installation

```bash
go build -o archiveFiles .
```

## Usage

### Basic Usage
```bash
# Archive a single database
./archiveFiles -source /path/to/database

# Archive multiple sources with compression
./archiveFiles -source "/path/to/db1,/path/to/db2" -compress

# Use configuration file
./archiveFiles -config backup-config.json
```

### Configuration File
Create a JSON configuration file for complex setups:

```json
{
  "source_paths": [
    "/path/to/rocksdb1",
    "/path/to/sqlite1.db",
    "/path/to/logs"
  ],
  "backup_path": "backup_2025",
  "method": "checkpoint",
  "compress": true,
  "verify": true,
  "include_pattern": "*.db,*.sqlite,*.log"
}
```

### Backup Methods

1. **Checkpoint Method** (Recommended)
   - Uses native database checkpoint APIs
   - Safe for live databases
   - Atomic and consistent snapshots
   
2. **Backup Method**
   - Uses database backup engines
   - Falls back to file copy if backup API unavailable
   
3. **Copy Method**
   - Copies data record-by-record
   - Slowest but most compatible

### Verification
Enable verification to ensure backup integrity:
```bash
./archiveFiles -source /path/to/db -verify
```

### Progress Tracking
View real-time progress for long operations:
```bash
./archiveFiles -source /path/to/large/db -progress
```

## Safety Features

### Production Database Safety
- **Lock Detection**: Automatically detects databases in use
- **Atomic Operations**: Uses checkpoint APIs for consistent snapshots
- **Read-Only Access**: Opens databases in read-only mode when possible
- **Fallback Mechanisms**: Graceful fallback to safe alternatives

### Data Integrity
- **Verification**: Compare backup data against source
- **Completeness Checks**: Verify all critical files are included
- **Checksum Validation**: Ensure data integrity during transfer

### Error Handling
- **Graceful Degradation**: Continue processing other databases if one fails
- **Detailed Logging**: Comprehensive error reporting and warnings
- **Recovery Options**: Multiple backup methods with automatic fallback

## Examples

### Archive Production Database
```bash
# Safe backup of live RocksDB
./archiveFiles -source /var/lib/rocksdb -method checkpoint -verify

# Safe backup of live SQLite with compression
./archiveFiles -source /var/lib/app.db -method backup -compress -verify
```

### Batch Processing
```bash
# Archive multiple database directories
./archiveFiles -source "/data/db1,/data/db2,/logs" -batch -compress
```

### Configuration-Based Backup
```bash
# Use predefined configuration
./archiveFiles -config production-backup.json
```

## Testing

Run the test suite:
```bash
go test -v
```

Generate test databases:
```bash
make generate-testdb
make generate-mixed-testdbs
```

## Requirements

- Go 1.22+
- RocksDB development libraries
- SQLite3 development libraries

## License

MIT License 