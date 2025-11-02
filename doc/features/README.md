# 功能实现文档

本目录包含 containerd-meta-viewer 各个功能的详细实现文档。

## 文档说明

每个功能都有一个独立的文档文件，详细说明：
- 功能的实现原理
- 代码流程
- 使用方法
- 性能考虑

## 文档列表

- [数据库读取功能](database_reader.md) - 核心数据库读取器实现
- [Buckets 命令](buckets_command.md) - buckets 命令实现
- [Snapshots 命令](snapshots_command.md) - snapshots 命令实现（待创建）
- [Devbox 命令](devbox_command.md) - devbox 命令实现（待创建）

## 如何添加新功能文档

1. 创建新的 `.md` 文件，以功能名称命名
2. 包含以下章节：
   - 概述
   - 实现位置
   - 实现原理
   - 使用示例
   - 性能考虑
3. 在本 README 中添加链接
4. 参考现有文档的格式和结构

## 文档维护

根据开发文档要求，每次修改功能实现后，必须同步更新对应的功能文档。

