package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"xsh/internal/devenv"
	"xsh/internal/diag"
	"xsh/internal/ui"
)

var diagCmd = &cobra.Command{
	Use:   "diag [模块]",
	Short: "快速故障排查（无参数进入交互向导）",
	Long: `快速故障排查:
  wizard  交互向导（默认）
  net     网络诊断（接口/网关/DNS/连通性）
  disk    磁盘诊断（分区/SMART/IO 延迟）
  perf    性能采样（CPU/内存/热点进程）
  event   关键系统事件（48 小时）
  bsod    蓝屏诊断（minidump/驱动/代码解释）
  driver  驱动诊断（异常驱动筛查）
  boot    开机启动诊断`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		module := ""
		if len(args) > 0 {
			module = args[0]
		}
		return diag.New(os.Stdout).Run(module)
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "开发环境体检（57 工具链探测 + PATH 审计）",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		return devenv.Report(os.Stdout)
	},
}
