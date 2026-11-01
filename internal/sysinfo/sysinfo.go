// Package sysinfo 提供跨模块共享的系统信息采集能力。
package sysinfo

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/yusufpapurcu/wmi"
)

// ProcInfo 进程摘要。
type ProcInfo struct {
	PID   int32
	Name  string
	CPU   float64 // 百分比（单核占比均值）
	MemMB float64
}

// ProcSnapshot 采样所有进程的 CPU/内存（阻塞 interval 时长）。
func ProcSnapshot(interval time.Duration) ([]ProcInfo, error) {
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	t1, err := procTimes()
	if err != nil {
		return nil, err
	}
	time.Sleep(interval)
	t2, err := procTimes()
	if err != nil {
		return nil, err
	}
	ncpu := float64(numCPU())
	var out []ProcInfo
	for pid, e2 := range t2 {
		cpu := 0.0
		if e1, ok := t1[pid]; ok {
			cpu = (e2 - e1) / interval.Seconds() / ncpu * 100
		}
		info := probePid(pid)
		if info == nil {
			continue
		}
		info.CPU = cpu
		out = append(out, *info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPU > out[j].CPU })
	return out, nil
}

// TopProcs 返回按 CPU 占用排序的前 n 个进程。
func TopProcs(n int) ([]ProcInfo, error) {
	all, err := ProcSnapshot(300 * time.Millisecond)
	if err != nil {
		return nil, err
	}
	if n > 0 && len(all) > n {
		return all[:n], nil
	}
	return all, nil
}

func numCPU() int { return runtimeNumCPU() }

// HumanDuration 秒数转为人类可读时长。
func HumanDuration(sec uint64) string {
	d := sec / 86400
	h := sec % 86400 / 3600
	m := sec % 3600 / 60
	if d > 0 {
		return fmt.Sprintf("%d天%d小时%d分", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%d小时%d分", h, m)
	}
	return fmt.Sprintf("%d分钟", m)
}

// LANIPv4s 返回非回环 IPv4 地址。
func LANIPv4s() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range ifaces {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				out = append(out, ip4.String())
			}
		}
	}
	return out
}

// DefaultGateway 返回默认网关（可能为空）。
func DefaultGateway() string {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1).NextHop").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// --- WMI 采集 ---

type wmiProcessor struct{ Name string }

// CPUModel 返回处理器型号。
func CPUModel() string {
	var dst []wmiProcessor
	if err := wmi.Query("SELECT Name FROM Win32_Processor", &dst); err != nil || len(dst) == 0 {
		return ""
	}
	return strings.TrimSpace(dst[0].Name)
}

type wmiVideo struct {
	Name                        string
	CurrentHorizontalResolution uint32
	CurrentVerticalResolution   uint32
}

// GPUInfo 返回主显卡名称与当前分辨率。
func GPUInfo() (name string, w, h uint32) {
	var dst []wmiVideo
	if err := wmi.Query("SELECT Name, CurrentHorizontalResolution, CurrentVerticalResolution FROM Win32_VideoController", &dst); err != nil || len(dst) == 0 {
		return "", 0, 0
	}
	return strings.TrimSpace(dst[0].Name), dst[0].CurrentHorizontalResolution, dst[0].CurrentVerticalResolution
}

type wmiBatteryStatic struct{ DesignedCapacity uint32 }
type wmiBatteryFull struct{ FullChargedCapacity uint32 }

// BatteryHealth 返回电池设计容量与满充容量。
func BatteryHealth() (design, full uint32, ok bool) {
	var d1 []wmiBatteryStatic
	if err := wmi.QueryNamespace("SELECT DesignedCapacity FROM BatteryStaticData", &d1, `root\wmi`); err != nil || len(d1) == 0 {
		return 0, 0, false
	}
	var d2 []wmiBatteryFull
	if err := wmi.QueryNamespace("SELECT FullChargedCapacity FROM BatteryFullChargedCapacity", &d2, `root\wmi`); err != nil || len(d2) == 0 {
		return 0, 0, false
	}
	return d1[0].DesignedCapacity, d2[0].FullChargedCapacity, true
}

type wmiPredict struct {
	InstanceName   string
	PredictFailure bool
}

// SmartWarning 返回 SMART 故障预测（仅传统磁盘，NVMe 见 PhysicalDisks）。
func SmartWarning() (string, bool) {
	var dst []wmiPredict
	if err := wmi.QueryNamespace("SELECT InstanceName, PredictFailure FROM MSStorageDriver_FailurePredictStatus", &dst, `root\wmi`); err != nil {
		return "", false
	}
	for _, d := range dst {
		if d.PredictFailure {
			return d.InstanceName, true
		}
	}
	return "", false
}

// DiskHealth 物理磁盘健康状态。
type DiskHealth struct {
	Name   string
	Status string
}

type wmiPhysDisk struct {
	FriendlyName string
	HealthStatus uint32
}

// PhysicalDisks 返回物理磁盘健康信息（NVMe/SSD，需要管理员或 Storage 权限）。
func PhysicalDisks() []DiskHealth {
	var dst []wmiPhysDisk
	if err := wmi.QueryNamespace("SELECT FriendlyName, HealthStatus FROM MSFT_PhysicalDisk", &dst, `root\Microsoft\Windows\Storage`); err != nil {
		return nil
	}
	out := make([]DiskHealth, 0, len(dst))
	for _, d := range dst {
		status := "unknown"
		switch d.HealthStatus {
		case 0:
			status = "healthy"
		case 1:
			status = "warning"
		case 2:
			status = "unhealthy"
		}
		out = append(out, DiskHealth{Name: d.FriendlyName, Status: status})
	}
	return out
}
