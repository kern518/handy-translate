# 翻译历史记录功能

## 功能概述

本版本新增了翻译和解释历史记录保存功能，可以自动将用户的翻译和解释记录保存到本地文件中，按照日期和类型进行分类存储。

## 功能特性

- ✅ 翻译历史记录保存
- ✅ 解释历史记录保存
- ✅ 按日期自动分类存储
- ✅ JSON格式存储，便于后续处理
- ✅ 异步保存，不影响翻译性能
- ✅ 可通过配置文件启用/禁用

## 配置说明

在 `config.toml` 文件中添加以下配置：

```toml
[history]
enabled = true           # 启用历史记录功能
storage_path = "./data"  # 存储路径
```

### 配置项说明

- `enabled`: 是否启用历史记录功能
  - `true`: 启用
  - `false`: 禁用

- `storage_path`: 历史记录文件存储路径
  - 默认值: `"./data"`
  - 支持相对路径和绝对路径

## 文件存储结构

历史记录按照以下结构存储：

```
data/
├── history/
│   ├── translate/           # 翻译记录
│   │   ├── 2024-01-15.json
│   │   ├── 2024-01-16.json
│   │   └── ...
│   └── explain/            # 解释记录
│       ├── 2024-01-15.json
│       ├── 2024-01-16.json
│       └── ...
```

## 数据格式

### 翻译记录格式 (`data/history/translate/YYYY-MM-DD.json`)

```json
[
  {
    "id": "uuid-string",
    "type": "translate",
    "source_text": "Hello world",
    "result": "你好世界",
    "from_lang": "en",
    "to_lang": "zh",
    "timestamp": "2024-01-15T14:30:25+08:00"
  }
]
```

### 解释记录格式 (`data/history/explain/YYYY-MM-DD.json`)

```json
[
  {
    "id": "uuid-string",
    "type": "explain",
    "source_text": "machine learning",
    "result": "机器学习是人工智能的一个分支...",
    "template_id": "template1",
    "timestamp": "2024-01-15T16:45:30+08:00"
  }
]
```

## 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `id` | string | 唯一标识符 (UUID) |
| `type` | string | 记录类型: "translate" 或 "explain" |
| `source_text` | string | 源文本 |
| `result` | string | 翻译/解释结果 |
| `from_lang` | string | 源语言 (仅翻译记录) |
| `to_lang` | string | 目标语言 (仅翻译记录) |
| `template_id` | string | 解释模板ID (仅解释记录) |
| `timestamp` | string | 时间戳 (ISO 8601格式) |

## 使用方式

### 1. 启用功能

1. 将 `config.toml.bak` 重命名为 `config.toml`
2. 在配置文件中添加 `[history]` 配置段
3. 设置 `enabled = true`
4. 重启应用程序

### 2. 验证功能

- 执行翻译操作后，检查 `data/history/translate/` 目录
- 执行解释操作后，检查 `data/history/explain/` 目录
- 文件按日期命名，内容为JSON格式

### 3. 禁用功能

设置 `enabled = false` 即可禁用历史记录保存，应用程序将不再创建历史记录文件。

## 技术实现

### 集成点

历史记录功能已集成到以下翻译流程中：

1. **Translate方法** (`app.go:56`) - 普通翻译
2. **TranslateMeanings方法** (`app.go:74`) - 流式翻译
3. **ExplainStream方法** (`app.go:169`) - 流式解释
4. **processTranslate方法** (`app.go:455`) - 内部翻译处理
5. **processExplain方法** (`app.go:511`) - 内部解释处理

### 性能优化

- 使用 `go` 协程异步保存，不阻塞翻译流程
- 文件追加写入，避免重复读取大量数据
- JSON格式化存储，便于后续处理和分析

## 注意事项

1. **磁盘空间**: 历史记录会持续累积，请定期清理不需要的文件
2. **隐私安全**: 历史记录包含翻译内容，请妥善保管存储目录
3. **文件权限**: 确保应用程序对存储目录有读写权限
4. **配置同步**: 修改配置后需要重启应用程序生效

## 示例用法

```bash
# 启用历史记录功能
# 编辑 config.toml
[history]
enabled = true
storage_path = "./data"

# 启动应用
wails3 dev

# 执行几次翻译操作
# 检查生成的文件
ls data/history/translate/
ls data/history/explain/

# 查看文件内容
cat data/history/translate/$(date +%Y-%m-%d).json
```

## 扩展开发

如需基于历史记录功能进行扩展开发，可以：

1. 在 `history/history.go` 中添加查询方法
2. 开发历史记录分析工具
3. 实现历史记录导入/导出功能
4. 添加数据统计和可视化功能