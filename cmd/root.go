// Package cmd XSH 命令树（cobra）：裸 xsh 进交互 shell，带子命令走单命令模式。
package cmd

import (
	"github.com/spf13/cobra"

	"xsh/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "xsh [命令]",
	Short: "玄枢 XSH — Windows 多合一 CLI 工具",
	Long: `玄枢 XSH — Windows 多合一 CLI 工具

直接运行 xsh 进入交互式 shell；带子命令则以单命令模式执行；
xsh -e "<命令>" 执行单条管道命令后退出。

核心命令:
  check    系统健康体检（14 项检查 + 综合评分）
  diag     故障排查（向导/网络/磁盘/性能/事件/蓝屏/驱动/开机）
  dev      开发环境体检（57 工具链探测 + PATH 审计）
  sec      授权安全评估（需 license + --authorized，全程审计）
  license  校验并显示授权文件
  version  版本信息

工具命令:
  fetch hash enc dec clip serve wifi startup snapshot`,
	Version: ui.Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		if evalLine != "" {
			return runEval(evalLine)
		}
		return runShell()
	},
	SilenceUsage: true,
}

// Execute 执行根命令。
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&evalLine, "eval", "e", "", "执行单条管道命令后退出")
	rootCmd.AddCommand(checkCmd, diagCmd, devCmd, secCmd, licenseCmd, versionCmd)
	rootCmd.AddCommand(toolCmds...)
}

var evalLine string
