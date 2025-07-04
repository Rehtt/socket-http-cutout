#!/bin/bash

# Linux网关HTTP原始socket数据包拦截和修改工具构建脚本
# 高性能Rust版本

set -e

echo "=== Linux网关HTTP原始socket数据包拦截和修改工具构建脚本 ==="
echo "构建高性能、内存安全的原始socket网络代理工具"
echo

# 检查Rust环境
if ! command -v cargo &> /dev/null; then
    echo "错误: 未找到Rust/Cargo，请先安装Rust"
    echo "安装命令: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
    exit 1
fi

echo "✓ Rust环境检查通过"
echo "Rust版本: $(rustc --version)"
echo "Cargo版本: $(cargo --version)"
echo

# 检查系统要求
echo "=== 系统要求检查 ==="
if [[ "$EUID" -eq 0 ]]; then
    echo "警告: 不建议以root用户编译，但运行时需要root权限"
fi

# 检查必要的系统库
echo "检查系统库..."
if ! ldconfig -p | grep -q "libc.so"; then
    echo "错误: 未找到libc库"
    exit 1
fi

echo "✓ 系统库检查通过"
echo

# 清理之前的构建
echo "=== 清理之前的构建 ==="
if [ -d "target" ]; then
    echo "清理target目录..."
    cargo clean
fi

echo "✓ 清理完成"
echo

# 构建项目
echo "=== 构建项目 ==="
echo "构建优化的release版本..."

# 设置构建环境变量
export RUSTFLAGS="-C target-cpu=native"

# 构建release版本
cargo build --release

# 检查构建结果
if [ -f "target/release/gateway-proxy-rust" ]; then
    echo "✓ 构建成功!"
    
    # 显示文件信息
    echo
    echo "=== 构建结果 ==="
    ls -lh target/release/gateway-proxy-rust
    
    # 显示文件大小
    SIZE=$(stat -c%s target/release/gateway-proxy-rust)
    SIZE_MB=$((SIZE / 1024 / 1024))
    echo "文件大小: ${SIZE_MB}MB"
    
    # 检查依赖
    echo
    echo "=== 依赖检查 ==="
    if command -v ldd &> /dev/null; then
        echo "动态链接库依赖:"
        ldd target/release/gateway-proxy-rust | head -10
    fi
    
    echo
    echo "=== 功能测试 ==="
    echo "测试帮助信息..."
    ./target/release/gateway-proxy-rust --help
    
    echo
    echo "测试版本信息..."
    ./target/release/gateway-proxy-rust --version
    
    echo
    echo "=== 构建完成 ==="
    echo "可执行文件: target/release/gateway-proxy-rust"
    echo "运行命令: sudo ./target/release/gateway-proxy-rust"
    echo
    echo "特性:"
    echo "  • 原始socket数据包捕获"
    echo "  • 实时HTTP数据包修改"
    echo "  • 零拷贝数据处理"
    echo "  • 内存安全保证"
    echo "  • 高性能异步处理"
    echo
    echo "注意事项:"
    echo "  • 需要root权限运行"
    echo "  • 支持Linux系统"
    echo "  • 监听网络接口流量"
    echo "  • 实时修改HTTP请求"
    echo
    echo "使用示例:"
    echo "  sudo ./target/release/gateway-proxy-rust -v"
    echo "  sudo ./target/release/gateway-proxy-rust -i eth0 -p 80"
    echo
    echo "详细文档: README_RUST.md"
    echo "使用指南: USAGE.md"
    
else
    echo "❌ 构建失败!"
    exit 1
fi 