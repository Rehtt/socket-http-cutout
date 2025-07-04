#!/bin/bash

# Linux网关HTTP原始socket数据包拦截和修改工具构建脚本 (Go版本)
# 高性能Go版本

set -e

echo "=== Linux网关HTTP原始socket数据包拦截和修改工具构建脚本 (Go版本) ==="
echo "构建高性能、跨平台的原始socket网络代理工具"
echo

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go编译器，请先安装Go"
    echo "安装命令: https://golang.org/doc/install"
    exit 1
fi

echo "✓ Go环境检查通过"
echo "Go版本: $(go version)"
echo

# 检查系统要求
echo "=== 系统要求检查 ==="
if [[ "$EUID" -eq 0 ]]; then
    echo "警告: 不建议以root用户编译，但运行时需要root权限"
fi

echo "✓ 系统检查通过"
echo

# 清理之前的构建
echo "=== 清理之前的构建 ==="
if [ -f "gateway-proxy-go" ]; then
    echo "清理旧的可执行文件..."
    rm -f gateway-proxy-go
fi

echo "✓ 清理完成"
echo

# 构建项目
echo "=== 构建项目 ==="
echo "构建优化的Go版本..."

# 设置构建环境变量
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

# 构建可执行文件
echo "正在编译..."
go build -ldflags="-w -s" -o gateway-proxy-go main.go

# 检查构建结果
if [ -f "gateway-proxy-go" ]; then
    echo "✓ 构建成功!"
    
    # 显示文件信息
    echo
    echo "=== 构建结果 ==="
    ls -lh gateway-proxy-go
    
    # 显示文件大小
    SIZE=$(stat -c%s gateway-proxy-go)
    SIZE_MB=$((SIZE / 1024 / 1024))
    echo "文件大小: ${SIZE_MB}MB"
    
    # 检查依赖
    echo
    echo "=== 依赖检查 ==="
    if command -v ldd &> /dev/null; then
        echo "动态链接库依赖:"
        ldd gateway-proxy-go | head -10 || echo "静态链接，无外部依赖"
    fi
    
    echo
    echo "=== 功能测试 ==="
    echo "测试帮助信息..."
    ./gateway-proxy-go -h
    
    echo
    echo "=== 构建完成 ==="
    echo "可执行文件: gateway-proxy-go"
    echo "运行命令: sudo ./gateway-proxy-go"
    echo
    echo "特性:"
    echo "  • 原始socket数据包捕获"
    echo "  • 实时HTTP数据包修改"
    echo "  • 高效的数据处理"
    echo "  • 详细的统计信息"
    echo "  • 跨平台支持"
    echo
    echo "注意事项:"
    echo "  • 需要root权限运行"
    echo "  • 支持Linux系统"
    echo "  • 监听网络接口流量"
    echo "  • 实时修改HTTP请求"
    echo
    echo "使用示例:"
    echo "  sudo ./gateway-proxy-go -v"
    echo "  sudo ./gateway-proxy-go -i eth0 -p 80"
    echo
    echo "详细文档: README_GO.md"
    echo "使用指南: USAGE.md"
    
else
    echo "❌ 构建失败!"
    exit 1
fi 