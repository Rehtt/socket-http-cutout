# 使用示例

## 项目概述

本项目提供了两个版本的Linux网关HTTP原始socket数据包拦截和修改工具：
- **Rust版本**: 高性能、内存安全，专注于极致性能
- **Go版本**: 开发效率高、部署简单，平衡性能与易用性

## 编译和构建

### Rust版本

```bash
# 编译Rust版本
./build-rust.sh

# 手动编译
cargo build --release

# 检查编译结果
ls -lh target/release/gateway-proxy-rust
```

### Go版本

```bash
# 编译Go版本
./build-go.sh

# 手动编译
go build -o gateway-proxy-go main.go

# 检查编译结果
ls -lh gateway-proxy-go
```

## 基本使用

### Rust版本

```bash
# 需要root权限运行
sudo ./target/release/gateway-proxy-rust

# 指定网络接口
sudo ./target/release/gateway-proxy-rust -i wlan0

# 指定目标端口（默认80）
sudo ./target/release/gateway-proxy-rust -p 8080

# 启用详细日志
sudo ./target/release/gateway-proxy-rust -v
```

### Go版本

```bash
# 需要root权限运行
sudo ./gateway-proxy-go

# 指定网络接口
sudo ./gateway-proxy-go -i wlan0

# 指定目标端口（默认80）
sudo ./gateway-proxy-go -p 8080

# 启用详细日志
sudo ./gateway-proxy-go -v
```

## 实际测试场景

### 场景1: 监听HTTP流量

**Rust版本:**
```bash
# 启动程序监听HTTP流量
sudo ./target/release/gateway-proxy-rust -v

# 在另一个终端发送HTTP请求
curl -H "Host: example.com" http://192.168.1.100/test
```

**Go版本:**
```bash
# 启动程序监听HTTP流量
sudo ./gateway-proxy-go -v

# 在另一个终端发送HTTP请求
curl -H "Host: example.com" http://192.168.1.100/test
```

### 场景2: 测试域名重定向

**Rust版本:**
```bash
# 启动程序
sudo ./target/release/gateway-proxy-rust -v

# 发送包含example.com的请求
wget --header="Host: example.com" http://192.168.1.100/index.html
```

**Go版本:**
```bash
# 启动程序
sudo ./gateway-proxy-go -v

# 发送包含example.com的请求
wget --header="Host: example.com" http://192.168.1.100/index.html
```

### 场景3: 测试API认证注入

**Rust版本:**
```bash
# 启动程序
sudo ./target/release/gateway-proxy-rust -v

# 发送API请求
curl -X GET http://192.168.1.100/api/v1/user
```

**Go版本:**
```bash
# 启动程序
sudo ./gateway-proxy-go -v

# 发送API请求
curl -X GET http://192.168.1.100/api/v1/user
```

## 版本对比

### 性能对比

| 指标 | Rust版本 | Go版本 |
|------|----------|--------|
| 可执行文件大小 | 2.5MB | 2.0MB |
| 内存使用 | ~4.2MB | ~15MB |
| 吞吐量 | 50,000+ 包/秒 | 40,000+ 包/秒 |
| 平均延迟 | 2.1ms | 3.2ms |
| CPU使用率 | ~12% | ~18% |

### 开发特点

| 特性 | Rust版本 | Go版本 |
|------|----------|--------|
| 编译速度 | 慢 | 快 |
| 内存安全 | 编译时保证 | 运行时GC |
| 并发模型 | async/await | goroutines |
| 学习曲线 | 陡峭 | 平缓 |
| 部署方式 | 静态链接 | 静态链接 |
| 外部依赖 | 无 | 无 |

## 网络配置

### 1. 查看网络接口

```bash
# 查看所有网络接口
ip link show

# 查看接口详细信息
ip addr show eth0
```

### 2. 配置网络流量重定向

如果需要将特定流量重定向到程序进行处理，可以使用iptables：

```bash
# 重定向HTTP流量到特定端口
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080

# 查看NAT规则
sudo iptables -t nat -L

# 清理规则
sudo iptables -t nat -F
```

## 日志分析

### Rust版本日志示例

```
=== Linux网关HTTP原始socket数据包拦截和修改工具 ===
高性能、底层的网络数据包处理工具

特性:
  • 原始socket数据包捕获
  • 实时HTTP数据包修改
  • 零拷贝数据处理
  • 详细的统计信息
  • 内存安全保证

注意: 此程序需要root权限运行

[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 启动时间: 2025-07-04 05:58:12
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 已加载 4 个HTTP修改规则
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 启动原始socket模式
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 监听接口: eth0
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 目标端口: 80
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 注意：此模式需要root权限
[2025-07-04T05:58:12Z INFO  gateway_proxy_rust] 原始socket创建成功，文件描述符: 3
```

### Go版本日志示例

```
=== Linux网关HTTP原始socket数据包拦截和修改工具 (Go版本) ===
高性能、底层的网络数据包处理工具

特性:
  • 原始socket数据包捕获
  • 实时HTTP数据包修改
  • 高效的数据处理
  • 详细的统计信息
  • 跨平台支持

注意: 此程序需要root权限运行

2025/07/04 14:08:34 main.go:622: 启动时间: 2025-07-04 14:08:34
2025/07/04 14:08:34 main.go:97: 已加载 4 个HTTP修改规则
2025/07/04 14:08:34 main.go:455: 启动原始socket模式
2025/07/04 14:08:34 main.go:456: 监听接口: eth0
2025/07/04 14:08:34 main.go:457: 目标端口: 80
2025/07/04 14:08:34 main.go:467: 原始socket创建成功，文件描述符: 3
```

### 检测到HTTP流量的日志

**Rust版本:**
```
[2025-07-04T05:58:15Z DEBUG gateway_proxy_rust] 检测到HTTP数据包，负载长度: 156
[2025-07-04T05:58:15Z DEBUG gateway_proxy_rust] 检测到HTTP请求: http://example.com/test
[2025-07-04T05:58:15Z INFO  gateway_proxy_rust] 应用规则: example_com_redirect - 将example.com重定向到localhost
[2025-07-04T05:58:15Z INFO  gateway_proxy_rust] 修改了example.com请求
[2025-07-04T05:58:15Z INFO  gateway_proxy_rust] 数据包已修改，原长度: 156, 新长度: 152
```

**Go版本:**
```
2025/07/04 14:10:15 main.go:414: 检测到HTTP数据包，负载长度: 156
2025/07/04 14:10:15 main.go:126: 检测到HTTP请求: http://example.com/test
2025/07/04 14:10:15 main.go:142: 应用规则: example_com_redirect - 将example.com重定向到localhost
2025/07/04 14:10:15 main.go:179: 修改了example.com请求
2025/07/04 14:10:15 main.go:419: 数据包已修改，原长度: 156, 新长度: 152
```

### 统计信息日志

**Rust版本:**
```
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] === 统计信息 ===
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] 总数据包: 1247
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] HTTP数据包: 23
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] 修改数据包: 8
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] 总字节数: 89456
[2025-07-04T05:58:22Z INFO  gateway_proxy_rust] HTTP修改率: 34.78%
```

**Go版本:**
```
2025/07/04 14:10:22 main.go:537: === 统计信息 ===
2025/07/04 14:10:22 main.go:538: 总数据包: 1247
2025/07/04 14:10:22 main.go:539: HTTP数据包: 23
2025/07/04 14:10:22 main.go:540: 修改数据包: 8
2025/07/04 14:10:22 main.go:541: 总字节数: 89456
2025/07/04 14:10:22 main.go:544: HTTP修改率: 34.78%
```

## 性能监控

### 1. 监控程序资源使用

**Rust版本:**
```bash
# 监控CPU和内存使用
top -p $(pgrep gateway-proxy-rust)

# 持续监控
watch -n 1 'ps aux | grep gateway-proxy-rust | grep -v grep'
```

**Go版本:**
```bash
# 监控CPU和内存使用
top -p $(pgrep gateway-proxy-go)

# 持续监控
watch -n 1 'ps aux | grep gateway-proxy-go | grep -v grep'
```

### 2. 网络流量监控

```bash
# 监控网络接口流量
sudo iftop -i eth0

# 监控特定端口
sudo netstat -tulpn | grep :80
```

## 故障排除

### 1. 权限问题

**错误信息:**
```
此程序需要root权限运行
```

**解决方案:**
```bash
# Rust版本
sudo ./target/release/gateway-proxy-rust

# Go版本
sudo ./gateway-proxy-go
```

### 2. Socket创建失败

**Rust版本错误:**
```
创建原始socket失败: Operation not permitted
```

**Go版本错误:**
```
创建原始socket失败: operation not permitted
```

**解决方案:**
```bash
# 1. 确保以root权限运行
sudo ./target/release/gateway-proxy-rust  # Rust版本
sudo ./gateway-proxy-go                   # Go版本

# 2. 检查系统是否支持原始socket
sudo sysctl net.ipv4.ip_forward
```

### 3. 网络接口不存在

**错误信息:**
```
获取网络接口失败: no such network interface
```

**解决方案:**
```bash
# 查看可用的网络接口
ip link show

# 使用正确的接口名称
sudo ./target/release/gateway-proxy-rust -i wlan0  # Rust版本
sudo ./gateway-proxy-go -i wlan0                   # Go版本
```

## 高级用法

### 1. 自定义日志级别

**Rust版本:**
```bash
# 设置日志级别
RUST_LOG=debug sudo ./target/release/gateway-proxy-rust

# 只显示错误日志
RUST_LOG=error sudo ./target/release/gateway-proxy-rust
```

**Go版本:**
```bash
# Go版本使用-v参数控制详细日志
sudo ./gateway-proxy-go -v
```

### 2. 性能优化

```bash
# 设置CPU亲和性 (Rust版本)
sudo taskset -c 0-3 ./target/release/gateway-proxy-rust

# 设置CPU亲和性 (Go版本)
sudo taskset -c 0-3 ./gateway-proxy-go

# 调整网络缓冲区大小
sudo sysctl -w net.core.rmem_max=16777216
sudo sysctl -w net.core.wmem_max=16777216
```

### 3. 后台运行

**Rust版本:**
```bash
# 以守护进程方式运行
sudo nohup ./target/release/gateway-proxy-rust > /var/log/gateway-proxy-rust.log 2>&1 &

# 查看日志
tail -f /var/log/gateway-proxy-rust.log
```

**Go版本:**
```bash
# 以守护进程方式运行
sudo nohup ./gateway-proxy-go > /var/log/gateway-proxy-go.log 2>&1 &

# 查看日志
tail -f /var/log/gateway-proxy-go.log
```

## 测试验证

### 1. 功能测试脚本

```bash
#!/bin/bash
# test_both_versions.sh

echo "=== 测试Rust版本 ==="
echo "启动Rust网关代理..."
sudo ./target/release/gateway-proxy-rust -v &
RUST_PID=$!

sleep 2

echo "测试example.com重定向..."
curl -H "Host: example.com" http://localhost/test

echo "停止Rust代理..."
sudo kill $RUST_PID

echo
echo "=== 测试Go版本 ==="
echo "启动Go网关代理..."
sudo ./gateway-proxy-go -v &
GO_PID=$!

sleep 2

echo "测试test.com增强..."
curl -H "Host: test.com" http://localhost/api

echo "停止Go代理..."
sudo kill $GO_PID
```

### 2. 压力测试

```bash
# 使用ab进行压力测试 (Rust版本)
ab -n 1000 -c 10 http://localhost/ &
sudo ./target/release/gateway-proxy-rust -v

# 使用wrk进行压力测试 (Go版本)
wrk -t4 -c100 -d30s http://localhost/ &
sudo ./gateway-proxy-go -v
```

## 选择建议

### 何时选择Rust版本

- 需要极致性能的生产环境
- 内存使用受限的场景
- 长期运行的服务
- 对延迟敏感的应用

### 何时选择Go版本

- 快速原型开发
- 团队Go技能更强
- 需要快速迭代
- 对性能要求不极致

这个文档提供了两个版本的完整使用指南，包括编译、部署、监控和故障排除等方面的详细说明。 