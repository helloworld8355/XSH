// Package diag 提供快速故障排查：交互向导、网络、磁盘、性能、事件、蓝屏、驱动诊断。
package diag

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"xsh/internal/ui"
)

// Runner 诊断器。
type Runner struct {
	Out io.Writer
}

// New 创建诊断器。
func New(out io.Writer) *Runner {
	return &Runner{Out: out}
}

// Run 按模块名执行诊断。
func (r *Runner) Run(module string) error {
	switch strings.ToLower(module) {
	case "", "wizard":
		return r.Wizard()
	case "net", "network":
		return r.Net()
	case "disk":
		return r.Disk()
	case "perf":
		return r.Perf()
	case "event":
		return r.Event()
	case "bsod":
		return r.BSOD()
	case "driver":
		return r.Driver()
	case "boot", "startup":
		return r.Startup()
	}
	return fmt.Errorf("未知诊断模块 %q（可用: wizard net disk perf event bsod driver boot）", module)
}

// Wizard 交互式排查向导。
func (r *Runner) Wizard() error {
	in := bufio.NewReader(os.Stdin)
	ui.Title(r.Out, "快速排查向导")
	ui.Info(r.Out, "请选择遇到的问题:")
	fmt.Fprintln(r.Out, "  1. 上网慢 / 断网")
	fmt.Fprintln(r.Out, "  2. 电脑卡顿 / 发热")
	fmt.Fprintln(r.Out, "  3. 磁盘 / 文件异常")
	fmt.Fprintln(r.Out, "  4. 蓝屏 / 崩溃")
	fmt.Fprintln(r.Out, "  5. 开机慢")
	fmt.Fprintln(r.Out, "  0. 退出")
	switch ask(in, r.Out, "选择> ") {
	case "1":
		return r.Net()
	case "2":
		if err := r.Perf(); err != nil {
			return err
		}
		return r.Disk()
	case "3":
		return r.Disk()
	case "4":
		if err := r.BSOD(); err != nil {
			return err
		}
		return r.Event()
	case "5":
		return r.Startup()
	case "0", "exit", "q", "":
		return nil
	}
	ui.Warn(r.Out, "无效选择，重新运行 xsh diag")
	return nil
}

func ask(in *bufio.Reader, out io.Writer, prompt string) string {
	fmt.Fprint(out, prompt)
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimSpace(line)
}
