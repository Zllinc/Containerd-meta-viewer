# Containerd Meta Viewer Makefile

# 变量定义
BINARY_NAME=containerd-meta-viewer
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# OSS 配置 (可以通过环境变量覆盖)
OSS_BUCKET?=$(shell echo $$OSS_BUCKET)
OSS_ENDPOINT?=$(shell echo $$OSS_ENDPOINT)
OSS_ACCESS_KEY_ID?=$(shell echo $$OSS_ACCESS_KEY_ID)
OSS_ACCESS_KEY_SECRET?=$(shell echo $$OSS_ACCESS_KEY_SECRET)
OSS_REGION?=$(shell echo $$OSS_REGION || echo "oss-cn-hangzhou")
OSS_PREFIX?=$(shell echo $$OSS_PREFIX || echo "containerd-meta-viewer")

# 默认目标
.PHONY: all
all: build

# 构建二进制文件
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) (version: $(VERSION))..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# 构建并推送到 OSS
.PHONY: build-and-push
build-and-push: build
	@echo "Building and pushing to OSS..."
	$(MAKE) oss-push

# 推送到阿里云 OSS
.PHONY: oss-push
oss-push: check-oss-config
	@echo "Uploading $(BINARY_NAME) to OSS..."
	@if [ ! -f "$(BINARY_NAME)" ]; then echo "Binary $(BINARY_NAME) not found, building first..."; $(MAKE) build; fi
	ossutil cp $(BINARY_NAME) oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION) --config-file=.ossutilconfig
	@echo "Creating latest version symlink..."
	ossutil cp oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION) oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-latest --config-file=.ossutilconfig
	@echo "✅ Upload completed!"
	@echo "Download URL: https://$(OSS_BUCKET).$(OSS_REGION).aliyuncs.com/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION)"
	@echo "Latest URL:   https://$(OSS_BUCKET).$(OSS_REGION).aliyuncs.com/$(OSS_PREFIX)/$(BINARY_NAME)-latest"

# 从 OSS 下载
.PHONY: oss-download
oss-download: check-oss-config
	@echo "Downloading $(BINARY_NAME) from OSS..."
	@if [ "$(VERSION)" = "latest" ]; then \
		ossutil cp oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-latest $(BINARY_NAME) --config-file=.ossutilconfig; \
	else \
		ossutil cp oss://$(OSS_BUCKET)/$(OSS_PREFIX)/$(BINARY_NAME)-$(VERSION) $(BINARY_NAME) --config-file=.ossutilconfig; \
	fi
	@echo "✅ Download completed!"

# 列出 OSS 上的版本
.PHONY: oss-list
oss-list: check-oss-config
	@echo "Listing available versions in OSS..."
	ossutil ls oss://$(OSS_BUCKET)/$(OSS_PREFIX)/ --config-file=.ossutilconfig

# 初始化 OSS 配置
.PHONY: oss-init
oss-init:
	@echo "Setting up OSS configuration..."
	@echo "Please set the following environment variables or update .ossutilconfig:"
	@echo "  OSS_BUCKET         - OSS bucket name"
	@echo "  OSS_ENDPOINT       - OSS endpoint"
	@echo "  OSS_ACCESS_KEY_ID  - OSS access key ID"
	@echo "  OSS_ACCESS_KEY_SECRET - OSS access key secret"
	@echo "  OSS_REGION         - OSS region (default: oss-cn-hangzhou)"
	@echo "  OSS_PREFIX         - OSS prefix path (default: devbox-meta-viewer)"
	@echo ""
	@echo "Creating .ossutilconfig template..."
	@echo "[Credentials]" > .ossutilconfig
	@echo "language=CH" >> .ossutilconfig
	@echo "endpoint=$(OSS_ENDPOINT)" >> .ossutilconfig
	@echo "accessKeyID=$(OSS_ACCESS_KEY_ID)" >> .ossutilconfig
	@echo "accessKeySecret=$(OSS_ACCESS_KEY_SECRET)" >> .ossutilconfig
	@echo ""
	@echo "✅ Please edit .ossutilconfig with your actual OSS credentials"

# 检查 OSS 配置
.PHONY: check-oss-config
check-oss-config:
	@if [ -z "$(OSS_BUCKET)" ] || [ -z "$(OSS_ENDPOINT)" ] || [ -z "$(OSS_ACCESS_KEY_ID)" ] || [ -z "$(OSS_ACCESS_KEY_SECRET)" ]; then \
		echo "❌ OSS configuration missing!"; \
		echo "Please set environment variables or run 'make oss-init'"; \
		echo "Required: OSS_BUCKET, OSS_ENDPOINT, OSS_ACCESS_KEY_ID, OSS_ACCESS_KEY_SECRET"; \
		exit 1; \
	fi
	@echo "✅ OSS configuration OK"

# 清理构建产物
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# 显示帮助信息
.PHONY: help
help:
	@echo "Containerd Meta Viewer Makefile"
	@echo ""
	@echo "Build & Deploy:"
	@echo "  build           - Build the binary"
	@echo "  build-and-push  - Build and push to OSS"
	@echo "  oss-push        - Push binary to OSS"
	@echo "  oss-download    - Download binary from OSS"
	@echo "  oss-list        - List versions in OSS"
	@echo "  oss-init        - Initialize OSS configuration"
	@echo ""
	@echo "Utility:"
	@echo "  clean           - Clean build artifacts"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "OSS Usage Examples:"
	@echo "  # Set environment variables"
	@echo "  export OSS_BUCKET='my-bucket'"
	@echo "  export OSS_ENDPOINT='oss-cn-hangzhou.aliyuncs.com'"
	@echo "  export OSS_ACCESS_KEY_ID='your-key-id'"
	@echo "  export OSS_ACCESS_KEY_SECRET='your-key-secret'"
	@echo ""
	@echo "  # Build and push"
	@echo "  make build-and-push"
	@echo ""
	@echo "  # Download from another machine"
	@echo "  wget https://your-bucket.oss-cn-hangzhou.aliyuncs.com/containerd-meta-viewer/containerd-meta-viewer-latest"
	@echo "  chmod +x containerd-meta-viewer"