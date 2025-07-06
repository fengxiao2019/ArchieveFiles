# Database Archive Tool

一个用于备份和归档多种数据库的Go工具，支持RocksDB和SQLite数据库，特别适用于被其他进程占用的数据库。

## 功能特性

- **多数据库支持**: 支持RocksDB和SQLite数据库
- **批量处理**: 可以扫描目录并批量处理多个数据库
- **智能检测**: 自动检测数据库类型（RocksDB目录或SQLite文件）
- **文件过滤**: 支持包含/排除文件模式过滤
- **只读模式访问**: 以只读模式打开数据库，不会影响正在运行的进程
- **多种备份方式**: 对于RocksDB支持三种不同的备份方法
  - `backup`: 使用RocksDB内置备份引擎（推荐）
  - `checkpoint`: 使用RocksDB checkpoint功能
  - `copy`: 手动遍历复制所有数据
- **自动压缩**: 支持将备份自动压缩为tar.gz格式
- **清理功能**: 可选择在压缩后删除原始备份目录

## 安装

### 前置条件

你需要先安装RocksDB的相关依赖库：

```bash
# macOS (使用Homebrew)
brew install rocksdb

# Ubuntu/Debian
sudo apt-get install librocksdb-dev

# CentOS/RHEL
sudo yum install rocksdb-devel
```

### 编译

```bash
git clone <your-repo>
cd archiveFiles
go mod tidy
go build -o rocksdb-archive
```

## 使用方法

### 基本用法

```bash
# 备份单个RocksDB数据库
./rocksdb-archive -source /path/to/rocksdb

# 备份单个SQLite数据库
./rocksdb-archive -source /path/to/database.db

# 批量处理目录中的所有数据库
./rocksdb-archive -source /path/to/database/directory -batch=true

# 自动检测目录（如果源是目录，会自动启用批量模式）
./rocksdb-archive -source /path/to/database/directory

# 使用文件过滤只处理特定类型的数据库
./rocksdb-archive -source /path/to/directory -include="*.db,*.sqlite"
```

### 命令行参数

- `-source`: 源数据库路径或目录（必需）
- `-backup`: 备份路径（默认：backup_timestamp）
- `-archive`: 归档文件路径（默认：backup_path.tar.gz）
- `-method`: RocksDB备份方法，可选值：backup, checkpoint, copy（默认：backup）
- `-batch`: 强制启用批量模式处理目录（默认：自动检测）
- `-include`: 包含文件模式，逗号分隔（例如："*.db,*.sqlite"）
- `-exclude`: 排除文件模式，逗号分隔
- `-compress`: 是否压缩备份（默认：true）
- `-remove-backup`: 压缩后是否删除备份目录（默认：true）

### 示例

```bash
# 单个RocksDB数据库完整备份流程
./rocksdb-archive \
  -source /var/lib/myapp/rocksdb \
  -backup /tmp/myapp-backup \
  -archive /backup/myapp-$(date +%Y%m%d).tar.gz \
  -method backup \
  -compress=true \
  -remove-backup=true

# 批量处理目录中的所有数据库
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/batch-backup \
  -archive /backup/all-databases-$(date +%Y%m%d).tar.gz \
  -batch=true \
  -compress=true \
  -remove-backup=true

# 只备份SQLite数据库
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/sqlite-backup \
  -archive /backup/sqlite-$(date +%Y%m%d).tar.gz \
  -include="*.db,*.sqlite,*.sqlite3" \
  -compress=true \
  -remove-backup=true

# 排除某些文件
./rocksdb-archive \
  -source /var/lib/databases \
  -backup /tmp/filtered-backup \
  -exclude="*temp*,*cache*" \
  -compress=true \
  -remove-backup=true
```

## 备份方法说明

### 1. backup（推荐）
- 使用RocksDB内置的备份引擎
- 效率最高，最安全
- 支持增量备份（虽然此工具当前只做全量备份）

### 2. checkpoint
- 创建数据库的一致性快照
- 速度快，占用空间小
- 适合快速备份

### 3. copy
- 手动遍历所有键值对进行复制
- 最灵活，但速度相对较慢
- 适合需要数据转换的场景

## 注意事项

1. **只读模式**: 此工具以只读模式打开数据库，不会影响正在运行的应用程序
2. **磁盘空间**: 确保有足够的磁盘空间存储备份和归档文件
3. **权限**: 确保有读取源数据库和写入目标路径的权限
4. **依赖**: 需要系统安装RocksDB库

## 故障排除

### 常见错误

1. **"failed to open source db"**: 检查源数据库路径是否正确，是否有读取权限
2. **"no required module provides package"**: 运行 `go mod tidy` 下载依赖
3. **链接错误**: 确保系统正确安装了RocksDB库

### 编译问题

如果遇到编译问题，可能需要指定RocksDB的路径：

```bash
CGO_CFLAGS="-I/path/to/rocksdb/include" \
CGO_LDFLAGS="-L/path/to/rocksdb/lib -lrocksdb -lstdc++ -lm -lz -lsnappy -llz4 -lzstd" \
go build -o rocksdb-archive
```

## 许可证

MIT License 