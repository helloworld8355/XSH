package sysinfo

import (
	"runtime"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func runtimeNumCPU() int { return runtime.NumCPU() }

func procTimes() (map[int32]float64, error) {
	ps, err := process.Processes()
	if err != nil {
		return nil, err
	}
	m := make(map[int32]float64, len(ps))
	for _, p := range ps {
		t, err := p.Times()
		if err != nil {
			continue
		}
		m[p.Pid] = t.Total()
	}
	return m, nil
}

func probePid(pid int32) *ProcInfo {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil
	}
	name, err := p.Name()
	if err != nil {
		name = ""
	}
	mem := 0.0
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		mem = float64(mi.RSS) / 1048576
	}
	return &ProcInfo{PID: pid, Name: name, MemMB: mem}
}

// KillPid 结束进程。
func KillPid(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// SvcInfo Windows 服务摘要。
type SvcInfo struct {
	Name    string
	Display string
	State   string
}

type wmiService struct {
	Name        string
	DisplayName string
	State       string
}

// ListServices 用 WMI 列出全部服务（无需管理员）。
func ListServices() ([]SvcInfo, error) {
	var dst []wmiService
	if err := wmi.Query("SELECT Name, DisplayName, State FROM Win32_Service", &dst); err != nil {
		return nil, err
	}
	out := make([]SvcInfo, 0, len(dst))
	for _, s := range dst {
		out = append(out, SvcInfo{
			Name:    s.Name,
			Display: s.DisplayName,
			State:   strings.ToLower(s.State),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ServiceStart 启动服务（需管理员）。
func ServiceStart(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

// ServiceStop 停止服务（需管理员）。
func ServiceStop(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}
