# 任务04: 列族管理器

## 任务描述
实现RocksDB列族的管理功能，包括列族的创建、删除、查询和切换操作。

## 功能需求
1. 列族创建和删除
2. 列族列表查询
3. 当前列族状态管理
4. 列族配置管理
5. 列族元数据缓存

## 技术要求
- 线程安全的列族操作
- 错误处理和回滚机制
- 列族状态一致性保证
- 高效的列族查询

## 接口设计
```cpp
// include/rcli/db/column_family_manager.h
namespace rcli::db {
    class ColumnFamilyManager {
    public:
        explicit ColumnFamilyManager(RocksDBWrapper& db);
        
        // 列族操作
        void create_column_family(const ColumnFamilyName& name);
        void drop_column_family(const ColumnFamilyName& name);
        std::vector<ColumnFamilyName> list_column_families() const;
        bool column_family_exists(const ColumnFamilyName& name) const;
        
        // 当前列族管理
        void set_current_column_family(const ColumnFamilyName& name);
        ColumnFamilyName get_current_column_family() const;
        
        // 列族信息
        size_t get_column_family_count() const;
        std::string get_column_family_info(const ColumnFamilyName& name) const;
        
    private:
        RocksDBWrapper& db_;
        ColumnFamilyName current_cf_;
        mutable std::shared_mutex cf_mutex_;
        std::unordered_set<ColumnFamilyName> cf_cache_;
    };
}
```

## 核心功能
1. **列族生命周期管理**
   - 创建新列族
   - 安全删除列族
   - 列族存在性检查

2. **状态跟踪**
   - 当前活动列族
   - 列族元数据缓存
   - 列族使用统计

3. **并发安全**
   - 读写锁保护
   - 原子操作支持

## 依赖关系
- 前置任务: 03-RocksDB封装层
- 被依赖: 06-基础数据库操作, 05-命令解析器

## 验收标准
1. [ ] 列族创建/删除功能正常
2. [ ] 列族列表查询准确
3. [ ] 当前列族切换正常
4. [ ] 并发操作安全
5. [ ] 异常情况处理完善
6. [ ] 单元测试覆盖完整

## 估时
3-4小时 