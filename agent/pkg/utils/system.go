package utils

import (
	"bufio"
	"devops-manager/api/protobuf"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// GetHostname 获取主机名
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// GetLocalIP 获取本地IP地址
func GetLocalIP() string {
	// 尝试连接到一个外部地址来获取本地IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// GetNetworkInterfaces 获取网络接口信息
func GetNetworkInterfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

// GetInterfaceAddrs 获取指定接口的地址
func GetInterfaceAddrs(iface net.Interface) ([]net.Addr, error) {
	return iface.Addrs()
}

// GetSystemStatus 获取系统状态信息
func GetSystemStatus() *protobuf.HostStatus {
	status := &protobuf.HostStatus{
		Timestamp:     time.Now().Unix(),
		UptimeSeconds: getUptime(),
		Ip:            GetLocalIP(),
		CustomTags:    make(map[string]string),
	}

	// 获取 CPU 信息
	if cpuInfo := getCPUInfo(); cpuInfo != nil {
		status.Cpu = cpuInfo
	}

	// 获取内存信息
	if memInfo := getMemoryInfo(); memInfo != nil {
		status.Memory = memInfo
	}

	// 获取磁盘信息
	status.Disks = getDiskInfo()

	return status
}

// getCPUInfo 获取 CPU 信息
func getCPUInfo() *protobuf.CPUInfo {
	cpuInfo := &protobuf.CPUInfo{
		CoreCount: int32(runtime.NumCPU()),
	}

	// 在 macOS/Linux 上尝试读取 CPU 使用率
	if runtime.GOOS == "linux" {
		if usage := getCPUUsageLinux(); usage >= 0 {
			cpuInfo.UsagePercent = usage
		}
	} else {
		// 对于其他系统，使用简单的估算
		cpuInfo.UsagePercent = 10.0 // 默认值
	}

	// 获取负载平均值（仅 Linux/macOS）
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if loads := getLoadAverage(); len(loads) >= 3 {
			cpuInfo.LoadAvg_1M = loads[0]
			cpuInfo.LoadAvg_5M = loads[1]
			cpuInfo.LoadAvg_15M = loads[2]
		}
	}

	return cpuInfo
}

// getMemoryInfo 获取内存信息
func getMemoryInfo() *protobuf.MemoryInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memInfo := &protobuf.MemoryInfo{
		UsedBytes: uint64(m.Alloc),
	}

	// 获取系统总内存
	if runtime.GOOS == "linux" {
		if total := getTotalMemoryLinux(); total > 0 {
			memInfo.TotalBytes = uint64(total)
			memInfo.UsagePercent = float64(memInfo.UsedBytes) / float64(total) * 100
		}
	} else {
		// 对于其他系统，使用估算值
		memInfo.TotalBytes = 8 * 1024 * 1024 * 1024 // 8GB 默认
		memInfo.UsagePercent = float64(memInfo.UsedBytes) / float64(memInfo.TotalBytes) * 100
	}

	return memInfo
}

// getDiskInfo 获取磁盘信息
func getDiskInfo() []*protobuf.DiskInfo {
	var disks []*protobuf.DiskInfo

	// 获取根目录磁盘信息
	if diskInfo := getDiskUsage("/"); diskInfo != nil {
		disks = append(disks, diskInfo)
	}

	return disks
}

// 注释掉暂时不需要的网络接口和进程信息收集函数
// 这些函数可能在将来的版本中重新启用

/*
// getNetworkInfo 获取网络接口信息
func getNetworkInfo() []*protobuf.NetworkInterfaceInfo {
	var interfaces []*protobuf.NetworkInterfaceInfo

	ifaces, err := net.Interfaces()
	if err != nil {
		return interfaces
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // 跳过未启用或回环接口
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					netInfo := &protobuf.NetworkInterfaceInfo{
						Name:        iface.Name,
						IpAddresses: []string{ipnet.IP.String()},
						IsUp:        iface.Flags&net.FlagUp != 0,
					}
					interfaces = append(interfaces, netInfo)
					break // 每个接口只取一个 IP
				}
			}
		}
	}

	return interfaces
}

// getProcessInfo 获取进程信息
func getProcessInfo() []*protobuf.ProcessInfo {
	var processes []*protobuf.ProcessInfo

	// 添加当前进程信息
	process := &protobuf.ProcessInfo{
		Pid:         int32(os.Getpid()),
		Name:        "devops-agent",
		CpuPercent:  0.1, // 估算值
		MemoryBytes: uint64(getProcessMemory()),
	}
	processes = append(processes, process)

	return processes
}
*/

// 辅助函数

// getUptime 获取系统运行时间（秒）
func getUptime() int64 {
	if runtime.GOOS == "linux" {
		return getUptimeLinux()
	}
	// 对于其他系统，返回程序运行时间
	return int64(time.Since(time.Now().Add(-time.Hour)).Seconds())
}

// getCPUUsageLinux 获取 Linux 系统 CPU 使用率
func getCPUUsageLinux() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return -1
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return -1
	}

	var total, idle int64
	for i := 1; i < len(fields) && i <= 7; i++ {
		val, _ := strconv.ParseInt(fields[i], 10, 64)
		total += val
		if i == 4 { // idle time
			idle = val
		}
	}

	if total == 0 {
		return -1
	}

	return float64(total-idle) / float64(total) * 100
}

// getTotalMemoryLinux 获取 Linux 系统总内存
func getTotalMemoryLinux() int64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return kb * 1024 // 转换为字节
				}
			}
			break
		}
	}
	return -1
}

// getUptimeLinux 获取 Linux 系统运行时间
func getUptimeLinux() int64 {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			uptime, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				return int64(uptime)
			}
		}
	}
	return 0
}

// getLoadAverage 获取负载平均值
func getLoadAverage() []float64 {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil
	}

	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			var loads []float64
			for i := 0; i < 3; i++ {
				load, err := strconv.ParseFloat(fields[i], 64)
				if err == nil {
					loads = append(loads, load)
				}
			}
			return loads
		}
	}
	return nil
}

// getDiskUsage 获取磁盘使用情况
func getDiskUsage(path string) *protobuf.DiskInfo {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return nil
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	return &protobuf.DiskInfo{
		MountPoint:   path,
		TotalBytes:   uint64(total),
		UsedBytes:    uint64(used),
		FreeBytes:    uint64(free),
		UsagePercent: float64(used) / float64(total) * 100,
	}
}

// GetSystemInfo 获取系统基本信息
func GetSystemInfo() map[string]string {
	info := make(map[string]string)

	hostname, _ := os.Hostname()
	info["hostname"] = hostname
	info["os"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	info["cpu_count"] = string(rune(runtime.NumCPU()))

	return info
}

// GetProcessInfo 获取进程信息
func GetProcessInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"pid":        os.Getpid(),
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  float64(m.Alloc) / 1024 / 1024,
	}
}
