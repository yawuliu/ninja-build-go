# 默认目标平台为当前平台
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# 项目名称
PROJECT_NAME := ninja-build

# 输出目录
DIST_DIR := dist
DIST_DIR_WIN := $(DIST_DIR)/windows
DIST_DIR_LINUX := $(DIST_DIR)/linux

# 默认目标
all: build

# 创建输出目录
$(DIST_DIR_WIN):
ifeq ($(GOOS),windows)
	if not exist "$(DIST_DIR_WIN)" mkdir "$(DIST_DIR_WIN)"
else ifeq ($(GOOS),linux)
	mkdir -p $(DIST_DIR_WIN)
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

$(DIST_DIR_LINUX):
ifeq ($(GOOS),windows)
	if not exist "$(DIST_DIR_LINUX)" mkdir "$(DIST_DIR_LINUX)"
else ifeq ($(GOOS),linux)
	mkdir -p $(DIST_DIR_LINUX)
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

# 构建项目
build:
ifeq ($(GOOS),windows)
	@echo "Building for Windows..."
	if not exist $(DIST_DIR_WIN) mkdir $(DIST_DIR_WIN)
	go build -o $(DIST_DIR_WIN)/$(PROJECT_NAME).exe
else ifeq ($(GOOS),linux)
	@echo "Building for Linux..."
	mkdir -p $(DIST_DIR_LINUX)
	go build -o $(DIST_DIR_LINUX)/$(PROJECT_NAME)
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

# 测试项目
test:
	go test ./...


run: build
ifeq ($(GOOS),windows)
	$(DIST_DIR_WIN)/$(PROJECT_NAME).exe -C test/test_01
else ifeq ($(GOOS),linux)
	$(DIST_DIR_LINUX)/$(PROJECT_NAME) -C test/test_01
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

# 清理项目
clean:
	@echo "Cleaning up..."
	go clean
ifeq ($(GOOS),windows)
	if exist $(DIST_DIR_WIN)/$(PROJECT_NAME).exe del $(DIST_DIR_WIN)/$(PROJECT_NAME).exe
else ifeq ($(GOOS),linux)
	rm -rf $(DIST_DIR_LINUX)/$(PROJECT_NAME)
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

# 编译为 Windows 平台
build-windows: $(DIST_DIR_WIN)
	@echo "Building for Windows..."
ifeq ($(GOOS),windows)
	SET GOOS=windows&&SET GOARCH=$(GOARCH)&&go build -o $(DIST_DIR_WIN)/$(PROJECT_NAME).exe
else ifeq ($(GOOS),linux)
	GOOS=windows GOARCH=$(GOARCH) go build -o $(DIST_DIR_WIN)/$(PROJECT_NAME).exe
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

# 编译为 Linux 平台
build-linux: $(DIST_DIR_LINUX)
	@echo "Building for Linux... "
ifeq ($(GOOS),windows)
	SET GOOS=linux&&SET GOARCH=$(GOARCH)&&go build -o $(DIST_DIR_LINUX)/$(PROJECT_NAME)
else ifeq ($(GOOS),linux)
	GOOS=linux GOARCH=$(GOARCH) go build -o $(DIST_DIR_LINUX)/$(PROJECT_NAME)
else
	@echo "Unsupported platform $(GOOS)"
	@exit 1
endif

.PHONY: all build test clean build-windows build-linux