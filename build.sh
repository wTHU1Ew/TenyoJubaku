#!/bin/bash
# TenyoJubaku Build Script
# 支持本地和跨平台编译 / Support local and cross-platform compilation

set -e  # Exit on error

# 默认目标平台 / Default target platform
TARGET_OS=${GOOS:-$(go env GOOS)}
TARGET_ARCH=${GOARCH:-$(go env GOARCH)}

# 输出目录 / Output directory
OUTPUT_DIR="./bin"
mkdir -p "$OUTPUT_DIR"

# 颜色输出 / Color output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  TenyoJubaku Build Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 显示编译信息 / Show build info
echo -e "${GREEN}Target OS:${NC}       $TARGET_OS"
echo -e "${GREEN}Target Arch:${NC}     $TARGET_ARCH"
echo -e "${GREEN}Go Version:${NC}      $(go version | awk '{print $3}')"
echo -e "${GREEN}Output Dir:${NC}      $OUTPUT_DIR"
echo ""

# 编译主服务 / Build main service
echo -e "${BLUE}Building main service (tenyojubaku)...${NC}"
GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build -o "$OUTPUT_DIR/tenyojubaku" ./cmd
echo -e "${GREEN}✓ Built:${NC} $OUTPUT_DIR/tenyojubaku"

# 编译 CLI 工具 / Build CLI tool
echo -e "${BLUE}Building CLI tool (tenyojubaku-cli)...${NC}"
GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build -o "$OUTPUT_DIR/tenyojubaku-cli" ./cmd/cli
echo -e "${GREEN}✓ Built:${NC} $OUTPUT_DIR/tenyojubaku-cli"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Binaries are in: $OUTPUT_DIR/"
echo ""

# 显示文件大小 / Show file sizes
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    ls -lh "$OUTPUT_DIR" | grep tenyojubaku
else
    # Linux
    ls -lh "$OUTPUT_DIR" | grep tenyojubaku
fi

echo ""
echo "Usage examples:"
echo "  Local build:              ./build.sh"
echo "  Linux AMD64:              GOOS=linux GOARCH=amd64 ./build.sh"
echo "  Linux ARM64:              GOOS=linux GOARCH=arm64 ./build.sh"
echo "  Windows:                  GOOS=windows GOARCH=amd64 ./build.sh"
