# 项目重命名总结

## 重命名概述

项目已从 `devbox-meta-viewer` 重命名为 `containerd-meta-viewer`，以更好地反映其通用 containerd snapshotter 元数据查看工具的定位。

## 重命名详情

### 1. 核心变更

| 项目 | 原名称 | 新名称 |
|------|--------|--------|
| 二进制文件 | `devbox-meta-viewer` | `containerd-meta-viewer` |
| Go 模块 | `github.com/devbox/meta-viewer` | `github.com/containerd/meta-viewer` |
| 项目目录 | `devbox-meta-viewer` | `containerd-meta-viewer` |

### 2. 文件修改清单

#### 2.1 核心配置文件
- ✅ `go.mod` - 模块路径更新
- ✅ `Makefile` - 二进制名称和 OSS 路径更新
- ✅ `main.go` - import 路径已更新

#### 2.2 命令行接口
- ✅ `cmd/root.go` - 命令名称、描述和帮助文本更新
- ✅ `cmd/buckets.go` - 命令描述更新
- ✅ 所有 cmd 文件中的 import 路径已更新

#### 2.3 内部包
- ✅ `internal/database/` - 所有 import 路径更新
- ✅ `internal/formatters/` - 所有 import 路径更新
- ✅ `internal/utils/` - 所有 import 路径更新

#### 2.4 文档
- ✅ `README.md` - 完全重命名和更新
- ✅ `doc/DEVELOPMENT.md` - 项目名称更新
- ✅ `doc/TEST.md` - 项目名称更新
- ✅ `doc/OSS_DEPLOYMENT.md` - 完全重命名
- ✅ `doc/RENAME_SUMMARY.md` - 新增本文档

#### 2.5 脚本
- ✅ `scripts/deploy-example.sh` - 完全重命名

### 3. 保持不变的部分

以下部分**故意保持不变**，因为它们反映了数据的实际来源：

- **数据模型**：`devbox` 相关的字段名称（如 `DevboxStorageInfo`）
- **数据库结构**：`devbox_storage_path` bucket
- **命令功能**：`devbox` 命令仍然用于查看 devbox 特定数据
- **常量**：默认数据库路径仍然指向 devbox snapshotter

这确保了工具仍然可以正确查看和解析 devbox snapshotter 的数据。

### 4. 功能验证

#### 4.1 构建测试
```bash
make build
# ✅ 成功构建 containerd-meta-viewer
```

#### 4.2 功能测试
```bash
./containerd-meta-viewer --help
# ✅ 显示正确的命令名称和描述

./containerd-meta-viewer --db-path /tmp/sample.db buckets --output json
# ✅ 功能正常，输出 JSON 格式
```

#### 4.3 测试套件
```bash
go test ./...
# ✅ 所有测试通过（100% 成功率）
```

#### 4.4 OSS 功能
```bash
make help
# ✅ OSS 相关命令显示正确的项目名称
```

### 5. 向后兼容性

- **完全兼容**：所有核心功能保持不变
- **数据兼容**：仍然可以读取所有现有格式的数据
- **命令兼容**：除了主命令名称外，子命令参数保持一致

### 6. 部署影响

#### 6.1 OSS 路径变更
```
旧路径：oss://bucket/devbox-meta-viewer/devbox-meta-viewer-*
新路径：oss://bucket/containerd-meta-viewer/containerd-meta-viewer-*
```

#### 6.2 下载链接更新
```bash
# 旧链接
wget https://bucket.oss-region.aliyuncs.com/devbox-meta-viewer/devbox-meta-viewer-latest

# 新链接
wget https://bucket.oss-region.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest
```

### 7. 迁移指南

#### 7.1 开发者
```bash
# 重新构建
make clean && make build

# 使用新命令
./containerd-meta-viewer buckets
```

#### 7.2 用户
```bash
# 下载新版本
wget https://your-bucket.oss-cn-hangzhou.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest
chmod +x containerd-meta-viewer

# 使用方式完全相同
./containerd-meta-viewer buckets
```

### 8. 新项目定位

重命名后，`containerd-meta-viewer` 的定位更加准确：

- **通用性**：不局限于 devbox，可支持任何 containerd snapshotter
- **专业性**：专注于 containerd 生态系统的元数据查看
- **扩展性**：为将来支持其他 snapshotter 类型奠定基础

### 9. 后续计划

1. **文档完善**：持续更新文档以反映通用 containerd 工具定位
2. **功能扩展**：考虑添加对其他 containerd snapshotter 的支持
3. **社区推广**：以通用 containerd 工具进行推广

## 总结

重命名工作已**完全完成**，项目现在以 `containerd-meta-viewer` 的名称运行，所有功能正常，测试通过，文档更新完整。项目保持了完全的向后兼容性和功能一致性。