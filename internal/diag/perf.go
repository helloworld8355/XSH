package diag

import (
	"fmt"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"xsh/internal/sysinfo"
	"xsh/internal/ui"
)

// Perf 性能诊断：CPU/内存概况 + Top 进程。
func (r *Runner) Perf() error {
	ui.Title(r.Out, "性能诊断")
	var problems []string

	pcts, err := cpu.Percent(time.Second, false)
	if err == nil && len(pcts) > 0 {
		p := pcts[0]
		ui.Field(r.Out, "CPU 占用", fmt.Sprintf("%.0f%%", p))
		if p >= 90 {
			problems = append(problems, "CPU 接近满载")
		}
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		ui.Field(r.Out, "内存占用", fmt.Sprintf("%.1fGB / %.1fGB (%.0f%%)",
			float64(vm.Used)/(1<<30), float64(vm.Total)/(1<<30), vm.UsedPercent))
		if vm.UsedPercent >= 95 {
			problems = append(problems, "内存接近耗尽")
		}
	}

	ui.Hdr(r.Out, "Top 进程 (CPU)")
	procs, err := sysinfo.ProcSnapshot(500 * time.Millisecond)
	if err != nil {
		return err
	}
	cpuRows := procRows(procs, 10, false)
	ui.PrintTable(r.Out, []string{"PID", "名称", "CPU%", "内存MB"}, cpuRows)

	sort.Slice(procs, func(i, j int) bool { return procs[i].MemMB > procs[j].MemMB })
	ui.Hdr(r.Out, "Top 进程 (内存)")
	ui.PrintTable(r.Out, []string{"PID", "名称", "CPU%", "内存MB"}, procRows(procs, 10, true))

	ui.Sep(r.Out, 60)
	if len(problems) == 0 {
		ui.OK(r.Out, "资源占用正常")
	} else {
		ui.Warn(r.Out, "发现 %d 个问题:", len(problems))
		for i, p := range problems {
			fmt.Fprintf(r.Out, "  %d. %s\n", i+1, p)
		}
		ui.Info(r.Out, "提示: 高占用进程可用 `xsh` 进入交互模式执行 kill <pid>")
	}
	return nil
}

func procRows(procs []sysinfo.ProcInfo, n int, renumber bool) [][]string {
	if len(procs) > n {
		procs = procs[:n]
	}
	rows := make([][]string, 0, len(procs))
	for _, p := range procs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", p.PID),
			p.Name,
			fmt.Sprintf("%.1f", p.CPU),
			fmt.Sprintf("%.0f", p.MemMB),
		})
	}
	_ = renumber
	return rows
}
