# Containerd Meta Viewer OSS 部署指南

## 概述

本文档描述了如何使用 Makefile 将 Containerd Meta Viewer 二进制文件自动推送到阿里云 OSS，以及如何在其他机器上下载和使用。

## 工作流程

### 开发机器
```bash
# 1. 修改代码后
make build-and-push

# 或者分步执行
make build
make oss-push
```

### 目标机器
```bash
# 1. 下载最新版本
wget https://your-bucket.oss-cn-hangzhou.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest

# 2. 赋予执行权限
chmod +x containerd-meta-viewer

# 3. 使用工具
./containerd-meta-viewer buckets
```

## 环境配置

### 方法一：环境变量配置（推荐）

```bash
# 设置 OSS 配置环境变量
export OSS_BUCKET="your-bucket-name"
export OSS_ENDPOINT="oss-cn-hangzhou.aliyuncs.com"
export OSS_ACCESS_KEY_ID="your-access-key-id"
export OSS_ACCESS_KEY_SECRET="your-access-key-secret"
export OSS_REGION="oss-cn-hangzhou"  # 可选，默认为 oss-cn-hangzhou
export OSS_PREFIX="containerd-meta-viewer"  # 可选，默认为 containerd-meta-viewer

# 添加到 ~/.bashrc 或 ~/.zshrc 以永久生效
echo 'export OSS_BUCKET="your-bucket-name"' >> ~/.bashrc
echo 'export OSS_ENDPOINT="oss-cn-hangzhou.aliyuncs.com"' >> ~/.bashrc
echo 'export OSS_ACCESS_KEY_ID="your-access-key-id"' >> ~/.bashrc
echo 'export OSS_ACCESS_KEY_SECRET="your-access-key-secret"' >> ~/.bashrc
```

### 方法二：配置文件

```bash
# 初始化配置文件
make oss-init

# 编辑生成的 .ossutilconfig 文件
vim .ossutilconfig
```

`.ossutilconfig` 文件示例：
```ini
[Credentials]
language=CH
endpoint=oss-cn-hangzhou.aliyuncs.com
accessKeyID=your-access-key-id
accessKeySecret=your-access-key-secret
```

## Makefile 命令详解

### 构建和推送

```bash
# 构建并推送到 OSS（推荐）
make build-and-push

# 单独构建
make build

# 单独推送（需要先构建）
make oss-push
```

### 下载和管理

```bash
# 下载最新版本
make oss-download

# 下载指定版本
VERSION=v1.2.3 make oss-download

# 列出所有可用版本
make oss-list
```

### 配置管理

```bash
# 初始化 OSS 配置
make oss-init

# 检查配置是否正确
make check-oss-config  # 内部调用，一般不直接使用
```

## OSS 存储结构

文件在 OSS 中的存储结构：

```
oss://your-bucket/containerd-meta-viewer/
├── containerd-meta-viewer-v1.2.3          # 版本化文件
├── containerd-meta-viewer-v1.2.4
├── containerd-meta-viewer-v1.2.5
├── containerd-meta-viewer-latest           # 最新版本的符号链接
└── containerd-meta-viewer-unknown          # 开发版本
```

## 版本管理

### 自动版本号

- 如果是 Git 仓库：使用 `git describe --tags --always --dirty`
- 如果不是 Git 仓库：使用 `unknown`

### 手动版本号

```bash
# 覆盖版本号
VERSION=v2.0.0 make build-and-push

# 或设置环境变量
export VERSION=v2.0.0
make build-and-push
```

### 版本下载

```bash
# 下载特定版本
wget https://your-bucket.oss-cn-hangzhou.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-v1.2.3

# 下载最新版本
wget https://your-bucket.oss-cn-hangzhou.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest
```

## 安装 ossutil

如果系统中没有安装 ossutil，请按以下步骤安装：

### Linux/macOS

```bash
# 下载 ossutil
wget https://gosspublic.alicdn.com/ossutil/1.7.16/ossutil64

# 赋予执行权限
chmod 755 ossutil64

# 移动到系统路径
sudo mv ossutil64 /usr/local/bin/ossutil

# 验证安装
ossutil --version
```

### 其他系统

参考阿里云官方文档：https://help.aliyun.com/document_detail/120072.html

## 安全最佳实践

### 1. 凭据管理

```bash
# 不要将凭据提交到版本控制
echo ".ossutilconfig" >> .gitignore

# 使用环境变量而不是配置文件（更安全）
export OSS_ACCESS_KEY_ID="your-key"
export OSS_ACCESS_KEY_SECRET="your-secret"
```

### 2. 权限控制

为 OSS 访问创建最小权限的 RAM 用户：

```json
{
  "Version": "1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "oss:PutObject",
        "oss:GetObject",
        "oss:DeleteObject",
        "oss:ListObjects"
      ],
      "Resource": [
        "acs:oss:*:*:your-bucket/containerd-meta-viewer/*"
      ]
    }
  ]
}
```

### 3. Bucket 安全配置

- 启用 Bucket 访问日志
- 配置适当的访问权限（私有读取，通过签名 URL 或 CDN 访问）
- 定期轮换访问密钥

## 故障排除

### 常见错误

1. **OSS 配置缺失**
   ```
   ❌ OSS configuration missing!
   Please set environment variables or run 'make oss-init'
   ```
   **解决方案**：设置环境变量或运行 `make oss-init`

2. **权限被拒绝**
   ```
   Error: oss: service returned error: StatusCode=403
   ```
   **解决方案**：检查 AccessKey ID 和 Secret，确保有足够的权限

3. **Bucket 不存在**
   ```
   Error: oss: service returned error: StatusCode=404
   ```
   **解决方案**：确认 Bucket 名称正确且已创建

4. **网络连接问题**
   ```
   Error: dial tcp: lookup oss-cn-hangzhou.aliyuncs.com: no such host
   ```
   **解决方案**：检查网络连接和 DNS 设置

### 调试技巧

```bash
# 显示详细信息
make oss-push VERBOSE=1

# 检查配置
make check-oss-config

# 列出 OSS 上的文件
make oss-list

# 手动测试 ossutil
ossutil ls oss://your-bucket --config-file=.ossutilconfig
```

## 自动化脚本示例

### 发布脚本

```bash
#!/bin/bash
# deploy.sh

set -e

echo "🚀 Starting deployment..."

# 检查环境
if [ -z "$OSS_BUCKET" ]; then
    echo "❌ OSS_BUCKET not set"
    exit 1
fi

# 运行测试
echo "🧪 Running tests..."
make test

# 构建和推送
echo "📦 Building and pushing..."
make build-and-push

echo "✅ Deployment completed successfully!"

# 显示下载信息
echo "📥 Download URL:"
echo "wget https://$OSS_BUCKET.$OSS_REGION.aliyuncs.com/$OSS_PREFIX/containerd-meta-viewer-latest"
```

### 安装脚本（目标机器）

```bash
#!/bin/bash
# install.sh

set -e

BUCKET=${1:-"your-bucket"}
REGION=${2:-"oss-cn-hangzhou"}
PREFIX=${3:-"containerd-meta-viewer"}

echo "📥 Installing Containerd Meta Viewer..."

# 下载
wget "https://$BUCKET.$REGION.aliyuncs.com/$PREFIX/containerd-meta-viewer-latest" -O containerd-meta-viewer

# 赋予权限
chmod +x containerd-meta-viewer

# 验证
./containerd-meta-viewer --version

echo "✅ Installation completed!"
echo "🔧 Usage: ./containerd-meta-viewer buckets"
```

## 复用这套流程

要将这套 OSS 部署流程应用到其他项目，需要：

1. **复制 Makefile 中的 OSS 相关目标**
2. **修改变量定义**
   ```makefile
   BINARY_NAME=your-project-name
   OSS_PREFIX=your-project-name
   ```
3. **复制 oss-init 配置逻辑**
4. **参考本文档创建项目特定的部署指南**

### 通用模板

```makefile
# 在其他项目的 Makefile 中添加这些目标
OSS_BUCKET?=$(shell echo $$OSS_BUCKET)
OSS_ENDPOINT?=$(shell echo $$OSS_ENDPOINT)
OSS_ACCESS_KEY_ID?=$(shell echo $$OSS_ACCESS_KEY_ID)
OSS_ACCESS_KEY_SECRET?=$(shell echo $$OSS_ACCESS_KEY_SECRET)
OSS_REGION?=$(shell echo $$OSS_REGION || echo "oss-cn-hangzhou")
OSS_PREFIX?=$(shell echo $$OSS_PREFIX || echo "your-project-name")

.PHONY: build-and-push
build-and-push: build
	$(MAKE) oss-push

.PHONY: oss-push
oss-push: check-oss-config
	ossutil cp $(BINARY_NAME) oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION) --config-file=.ossutilconfig
	ossutil cp oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION) oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-latest --config-file=.ossutilconfig
```

这样就可以在多个项目间复用相同的 OSS 部署流程了。