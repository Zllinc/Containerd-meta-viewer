# Containerd Meta Viewer

一个用于查看 containerd snapshotter 元数据的 CLI 工具。该工具允许用户查看存储在 bolt 数据库中的快照、存储信息和 LVM 映射。

## 功能特性

- 查看数据库中的所有 buckets
- 列出和搜索快照信息
- 查看 containerd 特定的存储信息
- 显示 LVM 卷名到挂载路径的映射
- 支持表格和 JSON 两种输出格式
- 提供详细和简洁的输出模式

## 安装

### 从源码构建

```bash
# 克隆项目
git clone <repository-url>
cd containerd-meta-viewer

# 下载依赖
go mod tidy

# 构建可执行文件
go build -o containerd-meta-viewer .

# 移动到系统路径（可选）
sudo mv containerd-meta-viewer /usr/local/bin/
```

## 使用方法

### 全局参数

- `--db-path, -p`: containerd metadata.db 文件路径（可选，默认为 `/var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db`）
- `--output, -o`: 输出格式，支持 `table`（默认）和 `json`
- `--verbose, -v`: 启用详细输出（仅在 JSON 格式下有效）

### 基本用法

现在工具会自动使用默认的数据库路径，您可以直接运行命令：

```bash
# 使用默认数据库路径
containerd-meta-viewer <command>

# 或者指定自定义数据库路径
containerd-meta-viewer --db-path /path/to/custom/metadata.db <command>
```

默认数据库路径：`/var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db`

### 命令参考

#### 1. 查看数据库 Buckets

列出数据库中的所有顶级 buckets：

```bash
# 使用默认数据库路径
containerd-meta-viewer buckets

# 或者指定自定义路径
containerd-meta-viewer --db-path /path/to/metadata.db buckets
```

输出示例：
```
NAME    KEYS
v1      1
```

#### 2. 快照管理

##### 列出所有快照

```bash
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list
```

输出示例：
```
ID    KEY            KIND       PARENT    CONTENT_ID    PATH                    INODES    SIZE    CREATED
1     sha256:abc...  active     -         abc123        /var/lib/containerd/...  1000      1024    2024-01-01 10:00:00
2     sha256:def...  committed  sha256:abc def456        /var/lib/containerd/...  1500      2048    2024-01-01 11:00:00
```

##### 查看特定快照详情

```bash
containerd-meta-viewer --db-path /path/to/metadata.db snapshots get <snapshot-key>
```

输出示例：
```
Snapshot Information:
====================
ID:       1
Key:      sha256:abcdef123456...
Kind:     active
Parent:   -
Created:  2024-01-01 10:00:00
Updated:  2024-01-01 10:00:00
Inodes:   1000
Size:     1024 bytes
ContentID: abc123
Path:      /var/lib/containerd/devbox/mounts/abc123

Labels:
  key1: value1
  key2: value2
```

##### 搜索快照

按内容 ID 或挂载路径搜索快照：

```bash
# 按内容 ID 搜索
containerd-meta-viewer --db-path /path/to/metadata.db snapshots search --content-id abc123

# 按路径搜索
containerd-meta-viewer --db-path /path/to/metadata.db snapshots search --path /var/lib/containerd/devbox/mounts/abc123

# 同时按多个条件搜索
containerd-meta-viewer --db-path /path/to/metadata.db snapshots search --content-id abc123 --path /var/lib/containerd/devbox/mounts/abc123
```

#### 3. Devbox 存储管理

##### 列出所有 Devbox 存储条目

```bash
containerd-meta-viewer --db-path /path/to/metadata.db devbox list
```

输出示例：
```
CONTENT_ID    LV_NAME            PATH                            STATUS  SNAPSHOT_KEY
abc123        lv-devbox-abc123   /var/lib/containerd/devbox/...  active  k8s.io/1234/sha256:abc...
def456        lv-devbox-def456   /var/lib/containerd/devbox/...  active  -
```

##### 查看特定 Devbox 存储条目

```bash
containerd-meta-viewer --db-path /path/to/metadata.db devbox get <content-id>
```

输出示例：
```
Devbox Storage Information:
==========================
ContentID:   abc123
LV Name:     lv-devbox-abc123
Path:        /var/lib/containerd/devbox/mounts/abc123
Status:      active
Snapshot Key: k8s.io/1234/sha256:abc123...
```

##### 查看 LVM 映射

显示 LVM 卷名到挂载路径的映射关系：

```bash
containerd-meta-viewer --db-path /path/to/metadata.db devbox lvm-map
```

输出示例：
```
LV_NAME            PATH
lv-devbox-abc123   /var/lib/containerd/devbox/mounts/abc123
lv-devbox-def456   /var/lib/containerd/devbox/mounts/def456
```

### 输出格式

#### 表格格式（默认）

```bash
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list
```

#### JSON 格式

```bash
# 紧凑 JSON
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list --output json

# 格式化 JSON
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list --output json --verbose
```

JSON 输出示例：
```json
[
  {
    "key": "sha256:abcdef123456...",
    "id": 1,
    "kind": "active",
    "parent": "",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z",
    "labels": {
      "key1": "value1"
    },
    "inodes": 1000,
    "size": 1024,
    "content_id": "abc123",
    "path": "/var/lib/containerd/devbox/mounts/abc123"
  }
]
```

### 常见使用场景

#### 1. 调试挂载问题

当容器挂载出现问题时，可以查看快照和对应的存储信息：

```bash
# 查看所有快照
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list

# 查看特定快照的详情
containerd-meta-viewer --db-path /path/to/metadata.db snapshots get <problematic-snapshot-key>

# 查看对应的 devbox 存储信息
containerd-meta-viewer --db-path /path/to/metadata.db devbox get <content-id>
```

#### 2. 检查 LVM 状态

查看 LVM 卷的映射关系和状态：

```bash
# 查看 LVM 映射
containerd-meta-viewer --db-path /path/to/metadata.db devbox lvm-map

# 查看所有 devbox 存储条目状态
containerd-meta-viewer --db-path /path/to/metadata.db devbox list
```

#### 3. 数据分析

使用 JSON 输出进行数据分析：

```bash
# 导出所有快照数据为 JSON
containerd-meta-viewer --db-path /path/to/metadata.db snapshots list --output json > snapshots.json

# 导出所有 devbox 存储数据为 JSON
containerd-meta-viewer --db-path /path/to/metadata.db devbox list --output json > devbox-storage.json
```

## 故障排除

### 常见错误

1. **"failed to open bolt database"**
   - 检查数据库文件是否存在（默认路径：`/var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db`）
   - 确保有读取该文件的权限
   - 如果默认路径不存在，使用 `--db-path` 指定正确的路径
   - **数据库被锁定处理**：如果数据库被 containerd 进程锁定，工具会自动复制数据库到临时文件进行读取，无需手动操作

2. **"v1 bucket not found"**
   - 数据库可能为空或损坏
   - 确认这是正确的 devbox metadata.db 文件

3. **权限被拒绝**
   - 确保当前用户有读取数据库文件的权限
   - 可能需要使用 `sudo` 运行命令

4. **数据库被锁定**
   - **自动处理**：工具会自动检测数据库锁定状态，如果被 containerd 进程锁定，会自动复制数据库到临时位置进行读取
   - 读取完成后会自动清理临时文件
   - 这是自动化过程，用户无需担心
   - 如果遇到临时文件相关的错误，可以手动清理 `/tmp/containerd-meta-viewer-*.db` 文件

### 调试技巧

1. **使用 JSON 输出获取详细信息**
   ```bash
   containerd-meta-viewer --db-path /path/to/metadata.db buckets --output json --verbose
   ```

2. **检查数据库结构**
   ```bash
   containerd-meta-viewer --db-path /path/to/metadata.db buckets
   ```

3. **验证数据库文件**
   - 确认文件大小合理（不为 0）
   - 确认文件权限可读

## 数据库结构说明

DevBox snapshotter 使用以下 BoltDB 结构：

```
v1/
├── snapshots/           # 标准containerd快照
│   └── <snapshot-key>   # 每个快照的bucket
│       ├── id           # 快照ID
│       ├── kind         # 快照类型(active/view/committed)
│       ├── parent       # 父快照
│       ├── inodes       # inode数量
│       ├── size         # 大小
│       ├── labels       # 标签
│       ├── content_id   # devbox特定: 内容ID
│       └── path         # devbox特定: 挂载路径
├── parents/            # 父子关系映射
└── devbox_storage_path/ # devbox特定bucket
    └── <content-id>    # 按contentID组织
        ├── lv_name     # LVM卷名
        ├── path        # 挂载路径
        ├── status      # 状态(active/removed)
        └── snapshot_key # 关联的快照key（可选）
```

## 贡献

欢迎提交 Issue 和 Pull Request。请参考 [DEVELOPMENT.md](DEVELOPMENT.md) 了解开发规范。

## 许可证

本项目使用 Apache License 2.0 许可证。