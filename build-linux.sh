#!/bin/bash

echo "正在编译Linux版本的DNS代理程序..."
echo "使用优化参数进行编译..."

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查Go是否可用
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 找不到Go编译器，请确保Go已安装并添加到PATH${NC}"
    exit 1
fi

# 设置编译参数
GOOS="linux"
GOARCH="amd64"
OUTPUT_FILE="dns-proxy-linux"

# 优化编译参数
BUILD_FLAGS="-ldflags=-s -w -X main.version=1.0.0 -trimpath"

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}    DNS代理程序 - Linux版本编译器${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}编译优化参数:${NC}"
echo -e "  ${GREEN}✓${NC} 去除符号表和调试信息 ${GREEN}(-s -w)${NC}     减小文件体积"
echo -e "  ${GREEN}✓${NC} 去除文件路径信息 ${GREEN}(-trimpath)${NC}    保护构建路径隐私"
echo -e "  ${GREEN}✓${NC} 设置版本信息 ${GREEN}(-X main.version)${NC}  可追踪版本"
echo ""
echo -e "${YELLOW}目标系统:${NC} Linux (${GOARCH})"
echo -e "${YELLOW}输出文件:${NC} ${OUTPUT_FILE}"
echo -e "${YELLOW}Go版本:${NC} $(go version)"
echo ""

# 执行编译
echo -e "${YELLOW}开始编译...${NC}"
if GOOS=${GOOS} GOARCH=${GOARCH} go build ${BUILD_FLAGS} -o ${OUTPUT_FILE} main.go; then
    echo ""
    echo -e "${GREEN}✓ 编译成功!${NC}"
    echo ""
    
    # 显示文件信息
    echo -e "${YELLOW}文件信息:${NC}"
    ls -lh ${OUTPUT_FILE} | awk -v YELLOW="${YELLOW}" -v GREEN="${GREEN}" -v NC="${NC}" '{
        printf "  大小: %s%s%s\n", YELLOW, $5, NC
        printf "  权限: %s%s%s\n", YELLOW, $1, NC
        printf "  修改时间: %s%s %s %s%s\n", YELLOW, $6, $7, $8, NC
    }'
    
    # 添加可执行权限
    chmod +x ${OUTPUT_FILE}
    echo ""
    echo -e "${GREEN}✓ 已添加可执行权限${NC}"
    
    # 显示优化效果
    ORIGINAL_SIZE=$(ls -l ${OUTPUT_FILE} | awk '{print $5}')
    echo ""
    echo -e "${GREEN}✓ 文件已优化，体积更小，运行更快!${NC}"
else
    echo ""
    echo -e "${RED}✗ 编译失败!${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}使用方法:${NC}"
echo -e "  ${GREEN}sudo ./${OUTPUT_FILE} -port 53 -upstream 8.8.8.8:53${NC}"
echo ""
echo -e "${YELLOW}参数说明:${NC}"
echo -e "  ${GREEN}-port${NC}          : 监听端口，默认53"
echo -e "  ${GREEN}-upstream${NC}      : 上游DNS服务器，默认8.8.8.8:53"
echo -e "  ${GREEN}-debug${NC}         : 启用调试模式"
echo -e "  ${GREEN}-test-interval${NC} : 速度测试间隔（秒）"
echo -e "  ${GREEN}-service${NC}       : 服务操作(install/uninstall/start/stop)"
echo ""
echo -e "${YELLOW}高级示例:${NC}"
echo -e "  ${GREEN}./${OUTPUT_FILE} -upstream \"8.8.8.8:53,1.1.1.1:53\" -test-interval 300 -debug${NC}"
echo ""
echo -e "${YELLOW}服务管理:${NC}"
echo -e "  ${GREEN}安装服务:${NC} sudo ./${OUTPUT_FILE} -service install -upstream \"8.8.8.8:53,1.1.1.1:53\""
echo -e "  ${GREEN}启动服务:${NC} sudo ./${OUTPUT_FILE} -service start"
echo -e "  ${GREEN}停止服务:${NC} sudo ./${OUTPUT_FILE} -service stop"
echo -e "  ${GREEN}卸载服务:${NC} sudo ./${OUTPUT_FILE} -service uninstall"
echo ""
echo -e "${YELLOW}测试方法:${NC}"
echo -e "  ${GREEN}dig @127.0.0.1 -p 53 example.com A${NC}"
echo -e "  ${GREEN}dig @127.0.0.1 -p 53 example.com AAAA${NC}"
echo ""