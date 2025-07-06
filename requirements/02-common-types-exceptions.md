# 任务02: 通用类型和异常处理

## 任务描述
定义项目中使用的通用类型、常量和异常处理机制，为整个项目提供统一的基础类型支持。

## 功能需求
1. 定义项目通用类型和别名
2. 实现异常处理体系
3. 定义常用常量和枚举
4. 提供错误码和错误信息映射

## 技术要求
- 使用现代C++类型系统
- 异常安全设计
- 强类型定义避免类型混淆
- 提供清晰的错误信息

## 输出产物
```cpp
// include/rcli/common/types.h
namespace rcli {
    using ColumnFamilyName = std::string;
    using DatabaseKey = std::string;
    using DatabaseValue = std::string;
    using Timestamp = std::chrono::system_clock::time_point;
    
    enum class ScanDirection { Forward, Reverse };
    enum class ExportFormat { CSV, JSON };
    enum class LogLevel { DEBUG, INFO, WARN, ERROR };
}

// include/rcli/common/exceptions.h
namespace rcli {
    class RcliException : public std::exception { ... };
    class DatabaseException : public RcliException { ... };
    class ColumnFamilyException : public RcliException { ... };
    class CommandException : public RcliException { ... };
    class JsonException : public RcliException { ... };
}
```

## 依赖关系
- 前置任务: 01-项目基础设施搭建
- 后续任务的基础依赖

## 验收标准
1. [ ] 通用类型定义完整
2. [ ] 异常类层次结构清晰
3. [ ] 异常信息描述详细
4. [ ] 错误码映射完整
5. [ ] 单元测试覆盖完整
6. [ ] 文档说明清晰

## 估时
1-2小时 