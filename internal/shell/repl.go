package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// RunREPL 启动交互式环境；info 为横幅输出回调。
func RunREPL(reg *Registry, info func(w io.Writer)) error {
	cwd, _ := os.Getwd()
	histDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "xsh")
	_ = os.MkdirAll(histDir, 0o755)
	histFile := filepath.Join(histDir, "history")

	ctx := &Ctx{Cwd: cwd, Out: os.Stdout, Registry: reg, Interactive: true, HistFile: histFile}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt(ctx),
		HistoryFile:     histFile,
		AutoComplete:    newCompleter(reg, ctx),
		InterruptPrompt: "\n(输入 exit 退出)\n",
		EOFPrompt:       "exit\n",
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	if info != nil {
		info(os.Stdout)
	}
	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		_ = reg.Execute(line, ctx)
		rl.SetPrompt(prompt(ctx))
	}
	return nil
}

func prompt(c *Ctx) string {
	d := c.Cwd
	if len(d) > 30 {
		d = "..." + d[len(d)-27:]
	}
	return fmt.Sprintf("\033[36;1mxsh\033[0m \033[90m%s\033[0m> ", d)
}
