.PHONY: build test clean install docker-build docker-run

BINARY_NAME=code-quality
BUILD_DIR=dist
VERSION ?= 1.0.0
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

build-linux:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_linux_amd64 .

build-darwin:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64 .

build-windows:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_windows_amd64.exe .

build-all: build-linux build-darwin build-windows

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

docker-build:
	docker build -t code-quality:$(VERSION) .

docker-run:
	docker run --rm -v $(PWD):/workspace code-quality:$(VERSION) /workspace

help:
	@echo "可用的 Make 目标:"
	@echo "  build          - 构建当前平台的二进制文件"
	@echo "  build-linux    - 构建 Linux 版本"
	@echo "  build-darwin   - 构建 macOS 版本"
	@echo "  build-windows  - 构建 Windows 版本"
	@echo "  build-all      - 构建所有平台版本"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  lint           - 运行代码检查"
	@echo "  clean          - 清理构建产物"
	@echo "  install        - 安装到 /usr/local/bin"
	@echo "  docker-build   - 构建 Docker 镜像"
	@echo "  docker-run     - 在 Docker 中运行"
