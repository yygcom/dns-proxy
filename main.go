package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

var (
	listenPort     int
	upstreamDNS    string
	upstreamList   []string
	currentIndex   int
	debugMode      bool
	testInterval   int    // 速度测试间隔（秒）
	speedTestDone  bool   // 是否已完成首次速度测试
	serviceAction  string // 服务操作: install, uninstall, start, stop
	serviceName    = "dns-proxy" // 服务名称
)

func init() {
	flag.IntVar(&listenPort, "port", 53, "DNS代理服务器监听端口")
	flag.StringVar(&upstreamDNS, "upstream", "8.8.8.8:53", "上游DNS服务器地址（多个用逗号分隔，如：8.8.8.8:53,1.1.1.1:53）")
	flag.BoolVar(&debugMode, "debug", false, "启用调试模式，显示详细日志")
	flag.IntVar(&testInterval, "test-interval", 300, "DNS服务器速度测试间隔（秒，默认300秒=5分钟，设为0禁用定时测试）")
	flag.StringVar(&serviceAction, "service", "", "服务操作: install(安装服务), uninstall(卸载服务), start(启动服务), stop(停止服务)")
}

// debugLog 只在调试模式下输出日志
func debugLog(format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// DNSServerSpeed 存储DNS服务器速度信息
type DNSServerSpeed struct {
	Server   string
	Speed    time.Duration
	Reachable bool
}

// testDNSServerSpeed 测试单个DNS服务器的响应速度
func testDNSServerSpeed(server string) DNSServerSpeed {
	debugLog("测试DNS服务器速度: %s", server)
	
	c := new(dns.Client)
	c.Net = "udp"
	c.Timeout = 3 * time.Second
	
	// 创建测试查询 - 查询google.com的A记录
	m := new(dns.Msg)
	m.SetQuestion("google.com.", dns.TypeA)
	m.RecursionDesired = true
	
	start := time.Now()
	_, _, err := c.Exchange(m, server)
	duration := time.Since(start)
	
	result := DNSServerSpeed{
		Server:    server,
		Speed:     duration,
		Reachable: err == nil,
	}
	
	if err == nil {
		debugLog("DNS服务器 %s 响应成功，耗时: %v", server, duration)
	} else {
		debugLog("DNS服务器 %s 响应失败: %v，耗时: %v", server, err, duration)
	}
	
	return result
}

// testAllDNSServerSpeeds 并发测试所有DNS服务器的速度
func testAllDNSServerSpeeds() []DNSServerSpeed {
	log.Printf("开始测试所有上游DNS服务器速度...")
	
	var results []DNSServerSpeed
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// 为每个服务器启动一个goroutine进行并发测试
	for _, server := range upstreamList {
		wg.Add(1)
		go func(srv string) {
			defer wg.Done()
			result := testDNSServerSpeed(srv)
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(server)
	}
	
	wg.Wait()
	
	// 按速度排序（快的在前）
	sort.Slice(results, func(i, j int) bool {
		// 不可用的服务器排在最后
		if !results[i].Reachable && results[j].Reachable {
			return false
		}
		if results[i].Reachable && !results[j].Reachable {
			return true
		}
		// 都可用时按速度排序
		return results[i].Speed < results[j].Speed
	})
	
	log.Printf("DNS服务器速度测试完成:")
	for i, result := range results {
		status := "✓"
		if !result.Reachable {
			status = "✗"
		}
		log.Printf("  [%d] %s - %v %s", i+1, result.Server, result.Speed, status)
	}
	
	return results
}

// startPeriodicSpeedTest 启动定期速度测试
func startPeriodicSpeedTest() {
	if testInterval <= 0 {
		return
	}
	
	ticker := time.NewTicker(time.Duration(testInterval) * time.Second)
	defer ticker.Stop()
	
	log.Printf("启动定时速度测试，间隔: %v", time.Duration(testInterval)*time.Second)
	
	for range ticker.C {
		log.Printf("开始定期DNS服务器速度测试...")
		results := testAllDNSServerSpeeds()
		
		if len(results) > 0 {
			// 更新上游列表为排序后的结果
			var newUpstreamList []string
			var newCurrentIndex int = 0
			
			for _, result := range results {
			if result.Reachable {
				newUpstreamList = append(newUpstreamList, result.Server)
				// 如果当前使用的服务器在新列表中，保持当前索引
				if result.Server == upstreamList[currentIndex] {
					newCurrentIndex = len(newUpstreamList) - 1
				}
			}
		}
			
			if len(newUpstreamList) > 0 {
				// 原子更新上游列表和当前索引
				upstreamList = newUpstreamList
				currentIndex = newCurrentIndex
				log.Printf("已更新上游DNS服务器排序: %v", upstreamList)
				debugLog("当前使用的主DNS服务器: %s", upstreamList[currentIndex])
			}
		}
	}
}

func main() {
	flag.Parse()

	// 处理服务操作
	if serviceAction != "" {
		handleServiceAction()
		return
	}

	log.Printf("DNS代理程序启动，调试模式: %v", debugMode)

	// 解析多个上游DNS服务器
	upstreamList = parseUpstreamServers(upstreamDNS)
	if len(upstreamList) == 0 {
		log.Fatal("错误: 没有有效的上游DNS服务器")
	}
	currentIndex = 0

	log.Printf("上游DNS服务器列表: %v", upstreamList)
	debugLog("监听端口: %d", listenPort)
	debugLog("速度测试间隔: %d 秒", testInterval)

	// 启动时进行速度测试
	if len(upstreamList) > 1 {
		results := testAllDNSServerSpeeds()
		if len(results) > 0 {
			// 更新上游列表为排序后的结果
			var newUpstreamList []string
			for _, result := range results {
				if result.Reachable {
					newUpstreamList = append(newUpstreamList, result.Server)
				}
			}
			if len(newUpstreamList) > 0 {
				upstreamList = newUpstreamList
				log.Printf("已按速度排序上游DNS服务器: %v", upstreamList)
			}
		}
		speedTestDone = true
	}

	// 启动定时速度测试（如果启用了）
	if testInterval > 0 && len(upstreamList) > 1 {
		go startPeriodicSpeedTest()
	}

	// 创建DNS服务器
	server := &dns.Server{
		Addr: fmt.Sprintf(":%d", listenPort),
		Net:  "udp",
	}

	// 直接运行DNS服务器，不再区分服务模式
	// Linux服务通过systemd直接调用程序本身

	// 注册DNS查询处理函数
	dns.HandleFunc(".", handleDNSRequest)

	log.Printf("DNS代理服务器启动，监听端口: %d，上游DNS: %s", listenPort, upstreamDNS)

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("正在关闭DNS代理服务器...")
		server.Shutdown()
	}()

	// 启动服务器
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("DNS服务器启动失败: %v", err)
	}
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	debugLog("收到DNS查询: ID=%d, 问题数=%d", r.Id, len(r.Question))
	if debugMode && len(r.Question) > 0 {
		for i, q := range r.Question {
			debugLog("  问题[%d]: %s %s", i, q.Name, dns.TypeToString[q.Qtype])
		}
	}

	// 转发请求到上游DNS服务器
	upstreamMsg, err := forwardToUpstream(r)
	if err != nil {
		log.Printf("转发请求到上游DNS失败: %v", err)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	// 过滤IPv6记录
	filteredMsg := filterIPv6Records(upstreamMsg, r)
	
	debugLog("发送DNS响应: ID=%d, 回答数=%d", filteredMsg.Id, len(filteredMsg.Answer))
	w.WriteMsg(filteredMsg)
}

func parseUpstreamServers(servers string) []string {
	var result []string
	serverList := strings.Split(servers, ",")
	
	debugLog("解析上游DNS服务器: %s", servers)
	for i, server := range serverList {
		server = strings.TrimSpace(server)
		if server != "" {
			// 智能端口检测 - 区分IPv6地址和端口缺失
			server = ensurePortForDNSServer(server)
			result = append(result, server)
			debugLog("  上游DNS[%d]: %s", i, server)
		}
	}
	
	debugLog("共解析到 %d 个上游DNS服务器", len(result))
	return result
}

// ensurePortForDNSServer 确保DNS服务器地址有正确的端口
func ensurePortForDNSServer(server string) string {
	// 检查是否已经有端口
	if strings.LastIndex(server, ":") > strings.LastIndex(server, "]") {
		// 已经包含端口 (处理IPv6情况)
		return server
	}
	
	// 检查是否是IPv6地址 (包含[]或有多组:)
	if strings.HasPrefix(server, "[") && strings.HasSuffix(server, "]") {
		// 已经是[IPv6]格式，添加端口
		return server + ":53"
	}
	
	if strings.Count(server, ":") > 1 {
		// 可能是IPv6地址，用[]包裹并添加端口
		return "[" + server + "]:53"
	}
	
	// IPv4地址，直接添加端口
	return server + ":53"
}

func getNextUpstreamServer() string {
	if len(upstreamList) == 0 {
		return ""
	}
	
	server := upstreamList[currentIndex]
	currentIndex = (currentIndex + 1) % len(upstreamList)
	return server
}

func forwardToUpstream(r *dns.Msg) (*dns.Msg, error) {
	c := new(dns.Client)
	c.Net = "udp"
	c.Timeout = 5 * time.Second // 设置超时时间

	// 使用当前首选DNS服务器
	preferredServer := upstreamList[currentIndex]
	
	debugLog("转发DNS查询到上游服务器: %s", preferredServer)
	msg, _, err := c.Exchange(r, preferredServer)
	if err == nil {
		debugLog("上游DNS服务器 %s 响应成功", preferredServer)
		return msg, nil
	}
	
	log.Printf("上游DNS服务器 %s 失败: %v，尝试备用服务器", preferredServer, err)
	
	// 首选服务器失败，尝试其他备用服务器
	for i := 1; i < len(upstreamList); i++ {
		nextIndex := (currentIndex + i) % len(upstreamList)
		backupServer := upstreamList[nextIndex]
		
		debugLog("尝试备用上游DNS服务器: %s", backupServer)
		msg, _, err := c.Exchange(r, backupServer)
		if err == nil {
			// 备用服务器成功，更新当前索引
			currentIndex = nextIndex
			log.Printf("已切换到上游DNS服务器: %s", backupServer)
			debugLog("DNS查询成功，使用备用服务器: %s", backupServer)
			return msg, nil
		}
		debugLog("备用上游DNS服务器 %s 也失败: %v", backupServer, err)
	}
	
	return nil, fmt.Errorf("所有上游DNS服务器都不可用，最后一个错误: %v", err)
}

// Windows服务相关函数
func installWindowsService() {
	log.Printf("Windows服务安装指南：")
	log.Printf("由于Windows服务安装需要管理员权限和特定环境，请手动执行以下步骤：")
	log.Printf("")
	log.Printf("1. 以管理员身份运行命令提示符(CMD)")
	log.Printf("2. 执行以下命令创建服务：")
	log.Printf("")
	log.Printf("   sc create %s binPath= \"C:\\path\\to\\dns-proxy.exe -upstream \\\"%s\\\" -port %d\" start= auto displayname= \"DNS Proxy Service\"", serviceName, upstreamDNS, listenPort)
	log.Printf("")
	log.Printf("3. 启动服务：")
	log.Printf("   sc start %s", serviceName)
	log.Printf("")
	log.Printf("4. 设置服务为自动启动（可选）：")
	log.Printf("   sc config %s start= auto", serviceName)
	log.Printf("")
	log.Printf("5. 查看服务状态：")
	log.Printf("   sc query %s", serviceName)
	log.Printf("")
	log.Printf("6. 停止服务：")
	log.Printf("   sc stop %s", serviceName)
	log.Printf("")
	log.Printf("7. 卸载服务：")
	log.Printf("   sc delete %s", serviceName)
	log.Printf("")
	log.Printf("注意：请将 C:\\path\\to\\dns-proxy.exe 替换为实际的程序路径")
	log.Printf("注意：需要管理员权限才能创建和管理服务")
}

func uninstallWindowsService() {
	log.Printf("Windows服务卸载指南：")
	log.Printf("1. 以管理员身份运行命令提示符(CMD)")
	log.Printf("2. 执行以下命令停止服务：")
	log.Printf("   sc stop %s", serviceName)
	log.Printf("3. 执行以下命令卸载服务：")
	log.Printf("   sc delete %s", serviceName)
	log.Printf("4. 验证服务已卸载：")
	log.Printf("   sc query %s", serviceName)
	log.Printf("")
	log.Printf("注意：需要管理员权限才能卸载服务")
}

func startWindowsService() {
	log.Printf("Windows服务启动指南：")
	log.Printf("1. 以管理员身份运行命令提示符(CMD)")
	log.Printf("2. 执行以下命令启动服务：")
	log.Printf("   sc start %s", serviceName)
	log.Printf("3. 验证服务状态：")
	log.Printf("   sc query %s", serviceName)
	log.Printf("")
	log.Printf("注意：需要管理员权限才能启动服务")
}

func stopWindowsService() {
	log.Printf("Windows服务停止指南：")
	log.Printf("1. 以管理员身份运行命令提示符(CMD)")
	log.Printf("2. 执行以下命令停止服务：")
	log.Printf("   sc stop %s", serviceName)
	log.Printf("3. 验证服务已停止：")
	log.Printf("   sc query %s", serviceName)
	log.Printf("")
	log.Printf("注意：需要管理员权限才能停止服务")
}

// Windows服务相关函数
func installLinuxService() {
	log.Printf("正在安装Linux systemd服务...")
	
	// 获取当前可执行文件路径
	exepath, err := os.Executable()
	if err != nil {
		log.Fatalf("获取可执行文件路径失败: %v", err)
	}
	
	// 创建systemd服务文件内容
	serviceContent := fmt.Sprintf(`[Unit]
Description=DNS Proxy Service
After=network.target

[Service]
Type=simple
ExecStart=%s -upstream "%s" -port %d
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
`, exepath, upstreamDNS, listenPort)
	
	// 写入服务文件
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	err = os.WriteFile(serviceFile, []byte(serviceContent), 0644)
	if err != nil {
		log.Fatalf("创建systemd服务文件失败: %v", err)
	}
	
	log.Printf("systemd服务文件已创建: %s", serviceFile)
	
	// 重新加载systemd配置
	result, err := runCommand("systemctl", "daemon-reload")
	if err != nil {
		log.Fatalf("重新加载systemd配置失败: %v\n输出: %s", err, result)
	}
	
	// 启用服务
	result, err = runCommand("systemctl", "enable", serviceName)
	if err != nil {
		log.Fatalf("启用服务失败: %v\n输出: %s", err, result)
	}
	
	log.Printf("Linux systemd服务安装成功！")
	log.Printf("服务文件: %s", serviceFile)
	log.Printf("启动命令: systemctl start %s", serviceName)
	log.Printf("停止命令: systemctl stop %s", serviceName)
	log.Printf("状态命令: systemctl status %s", serviceName)
	log.Printf("卸载命令: systemctl disable %s && rm %s", serviceName, serviceFile)
}

func uninstallLinuxService() {
	log.Printf("正在卸载Linux systemd服务...")
	
	// 先停止服务
	stopLinuxService()
	
	// 禁用服务
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	result, err := runCommand("systemctl", "disable", serviceName)
	if err != nil {
		log.Printf("禁用服务失败: %v\n输出: %s", err, result)
	}
	
	// 删除服务文件
	err = os.Remove(serviceFile)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("删除服务文件失败: %v", err)
	}
	
	// 重新加载systemd配置
	result, err = runCommand("systemctl", "daemon-reload")
	if err != nil {
		log.Printf("重新加载systemd配置失败: %v\n输出: %s", err, result)
	}
	
	log.Printf("Linux systemd服务卸载成功！")
}

func startLinuxService() {
	log.Printf("正在启动Linux服务...")
	
	result, err := runCommand("systemctl", "start", serviceName)
	if err != nil {
		log.Fatalf("启动Linux服务失败: %v\n输出: %s", err, result)
	}
	
	log.Printf("Linux服务启动成功！")
	
	// 显示服务状态
	result, _ = runCommand("systemctl", "status", serviceName)
	log.Printf("服务状态:\n%s", result)
}

func stopLinuxService() {
	log.Printf("正在停止Linux服务...")
	
	result, err := runCommand("systemctl", "stop", serviceName)
	if err != nil {
		log.Printf("停止Linux服务失败: %v\n输出: %s", err, result)
		return
	}
	
	log.Printf("Linux服务停止成功！")
}

// 辅助函数：执行系统命令
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// handleServiceAction 处理服务安装/卸载/启动/停止操作
func handleServiceAction() {
	switch serviceAction {
	case "install":
		installService()
	case "uninstall":
		uninstallService()
	case "start":
		startService()
	case "stop":
		stopService()
	default:
		log.Fatalf("未知的服务操作: %s", serviceAction)
	}
}

// installService 安装系统服务
func installService() {
	log.Printf("正在安装 %s 服务...", serviceName)
	
	if runtime.GOOS == "windows" {
		installWindowsService()
	} else {
		installLinuxService()
	}
}

// uninstallService 卸载系统服务
func uninstallService() {
	log.Printf("正在卸载 %s 服务...", serviceName)
	
	if runtime.GOOS == "windows" {
		uninstallWindowsService()
	} else {
		uninstallLinuxService()
	}
}

// startService 启动服务
func startService() {
	log.Printf("正在启动 %s 服务...", serviceName)
	
	if runtime.GOOS == "windows" {
		startWindowsService()
	} else {
		startLinuxService()
	}
}

// stopService 停止服务
func stopService() {
	log.Printf("正在停止 %s 服务...", serviceName)
	
	if runtime.GOOS == "windows" {
		stopWindowsService()
	} else {
		stopLinuxService()
	}
}

func filterIPv6Records(msg *dns.Msg, originalQuery *dns.Msg) *dns.Msg {
	filteredMsg := new(dns.Msg)
	filteredMsg.SetReply(originalQuery)
	filteredMsg.Compress = msg.Compress

	// 检查原始查询类型
	queryType := originalQuery.Question[0].Qtype
	
	debugLog("开始过滤IPv6记录，原始记录数: %d", len(msg.Answer))
	
	// 过滤IPv6记录 (AAAA记录)
	var filteredAnswers []dns.RR
	ipv6Count := 0
	for _, rr := range msg.Answer {
		if rr.Header().Rrtype == dns.TypeAAAA {
			ipv6Count++
			debugLog("过滤掉IPv6记录: %s", rr.String())
			continue // 跳过IPv6记录
		}
		filteredAnswers = append(filteredAnswers, rr)
		debugLog("保留记录: %s", rr.String())
	}
	
	debugLog("过滤完成，IPv6记录数: %d, 保留记录数: %d", ipv6Count, len(filteredAnswers))

	// 根据查询类型和过滤结果决定响应
	if len(filteredAnswers) == 0 {
		if queryType == dns.TypeAAAA {
			// 查询AAAA记录但全部被过滤，返回NXDOMAIN
			filteredMsg.Rcode = dns.RcodeNameError
			filteredMsg.Answer = []dns.RR{}
			debugLog("查询类型为AAAA且全部被过滤，返回NXDOMAIN")
		} else {
			// 查询其他类型但无结果，保持原始响应码
			filteredMsg.Rcode = msg.Rcode
			filteredMsg.Answer = []dns.RR{}
			debugLog("查询类型为%d，过滤后无结果，保持响应码: %d", queryType, msg.Rcode)
		}
	} else {
		// 有过滤后的记录
		filteredMsg.Answer = filteredAnswers
		filteredMsg.Rcode = msg.Rcode
		debugLog("有过滤后的记录，响应码: %d, 记录数: %d", msg.Rcode, len(filteredAnswers))
	}
	
	// 保留其他部分
	filteredMsg.Ns = msg.Ns
	filteredMsg.Extra = msg.Extra

	return filteredMsg
}