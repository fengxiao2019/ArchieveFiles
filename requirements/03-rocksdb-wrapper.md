# 任务03: RocksDB封装层

## 任务描述
实现RocksDB的C++封装层，提供线程安全的数据库操作接口，隐藏RocksDB的复杂性。

## 功能需求
1. 数据库连接管理（打开/关闭）
2. 基础CRUD操作（Get/Put/Delete）
3. 批量操作支持
4. 迭代器封装
5. 事务支持（可选）
6. 资源自动管理

## 技术要求
- RAII资源管理模式
- 异常安全保证
- 线程安全设计
- 智能指针管理资源
- 提供统一的错误处理

## 接口设计
```cpp
// include/rcli/db/rocksdb_wrapper.h
namespace rcli::db {
    class RocksDBWrapper {
    public:
        explicit RocksDBWrapper(const std::string& db_path);
        ~RocksDBWrapper();
        
        // 基础操作
        std::optional<DatabaseValue> get(const DatabaseKey& key, 
                                       const ColumnFamilyName& cf = "default");
        void put(const DatabaseKey& key, const DatabaseValue& value,
                const ColumnFamilyName& cf = "default");
        void remove(const DatabaseKey& key, 
                   const ColumnFamilyName& cf = "default");
        
        // 批量操作
        void write_batch(const std::vector<WriteOperation>& operations);
        
        // 迭代器
        std::unique_ptr<Iterator> new_iterator(const ColumnFamilyName& cf = "default");
        
        // 状态检查
        bool is_open() const noexcept;
        
    private:
        class Impl;
        std::unique_ptr<Impl> pimpl_;
    };
}
```

## 依赖关系
- 前置任务: 02-通用类型和异常处理
- 被依赖: 04-列族管理, 06-基础数据库操作

## 验收标准
1. [ ] 数据库连接管理正常
2. [ ] 基础CRUD操作功能完整
3. [ ] 异常处理机制完善
4. [ ] 资源泄漏测试通过
5. [ ] 线程安全测试通过
6. [ ] 性能测试达标
7. [ ] 单元测试覆盖率 >90%

## 估时
4-6小时 