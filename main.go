package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// 命令行参数
var (
	interfaceName = flag.String("i", "eth0", "网络接口名称")
	targetPort    = flag.Int("p", 80, "监听的目标端口")
	verbose       = flag.Bool("v", false, "启用详细日志")
	showHelp      = flag.Bool("h", false, "显示帮助信息")
)

// IP头结构
type IPHeader struct {
	VersionIHL uint8  // 版本(4位) + 头长度(4位)
	TOS        uint8  // 服务类型
	Length     uint16 // 总长度
	ID         uint16 // 标识
	FragOff    uint16 // 片偏移
	TTL        uint8  // 生存时间
	Protocol   uint8  // 协议
	Checksum   uint16 // 头校验和
	SrcIP      uint32 // 源IP地址
	DstIP      uint32 // 目标IP地址
}

// TCP头结构
type TCPHeader struct {
	SrcPort    uint16 // 源端口
	DstPort    uint16 // 目标端口
	SeqNum     uint32 // 序列号
	AckNum     uint32 // 确认序列号
	DataOffset uint8  // 数据偏移
	Flags      uint8  // 标志位
	Window     uint16 // 窗口大小
	Checksum   uint16 // 校验和
	UrgPtr     uint16 // 紧急指针
}

// HTTP修改规则
type HTTPModifyRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
}

// 代理统计信息
type ProxyStats struct {
	TotalPackets    uint64
	HTTPPackets     uint64
	ModifiedPackets uint64
	TotalBytes      uint64
	mutex           sync.RWMutex
}

// 网关代理结构
type GatewayProxy struct {
	rules      []HTTPModifyRule
	stats      *ProxyStats
	targetPort int
	socket     int
}

// 创建新的网关代理实例
func NewGatewayProxy(targetPort int) *GatewayProxy {
	proxy := &GatewayProxy{
		rules:      make([]HTTPModifyRule, 0),
		stats:      &ProxyStats{},
		targetPort: targetPort,
	}
	proxy.addDefaultRules()
	return proxy
}

// 添加默认的HTTP修改规则
func (gp *GatewayProxy) addDefaultRules() {
	rules := []HTTPModifyRule{
		{
			Name:        "example_com_redirect",
			Pattern:     regexp.MustCompile(`example\.com`),
			Description: "将example.com重定向到localhost",
		},
		{
			Name:        "test_com_enhance",
			Pattern:     regexp.MustCompile(`test\.com`),
			Description: "为test.com请求添加自定义头",
		},
		{
			Name:        "api_auth",
			Pattern:     regexp.MustCompile(`/api/v1/user`),
			Description: "为API请求添加认证头",
		},
		{
			Name:        "admin_security",
			Pattern:     regexp.MustCompile(`admin`),
			Description: "为admin请求添加安全头",
		},
	}

	gp.rules = append(gp.rules, rules...)
	log.Printf("已加载 %d 个HTTP修改规则", len(gp.rules))
}

// 检查是否是HTTP请求
func (gp *GatewayProxy) isHTTPRequest(data []byte) bool {
	if len(data) < 10 {
		return false
	}

	dataStr := string(data[:min(len(data), 100)])
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH "}

	for _, method := range httpMethods {
		if strings.HasPrefix(dataStr, method) {
			return true
		}
	}
	return false
}

// 提取HTTP URL
func (gp *GatewayProxy) extractHTTPURL(data []byte) string {
	dataStr := string(data)
	lines := strings.Split(dataStr, "\n")

	if len(lines) == 0 {
		return ""
	}

	// 解析请求行
	requestLine := lines[0]
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return ""
	}

	// 获取Host头
	var host string
	for i := 1; i < len(lines); i++ {
		line := strings.ToLower(strings.TrimSpace(lines[i]))
		if strings.HasPrefix(line, "host:") {
			hostParts := strings.Split(line, ":")
			if len(hostParts) > 1 {
				host = strings.TrimSpace(hostParts[1])
				break
			}
		}
	}

	if host != "" {
		return fmt.Sprintf("http://%s%s", host, parts[1])
	}
	return parts[1]
}

// 应用修改规则
func (gp *GatewayProxy) applyModifyRules(data []byte) ([]byte, bool) {
	if !gp.isHTTPRequest(data) {
		return data, false
	}

	url := gp.extractHTTPURL(data)
	if url != "" {
		if *verbose {
			log.Printf("检测到HTTP请求: %s", url)
		}

		dataStr := string(data)
		for _, rule := range gp.rules {
			if rule.Pattern.MatchString(url) || rule.Pattern.MatchString(dataStr) {
				log.Printf("应用规则: %s - %s", rule.Name, rule.Description)

				var modifiedData []byte
				switch rule.Name {
				case "example_com_redirect":
					modifiedData = gp.modifyExampleComRequest(data)
				case "test_com_enhance":
					modifiedData = gp.modifyTestComRequest(data)
				case "api_auth":
					modifiedData = gp.modifyAPIRequest(data)
				case "admin_security":
					modifiedData = gp.modifyAdminRequest(data)
				default:
					modifiedData = data
				}

				// 更新统计信息
				gp.stats.mutex.Lock()
				gp.stats.ModifiedPackets++
				gp.stats.mutex.Unlock()

				return modifiedData, true
			}
		}
	}

	return data, false
}

// 修改example.com请求
func (gp *GatewayProxy) modifyExampleComRequest(data []byte) []byte {
	dataStr := string(data)
	modified := strings.ReplaceAll(dataStr, "example.com", "localhost")
	log.Println("修改了example.com请求")
	return []byte(modified)
}

// 修改test.com请求
func (gp *GatewayProxy) modifyTestComRequest(data []byte) []byte {
	dataStr := string(data)
	if strings.Contains(dataStr, "Host: test.com") {
		modified := strings.ReplaceAll(dataStr,
			"Host: test.com",
			"Host: test.com\r\nX-Gateway-Modified: true\r\nX-Go-Proxy: enabled")
		log.Println("修改了test.com请求，添加了自定义头")
		return []byte(modified)
	}
	return data
}

// 修改API请求
func (gp *GatewayProxy) modifyAPIRequest(data []byte) []byte {
	dataStr := string(data)
	if strings.Contains(dataStr, "/api/v1/user") {
		lines := strings.Split(dataStr, "\r\n")
		if len(lines) > 1 {
			newLines := []string{lines[0]}
			newLines = append(newLines, "Authorization: Bearer go-gateway-token")
			newLines = append(newLines, "X-API-Version: v1")
			newLines = append(newLines, lines[1:]...)
			modified := strings.Join(newLines, "\r\n")
			log.Println("修改了API请求，添加了认证头")
			return []byte(modified)
		}
	}
	return data
}

// 修改admin请求
func (gp *GatewayProxy) modifyAdminRequest(data []byte) []byte {
	dataStr := string(data)
	if strings.Contains(dataStr, "admin") {
		lines := strings.Split(dataStr, "\r\n")
		if len(lines) > 1 {
			newLines := []string{lines[0]}
			newLines = append(newLines, "X-Admin-Access: restricted")
			newLines = append(newLines, "X-Security-Check: enabled")
			newLines = append(newLines, "X-Go-Security: active")
			newLines = append(newLines, lines[1:]...)
			modified := strings.Join(newLines, "\r\n")
			log.Println("修改了admin请求，添加了安全头")
			return []byte(modified)
		}
	}
	return data
}

// 解析IP头
func (gp *GatewayProxy) parseIPHeader(data []byte) *IPHeader {
	if len(data) < 20 {
		return nil
	}

	header := (*IPHeader)(unsafe.Pointer(&data[0]))

	// 检查IP版本
	if (header.VersionIHL >> 4) != 4 {
		return nil
	}

	// 转换字节序
	header.Length = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.Length))[:])
	header.ID = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.ID))[:])
	header.FragOff = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.FragOff))[:])
	header.Checksum = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.Checksum))[:])
	header.SrcIP = binary.BigEndian.Uint32((*[4]byte)(unsafe.Pointer(&header.SrcIP))[:])
	header.DstIP = binary.BigEndian.Uint32((*[4]byte)(unsafe.Pointer(&header.DstIP))[:])

	return header
}

// 解析TCP头
func (gp *GatewayProxy) parseTCPHeader(data []byte, ipHeaderLen int) *TCPHeader {
	if len(data) < ipHeaderLen+20 {
		return nil
	}

	tcpData := data[ipHeaderLen:]
	header := (*TCPHeader)(unsafe.Pointer(&tcpData[0]))

	// 转换字节序
	header.SrcPort = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.SrcPort))[:])
	header.DstPort = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.DstPort))[:])
	header.SeqNum = binary.BigEndian.Uint32((*[4]byte)(unsafe.Pointer(&header.SeqNum))[:])
	header.AckNum = binary.BigEndian.Uint32((*[4]byte)(unsafe.Pointer(&header.AckNum))[:])
	header.Window = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.Window))[:])
	header.Checksum = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.Checksum))[:])
	header.UrgPtr = binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&header.UrgPtr))[:])

	return header
}

// 获取TCP负载
func (gp *GatewayProxy) getTCPPayload(data []byte, ipHeaderLen, tcpHeaderLen int) []byte {
	payloadOffset := ipHeaderLen + tcpHeaderLen
	if len(data) <= payloadOffset {
		return nil
	}
	return data[payloadOffset:]
}

// 计算校验和
func (gp *GatewayProxy) calculateChecksum(data []byte) uint16 {
	var sum uint32

	// 按16位累加
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 + uint32(data[i+1])
	}

	// 处理奇数字节
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// 处理进位
	for (sum >> 16) != 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return uint16(^sum)
}

// 重建数据包
func (gp *GatewayProxy) rebuildPacket(originalData, newPayload []byte) []byte {
	ipHeader := gp.parseIPHeader(originalData)
	if ipHeader == nil {
		return nil
	}

	ipHeaderLen := int(ipHeader.VersionIHL&0x0F) * 4
	tcpHeader := gp.parseTCPHeader(originalData, ipHeaderLen)
	if tcpHeader == nil {
		return nil
	}

	tcpHeaderLen := int(tcpHeader.DataOffset>>4) * 4

	// 计算新的总长度
	newTotalLen := ipHeaderLen + tcpHeaderLen + len(newPayload)
	newPacket := make([]byte, newTotalLen)

	// 复制IP头
	copy(newPacket[:ipHeaderLen], originalData[:ipHeaderLen])

	// 复制TCP头
	copy(newPacket[ipHeaderLen:ipHeaderLen+tcpHeaderLen],
		originalData[ipHeaderLen:ipHeaderLen+tcpHeaderLen])

	// 复制新负载
	copy(newPacket[ipHeaderLen+tcpHeaderLen:], newPayload)

	// 更新IP头的总长度
	binary.BigEndian.PutUint16(newPacket[2:4], uint16(newTotalLen))

	// 重新计算IP校验和
	binary.BigEndian.PutUint16(newPacket[10:12], 0) // 清零校验和字段
	ipChecksum := gp.calculateChecksum(newPacket[:ipHeaderLen])
	binary.BigEndian.PutUint16(newPacket[10:12], ipChecksum)

	// 重新计算TCP校验和（简化实现）
	tcpStart := ipHeaderLen
	tcpEnd := newTotalLen
	binary.BigEndian.PutUint16(newPacket[tcpStart+16:tcpStart+18], 0) // 清零TCP校验和
	tcpChecksum := gp.calculateChecksum(newPacket[tcpStart:tcpEnd])
	binary.BigEndian.PutUint16(newPacket[tcpStart+16:tcpStart+18], tcpChecksum)

	return newPacket
}

// 处理数据包
func (gp *GatewayProxy) processPacket(data []byte) []byte {
	// 更新统计信息
	gp.stats.mutex.Lock()
	gp.stats.TotalPackets++
	gp.stats.TotalBytes += uint64(len(data))
	gp.stats.mutex.Unlock()

	// 解析IP头
	ipHeader := gp.parseIPHeader(data)
	if ipHeader == nil {
		return nil
	}

	ipHeaderLen := int(ipHeader.VersionIHL&0x0F) * 4

	// 检查是否是TCP协议
	if ipHeader.Protocol != 6 {
		return nil
	}

	// 解析TCP头
	tcpHeader := gp.parseTCPHeader(data, ipHeaderLen)
	if tcpHeader == nil {
		return nil
	}

	tcpHeaderLen := int(tcpHeader.DataOffset>>4) * 4

	// 检查是否是目标端口
	if int(tcpHeader.DstPort) != gp.targetPort {
		return nil
	}

	// 获取TCP负载
	payload := gp.getTCPPayload(data, ipHeaderLen, tcpHeaderLen)
	if len(payload) == 0 {
		return nil
	}

	// 检查是否是HTTP数据
	if !gp.isHTTPRequest(payload) {
		return nil
	}

	// 更新HTTP包统计
	gp.stats.mutex.Lock()
	gp.stats.HTTPPackets++
	gp.stats.mutex.Unlock()

	if *verbose {
		log.Printf("检测到HTTP数据包，负载长度: %d", len(payload))
	}

	// 应用修改规则
	modifiedPayload, wasModified := gp.applyModifyRules(payload)

	if wasModified {
		log.Printf("数据包已修改，原长度: %d, 新长度: %d", len(payload), len(modifiedPayload))
		return gp.rebuildPacket(data, modifiedPayload)
	}

	return nil
}

// 启动原始socket监听
func (gp *GatewayProxy) startRawSocket(interfaceName string) error {
	log.Println("启动原始socket模式")
	log.Printf("监听接口: %s", interfaceName)
	log.Printf("目标端口: %d", gp.targetPort)
	log.Println("注意：此模式需要root权限")

	// 创建原始socket
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		return fmt.Errorf("创建原始socket失败: %v", err)
	}

	gp.socket = fd
	log.Printf("原始socket创建成功，文件描述符: %d", fd)

	// 绑定到指定网络接口
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return fmt.Errorf("获取网络接口失败: %v", err)
	}

	addr := &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  iface.Index,
	}

	err = syscall.Bind(fd, addr)
	if err != nil {
		return fmt.Errorf("绑定socket失败: %v", err)
	}

	log.Printf("socket已绑定到接口: %s (index: %d)", interfaceName, iface.Index)

	// 启动统计信息定时器
	go gp.statsTimer()

	// 主循环：读取和处理数据包
	buffer := make([]byte, 65536)
	for {
		n, err := syscall.Read(fd, buffer)
		if err != nil {
			log.Printf("读取数据包失败: %v", err)
			continue
		}

		if n == 0 {
			continue
		}

		packetData := buffer[:n]

		// 跳过以太网头（14字节）获取IP数据包
		if len(packetData) > 14 {
			ipData := packetData[14:]

			// 处理数据包
			if modifiedPacket := gp.processPacket(ipData); modifiedPacket != nil {
				// 保存以太网头
				ethernetHeader := packetData[:14]
				// 构建新的以太网帧
				newFrame := append(ethernetHeader, modifiedPacket...)

				// 注入修改后的数据包
				_, err := syscall.Write(fd, newFrame)
				if err != nil {
					log.Printf("注入数据包失败: %v", err)
				} else {
					if *verbose {
						log.Printf("成功注入 %d 字节的修改后数据包", len(newFrame))
					}
				}
			}
		}
	}
}

// 统计信息定时器
func (gp *GatewayProxy) statsTimer() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		gp.showStats()
	}
}

// 显示统计信息
func (gp *GatewayProxy) showStats() {
	gp.stats.mutex.RLock()
	defer gp.stats.mutex.RUnlock()

	log.Println("=== 统计信息 ===")
	log.Printf("总数据包: %d", gp.stats.TotalPackets)
	log.Printf("HTTP数据包: %d", gp.stats.HTTPPackets)
	log.Printf("修改数据包: %d", gp.stats.ModifiedPackets)
	log.Printf("总字节数: %d", gp.stats.TotalBytes)

	if gp.stats.HTTPPackets > 0 {
		modifyRate := float64(gp.stats.ModifiedPackets) / float64(gp.stats.HTTPPackets) * 100.0
		log.Printf("HTTP修改率: %.2f%%", modifyRate)
	}
}

// 显示最终统计信息
func (gp *GatewayProxy) showFinalStats() {
	gp.stats.mutex.RLock()
	defer gp.stats.mutex.RUnlock()

	log.Println("=== 最终统计信息 ===")
	log.Printf("总数据包数: %d", gp.stats.TotalPackets)
	log.Printf("HTTP数据包数: %d", gp.stats.HTTPPackets)
	log.Printf("修改数据包数: %d", gp.stats.ModifiedPackets)
	log.Printf("总字节数: %d", gp.stats.TotalBytes)

	if gp.stats.HTTPPackets > 0 {
		modifyRate := float64(gp.stats.ModifiedPackets) / float64(gp.stats.HTTPPackets) * 100.0
		log.Printf("HTTP修改率: %.2f%%", modifyRate)
	}
}

// 检查权限
func checkPermissions() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("此程序需要root权限运行")
	}
	return nil
}

// 显示使用帮助
func showUsage() {
	fmt.Println("=== Linux网关HTTP原始socket数据包拦截和修改工具 (Go版本) ===")
	fmt.Println("高性能、底层的网络数据包处理工具")
	fmt.Println()
	fmt.Println("特性:")
	fmt.Println("  • 原始socket数据包捕获")
	fmt.Println("  • 实时HTTP数据包修改")
	fmt.Println("  • 高效的数据处理")
	fmt.Println("  • 详细的统计信息")
	fmt.Println("  • 跨平台支持")
	fmt.Println()
	fmt.Println("注意: 此程序需要root权限运行")
	fmt.Println()
}

// 网络字节序转换
func htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

// 辅助函数：取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	flag.Parse()

	if *showHelp {
		showUsage()
		flag.PrintDefaults()
		return
	}

	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	showUsage()
	log.Printf("启动时间: %s", time.Now().Format("2006-01-02 15:04:05"))

	// 检查权限
	if err := checkPermissions(); err != nil {
		log.Fatalf("权限检查失败: %v", err)
	}

	// 创建网关代理
	gateway := NewGatewayProxy(*targetPort)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("收到中断信号，正在退出...")
		gateway.showFinalStats()
		if gateway.socket != 0 {
			syscall.Close(gateway.socket)
		}
		os.Exit(0)
	}()

	// 启动原始socket模式
	if err := gateway.startRawSocket(*interfaceName); err != nil {
		log.Fatalf("启动原始socket失败: %v", err)
	}
}
