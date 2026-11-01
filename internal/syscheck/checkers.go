package syscheck

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"

	"xsh/internal/config"
	"xsh/internal/sysinfo"
	"xsh/internal/toolset"
	"xsh/internal/ui"
)

func init() {
	registerChecker("cpu", checkCPU)
	registerChecker("memory", checkMemory)
	registerChecker("disk-space", checkDiskSpace)
	registerChecker("disk-health", checkDiskHealth)
	registerChecker("battery", checkBattery)
	registerChecker("uptime", checkUptime)
	registerChecker("security-services", checkSecurityServices)
	registerChecker("startup", checkStartup)
	registerChecker("temp", checkTemp)
	registerChecker("gateway", checkGateway)
	registerChecker("dns", checkDNS)
	registerChecker("uac", checkUAC)
	registerChecker("pending-reboot", checkPendingReboot)
	registerChecker("events", checkEvents)
}

func checkCPU() Check {
	c := Check{ID: "cpu", Name: "CPU 负载", Score: 10}
	pcts, err := cpu.Percent(time.Second, false)
	if err != nil || len(pcts) == 0 {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取 CPU 占用"
		return c
	}
	p := pcts[0]
	c.Detail = fmt.Sprintf("CPU 占用 %.0f%%", p)
	switch {
	case p >= 90:
		c.Sev, c.Score = SevCrit, 2
		c.Advice = "CPU 持续高位，用 `xsh diag perf` 定位占用进程"
	case p >= 75:
		c.Sev, c.Score = SevWarn, 5
		c.Advice = "CPU 占用偏高，关注后台进程"
	}
	return c
}

func checkMemory() Check {
	c := Check{ID: "memory", Name: "内存占用", Score: 10}
	vm, err := mem.VirtualMemory()
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取内存信息"
		return c
	}
	c.Detail = fmt.Sprintf("已用 %.1fGB / %.1fGB (%.0f%%)",
		float64(vm.Used)/(1<<30), float64(vm.Total)/(1<<30), vm.UsedPercent)
	switch {
	case vm.UsedPercent >= 95:
		c.Sev, c.Score = SevCrit, 2
		c.Advice = "内存接近耗尽，检查内存泄漏或增加内存"
	case vm.UsedPercent >= 85:
		c.Sev, c.Score = SevWarn, 5
		c.Advice = "内存占用偏高，关闭不需要的程序"
	}
	return c
}

func checkDiskSpace() Check {
	c := Check{ID: "disk-space", Name: "磁盘空间", Score: 10}
	parts, err := disk.Partitions(false)
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取分区"
		return c
	}
	var lines []string
	for _, p := range parts {
		du, err := disk.Usage(p.Mountpoint)
		if err != nil || du.Total == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %.0f%%", p.Mountpoint, du.UsedPercent))
		switch {
		case du.UsedPercent >= 95:
			c.Sev, c.Score = SevCrit, 2
			c.Advice = fmt.Sprintf("%s 剩余空间不足 5%%，请清理或扩容", p.Mountpoint)
		case du.UsedPercent >= 90 && c.Sev < SevWarn:
			c.Sev, c.Score = SevWarn, 5
			c.Advice = fmt.Sprintf("%s 剩余空间不足 10%%，建议清理", p.Mountpoint)
		}
	}
	c.Detail = strings.Join(lines, "  ")
	if c.Detail == "" {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "未获取到分区用量"
	}
	return c
}

func checkDiskHealth() Check {
	c := Check{ID: "disk-health", Name: "硬盘健康(SMART)", Score: 10}
	if msg, ok := sysinfo.SmartWarning(); ok {
		c.Sev, c.Score = SevCrit, 2
		c.Detail = fmt.Sprintf("SMART 故障预测: %s", msg)
		c.Advice = "立即备份数据并更换硬盘"
		return c
	}
	disks := sysinfo.PhysicalDisks()
	var lines []string
	for _, d := range disks {
		lines = append(lines, fmt.Sprintf("%s: %s", d.Name, d.Status))
		if d.Status != "healthy" {
			c.Sev, c.Score = SevCrit, 3
			c.Advice = "磁盘报告异常状态，请备份数据"
		}
	}
	c.Detail = strings.Join(lines, "  ")
	if c.Detail == "" {
		c.Detail = "未获取到 SMART 信息（可能需要管理员权限）"
	}
	return c
}

func checkBattery() Check {
	c := Check{ID: "battery", Name: "电池健康", Score: 10}
	design, full, ok := sysinfo.BatteryHealth()
	if !ok || design == 0 {
		c.Detail = "无电池或未获取到电池信息"
		return c
	}
	ratio := float64(full) / float64(design) * 100
	c.Detail = fmt.Sprintf("设计 %dmWh / 满充 %dmWh (%.0f%%)", design, full, ratio)
	switch {
	case ratio < 60:
		c.Sev, c.Score = SevWarn, 4
		c.Advice = "电池损耗较大，如需长续航建议更换"
	case ratio < 80:
		c.Sev, c.Score = SevInfo, 8
	}
	return c
}

func checkUptime() Check {
	c := Check{ID: "uptime", Name: "运行时长", Score: 10}
	up, err := host.Uptime()
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取运行时长"
		return c
	}
	c.Detail = sysinfo.HumanDuration(up)
	if up > 30*86400 {
		c.Sev, c.Score = SevWarn, 7
		c.Advice = "长时间未重启，建议重启以完成更新并释放资源"
	}
	return c
}

func checkSecurityServices() Check {
	c := Check{ID: "security-services", Name: "安全服务", Score: 10}
	svcs, err := sysinfo.ListServices()
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取服务状态"
		return c
	}
	find := func(name string) (string, bool) {
		for _, s := range svcs {
			if strings.EqualFold(s.Name, name) || strings.EqualFold(s.Display, name) {
				return s.State, true
			}
		}
		return "", false
	}
	var lines []string
	if st, ok := find("WinDefend"); ok {
		lines = append(lines, "Defender:"+st)
		if !strings.EqualFold(st, "running") {
			c.Sev, c.Score = SevWarn, 5
			c.Advice = "Windows Defender 未运行，确认是否安装第三方杀毒软件"
		}
	}
	if st, ok := find("wuauserv"); ok {
		lines = append(lines, "更新服务:"+st)
	}
	if st, ok := find("mpssvc"); ok {
		lines = append(lines, "防火墙:"+st)
		if !strings.EqualFold(st, "running") && c.Sev < SevWarn {
			c.Sev, c.Score = SevWarn, 5
			if c.Advice == "" {
				c.Advice = "Windows 防火墙服务未运行"
			}
		}
	}
	c.Detail = strings.Join(lines, "  ")
	if c.Detail == "" {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "未获取到安全服务状态"
	}
	return c
}

func checkStartup() Check {
	c := Check{ID: "startup", Name: "启动项", Score: 10}
	items, err := toolset.StartupList()
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取启动项"
		return c
	}
	c.Detail = fmt.Sprintf("共 %d 项", len(items))
	if len(items) > 25 {
		c.Sev, c.Score = SevWarn, 6
		c.Advice = "启动项较多影响开机速度，用 `xsh startup` 查看并清理"
	}
	return c
}

func checkTemp() Check {
	c := Check{ID: "temp", Name: "临时文件", Score: 10}
	dir := os.Getenv("TEMP")
	if dir == "" {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "未找到 TEMP 目录"
		return c
	}
	var size int64
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if size > 8<<30 {
			return filepath.SkipAll
		}
		if !d.IsDir() {
			if fi, e := d.Info(); e == nil {
				size += fi.Size()
			}
		}
		return nil
	})
	c.Detail = fmt.Sprintf("%s: %s", dir, ui.HumanBytes(float64(size)))
	if size > 5<<30 {
		c.Sev, c.Score = SevWarn, 6
		c.Advice = "临时目录过大，建议使用磁盘清理或存储感知"
	}
	return c
}

func checkGateway() Check {
	c := Check{ID: "gateway", Name: "网关连通", Score: 10}
	gw := sysinfo.DefaultGateway()
	if gw == "" {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "未检测到默认网关"
		return c
	}
	out, _ := exec.Command("ping", "-n", "2", "-w", "1000", gw).Output()
	s := string(out)
	c.Detail = fmt.Sprintf("网关 %s", gw)
	switch {
	case strings.Contains(s, "丢失 = 0") || strings.Contains(s, "Lost = 0"):
	case strings.Contains(s, "无法访问") || strings.Contains(s, "unreachable") || s == "":
		c.Sev, c.Score = SevCrit, 3
		c.Detail += " 不可达"
		c.Advice = "检查网线/Wi-Fi 连接与路由器状态"
	default:
		c.Sev, c.Score = SevWarn, 5
		c.Detail += " 存在丢包"
		c.Advice = "链路存在丢包，检查网络质量"
	}
	return c
}

func checkDNS() Check {
	c := Check{ID: "dns", Name: "DNS 解析", Score: 10}
	domain := config.Load().ProbeDomain
	if domain == "" {
		domain = "www.msftconnecttest.com"
	}
	if _, err := net.LookupIP(domain); err != nil {
		c.Sev, c.Score = SevCrit, 3
		c.Detail = fmt.Sprintf("解析 %s 失败", domain)
		c.Advice = "DNS 不可用，运行 `xsh diag net` 排查"
		return c
	}
	c.Detail = fmt.Sprintf("解析 %s 正常", domain)
	return c
}

func checkUAC() Check {
	c := Check{ID: "uac", Name: "UAC 状态", Score: 10}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, registry.QUERY_VALUE)
	if err != nil {
		c.Sev, c.Score = SevInfo, 5
		c.Detail = "无法读取 UAC 配置"
		return c
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("EnableLUA")
	if err != nil || v == 0 {
		c.Sev, c.Score = SevWarn, 5
		c.Detail = "UAC 已禁用"
		c.Advice = "建议启用 UAC（用户账户控制）降低恶意程序提权风险"
		return c
	}
	c.Detail = "UAC 已启用"
	return c
}

func checkPendingReboot() Check {
	c := Check{ID: "pending-reboot", Name: "待重启更新", Score: 10}
	keys := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
	}
	for _, p := range keys {
		if k, err := registry.OpenKey(registry.LOCAL_MACHINE, p, registry.QUERY_VALUE); err == nil {
			k.Close()
			c.Sev, c.Score = SevInfo, 8
			c.Detail = "有待完成安装的系统更新"
			c.Advice = "重启电脑以完成系统更新"
			return c
		}
	}
	c.Detail = "无待重启更新"
	return c
}

type wmiEvent struct {
	EventCode uint32
}

func checkEvents() Check {
	c := Check{ID: "events", Name: "系统错误事件", Score: 10}
	since := dmtfDateTime(time.Now().Add(-24 * time.Hour))
	q := fmt.Sprintf("SELECT EventCode FROM Win32_NTLogEvent WHERE Logfile='System' AND EventType=1 AND TimeGenerated >= '%s'", since)
	var evts []wmiEvent
	type evRes struct {
		evts []wmiEvent
		err  error
	}
	ch := make(chan evRes, 1)
	go func() {
		var es []wmiEvent
		err := wmi.QueryNamespace(q, &es, `root\cimv2`)
		ch <- evRes{es, err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			c.Sev, c.Score = SevInfo, 8
			c.Detail = "无法读取事件日志（新系统 API 变更或权限不足）"
			return c
		}
		evts = res.evts
	case <-time.After(8 * time.Second):
		c.Sev, c.Score = SevInfo, 8
		c.Detail = "事件日志查询超时（跳过）"
		return c
	}
	n := len(evts)
	c.Detail = fmt.Sprintf("最近 24 小时 %d 条错误事件", n)
	switch {
	case n > 50:
		c.Sev, c.Score = SevCrit, 4
		c.Advice = "错误事件密集，用 `xsh diag event` 查看详情"
	case n > 10:
		c.Sev, c.Score = SevWarn, 6
		c.Advice = "存在较多错误事件，用 `xsh diag event` 查看详情"
	}
	return c
}

// dmtfDateTime 生成 WMI DMTF 时间字符串（如 20250101120000.000000+480）。
func dmtfDateTime(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign = "-"
		off = -off
	}
	return t.Format("20060102150405.000000") + fmt.Sprintf("%s%03d", sign, off/60)
}
