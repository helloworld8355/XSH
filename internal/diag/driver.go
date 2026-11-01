package diag

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strings"

	"xsh/internal/ui"
)

// Driver 驱动诊断：driverquery 解析，筛出异常驱动。
func (r *Runner) Driver() error {
	ui.Title(r.Out, "驱动诊断")
	out, err := exec.Command("driverquery", "/fo", "csv").Output()
	if err != nil {
		return fmt.Errorf("driverquery 执行失败: %w", err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(out))).ReadAll()
	if err != nil || len(rows) < 2 {
		return fmt.Errorf("driverquery 输出解析失败")
	}
	// 列顺序: 模块名,显示名,驱动类型,启动模式,状态,状态(默认),接受暂停,接受停止...
	var normal, abnormal int
	var abRows [][]string
	for _, row := range rows[1:] {
		if len(row) < 5 {
			continue
		}
		name, display := row[0], row[1]
		startMode, state := row[3], row[4]
		normal++
		// 系统/自动启动但未运行 → 疑似异常；已禁用的系统驱动跳过
		if (startMode == "系统" || startMode == "System" || startMode == "自动" || startMode == "Auto") &&
			(state != "正在运行" && state != "Running") {
			abnormal++
			if len(abRows) < 20 {
				abRows = append(abRows, []string{name, display, startMode, state})
			}
		}
	}
	ui.Info(r.Out, "共 %d 个驱动", len(rows)-1)
	if len(abRows) > 0 {
		ui.Warn(r.Out, "以下 %d 个驱动设置为自动/系统启动但未运行:", abnormal)
		ui.PrintTable(r.Out, []string{"模块", "名称", "启动模式", "状态"}, abRows)
		ui.Info(r.Out, "提示: 部分服务按需启动属正常现象；频繁蓝屏时优先排查这些驱动")
	} else {
		ui.OK(r.Out, "所有自动启动的驱动均正常运行")
	}
	ui.Info(r.Out, "更新驱动建议: 优先使用设备管理器/厂商官网，避免第三方驱动管家")
	return nil
}
