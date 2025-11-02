# Buckets 命令功能实现

## 概述

`buckets` 命令用于列出 containerd metadata 数据库中的所有顶级 bucket，帮助用户了解数据库结构。

## 命令信息

- **命令名称**: `buckets`
- **实现文件**: `cmd/buckets.go`
- **依赖组件**: `database.MetaReader`, `formatters.TableFormatter`, `formatters.JSONFormatter`

## 实现原理

### 命令注册

```go
var bucketsCmd = &cobra.Command{
    Use:   "buckets",
    Short: "List all top-level buckets in the database",
    Long:  `List all top-level buckets in the containerd metadata database.`,
    RunE:  runBuckets,
}
```

### 执行流程

```20:42:cmd/buckets.go
func runBuckets(cmd *cobra.Command, args []string) error {
	// Create database reader
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	// Get buckets
	buckets, err := reader.ListBuckets()
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	// Format output
	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatBuckets(buckets)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatBuckets(buckets)
	}
}
```

### 执行步骤

1. **创建数据库读取器**
   - 调用 `database.NewMetaReader(dbPath)`
   - 如果数据库被锁定，会自动处理（见 `database_reader.md`）

2. **获取 bucket 列表**
   - 调用 `reader.ListBuckets()`
   - 返回所有顶级 bucket 及其 key 数量

3. **格式化输出**
   - 根据 `--output` 参数选择格式化器
   - `table`: 使用 `TableFormatter` 输出表格格式
   - `json`: 使用 `JSONFormatter` 输出 JSON 格式

4. **资源清理**
   - `defer reader.Close()` 确保数据库连接和临时文件被清理

## 输出格式

### 表格格式（默认）

```
NAME  KEYS
v1    1411
```

### JSON 格式

```json
[
  {
    "name": "v1",
    "key_count": 1411
  }
]
```

## 使用示例

```bash
# 使用默认数据库路径
./containerd-meta-viewer buckets

# 指定数据库路径
./containerd-meta-viewer --db-path /path/to/metadata.db buckets

# JSON 输出
./containerd-meta-viewer buckets --output json

# 格式化 JSON 输出
./containerd-meta-viewer buckets --output json --verbose
```

## 数据来源

数据来自 BoltDB 数据库的顶级 bucket，通过 `database.MetaReader.ListBuckets()` 方法获取。

## 错误处理

- **数据库打开失败**: 返回详细错误信息，包括文件路径和错误原因
- **读取失败**: 返回具体错误，帮助用户诊断问题
- **格式化失败**: 返回格式化器相关错误

## 性能考虑

- 使用只读事务，不影响数据库性能
- 一次性读取所有 bucket，操作高效
- 如果数据库被锁定，会自动复制（见 `database_reader.md`）

