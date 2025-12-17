#!/bin/bash
# 权限检查脚本 / Permission Check Script
# 用于验证 TenyoJubaku 部署的文件权限是否正确

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 设置工作目录
WORK_DIR="${1:-$HOME/tenyojubaku}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  TenyoJubaku 权限检查${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "检查目录: ${WORK_DIR}"
echo ""

# 检查目录是否存在
if [ ! -d "$WORK_DIR" ]; then
    echo -e "${RED}✗ 错误: 目录不存在: $WORK_DIR${NC}"
    exit 1
fi

cd "$WORK_DIR"

# 检查函数
check_permission() {
    local file=$1
    local expected_perm=$2
    local description=$3

    if [ ! -e "$file" ]; then
        echo -e "${YELLOW}⚠ 跳过: $file (文件不存在)${NC}"
        return
    fi

    # 获取实际权限（数字形式）
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        actual_perm=$(stat -f "%Lp" "$file")
    else
        # Linux
        actual_perm=$(stat -c "%a" "$file")
    fi

    if [ "$actual_perm" == "$expected_perm" ]; then
        echo -e "${GREEN}✓ $description${NC}"
        echo "  文件: $file"
        echo "  权限: $actual_perm (正确)"
    else
        echo -e "${RED}✗ $description${NC}"
        echo "  文件: $file"
        echo "  期望: $expected_perm"
        echo "  实际: $actual_perm"
        echo -e "  ${YELLOW}修复命令: chmod $expected_perm $file${NC}"
    fi
    echo ""
}

# 开始检查
echo -e "${BLUE}=== 检查二进制文件 ===${NC}"
check_permission "tenyojubaku" "755" "主服务二进制文件"
check_permission "tenyojubaku-cli" "755" "CLI 工具二进制文件"

echo -e "${BLUE}=== 检查关键目录 ===${NC}"
check_permission "configs" "700" "配置目录（最重要！）"
check_permission "data" "700" "数据目录"
check_permission "logs" "750" "日志目录"

echo -e "${BLUE}=== 检查配置文件 ===${NC}"
check_permission "configs/config.yaml" "600" "配置文件（包含 API 凭证）"

echo -e "${BLUE}=== 检查数据库文件 ===${NC}"
if [ -f "data/tenyojubaku.db" ]; then
    check_permission "data/tenyojubaku.db" "600" "数据库文件"
else
    echo -e "${YELLOW}⚠ 跳过: data/tenyojubaku.db (数据库尚未创建)${NC}"
    echo ""
fi

# 检查所有者
echo -e "${BLUE}=== 检查文件所有者 ===${NC}"
current_user=$(whoami)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    owner=$(stat -f "%Su" "configs/config.yaml")
else
    # Linux
    owner=$(stat -c "%U" "configs/config.yaml")
fi

if [ "$owner" == "$current_user" ]; then
    echo -e "${GREEN}✓ 文件所有者正确: $owner${NC}"
else
    echo -e "${RED}✗ 文件所有者不匹配${NC}"
    echo "  期望: $current_user"
    echo "  实际: $owner"
    echo -e "  ${YELLOW}修复命令: chown -R $current_user:$current_user $WORK_DIR${NC}"
fi
echo ""

# 总结
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  检查完成${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "如果发现权限问题，可以运行以下命令修复："
echo ""
echo "cd $WORK_DIR"
echo "chmod 755 tenyojubaku tenyojubaku-cli"
echo "chmod 700 configs data"
echo "chmod 750 logs"
echo "chmod 600 configs/config.yaml"
echo "chmod 600 data/*.db 2>/dev/null || true"
echo "chown -R $current_user:$current_user ."
echo ""
