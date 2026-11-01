package devenv

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"xsh/internal/ui"
)

// PathAudit PATH 审计结果。
type PathAudit struct {
	Total      int
	Duplicates []string
	Missing    []string
	Warnings   []string
}

// AuditPATH 审计 PATH 环境变量：重复、失效目录、长度风险。
func AuditPATH() *PathAudit {
	raw := os.Getenv("PATH")
	a := &PathAudit{}
	entries := filepath.SplitList(raw)
	a.Total = len(entries)
	seen := map[string]int{}
	for _, e := range entries {
		if e == "" {
			continue
		}
		k := strings.ToLower(strings.TrimSuffix(e, `\`))
		seen[k]++
		if seen[k] == 2 {
			a.Duplicates = append(a.Duplicates, e)
		}
		if fi, err := os.Stat(e); err != nil || !fi.IsDir() {
			a.Missing = append(a.Missing, e)
		}
	}
	if len(raw) > 2048 {
		a.Warnings = append(a.Warnings, fmt.Sprintf("PATH 长度 %d 接近系统上限，存在被截断风险", len(raw)))
	}
	if a.Total > 40 {
		a.Warnings = append(a.Warnings, fmt.Sprintf("PATH 条目多达 %d 项，命令查找性能下降", a.Total))
	}
	if len(a.Missing) > 5 {
		a.Warnings = append(a.Warnings, fmt.Sprintf("存在 %d 个不存在的目录（多为卸载残留）", len(a.Missing)))
	}
	return a
}

// Report 探测并渲染开发环境体检报告。
func Report(w io.Writer) error {
	ui.Title(w, "开发环境体检")
	ui.Info(w, "正在探测 %d 个工具链…", len(detectors))
	tools := Detect()
	audit := AuditPATH()

	found := 0
	rows := make([][]string, 0, len(tools))
	for _, t := range tools {
		if t.Found {
			found++
			rows = append(rows, []string{t.Name, t.Version, t.Env, t.Path})
		}
	}
	ui.Hdr(w, fmt.Sprintf("已安装工具 (%d/%d)", found, len(tools)))
	ui.PrintTable(w, []string{"工具", "版本", "环境变量", "路径"}, rows)

	var missing []string
	for _, t := range tools {
		if !t.Found {
			missing = append(missing, t.Name)
		}
	}
	if len(missing) > 0 {
		ui.Dim(w, "未检测到: "+strings.Join(missing, ", "))
	}

	ui.Sep(w, 72)
	ui.Title(w, "PATH 审计")
	ui.Field(w, "条目数", fmt.Sprintf("%d", audit.Total))
	ui.Field(w, "重复条目", fmt.Sprintf("%d", len(audit.Duplicates)))
	for _, d := range audit.Duplicates {
		ui.Dim(w, "  重复: "+d)
	}
	ui.Field(w, "失效目录", fmt.Sprintf("%d", len(audit.Missing)))
	for _, m := range audit.Missing {
		ui.Dim(w, "  不存在: "+m)
	}
	if len(audit.Warnings) == 0 {
		ui.OK(w, "PATH 配置健康")
	} else {
		for _, warn := range audit.Warnings {
			ui.Warn(w, "%s", warn)
		}
	}
	return nil
}
