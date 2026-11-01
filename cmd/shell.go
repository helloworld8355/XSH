package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v3/host"

	"xsh/internal/shell"
	"xsh/internal/shell/builtins"
	"xsh/internal/sysinfo"
	"xsh/internal/ui"
	"xsh/internal/winutil"
)

func newRegistry() *shell.Registry {
	reg := shell.NewRegistry()
	builtins.Register(reg)
	return reg
}

// runShell 启动交互式 REPL。
func runShell() error {
	ui.InitConsole()
	style, err := ui.ParseBannerStyle(bannerStyle)
	if err != nil {
		return err
	}
	bannerStyle = style
	return shell.RunREPL(newRegistry(), bannerInfo)
}

// runEval 执行单条管道命令（非交互）。
func runEval(line string) error {
	ui.InitConsole()
	reg := newRegistry()
	cwd, _ := os.Getwd()
	ctx := &shell.Ctx{Cwd: cwd, Out: os.Stdout, Registry: reg}
	return reg.Execute(line, ctx)
}

// bannerInfo 渲染启动横幅与系统摘要。
func bannerInfo(w io.Writer) {
	ui.Banner(w, bannerStyle)
	hn, _ := os.Hostname()
	ui.Field(w, "主机名", hn)
	if m := sysinfo.CPUModel(); m != "" {
		ui.Field(w, "处理器", m)
	}
	if ips := sysinfo.LANIPv4s(); len(ips) > 0 {
		ui.Field(w, "内网地址", strings.Join(ips, ", "))
	}
	if hi, err := host.Info(); err == nil {
		ui.Field(w, "运行时长", sysinfo.HumanDuration(hi.Uptime))
	}
	if winutil.IsElevated() {
		ui.Warn(w, "当前为管理员权限，请谨慎执行系统修改类命令")
	}
	fmt.Fprintln(w)
	ui.Dim(w, "输入 help 查看命令列表 · Tab 补全 · exit 退出")
}
