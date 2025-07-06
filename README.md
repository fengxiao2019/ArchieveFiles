# RocksDB Archive Tool

一个用于备份和归档RocksDB数据库的Go工具，特别适用于被其他进程占用的数据库。

## 功能特性

- **只读模式访问**: 以只读模式打开RocksDB，不会影响正在运行的进程
- **多种备份方式**: 支持三种不同的备份方法
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
# 使用默认设置备份数据库
./rocksdb-archive -source /path/to/rocksdb

# 指定备份路径
./rocksdb-archive -source /path/to/rocksdb -backup /path/to/backup

# 使用checkpoint方法
./rocksdb-archive -source /path/to/rocksdb -method checkpoint

# 不压缩备份
./rocksdb-archive -source /path/to/rocksdb -compress=false
```

### 命令行参数

- `-source`: 源RocksDB路径（必需）
- `-backup`: 备份路径（默认：backup_timestamp）
- `-archive`: 归档文件路径（默认：backup_path.tar.gz）
- `-method`: 备份方法，可选值：backup, checkpoint, copy（默认：backup）
- `-compress`: 是否压缩备份（默认：true）
- `-remove-backup`: 压缩后是否删除备份目录（默认：true）

### 示例

```bash
# 完整的备份和归档流程
./rocksdb-archive \
  -source /var/lib/myapp/rocksdb \
  -backup /tmp/myapp-backup \
  -archive /backup/myapp-$(date +%Y%m%d).tar.gz \
  -method backup \
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