# 数据库读取功能实现

## 概述

数据库读取功能是 containerd-meta-viewer 的核心功能，负责从 BoltDB 数据库中安全地读取 containerd snapshotter 的元数据。

## 实现位置

- **核心实现**: `internal/database/reader.go`
- **数据模型**: `internal/database/models.go`
- **调用位置**: `cmd/*.go` 中的所有命令

## 核心组件

### MetaReader 结构

```go
type MetaReader struct {
    db       *bolt.DB      // BoltDB 数据库连接
    tempPath string        // 临时文件路径（如果数据库被复制）
}
```

### 关键方法

#### NewMetaReader(dbPath string) (*MetaReader, error)

创建数据库读取器实例。这是整个读取流程的入口点。

**工作原理**:

1. **首次尝试直接打开**:
   - 使用 ReadOnly 模式尝试打开数据库
   - 设置 1 秒超时，快速检测数据库是否被锁定
   - 如果成功，直接返回 MetaReader

2. **数据库被锁定时自动复制**:
   - 检测到超时错误（数据库被 containerd 进程以写模式锁定）
   - 创建临时文件（`/tmp/containerd-meta-viewer-*.db`）
   - 复制整个数据库文件到临时位置
   - 打开临时文件副本进行读取
   - 保存临时文件路径，以便后续清理

3. **错误处理**:
   - 文件不存在、权限问题等其他错误直接返回

**代码流程**:

```
NewMetaReader(dbPath)
  ├─> 尝试 ReadOnly 打开（1秒超时）
  ├─> 成功？
  │   └─> 返回 MetaReader（tempPath 为空）
  └─> 超时错误？
      ├─> 创建临时文件
      ├─> 复制数据库文件
      ├─> 打开临时副本
      └─> 返回 MetaReader（tempPath 为临时文件路径）
```

#### Close() error

关闭数据库连接并清理资源。

- 关闭 BoltDB 连接
- 如果存在临时文件，自动删除
- 确保资源完全释放

#### ListBuckets() ([]BucketInfo, error)

列出数据库中的所有顶级 bucket。

**实现原理**:
- 使用 `db.View()` 创建只读事务
- 遍历所有顶级 bucket
- 统计每个 bucket 的 key 数量
- 返回 BucketInfo 列表

#### ListSnapshots() ([]SnapshotInfo, error)

列出所有快照信息。

**实现原理**:
- 访问 `v1/snapshots` bucket
- 遍历每个快照 bucket
- 调用 `readSnapshotInfo()` 解析快照数据
- 返回完整的快照信息列表

#### GetSnapshot(key string) (*SnapshotInfo, error)

获取特定快照的详细信息。

#### ListDevboxStorage() ([]DevboxStorageInfo, error)

列出所有 devbox 存储条目。

#### GetDevboxStorage(contentID string) (*DevboxStorageInfo, error)

获取特定 devbox 存储条目。

#### SearchSnapshots(contentID, path string) ([]SnapshotInfo, error)

根据内容 ID 或路径搜索快照。

## 数据库锁定处理机制

### 问题背景

containerd 进程会以写模式（独占锁）打开 metadata.db 文件。当工具尝试读取时，如果数据库被锁定，直接打开会超时。

### 解决方案

**自动复制机制**:

1. **检测锁定**: 使用短超时（1秒）快速检测数据库是否被锁定
2. **复制策略**: 如果被锁定，自动复制整个数据库文件到临时位置
3. **从副本读取**: 从临时副本读取数据，避免等待锁定
4. **自动清理**: 读取完成后自动删除临时文件

### 优势

- **无阻塞**: 不会因为数据库被锁定而卡住
- **自动化**: 用户无需手动操作
- **安全**: 读取完成后自动清理临时文件
- **透明**: 对上层调用完全透明

### 注意事项

- 每次命令执行如果检测到锁定，都会复制一次
- 复制的是整个数据库文件（通常 512KB-几MB）
- 临时文件会在命令执行完成后自动删除
- 如果程序异常退出，临时文件可能残留（但会被下次清理或系统自动清理）

## 数据模型

### BucketInfo

```go
type BucketInfo struct {
    Name     string `json:"name"`      // Bucket 名称
    KeyCount int    `json:"key_count"` // Key 数量
}
```

### SnapshotInfo

包含快照的完整信息：ID、类型、父快照、创建时间、内容 ID、挂载路径等。

### DevboxStorageInfo

包含 devbox 存储信息：内容 ID、LVM 卷名、路径、状态等。

## 使用示例

```go
// 创建读取器
reader, err := database.NewMetaReader("/path/to/metadata.db")
if err != nil {
    return err
}
defer reader.Close() // 确保资源释放

// 列出所有 bucket
buckets, err := reader.ListBuckets()

// 列出所有快照
snapshots, err := reader.ListSnapshots()

// 获取特定快照
snapshot, err := reader.GetSnapshot("sha256:abc123")
```

## 性能考虑

- **只读事务**: 所有操作使用只读事务，不影响数据库性能
- **批量读取**: List 方法一次性读取所有数据，减少事务开销
- **临时文件**: 仅在必要时复制，避免不必要的 I/O
- **连接复用**: 同一 MetaReader 实例的多次操作复用同一个数据库连接

## 错误处理

所有方法都返回详细的错误信息，包含上下文：
- 文件不存在
- 权限被拒绝
- 数据库损坏
- 数据格式错误

