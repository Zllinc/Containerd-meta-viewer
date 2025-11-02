# Containerd Meta Viewer 开发文档

## 项目概述

Containerd Meta Viewer 是一个用于查看 containerd containerd snapshotter 元数据的 CLI 工具。该工具允许用户查看存储在 bolt 数据库中的快照、containerd 存储信息和 LVM 映射。

## 项目结构

```
containerd-meta-viewer/
├── cmd/                    # 命令行接口实现
│   ├── root.go            # 根命令和全局参数
│   ├── buckets.go         # buckets 相关命令
│   ├── snapshots.go       # snapshots 相关命令
│   └── containerd.go          # containerd 相关命令
├── internal/              # 内部实现包
│   ├── database/          # 数据库操作
│   │   ├── reader.go      # BoltDB 读取器
│   │   └── models.go      # 数据模型定义
│   ├── formatters/        # 输出格式化器
│   │   ├── table.go       # 表格格式输出
│   │   └── json.go        # JSON 格式输出
│   └── utils/             # 工具函数
│       └── binary.go      # 二进制数据处理
├── go.mod                 # Go 模块定义
├── go.sum                 # Go 模块校验
├── main.go                # 程序入口点
├── DEVELOPMENT.md         # 开发文档
└── README.md              # 使用文档
```

## 开发规范

### 代码规范

1. **语言**：所有代码使用英语编写，包括注释、变量名、函数名
2. **包命名**：使用简短、清晰的包名，遵循 Go 官方规范
3. **错误处理**：使用 `fmt.Errorf` 包装错误，提供上下文信息
4. **注释**：所有公开的函数、类型、常量必须有注释

### 代码结构规范

1. **命令结构**：
   - 每个命令模块一个文件（如 `buckets.go`, `snapshots.go`）
   - 命令使用 `cobra.Command` 结构
   - 每个子命令实现独立的 `runXxx` 函数

2. **数据访问层**：
   - 所有数据库操作封装在 `database/reader.go` 中
   - 使用只读事务确保数据安全
   - 错误处理包含详细的上下文信息

3. **输出格式化**：
   - 支持表格和 JSON 两种格式
   - 格式化器实现在 `formatters/` 包中
   - 统一的格式化接口

### 开发流程

1. **添加新功能**：
   - 在 `internal/database/models.go` 中定义数据模型
   - 在 `internal/database/reader.go` 中实现数据访问方法
   - 在 `cmd/` 中添加对应的命令处理
   - 在 `internal/formatters/` 中添加输出格式化支持

2. **修改现有功能**：
   - 确保向后兼容性
   - 更新相应的测试用例
   - 更新文档

## 核心组件说明

### 数据库读取器 (`database/reader.go`)

负责所有 BoltDB 的读取操作：
- `NewMetaReader()`: 创建数据库读取器，自动处理数据库锁定情况
- `ListBuckets()`: 列出所有顶级 bucket
- `ListSnapshots()`: 列出所有快照
- `GetSnapshot(key)`: 获取特定快照
- `ListDevboxStorage()`: 列出所有 containerd 存储条目
- `GetDevboxStorage(contentID)`: 获取特定 containerd 存储条目
- `SearchSnapshots()`: 搜索快照

**数据库锁定处理**: 当数据库被 containerd 进程锁定时，会自动复制数据库到临时文件进行读取。详细原理见 `doc/features/database_reader.md`。

### 数据模型 (`database/models.go`)

定义所有数据结构：
- `SnapshotInfo`: 快照信息
- `DevboxStorageInfo`: Devbox 存储信息
- `BucketInfo`: Bucket 信息

### 命令处理 (`cmd/`)

使用 Cobra 框架处理命令行参数和子命令：
- `root.go`: 根命令，定义全局参数
- `buckets.go`: bucket 相关命令
- `snapshots.go`: 快照相关命令
- `containerd.go`: containerd 相关命令

### 格式化器 (`formatters/`)

负责输出格式化：
- `table.go`: 表格格式输出
- `json.go`: JSON 格式输出

## 开发环境设置

### 前置要求

- Go 1.21 或更高版本
- Git

### 构建项目

```bash
# 克隆项目（如果适用）
git clone <repository-url>
cd containerd-meta-viewer

# 下载依赖
go mod tidy

# 构建可执行文件
go build -o containerd-meta-viewer .

# 运行
./containerd-meta-viewer --help
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/database

# 运行测试并显示覆盖率
go test -cover ./...
```

## 添加新功能指南

### 1. 添加新的数据模型

在 `internal/database/models.go` 中定义新的数据结构：

```go
// NewFeatureInfo represents information about a new feature
type NewFeatureInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### 2. 实现数据访问方法

在 `internal/database/reader.go` 中添加读取方法：

```go
// ListNewFeatures returns all new features
func (r *MetaReader) ListNewFeatures() ([]NewFeatureInfo, error) {
    // 实现逻辑
}
```

### 3. 添加命令处理

创建新的命令文件或在现有文件中添加子命令：

```go
// newFeatureCmd represents the new-feature command
var newFeatureCmd = &cobra.Command{
    Use:   "new-feature",
    Short: "Handle new feature",
    RunE:  runNewFeature,
}

func runNewFeature(cmd *cobra.Command, args []string) error {
    // 实现逻辑
}
```

### 4. 添加输出格式化支持

在 `formatters/table.go` 和 `formatters/json.go` 中添加格式化方法：

```go
// FormatNewFeatures formats new feature information as a table
func (f *TableFormatter) FormatNewFeatures(features []NewFeatureInfo) error {
    // 实现表格格式化
}
```

## 数据库锁定处理

### 自动复制机制

当 containerd 进程以写模式锁定数据库时，工具会自动处理：

1. **检测锁定**: 尝试以 ReadOnly 模式打开数据库（1秒超时）
2. **自动复制**: 如果检测到锁定，自动复制数据库文件到临时位置
3. **从副本读取**: 从临时副本读取数据，完全透明
4. **自动清理**: 读取完成后自动删除临时文件

详细实现原理请参考 `doc/features/database_reader.md`。

## 文档维护规范

### 必须同步的文档

每次修改代码后，**必须**同步更新以下文档：

1. **功能实现文档** (`doc/features/`)
   - 如果修改了功能实现逻辑，更新对应的功能文档
   - 例如：修改了数据库读取逻辑 → 更新 `doc/features/database_reader.md`
   - 添加了新功能 → 创建新的功能文档

2. **功能迭代文档** (`doc/changelog/features/`)
   - **必须**记录每次功能变更
   - 文件命名：以功能名称命名（如 `database_reader.md`）
   - 每次变更都要记录：
     - 变更日期（格式：YYYY-MM-DD）
     - 变更前的实现方式
     - 变更后的实现方式
     - 变更原因和影响
   - 例如：修改了数据库锁定处理 → 更新 `doc/changelog/features/database_reader.md`

3. **开发文档** (`doc/DEVELOPMENT.md`)
   - 如果修改了开发规范、架构或流程，更新本文档

4. **用户文档** (`README.md`)
   - 如果添加了新功能或修改了用户可见的行为，更新用户文档
   - 特别是故障排除部分，要包含常见问题和解决方案

### 文档同步检查清单

在提交代码前，检查以下事项：

- [ ] 是否修改了核心功能逻辑？
  - [ ] 是 → 更新 `doc/features/` 中对应的功能文档
- [ ] 是否修改了任何功能的行为？
  - [ ] 是 → 在 `doc/changelog/features/` 中记录变更
- [ ] 是否添加了新功能？
  - [ ] 是 → 创建新的功能文档和变更记录
- [ ] 是否修改了开发规范或流程？
  - [ ] 是 → 更新 `doc/DEVELOPMENT.md`
- [ ] 是否修改了用户可见的行为？
  - [ ] 是 → 更新 `README.md`

### 文档目录结构

```
doc/
├── DEVELOPMENT.md              # 开发文档
├── TEST.md                     # 测试文档
├── OSS_DEPLOYMENT.md           # OSS 部署文档
├── RENAME_SUMMARY.md           # 重命名总结
├── features/                   # 功能实现文档（每个功能一个文件）
│   ├── database_reader.md     # 数据库读取功能实现
│   ├── buckets_command.md     # buckets 命令实现（待创建）
│   └── ...
└── changelog/                   # 功能变更记录
    └── features/               # 按功能分类的变更记录
        ├── database_reader.md  # 数据库读取功能变更历史
        └── ...
```

## 注意事项

1. **数据库安全**：所有数据库操作使用只读模式，确保不会修改原始数据
2. **错误处理**：提供清晰的错误信息，包含足够的上下文用于调试
3. **向后兼容**：任何修改都要确保向后兼容性
4. **性能考虑**：对于大型数据库，考虑分页或限制输出结果数量
5. **内存使用**：及时关闭数据库连接，避免内存泄漏
6. **文档同步**：修改代码后必须同步更新相关文档（见上文"文档维护规范"）

## 调试技巧

1. **启用详细输出**：使用 `--verbose` 标志获取更详细的错误信息
2. **JSON 输出**：使用 `--output json` 获取结构化数据便于调试
3. **日志记录**：在关键操作点添加日志记录（如果需要）

## 发布流程

1. 更新版本号（如果适用）
2. 更新 CHANGELOG.md（如果存在）
3. 运行完整测试套件
4. 构建发布版本
5. 创建 Git 标签（如果使用 Git）