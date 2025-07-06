# 任务01: 项目基础设施搭建

## 任务描述
搭建C++ RocksDB CLI项目的基础设施，包括目录结构、构建系统和基本配置文件。

## 功能需求
1. 创建标准的C++项目目录结构
2. 配置CMake构建系统
3. 设置依赖管理（vcpkg或Conan）
4. 创建基本的配置文件

## 技术要求
- 使用CMake 3.15+作为构建系统
- 支持C++17标准
- 跨平台兼容（Linux/macOS/Windows）
- 现代C++项目最佳实践

## 输出产物
```
rcli/
├── CMakeLists.txt              # 主CMake配置
├── cmake/                      # CMake模块
│   ├── FindRocksDB.cmake
│   └── CompilerWarnings.cmake
├── vcpkg.json                  # vcpkg依赖配置
├── .gitignore                  # Git忽略规则
├── include/rcli/               # 头文件目录
├── src/                        # 源文件目录
├── tests/                      # 测试目录
├── scripts/                    # 脚本目录
├── examples/                   # 示例目录
└── docs/                       # 文档目录
```

## 依赖关系
- 无前置任务
- 后续所有任务的基础

## 验收标准
1. [ ] 项目目录结构完整创建
2. [ ] CMake配置能成功生成构建文件
3. [ ] 依赖管理配置正确
4. [ ] 能在三大平台成功构建空项目
5. [ ] 编译器警告级别设置合理
6. [ ] 代码格式化配置（.clang-format）

## 估时
2-3小时 