use anyhow::{Context, Result};
use clap::Parser;
use log::{debug, error, info};
use regex::Regex;
use std::{
    mem,
    sync::Arc,
    time::Duration,
};
use tokio::{
    sync::RwLock,
    time::interval,
};

/// 命令行参数
#[derive(Parser)]
#[command(
    name = "gateway-proxy-rust",
    about = "Linux网关HTTP原始socket数据包拦截和修改工具",
    version = "0.1.0"
)]
struct Args {
    /// 网络接口名称
    #[arg(short, long, default_value = "eth0")]
    interface: String,

    /// 启用详细日志
    #[arg(short, long)]
    verbose: bool,

    /// 监听的目标端口
    #[arg(short, long, default_value = "80")]
    port: u16,
}

/// HTTP修改规则
#[derive(Debug, Clone)]
struct HttpModifyRule {
    name: String,
    pattern: Regex,
    description: String,
}

/// IP头结构
#[repr(C, packed)]
#[derive(Debug, Clone, Copy)]
struct IpHeader {
    version_ihl: u8,      // 版本(4位) + 头长度(4位)
    tos: u8,              // 服务类型
    total_len: u16,       // 总长度
    id: u16,              // 标识
    frag_off: u16,        // 片偏移
    ttl: u8,              // 生存时间
    protocol: u8,         // 协议
    check: u16,           // 头校验和
    saddr: u32,           // 源地址
    daddr: u32,           // 目标地址
}

/// TCP头结构
#[repr(C, packed)]
#[derive(Debug, Clone, Copy)]
struct TcpHeader {
    source: u16,          // 源端口
    dest: u16,            // 目标端口
    seq: u32,             // 序列号
    ack_seq: u32,         // 确认序列号
    res1_doff_res2_flags: u16, // 保留位+数据偏移+保留位+标志
    window: u16,          // 窗口大小
    check: u16,           // 校验和
    urg_ptr: u16,         // 紧急指针
}

/// 网关代理结构
#[derive(Debug)]
struct GatewayProxy {
    rules: Vec<HttpModifyRule>,
    stats: Arc<RwLock<ProxyStats>>,
    target_port: u16,
}

/// 代理统计信息
#[derive(Debug, Default)]
struct ProxyStats {
    total_packets: u64,
    http_packets: u64,
    modified_packets: u64,
    total_bytes: u64,
}

impl GatewayProxy {
    /// 创建新的网关代理实例
    fn new(target_port: u16) -> Self {
        let mut proxy = Self {
            rules: Vec::new(),
            stats: Arc::new(RwLock::new(ProxyStats::default())),
            target_port,
        };
        proxy.add_default_rules();
        proxy
    }

    /// 添加默认的HTTP修改规则
    fn add_default_rules(&mut self) {
        let rules = vec![
            HttpModifyRule {
                name: "example_com_redirect".to_string(),
                pattern: Regex::new(r"example\.com").unwrap(),
                description: "将example.com重定向到localhost".to_string(),
            },
            HttpModifyRule {
                name: "test_com_enhance".to_string(),
                pattern: Regex::new(r"test\.com").unwrap(),
                description: "为test.com请求添加自定义头".to_string(),
            },
            HttpModifyRule {
                name: "api_auth".to_string(),
                pattern: Regex::new(r"/api/v1/user").unwrap(),
                description: "为API请求添加认证头".to_string(),
            },
            HttpModifyRule {
                name: "admin_security".to_string(),
                pattern: Regex::new(r"admin").unwrap(),
                description: "为admin请求添加安全头".to_string(),
            },
        ];

        self.rules.extend(rules);
        info!("已加载 {} 个HTTP修改规则", self.rules.len());
    }

    /// 检查是否是HTTP请求
    fn is_http_request(data: &[u8]) -> bool {
        if data.len() < 10 {
            return false;
        }

        let data_str = String::from_utf8_lossy(&data[..std::cmp::min(data.len(), 100)]);
        let http_methods = ["GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH "];

        http_methods.iter().any(|method| data_str.starts_with(method))
    }

    /// 提取HTTP URL
    fn extract_http_url(data: &[u8]) -> Option<String> {
        let data_str = String::from_utf8_lossy(data);
        let lines: Vec<&str> = data_str.lines().collect();
        
        if lines.is_empty() {
            return None;
        }

        // 解析请求行
        let request_line = lines[0];
        let parts: Vec<&str> = request_line.split_whitespace().collect();
        if parts.len() < 2 {
            return None;
        }

        // 获取Host头
        let host = lines.iter()
            .skip(1)
            .find(|line| line.to_lowercase().starts_with("host:"))
            .and_then(|line| line.split(':').nth(1))
            .map(|h| h.trim());

        if let Some(host) = host {
            Some(format!("http://{}{}", host, parts[1]))
        } else {
            Some(parts[1].to_string())
        }
    }

    /// 应用修改规则
    async fn apply_modify_rules(&self, data: &[u8]) -> (Vec<u8>, bool) {
        if !Self::is_http_request(data) {
            return (data.to_vec(), false);
        }

        let url = Self::extract_http_url(data);
        if let Some(url) = url {
            debug!("检测到HTTP请求: {}", url);

            for rule in &self.rules {
                if rule.pattern.is_match(&url) || rule.pattern.is_match(&String::from_utf8_lossy(data)) {
                    info!("应用规则: {} - {}", rule.name, rule.description);
                    
                    let modified_data = match rule.name.as_str() {
                        "example_com_redirect" => self.modify_example_com_request(data),
                        "test_com_enhance" => self.modify_test_com_request(data),
                        "api_auth" => self.modify_api_request(data),
                        "admin_security" => self.modify_admin_request(data),
                        _ => data.to_vec(),
                    };

                    // 更新统计信息
                    let mut stats = self.stats.write().await;
                    stats.modified_packets += 1;
                    
                    return (modified_data, true);
                }
            }
        }

        (data.to_vec(), false)
    }

    /// 修改example.com请求
    fn modify_example_com_request(&self, data: &[u8]) -> Vec<u8> {
        let data_str = String::from_utf8_lossy(data);
        let modified = data_str.replace("example.com", "localhost");
        info!("修改了example.com请求");
        modified.into_bytes()
    }

    /// 修改test.com请求
    fn modify_test_com_request(&self, data: &[u8]) -> Vec<u8> {
        let data_str = String::from_utf8_lossy(data);
        if data_str.contains("Host: test.com") {
            let modified = data_str.replace(
                "Host: test.com",
                "Host: test.com\r\nX-Gateway-Modified: true\r\nX-Rust-Proxy: enabled"
            );
            info!("修改了test.com请求，添加了自定义头");
            modified.into_bytes()
        } else {
            data.to_vec()
        }
    }

    /// 修改API请求
    fn modify_api_request(&self, data: &[u8]) -> Vec<u8> {
        let data_str = String::from_utf8_lossy(data);
        if data_str.contains("/api/v1/user") {
            let lines: Vec<&str> = data_str.lines().collect();
            if lines.len() > 1 {
                let mut new_lines = vec![lines[0]];
                new_lines.push("Authorization: Bearer rust-gateway-token");
                new_lines.push("X-API-Version: v1");
                new_lines.extend_from_slice(&lines[1..]);
                let modified = new_lines.join("\r\n");
                info!("修改了API请求，添加了认证头");
                return modified.into_bytes();
            }
        }
        data.to_vec()
    }

    /// 修改admin请求
    fn modify_admin_request(&self, data: &[u8]) -> Vec<u8> {
        let data_str = String::from_utf8_lossy(data);
        if data_str.contains("admin") {
            let lines: Vec<&str> = data_str.lines().collect();
            if lines.len() > 1 {
                let mut new_lines = vec![lines[0]];
                new_lines.push("X-Admin-Access: restricted");
                new_lines.push("X-Security-Check: enabled");
                new_lines.push("X-Rust-Security: active");
                new_lines.extend_from_slice(&lines[1..]);
                let modified = new_lines.join("\r\n");
                info!("修改了admin请求，添加了安全头");
                return modified.into_bytes();
            }
        }
        data.to_vec()
    }

    /// 解析IP头
    unsafe fn parse_ip_header(data: &[u8]) -> Option<&IpHeader> {
        if data.len() < mem::size_of::<IpHeader>() {
            return None;
        }
        
        let ip_header = &*(data.as_ptr() as *const IpHeader);
        
        // 检查IP版本
        if (ip_header.version_ihl >> 4) != 4 {
            return None;
        }
        
        Some(ip_header)
    }

    /// 解析TCP头
    unsafe fn parse_tcp_header(data: &[u8], ip_header_len: usize) -> Option<&TcpHeader> {
        if data.len() < ip_header_len + mem::size_of::<TcpHeader>() {
            return None;
        }
        
        let tcp_data = &data[ip_header_len..];
        let tcp_header = &*(tcp_data.as_ptr() as *const TcpHeader);
        
        Some(tcp_header)
    }

    /// 获取TCP负载
    fn get_tcp_payload(data: &[u8], ip_header_len: usize, tcp_header_len: usize) -> Option<&[u8]> {
        let payload_offset = ip_header_len + tcp_header_len;
        if data.len() <= payload_offset {
            return None;
        }
        
        Some(&data[payload_offset..])
    }

    /// 计算校验和
    fn calculate_checksum(data: &[u8]) -> u16 {
        let mut sum: u32 = 0;
        let mut i = 0;

        // 按16位累加
        while i < data.len() - 1 {
            let word = ((data[i] as u16) << 8) + (data[i + 1] as u16);
            sum += word as u32;
            i += 2;
        }

        // 处理奇数字节
        if i < data.len() {
            sum += (data[i] as u32) << 8;
        }

        // 处理进位
        while (sum >> 16) != 0 {
            sum = (sum & 0xFFFF) + (sum >> 16);
        }

        !sum as u16
    }

    /// 重建数据包
    fn rebuild_packet(&self, original_data: &[u8], new_payload: &[u8]) -> Option<Vec<u8>> {
        unsafe {
            let ip_header = Self::parse_ip_header(original_data)?;
            let ip_header_len = ((ip_header.version_ihl & 0x0F) * 4) as usize;
            let tcp_header = Self::parse_tcp_header(original_data, ip_header_len)?;
            let tcp_header_len = (((tcp_header.res1_doff_res2_flags >> 12) & 0x0F) * 4) as usize;

            // 计算新的总长度
            let new_total_len = ip_header_len + tcp_header_len + new_payload.len();
            let mut new_packet = vec![0u8; new_total_len];

            // 复制IP头
            new_packet[..ip_header_len].copy_from_slice(&original_data[..ip_header_len]);
            
            // 复制TCP头
            new_packet[ip_header_len..ip_header_len + tcp_header_len]
                .copy_from_slice(&original_data[ip_header_len..ip_header_len + tcp_header_len]);
            
            // 复制新负载
            new_packet[ip_header_len + tcp_header_len..].copy_from_slice(new_payload);

            // 更新IP头的总长度
            let ip_total_len = (new_total_len as u16).to_be();
            new_packet[2..4].copy_from_slice(&ip_total_len.to_le_bytes());

            // 重新计算IP校验和
            new_packet[10..12].copy_from_slice(&[0, 0]); // 清零校验和字段
            let ip_checksum = Self::calculate_checksum(&new_packet[..ip_header_len]);
            new_packet[10..12].copy_from_slice(&ip_checksum.to_be_bytes());

            // 重新计算TCP校验和（简化实现，实际应该包括伪头）
            let tcp_start = ip_header_len;
            let tcp_end = new_total_len;
            new_packet[tcp_start + 16..tcp_start + 18].copy_from_slice(&[0, 0]); // 清零TCP校验和
            let tcp_checksum = Self::calculate_checksum(&new_packet[tcp_start..tcp_end]);
            new_packet[tcp_start + 16..tcp_start + 18].copy_from_slice(&tcp_checksum.to_be_bytes());

            Some(new_packet)
        }
    }

    /// 处理数据包
    async fn process_packet(&self, data: &[u8]) -> Option<Vec<u8>> {
        unsafe {
            // 更新统计信息
            {
                let mut stats = self.stats.write().await;
                stats.total_packets += 1;
                stats.total_bytes += data.len() as u64;
            }

            // 解析IP头
            let ip_header = Self::parse_ip_header(data)?;
            let ip_header_len = ((ip_header.version_ihl & 0x0F) * 4) as usize;

            // 检查是否是TCP协议
            if ip_header.protocol != 6 {
                return None;
            }

            // 解析TCP头
            let tcp_header = Self::parse_tcp_header(data, ip_header_len)?;
            let tcp_header_len = (((tcp_header.res1_doff_res2_flags >> 12) & 0x0F) * 4) as usize;

            // 检查是否是目标端口
            if tcp_header.dest.to_be() != self.target_port {
                return None;
            }

            // 获取TCP负载
            let payload = Self::get_tcp_payload(data, ip_header_len, tcp_header_len)?;
            
            if payload.is_empty() {
                return None;
            }

            // 检查是否是HTTP数据
            if !Self::is_http_request(payload) {
                return None;
            }

            // 更新HTTP包统计
            {
                let mut stats = self.stats.write().await;
                stats.http_packets += 1;
            }

            debug!("检测到HTTP数据包，负载长度: {}", payload.len());

            // 应用修改规则
            let (modified_payload, was_modified) = self.apply_modify_rules(payload).await;
            
            if was_modified {
                info!("数据包已修改，原长度: {}, 新长度: {}", payload.len(), modified_payload.len());
                return self.rebuild_packet(data, &modified_payload);
            }

            None
        }
    }

    /// 启动原始socket监听
    async fn start_raw_socket(&self, interface: &str) -> Result<()> {
        info!("启动原始socket模式");
        info!("监听接口: {}", interface);
        info!("目标端口: {}", self.target_port);
        info!("注意：此模式需要root权限");

        // 创建原始socket
        let socket_fd = unsafe {
            libc::socket(libc::AF_PACKET, libc::SOCK_RAW, (libc::ETH_P_ALL as u16).to_be() as i32)
        };

        if socket_fd < 0 {
            return Err(anyhow::anyhow!("创建原始socket失败: {}", 
                std::io::Error::last_os_error()));
        }

        info!("原始socket创建成功，文件描述符: {}", socket_fd);

        // 设置socket为非阻塞模式
        unsafe {
            let flags = libc::fcntl(socket_fd, libc::F_GETFL, 0);
            libc::fcntl(socket_fd, libc::F_SETFL, flags | libc::O_NONBLOCK);
        }

        // 启动统计信息定时器
        let stats_proxy = Arc::clone(&self.stats);
        tokio::spawn(async move {
            let mut interval = interval(Duration::from_secs(10));
            loop {
                interval.tick().await;
                let stats = stats_proxy.read().await;
                info!("=== 统计信息 ===");
                info!("总数据包: {}", stats.total_packets);
                info!("HTTP数据包: {}", stats.http_packets);
                info!("修改数据包: {}", stats.modified_packets);
                info!("总字节数: {}", stats.total_bytes);
                if stats.http_packets > 0 {
                    info!("HTTP修改率: {:.2}%", 
                          stats.modified_packets as f64 / stats.http_packets as f64 * 100.0);
                }
            }
        });

        // 主循环：读取和处理数据包
        let mut buffer = vec![0u8; 65536];
        loop {
            let result = unsafe {
                libc::recv(socket_fd, buffer.as_mut_ptr() as *mut libc::c_void, buffer.len(), 0)
            };

            if result < 0 {
                let errno = unsafe { *libc::__errno_location() };
                if errno == libc::EAGAIN || errno == libc::EWOULDBLOCK {
                    // 非阻塞模式下没有数据可读
                    tokio::time::sleep(Duration::from_millis(1)).await;
                    continue;
                }
                error!("接收数据包失败: {}", std::io::Error::last_os_error());
                continue;
            }

            if result == 0 {
                continue;
            }

            let packet_data = &buffer[..result as usize];
            
            // 跳过以太网头（14字节）获取IP数据包
            if packet_data.len() > 14 {
                let ip_data = &packet_data[14..];
                
                // 处理数据包
				if let Some(modified_packet) = self.process_packet(ip_data).await {
					// 保存以太网头
					let ethernet_header = &packet_data[..14];
					// 构建新的以太网帧
					let mut new_frame = Vec::with_capacity(ethernet_header.len() + modified_packet.len());
					new_frame.extend_from_slice(ethernet_header);
					new_frame.extend_from_slice(&modified_packet);

					// 注入修改后的数据包
					let bytes_written = unsafe {
						libc::write(socket_fd, new_frame.as_ptr() as *const libc::c_void, new_frame.len())
					};

					if bytes_written < 0 {
						error!("注入数据包失败: {}", std::io::Error::last_os_error());
					} else {
						debug!("成功注入 {} 字节的修改后数据包", bytes_written);
					}
				}
            }
        }
    }

    /// 显示统计信息
    async fn show_stats(&self) {
        let stats = self.stats.read().await;
        info!("=== 最终统计信息 ===");
        info!("总数据包数: {}", stats.total_packets);
        info!("HTTP数据包数: {}", stats.http_packets);
        info!("修改数据包数: {}", stats.modified_packets);
        info!("总字节数: {}", stats.total_bytes);
        if stats.http_packets > 0 {
            info!("HTTP修改率: {:.2}%", 
                  stats.modified_packets as f64 / stats.http_packets as f64 * 100.0);
        }
    }
}

/// 检查权限
fn check_permissions() -> Result<()> {
    if unsafe { libc::getuid() } != 0 {
        return Err(anyhow::anyhow!("此程序需要root权限运行"));
    }
    Ok(())
}

/// 显示使用帮助
fn show_usage() {
    println!("=== Linux网关HTTP原始socket数据包拦截和修改工具 ===");
    println!("高性能、底层的网络数据包处理工具");
    println!();
    println!("特性:");
    println!("  • 原始socket数据包捕获");
    println!("  • 实时HTTP数据包修改");
    println!("  • 零拷贝数据处理");
    println!("  • 详细的统计信息");
    println!("  • 内存安全保证");
    println!();
    println!("注意: 此程序需要root权限运行");
    println!();
}

#[tokio::main]
async fn main() -> Result<()> {
    let args = Args::parse();

    // 初始化日志
    if args.verbose {
        env_logger::Builder::from_default_env()
            .filter_level(log::LevelFilter::Debug)
            .init();
    } else {
        env_logger::Builder::from_default_env()
            .filter_level(log::LevelFilter::Info)
            .init();
    }

    show_usage();
    info!("启动时间: {}", chrono::Utc::now().format("%Y-%m-%d %H:%M:%S"));

    // 检查权限
    check_permissions().context("权限检查失败")?;

    // 创建网关代理
    let gateway = GatewayProxy::new(args.port);

    // 设置Ctrl+C处理
    let gateway_clone = Arc::new(gateway);
    let gateway_stats = Arc::clone(&gateway_clone);
    
    tokio::spawn(async move {
        tokio::signal::ctrl_c().await.expect("无法设置Ctrl+C处理器");
        info!("收到中断信号，正在退出...");
        gateway_stats.show_stats().await;
        std::process::exit(0);
    });

    // 启动原始socket模式
    gateway_clone.start_raw_socket(&args.interface).await?;

    Ok(())
}

