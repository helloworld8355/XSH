package builtins

import (
	"fmt"
	"os/user"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/host"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/sysinfo"
	"xsh/internal/ui"
	"xsh/internal/winutil"
)

func registerMisc(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "echo", Summary: "输出文本", Fn: cmdEcho})
	reg.Add(&shell.Command{Name: "date", Aliases: []string{"Get-Date"}, Summary: "当前时间", Fn: cmdDate})
	reg.Add(&shell.Command{Name: "whoami", Summary: "当前用户", Fn: cmdWhoami})
	reg.Add(&shell.Command{Name: "uptime", Summary: "运行时长", Fn: cmdUptime})
	reg.Add(&shell.Command{Name: "admin", Summary: "是否管理员", Fn: cmdAdmin})
	reg.Add(&shell.Command{Name: "clear", Aliases: []string{"cls"}, Summary: "清屏", Fn: cmdClear})
	reg.Add(&shell.Command{Name: "history", Summary: "历史命令", Fn: cmdHistory})
	reg.Add(&shell.Command{Name: "run", Summary: "执行脚本 <file.xsh>", Fn: cmdRun})
	reg.Add(&shell.Command{Name: "help", Aliases: []string{"Get-Help", "?"}, Summary: "命令帮助", Fn: cmdHelp})
}

func cmdEcho(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	text := strings.Join(args, " ")
	return []pipeline.Record{{"line": text}}, pipeline.LineSchema(), nil
}

func cmdDate(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return []pipeline.Record{{"now": time.Now().Format("2006-01-02 15:04:05")}}, pipeline.NewSchema("now"), nil
}

func cmdWhoami(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	u, err := user.Current()
	name := "?"
	if err == nil {
		name = u.Username
	}
	elevated := "否"
	if winutil.IsElevated() {
		elevated = "是"
	}
	rec := pipeline.Record{"user": name, "elevated": elevated}
	return []pipeline.Record{rec}, pipeline.NewSchema("user", "elevated"), nil
}

func cmdUptime(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	up, err := host.Uptime()
	if err != nil {
		return nil, nil, err
	}
	return []pipeline.Record{{"uptime": sysinfo.HumanDuration(up)}}, pipeline.NewSchema("uptime"), nil
}

func cmdAdmin(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	v := "false"
	if winutil.IsElevated() {
		v = "true"
	}
	return []pipeline.Record{{"elevated": v}}, pipeline.NewSchema("elevated"), nil
}

func cmdClear(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	fmt.Fprint(c.Out, "\x1b[2J\x1b[H")
	return nil, nil, nil
}

func cmdHistory(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	if c.HistFile == "" {
		return nil, nil, nil
	}
	lines, err := readFileLines(c.HistFile, 0)
	if err != nil {
		return nil, nil, nil
	}
	var recs []pipeline.Record
	for _, ln := range lines {
		recs = append(recs, pipeline.Record{"line": ln})
	}
	return recs, pipeline.LineSchema(), nil
}

func cmdRun(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	if c.RunDepth >= 5 {
		return nil, nil, fmt.Errorf("脚本嵌套过深（最多 5 层）")
	}
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("run", "<脚本.xsh>")
	}
	lines, err := readFileLines(c.Resolve(rest[0]), 0)
	if err != nil {
		return nil, nil, err
	}
	sub := *c
	sub.RunDepth++
	for i, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if err := c.Registry.Execute(ln, &sub); err != nil {
			return nil, nil, fmt.Errorf("%s 第 %d 行执行失败: %w", rest[0], i+1, err)
		}
	}
	return nil, nil, nil
}

func cmdHelp(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	rows := make([][]string, 0, 64)
	for _, cmd := range c.Registry.Commands() {
		rows = append(rows, []string{cmd.Name, cmd.Summary})
	}
	ui.PrintTable(c.Out, []string{"命令", "说明"}, rows)
	ui.Dim(c.Out, "别名可用 Tab 补全查看；管道: cmd1 | cmd2 | ...")
	return nil, nil, nil
}
