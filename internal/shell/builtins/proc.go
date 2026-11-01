package builtins

import (
	"fmt"
	"strconv"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/sysinfo"
)

func registerProc(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "ps", Aliases: []string{"Get-Process"}, Summary: "进程列表 [名称过滤]", Fn: cmdPS})
	reg.Add(&shell.Command{Name: "kill", Aliases: []string{"Stop-Process"}, Summary: "结束进程 <pid...>", Fn: cmdKill})
}

func cmdPS(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	procs, err := sysinfo.ProcSnapshot(300 * 1e6) // 300ms
	if err != nil {
		return nil, nil, err
	}
	filter := strings.ToLower(first(rest))
	var recs []pipeline.Record
	for _, p := range procs {
		if filter != "" && !strings.Contains(strings.ToLower(p.Name), filter) && strconv.Itoa(int(p.PID)) != filter {
			continue
		}
		recs = append(recs, pipeline.Record{
			"name": p.Name,
			"pid":  strconv.Itoa(int(p.PID)),
			"cpu":  pipeline.TrimF(p.CPU),
			"mem":  pipeline.TrimF(p.MemMB),
		})
	}
	sc := pipeline.NewSchema("name", "pid", "cpu", "mem").WithNumeric("pid", "cpu", "mem")
	return recs, sc, nil
}

func cmdKill(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("kill", "<pid...>")
	}
	var errs []string
	for _, s := range rest {
		pid, err := strconv.Atoi(s)
		if err != nil {
			errs = append(errs, fmt.Sprintf("无效 PID %q", s))
			continue
		}
		if err := sysinfo.KillPid(int32(pid)); err != nil {
			errs = append(errs, fmt.Sprintf("kill %d: %v", pid, err))
		}
	}
	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil, nil, nil
}
