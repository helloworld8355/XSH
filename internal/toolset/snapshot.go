package toolset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"xsh/internal/sysinfo"
)

// Snapshot 系统快照。
type Snapshot struct {
	Time       string      `json:"time"`
	Host       string      `json:"host"`
	OS         string      `json:"os"`
	Uptime     string      `json:"uptime"`
	CPU        string      `json:"cpu"`
	Cores      int         `json:"cores"`
	MemTotal   uint64      `json:"mem_total"`
	MemUsed    uint64      `json:"mem_used"`
	MemPercent float64     `json:"mem_percent"`
	Disks      []DiskBrief `json:"disks"`
	Gateway    string      `json:"gateway"`
	LANIPs     []string    `json:"lan_ips"`
	TopProcs   []ProcBrief `json:"top_procs"`
}

// DiskBrief 磁盘用量摘要。
type DiskBrief struct {
	Mount   string  `json:"mount"`
	TotalGB float64 `json:"total_gb"`
	UsedGB  float64 `json:"used_gb"`
	Percent float64 `json:"percent"`
}

// ProcBrief 进程摘要。
type ProcBrief struct {
	PID   int32   `json:"pid"`
	Name  string  `json:"name"`
	CPU   float64 `json:"cpu"`
	MemMB float64 `json:"mem_mb"`
}

// TakeSnapshot 采集当前系统快照（阻塞约 0.3 秒采样进程）。
func TakeSnapshot() (*Snapshot, error) {
	s := &Snapshot{
		Time:  time.Now().Format("2006-01-02 15:04:05"),
		CPU:   sysinfo.CPUModel(),
		Cores: runtime.NumCPU(),
	}
	if hn, err := os.Hostname(); err == nil {
		s.Host = hn
	}
	if hi, err := host.Info(); err == nil {
		s.OS = hi.Platform + " " + hi.PlatformVersion
		s.Uptime = sysinfo.HumanDuration(hi.Uptime)
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		s.MemTotal, s.MemUsed, s.MemPercent = vm.Total, vm.Used, vm.UsedPercent
	}
	if parts, err := disk.Partitions(false); err == nil {
		for _, p := range parts {
			du, err := disk.Usage(p.Mountpoint)
			if err != nil || du.Total == 0 {
				continue
			}
			s.Disks = append(s.Disks, DiskBrief{
				Mount:   p.Mountpoint,
				TotalGB: float64(du.Total) / (1 << 30),
				UsedGB:  float64(du.Used) / (1 << 30),
				Percent: du.UsedPercent,
			})
		}
	}
	s.Gateway = sysinfo.DefaultGateway()
	s.LANIPs = sysinfo.LANIPv4s()
	if procs, err := sysinfo.TopProcs(15); err == nil {
		for _, p := range procs {
			s.TopProcs = append(s.TopProcs, ProcBrief{PID: p.PID, Name: p.Name, CPU: p.CPU, MemMB: p.MemMB})
		}
	}
	return s, nil
}

// SaveSnapshot 保存快照 JSON，返回文件路径。
func SaveSnapshot(s *Snapshot, dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("snapshot-%s.json", time.Now().Format("20060102-150405")))
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// LoadSnapshot 读取快照文件。
func LoadSnapshot(path string) (*Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSnapshots 列出目录中的快照文件（新→旧）。
func ListSnapshots(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for i := len(ents) - 1; i >= 0; i-- {
		name := ents[i].Name()
		if strings.HasPrefix(name, "snapshot-") && strings.HasSuffix(name, ".json") {
			out = append(out, filepath.Join(dir, name))
		}
	}
	return out, nil
}
