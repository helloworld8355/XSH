package diag

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/yusufpapurcu/wmi"

	"xsh/internal/sysinfo"
	"xsh/internal/toolset"
	"xsh/internal/ui"
)

type wmiDiskPerf struct {
	Name                  string
	AvgDiskSecPerTransfer uint64
	PercentDiskTime       uint64
}

// Disk 磁盘诊断：分区用量、SMART 健康、磁盘响应延迟。
func (r *Runner) Disk() error {
	ui.Title(r.Out, "磁盘诊断")
	var problems []string

	parts, _ := disk.Partitions(false)
	rows := make([][]string, 0, len(parts))
	for _, p := range parts {
		du, err := disk.Usage(p.Mountpoint)
		if err != nil || du.Total == 0 {
			continue
		}
		rows = append(rows, []string{
			p.Mountpoint,
			fmt.Sprintf("%.0f%%", du.UsedPercent),
			ui.HumanBytes(float64(du.Free)) + " / " + ui.HumanBytes(float64(du.Total)),
		})
		if du.UsedPercent >= 95 {
			problems = append(problems, p.Mountpoint+" 剩余空间不足 5%")
		}
	}
	ui.PrintTable(r.Out, []string{"卷", "使用率", "剩余/总量"}, rows)

	if msg, ok := sysinfo.SmartWarning(); ok {
		ui.Err(r.Out, "SMART 故障预测: %s（请立即备份数据）", msg)
		problems = append(problems, "SMART 预测磁盘故障")
	}
	for _, d := range sysinfo.PhysicalDisks() {
		if d.Status != "healthy" {
			ui.Err(r.Out, "物理磁盘 %s 状态: %s", d.Name, d.Status)
			problems = append(problems, "磁盘 "+d.Name+" 状态异常")
		} else {
			ui.OK(r.Out, "物理磁盘 %s 健康", d.Name)
		}
	}

	var perfs []wmiDiskPerf
	if err := wmi.QueryNamespace("SELECT Name, AvgDiskSecPerTransfer, PercentDiskTime FROM Win32_PerfFormattedData_PerfDisk_PhysicalDisk WHERE Name != '_Total'", &perfs, `root\cimv2`); err == nil {
		prows := make([][]string, 0, len(perfs))
		for _, p := range perfs {
			ms := float64(p.AvgDiskSecPerTransfer) / 10000 // 100ns → ms
			prows = append(prows, []string{p.Name, fmt.Sprintf("%.2fms", ms), fmt.Sprintf("%d%%", p.PercentDiskTime)})
			if ms > 60 {
				problems = append(problems, fmt.Sprintf("%s 平均 IO 延迟 %.0fms 过高（机械盘满载或硬盘降速）", p.Name, ms))
			}
		}
		if len(prows) > 0 {
			ui.Hdr(r.Out, "磁盘 IO")
			ui.PrintTable(r.Out, []string{"磁盘", "平均 IO 延迟", "忙碌率"}, prows)
		}
	}

	dumps := filepath.Join(os.Getenv("WINDIR"), "Minidump")
	if ents, err := os.ReadDir(dumps); err == nil && len(ents) > 0 {
		ui.Warn(r.Out, "存在 %d 个蓝屏转储（xsh diag bsod 查看详情）", len(ents))
	}

	ui.Sep(r.Out, 60)
	if len(problems) == 0 {
		ui.OK(r.Out, "磁盘状态正常")
	} else {
		ui.Warn(r.Out, "发现 %d 个问题:", len(problems))
		for i, p := range problems {
			fmt.Fprintf(r.Out, "  %d. %s\n", i+1, p)
		}
	}
	return nil
}

// Startup 开机启动诊断。
func (r *Runner) Startup() error {
	ui.Title(r.Out, "开机启动诊断")
	items, err := toolset.StartupList()
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Source, it.Name, it.Command})
	}
	if len(rows) > 0 {
		ui.PrintTable(r.Out, []string{"来源", "名称", "命令"}, rows)
	}
	ui.Info(r.Out, "共 %d 个启动项", len(items))
	if len(items) > 25 {
		ui.Warn(r.Out, "启动项过多影响开机速度；用 `xsh startup remove` 清理不需要的项")
	}
	return nil
}
