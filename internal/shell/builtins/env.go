package builtins

import (
	"os"
	"sort"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/ui"
)

func registerEnv(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "env", Aliases: []string{"Get-ChildItem-Env"}, Summary: "环境变量列表", Fn: cmdEnv})
	reg.Add(&shell.Command{Name: "genv", Summary: "读取环境变量 <名称>", Fn: cmdGEnv})
	reg.Add(&shell.Command{Name: "senv", Summary: "设置环境变量 <名称> <值> [--persist]", Fn: cmdSEnv})
}

func cmdEnv(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	env := os.Environ()
	sort.Strings(env)
	var recs []pipeline.Record
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		recs = append(recs, pipeline.Record{"name": k, "value": v})
	}
	return recs, pipeline.NewSchema("name", "value"), nil
}

func cmdGEnv(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("genv", "<名称>")
	}
	return []pipeline.Record{{rest[0]: os.Getenv(rest[0])}}, pipeline.NewSchema(rest[0]), nil
}

func cmdSEnv(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, nil)
	if len(rest) < 2 {
		return argErr("senv", "<名称> <值> [--persist]")
	}
	if err := os.Setenv(rest[0], rest[1]); err != nil {
		return nil, nil, err
	}
	if flags["persist"] != "" {
		k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
		if err != nil {
			return nil, nil, err
		}
		defer k.Close()
		if err := k.SetStringValue(rest[0], rest[1]); err != nil {
			return nil, nil, err
		}
		broadcastEnvChange()
		ui.OK(c.Out, "已持久化 %s（新进程生效）", rest[0])
	}
	return nil, nil, nil
}

// broadcastEnvChange 广播 WM_SETTINGCHANGE，使资源管理器感知环境变量变化。
func broadcastEnvChange() {
	user32 := syscall.NewLazyDLL("user32.dll")
	send := user32.NewProc("SendMessageTimeoutW")
	envStr, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	var res uintptr
	send.Call(0xFFFF, 0x1A, 0, uintptr(unsafe.Pointer(envStr)), 0x2, 5000, uintptr(unsafe.Pointer(&res)))
}
