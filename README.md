# Containerd Meta Viewer

一个用于查看和管理 containerd snapshotter 元数据的 CLI 工具。该工具允许用户查看存储在 bolt 数据库中的快照、存储信息和 LVM 映射，并提供清理孤儿快照、分析依赖关系等功能。

## 功能特性

### 核心功能
- 查看数据库中的所有 buckets
- 列出和搜索快照信息
- 查看 containerd 特定的存储信息
- 显示 LVM 卷名到挂载路径的映射
- 支持表格和 JSON 两种输出格式

### 快照管理
- **孤儿检测**: 检测 metadata.db 中存在但 containerd 中不存在的快照
- **未使用检测**: 检测没有被任何容器使用的快照
- **安全清理**: 多重检查确保安全删除快照
- **依赖分析**: 分析快照被多少容器直接或间接依赖

### Containerd Core 管理
- **Ghost Children 检测**: 检测 containerd core 中存在但快照已删除的残留引用
- **Ghost Children 清理**: 清理 containerd core 中的残留引用

## 安装

### 从源码构建

```bash
git clone <repository-url>
cd containerd-meta-viewer
go mod tidy
go build -o containerd-meta-viewer .
sudo mv containerd-meta-viewer /usr/local/bin/
```

## 全局参数

- `--db-path, -p`: containerd metadata.db 文件路径（默认 `/var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db`）
- `--output, -o`: 输出格式 `table`（默认）或 `json`
- `--verbose, -v`: 启用详细输出

---

## 命令参考

### 1. 查看数据库 Buckets

```bash
containerd-meta-viewer buckets
```

---

### 2. 快照管理 (snapshots)

#### 列出所有快照

```bash
containerd-meta-viewer snapshots list
```

#### 查看特定快照详情

```bash
containerd-meta-viewer snapshots get <snapshot-key>
```

#### 搜索快照

```bash
# 按内容 ID 搜索
containerd-meta-viewer snapshots search --content-id abc123

# 按路径搜索
containerd-meta-viewer snapshots search --path /var/lib/containerd/devbox/mounts/abc123
```

#### 查看快照父链

```bash
containerd-meta-viewer snapshots parents <snapshot-key>
```

#### 查看快照子链

```bash
containerd-meta-viewer snapshots children <snapshot-key>
```

---

### 3. 孤儿快照检测与清理 (orphan)

检测在 metadata.db 中存在但在 containerd 中不存在的快照。

```bash
# 检测孤儿快照
containerd-meta-viewer snapshots orphan --namespace k8s.io

# 导出到文件
containerd-meta-viewer snapshots orphan --namespace k8s.io --export /tmp/orphans.json

# 清理孤儿快照
containerd-meta-viewer snapshots cleanup --namespace k8s.io --file /tmp/orphans.json
```

---

### 4. 未使用快照检测 (unused)

检测没有被其他快照引用为 parent 且不是 Active 状态的快照。

```bash
# 列出未使用的快照
containerd-meta-viewer snapshots unused --namespace k8s.io

# 导出到文件
containerd-meta-viewer snapshots unused --namespace k8s.io --export /tmp/unused.json
```

---

### 5. 安全未使用快照检测 (safe-unused)

进行多重安全检查，确保快照真正可以安全删除。

**检查项目：**
1. **Kind 检查**: 不是 Active 状态
2. **Parent 检查**: 没有被其他快照引用为 parent
3. **Container 检查**: 没有被任何容器使用
4. **Mount 检查**: 没有被挂载

```bash
# 列出所有快照的安全检查结果
containerd-meta-viewer snapshots safe-unused --namespace k8s.io

# 只显示安全可删除的快照
containerd-meta-viewer snapshots safe-unused --namespace k8s.io --only-safe

# 导出安全快照到文件
containerd-meta-viewer snapshots safe-unused --namespace k8s.io --export /tmp/safe.json --only-safe
```

---

### 6. 安全清理 (safe-cleanup)

根据导出的文件来安全删除快照。

```bash
# 预览删除（dry-run）
containerd-meta-viewer snapshots safe-cleanup --namespace k8s.io --file /tmp/safe.json --dry-run

# 实际删除
containerd-meta-viewer snapshots safe-cleanup --namespace k8s.io --file /tmp/safe.json
```

---

### 7. 快照依赖分析 (deps)

分析每个快照被多少容器直接或间接依赖。

```bash
# 基本分析
containerd-meta-viewer snapshots deps --namespace k8s.io

# 按容器数量分组显示
containerd-meta-viewer snapshots deps --namespace k8s.io --group-by count

# 按层级深度分组显示
containerd-meta-viewer snapshots deps --namespace k8s.io --group-by depth

# 只显示被至少 5 个容器使用的快照
containerd-meta-viewer snapshots deps --namespace k8s.io --min-count 5
```

**输出示例：**
```
=== Summary ===
Total snapshots: 2000
  Unused (0 containers):       100
  Direct container usage:      50
  Indirect only (base layers): 1850
  Max containers per snapshot: 45

=== Distribution by Container Count ===
Container Count      Snapshot Count
---------------      ---------------
0                    100        ██████████
1                    50         █████
5                    200        ████████████████████
```

---

### 8. Ghost 检测与清理 (ghost)

检测 devbox metadata.db 中的 ghost 引用。

```bash
# 检测 ghost
containerd-meta-viewer snapshots ghost --namespace k8s.io

# 预览清理
containerd-meta-viewer snapshots ghost-cleanup --dry-run

# 实际清理
containerd-meta-viewer snapshots ghost-cleanup
```

---

### 9. Devbox 存储管理 (devbox)

```bash
# 列出所有条目
containerd-meta-viewer devbox list

# 查看特定条目
containerd-meta-viewer devbox get <content-id>

# 查看 LVM 映射
containerd-meta-viewer devbox lvm-map

# 清除 snapshot key
containerd-meta-viewer devbox remove_key <content-id>
```

---

### 10. Containerd Core 管理 (containerd)

用于检查和管理 containerd core 的 `meta.db` 数据库（路径：`/var/lib/containerd/io.containerd.metadata.v1.bolt/meta.db`）。

```bash
# 列出 Buckets
containerd-meta-viewer containerd buckets

# 列出命名空间
containerd-meta-viewer containerd namespaces

# 列出快照
containerd-meta-viewer containerd snapshots --namespace k8s.io

# 查看快照详情
containerd-meta-viewer containerd get --namespace k8s.io <snapshot-key>

# 查看快照 Children
containerd-meta-viewer containerd children --namespace k8s.io <snapshot-key>

# 检测 Ghost Children
containerd-meta-viewer containerd ghost --namespace k8s.io

# 清理 Ghost Children（预览）
containerd-meta-viewer containerd ghost-cleanup --namespace k8s.io --dry-run

# 清理 Ghost Children（实际）
containerd-meta-viewer containerd ghost-cleanup --namespace k8s.io

# 导出结构
containerd-meta-viewer containerd dump
```

---

## 常见使用场景

### 场景 1: 清理泄露的快照

```bash
# 1. 先清理 containerd core 中的 ghost children
containerd-meta-viewer containerd ghost --namespace k8s.io
containerd-meta-viewer containerd ghost-cleanup --namespace k8s.io --dry-run
containerd-meta-viewer containerd ghost-cleanup --namespace k8s.io

# 2. 检测并清理未使用的快照
containerd-meta-viewer snapshots unused --namespace k8s.io --export /tmp/unused.json
containerd-meta-viewer snapshots safe-cleanup --namespace k8s.io --file /tmp/unused.json --dry-run
containerd-meta-viewer snapshots safe-cleanup --namespace k8s.io --file /tmp/unused.json
```

### 场景 2: 分析快照使用情况

```bash
containerd-meta-viewer snapshots deps --namespace k8s.io
containerd-meta-viewer snapshots deps --namespace k8s.io --group-by count --min-count 10
```

### 场景 3: 调试删除失败 (cannot remove snapshot with child)

```bash
# 1. 检查 ghost children
containerd-meta-viewer containerd children --namespace k8s.io <snapshot-key>

# 2. 清理 ghost children
containerd-meta-viewer containerd ghost-cleanup --namespace k8s.io

# 3. 重试删除
ctr -n k8s.io snapshots --snapshotter devbox rm <snapshot-key>
```

### 场景 4: 追溯快照来源

```bash
containerd-meta-viewer snapshots parents sha256:abc123...
containerd-meta-viewer snapshots children sha256:abc123...
```

---

## 数据库结构

### DevBox Snapshotter (metadata.db)

```
v1/
├── snapshots/           # 快照信息
├── parents/             # 父子关系 (child → parent)
└── devbox_storage_path/ # devbox 存储
```

### Containerd Core (meta.db)

```
v1/
└── <namespace>/
    └── snapshots/
        └── <snapshotter>/
            ├── <snapshot-key>
            └── children/
                └── <parent-key> → [child-keys]
```

---

## 故障排除

1. **"failed to open bolt database"** - 检查文件是否存在和权限，数据库被锁定时工具会自动复制到临时文件
2. **"cannot remove snapshot with child"** - 使用 `containerd ghost-cleanup` 清理残留引用
3. **权限被拒绝** - 使用 `sudo` 运行

---

## 贡献

欢迎提交 Issue 和 Pull Request。请参考 [doc/DEVELOPMENT.md](doc/DEVELOPMENT.md) 了解开发规范。

## 许可证

Apache License 2.0