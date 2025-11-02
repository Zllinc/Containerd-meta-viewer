# Containerd Meta Viewer 架构与技术文档

## 目录

1. [项目概述](#项目概述)
2. [整体架构](#整体架构)
3. [技术栈](#技术栈)
4. [架构层次](#架构层次)
5. [数据流](#数据流)
6. [设计模式](#设计模式)
7. [关键组件](#关键组件)

## 项目概述

Containerd Meta Viewer 是一个命令行工具，用于读取和查看 containerd snapshotter 存储在 BoltDB 数据库中的元数据。它采用分层架构设计，实现了命令处理、数据访问、格式化输出的清晰分离。

## 整体架构

### 架构图

```
┌─────────────────────────────────────────────────────────┐
│                     CLI 用户界面层                        │
│              (Cobra Command Framework)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ buckets  │  │snapshots │  │  devbox  │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    业务逻辑层                              │
│                    (cmd/*.go)                            │
│  • 命令处理逻辑                                           │
│  • 参数验证                                               │
│  • 错误处理                                               │
└─────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┴─────────────────┐
        │                                     │
        ▼                                     ▼
┌──────────────────────┐          ┌──────────────────────┐
│   数据访问层           │          │   格式化层            │
│ (database/reader.go) │          │ (formatters/*.go)    │
│                      │          │                      │
│ • MetaReader         │          │ • TableFormatter     │
│ • 数据库读取          │          │ • JSONFormatter      │
│ • 数据模型            │          │ • 输出格式化          │
└──────────────────────┘          └──────────────────────┘
        │                                     │
        │                                     │
        └─────────────────┬─────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    存储层                                 │
│              BoltDB (bbolt)                              │
│  • 只读访问                                               │
│  • 自动锁定处理                                           │
│  • 临时文件复制机制                                       │
└─────────────────────────────────────────────────────────┘
```

### 目录结构

```
containerd-meta-viewer/
├── cmd/                    # CLI 命令层
│   ├── root.go            # 根命令，全局参数定义
│   ├── buckets.go         # buckets 命令实现
│   ├── snapshots.go       # snapshots 命令实现
│   └── devbox.go          # devbox 命令实现
│
├── internal/              # 内部实现（不对外暴露）
│   ├── database/          # 数据访问层
│   │   ├── reader.go      # BoltDB 读取器核心实现
│   │   └── models.go      # 数据模型定义
│   │
│   ├── formatters/        # 格式化层
│   │   ├── table.go       # 表格格式输出
│   │   └── json.go        # JSON 格式输出
│   │
│   └── utils/             # 工具函数
│       └── binary.go      # 二进制数据处理
│
├── main.go                # 程序入口点
├── go.mod                 # Go 模块依赖
└── doc/                   # 文档目录
```

## 技术栈

### 核心依赖

#### 1. **Go 语言**
- **版本**: Go 1.21+
- **选择原因**: 
  - 高性能和并发能力
  - 优秀的标准库
  - 适合 CLI 工具开发
  - 静态编译，部署简单

#### 2. **Cobra** (`github.com/spf13/cobra`)
- **版本**: v1.7.0
- **用途**: CLI 命令行框架
- **特性**:
  - 命令和子命令结构
  - 参数解析和验证
  - 自动生成帮助文档
  - 支持命令行补全
- **使用场景**:
  - 所有命令的定义和注册
  - 全局参数管理（`--db-path`, `--output`, `--verbose`）
  - 命令链的组织（root → buckets/snapshots/devbox）

#### 3. **BoltDB** (`go.etcd.io/bbolt`)
- **版本**: v1.3.7
- **用途**: 嵌入式键值数据库，用于读取 containerd metadata
- **特性**:
  - 纯 Go 实现，无外部依赖
  - ACID 事务支持
  - 高效的 B+ 树索引
  - 支持只读模式
- **使用场景**:
  - 读取 containerd snapshotter 的元数据
  - 遍历 bucket 结构
  - 查询快照和存储信息

#### 4. **Containerd** (`github.com/containerd/containerd`)
- **版本**: v1.7.0
- **用途**: 复用 containerd 的数据模型和工具函数
- **依赖组件**:
  - `metadata/boltutil`: BoltDB 工具函数（读取时间戳、标签等）
  - `snapshots`: 快照类型定义（KindActive, KindCommitted 等）
- **使用场景**:
  - 解析快照类型
  - 读取时间戳和标签
  - 保持与 containerd 数据格式兼容

### 间接依赖

- **golang.org/x/sys**: 系统调用（用于文件操作）
- **golang.org/x/sync**: 并发同步原语
- **Google Protobuf**: containerd 的数据序列化格式

## 架构层次

### 1. 表现层（Presentation Layer）

**位置**: `cmd/` 目录

**职责**:
- 命令行接口定义
- 参数解析和验证
- 用户交互

**实现**:
- 使用 Cobra 框架定义命令
- 每个命令对应一个文件
- 统一的错误处理和输出格式

**示例**:
```go
// cmd/buckets.go
var bucketsCmd = &cobra.Command{
    Use:   "buckets",
    Short: "List all top-level buckets",
    RunE:  runBuckets,
}

func runBuckets(cmd *cobra.Command, args []string) error {
    // 1. 创建数据访问层
    reader, err := database.NewMetaReader(dbPath)
    
    // 2. 获取数据
    buckets, err := reader.ListBuckets()
    
    // 3. 格式化输出
    formatter.FormatBuckets(buckets)
}
```

### 2. 业务逻辑层（Business Logic Layer）

**位置**: `cmd/*.go` 中的 `runXxx` 函数

**职责**:
- 业务流程编排
- 数据验证
- 错误处理

**特点**:
- 薄层设计，主要协调数据访问和格式化
- 不包含复杂业务逻辑
- 保持命令处理函数的简洁

### 3. 数据访问层（Data Access Layer）

**位置**: `internal/database/`

**职责**:
- 数据库连接管理
- 数据读取和转换
- 数据模型定义

**核心组件**:

#### MetaReader
```go
type MetaReader struct {
    db       *bolt.DB    // BoltDB 连接
    tempPath string      // 临时文件路径（如果数据库被复制）
}
```

**关键特性**:
- **只读访问**: 所有操作使用 `db.View()`，确保不修改数据
- **自动锁定处理**: 检测到数据库被锁定时，自动复制到临时文件
- **资源管理**: `Close()` 方法确保数据库连接和临时文件被清理

**主要方法**:
- `NewMetaReader()`: 创建读取器，处理数据库锁定
- `ListBuckets()`: 列出所有顶级 bucket
- `ListSnapshots()`: 列出所有快照
- `ListDevboxStorage()`: 列出 devbox 存储条目
- `GetSnapshot()`: 获取特定快照
- `SearchSnapshots()`: 搜索快照

### 4. 格式化层（Formatting Layer）

**位置**: `internal/formatters/`

**职责**:
- 数据格式化
- 输出呈现

**设计模式**: 策略模式（Strategy Pattern）

#### 格式化器接口

虽然没有显式接口，但两个格式化器实现了相同的模式：

```go
// TableFormatter - 表格格式
type TableFormatter struct {
    writer *tabwriter.Writer
}

// JSONFormatter - JSON 格式
type JSONFormatter struct {
    pretty bool  // 是否美化输出
}
```

**支持的格式**:
- **表格格式** (`table`): 人类可读的表格输出
- **JSON 格式** (`json`): 机器可读的结构化数据
  - 紧凑模式: 单行 JSON
  - 美化模式: 格式化的多行 JSON

### 5. 工具层（Utility Layer）

**位置**: `internal/utils/`

**职责**:
- 通用工具函数
- 二进制数据处理

**主要功能**:
- `ReadID()`: 读取 uint64 ID
- `ReadInodes()`: 读取 inode 数量
- `ReadSize()`: 读取大小信息

## 数据流

### 典型命令执行流程

以 `containerd-meta-viewer buckets` 为例：

```
1. 用户输入命令
   ↓
2. Cobra 解析命令和参数
   rootCmd.Execute()
   ↓
3. 调用 runBuckets()
   cmd/buckets.go
   ↓
4. 创建数据读取器
   database.NewMetaReader(dbPath)
   ├─> 尝试打开数据库
   ├─> 如果被锁定 → 复制到临时文件
   └─> 返回 MetaReader 实例
   ↓
5. 读取数据
   reader.ListBuckets()
   ├─> db.View() 创建只读事务
   ├─> 遍历顶级 bucket
   └─> 返回 []BucketInfo
   ↓
6. 格式化输出
   formatter.FormatBuckets(buckets)
   ├─> 根据 --output 参数选择格式化器
   ├─> table → TableFormatter
   └─> json → JSONFormatter
   ↓
7. 输出到标准输出
   ↓
8. 清理资源
   reader.Close()
   └─> 删除临时文件（如果存在）
```

### 数据模型流转

```
BoltDB 原始数据
    │
    ▼
[]byte (二进制)
    │
    ▼
readXxxInfo() 解析
    │
    ▼
Info 结构体 (SnapshotInfo/DevboxStorageInfo/BucketInfo)
    │
    ▼
格式化器处理
    │
    ▼
字符串输出 (表格或 JSON)
```

## 设计模式

### 1. 分层架构（Layered Architecture）

```
表现层 (cmd/)
    ↓
业务逻辑层 (cmd/ runXxx)
    ↓
数据访问层 (database/)
    ↓
存储层 (BoltDB)
```

**优点**:
- 关注点分离
- 易于测试
- 便于维护和扩展

### 2. 策略模式（Strategy Pattern）

**应用**: 输出格式化

```go
// 根据用户选择使用不同的格式化策略
if output == "json" {
    formatter := formatters.NewJSONFormatter(verbose)
    return formatter.FormatBuckets(buckets)
} else {
    formatter := formatters.NewTableFormatter()
    return formatter.FormatBuckets(buckets)
}
```

### 3. 资源管理模式（RAII 类似）

**应用**: 数据库连接和临时文件管理

```go
reader, err := database.NewMetaReader(dbPath)
if err != nil {
    return err
}
defer reader.Close()  // 确保资源被清理
```

### 4. 门面模式（Facade Pattern）

**应用**: MetaReader 封装复杂的数据库操作

`MetaReader` 提供了简洁的接口，隐藏了 BoltDB 的复杂性：
- 事务管理
- 锁定处理
- 错误处理

## 关键组件

### 1. 数据库锁定处理机制

**问题**: containerd 进程以写模式锁定数据库，直接读取会阻塞或失败。

**解决方案**: 智能复制机制

```go
// 1. 快速检测（1秒超时）
opts := &bolt.Options{
    ReadOnly: true,
    Timeout:  1 * time.Second,
}
db, err := bolt.Open(dbPath, 0400, opts)

// 2. 如果超时，复制数据库
if err != nil && err.Error() == "timeout" {
    copyFile(dbPath, tempPath)
    db, err = bolt.Open(tempPath, 0400, opts)
}

// 3. 使用后自动清理
defer reader.Close()  // 删除临时文件
```

### 2. 只读事务保证

所有数据库操作都使用 `db.View()`，确保：
- 不会修改原始数据
- 可以并发读取
- 数据一致性

```go
err := r.db.View(func(tx *bolt.Tx) error {
    // 只读操作
    bucket := tx.Bucket([]byte("v1"))
    return nil
})
```

### 3. 数据模型映射

将 BoltDB 的二进制数据映射为 Go 结构体：

```go
type SnapshotInfo struct {
    Key       string
    ID        uint64
    Kind      snapshots.Kind
    CreatedAt time.Time
    // ...
}
```

### 4. 格式化器设计

**表格格式化器**:
- 使用 `tabwriter` 对齐列
- 自动截断长字符串
- 处理空值和默认值

**JSON 格式化器**:
- 使用标准 `encoding/json`
- 支持紧凑和美化两种模式
- 自动处理 `omitempty` 标签

## 性能优化

### 1. 批量读取

一次性读取所有数据，而不是多次查询：

```go
// 好的做法：一次读取所有
snapshots, err := reader.ListSnapshots()

// 避免：多次查询
for id := range ids {
    snapshot, _ := reader.GetSnapshot(id)  // 不推荐
}
```

### 2. 只读事务

只读事务比读写事务更轻量，性能更好。

### 3. 临时文件策略

仅在必要时复制数据库（检测到锁定时），避免不必要的 I/O。

### 4. 内存管理

- 及时关闭数据库连接
- 自动清理临时文件
- 避免内存泄漏

## 扩展性

### 添加新命令

1. 在 `cmd/` 创建新文件
2. 实现 `runXxx` 函数
3. 在 `init()` 中注册命令

### 添加新的数据源

1. 在 `database/reader.go` 添加读取方法
2. 在 `database/models.go` 定义数据模型
3. 在格式化器中添加格式化方法

### 添加新的输出格式

1. 在 `formatters/` 创建新的格式化器
2. 实现格式化方法
3. 在命令处理中选择格式化器

## 安全性

### 1. 只读访问

所有数据库操作都是只读的，不会修改原始数据。

### 2. 错误处理

所有错误都被适当处理，不会暴露敏感信息。

### 3. 资源清理

使用 `defer` 确保资源（数据库连接、临时文件）被正确清理。

## 总结

Containerd Meta Viewer 采用了清晰的分层架构：
- **表现层**: Cobra CLI 框架
- **业务逻辑层**: 薄层协调
- **数据访问层**: 封装 BoltDB 操作
- **格式化层**: 策略模式实现多格式输出
- **工具层**: 通用功能支持

这种设计使得代码：
- ✅ 易于理解和维护
- ✅ 便于测试和扩展
- ✅ 性能优秀
- ✅ 安全可靠

