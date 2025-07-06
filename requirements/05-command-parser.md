# 任务05: 命令解析器

## 任务描述
实现灵活的命令行解析器，支持交互式和直接命令执行模式，处理各种命令格式和参数。

## 功能需求
1. 命令行参数解析（CLI11）
2. 交互式命令解析
3. 命令验证和提示
4. 帮助系统
5. 命令历史和补全

## 支持的命令格式
```bash
# 列族管理
usecf <cf>                   # 切换列族
listcf                       # 列出所有列族
createcf <cf>                # 创建列族
dropcf <cf>                  # 删除列族

# 数据操作
get [<cf>] <key> [--pretty]  # 查询键值
put [<cf>] <key> <value>     # 插入/更新
prefix [<cf>] <prefix>       # 前缀查询
last [<cf>] [--pretty]       # 获取最后一个键值

# 高级操作
scan [<cf>] [start] [end] [options]
jsonquery [<cf>] <field> <value> [--pretty]
export [<cf>] <file_path>

# 系统命令
help                         # 显示帮助
exit/quit                    # 退出
```

## 技术要求
- 使用CLI11库进行参数解析
- 支持命令别名和简写
- 提供清晰的错误提示
- 支持命令补全（readline）

## 接口设计
```cpp
// include/rcli/repl/command_parser.h
namespace rcli::repl {
    struct ParsedCommand {
        std::string command;
        std::vector<std::string> args;
        std::unordered_map<std::string, std::string> options;
        std::optional<ColumnFamilyName> column_family;
    };
    
    class CommandParser {
    public:
        CommandParser();
        
        // 解析方法
        ParsedCommand parse_interactive_command(const std::string& input);
        ParsedCommand parse_cli_arguments(int argc, char* argv[]);
        
        // 验证和帮助
        bool validate_command(const ParsedCommand& cmd);
        std::string get_command_help(const std::string& command = "");
        std::vector<std::string> get_command_suggestions(const std::string& partial);
        
    private:
        void setup_cli_parser();
        std::unique_ptr<CLI::App> cli_app_;
        std::unordered_map<std::string, std::string> command_aliases_;
    };
}
```

## 依赖关系
- 前置任务: 02-通用类型和异常处理
- 被依赖: 07-REPL交互界面, 06-基础数据库操作

## 验收标准
1. [ ] CLI参数解析正确
2. [ ] 交互式命令解析准确
3. [ ] 命令验证机制完善
4. [ ] 帮助系统完整
5. [ ] 错误提示清晰
6. [ ] 支持命令补全
7. [ ] 单元测试覆盖完整

## 估时
4-5小时 