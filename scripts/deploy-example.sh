#!/bin/bash
# Containerd Meta Viewer 部署示例脚本

set -e

echo "🚀 Containerd Meta Viewer OSS 部署示例"
echo "=================================="

# 检查是否在正确的目录
if [ ! -f "main.go" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

# 步骤 1: 设置环境变量（示例）
echo ""
echo "📋 步骤 1: 设置 OSS 环境变量"
echo "请设置以下环境变量："
echo "export OSS_BUCKET='your-bucket-name'"
echo "export OSS_ENDPOINT='oss-cn-hangzhou.aliyuncs.com'"
echo "export OSS_ACCESS_KEY_ID='your-access-key-id'"
echo "export OSS_ACCESS_KEY_SECRET='your-access-key-secret'"
echo ""

# 检查环境变量
if [ -z "$OSS_BUCKET" ] || [ -z "$OSS_ENDPOINT" ] || [ -z "$OSS_ACCESS_KEY_ID" ] || [ -z "$OSS_ACCESS_KEY_SECRET" ]; then
    echo "⚠️  OSS 环境变量未设置，将跳过实际推送"
    DRY_RUN=true
else
    echo "✅ OSS 环境变量已设置"
    DRY_RUN=false
fi

# 步骤 2: 运行测试
echo ""
echo "🧪 步骤 2: 运行测试"
if command -v go &> /dev/null; then
    echo "运行单元测试..."
    go test ./... -v
    echo "✅ 测试通过"
else
    echo "⚠️  Go 未安装，跳过测试"
fi

# 步骤 3: 构建二进制文件
echo ""
echo "🔨 步骤 3: 构建二进制文件"
make build
echo "✅ 构建完成"

# 显示二进制文件信息
if [ -f "containerd-meta-viewer" ]; then
    echo "二进制文件信息:"
    ls -lh containerd-meta-viewer
    echo "版本信息:"
    ./containerd-meta-viewer --version 2>/dev/null || echo "版本信息不可用"
fi

# 步骤 4: 推送到 OSS（如果配置了）
echo ""
echo "📤 步骤 4: 推送到 OSS"
if [ "$DRY_RUN" = true ]; then
    echo "⚠️  跳过 OSS 推送（未配置环境变量）"
    echo "要启用推送，请设置 OSS 环境变量后运行："
    echo "make build-and-push"
else
    echo "推送到 OSS..."
    make oss-push
    echo "✅ 推送完成"
fi

# 步骤 5: 显示下载指令
echo ""
echo "📥 步骤 5: 在其他机器上的下载指令"
echo "在目标机器上执行以下命令下载最新版本："
echo ""
echo "# 下载二进制文件"
echo "wget https://$OSS_BUCKET.$OSS_REGION.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest"
echo ""
echo "# 赋予执行权限"
echo "chmod +x containerd-meta-viewer"
echo ""
echo "# 验证安装"
echo "./containerd-meta-viewer --version"
echo ""
echo "# 使用工具"
echo "./containerd-meta-viewer buckets"

# 步骤 6: 清理
echo ""
echo "🧹 步骤 6: 清理"
echo "是否清理构建文件？(y/N)"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    make clean
    echo "✅ 清理完成"
fi

echo ""
echo "🎉 部署流程演示完成！"
echo ""
echo "实际使用时："
echo "1. 设置 OSS 环境变量"
echo "2. 运行: make build-and-push"
echo "3. 在目标机器上: wget <下载URL> && chmod +x containerd-meta-viewer"