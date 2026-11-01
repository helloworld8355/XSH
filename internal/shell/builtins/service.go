package builtins

import (
	"fmt"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/sysinfo"
	"xsh/internal/winutil"
)

func registerService(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "svc", Aliases: []string{"Get-Service"}, Summary: "服务列表 [名称过滤]", Fn: cmdSvc})
	reg.Add(&shell.Command{Name: "svc-start", Summary: "启动服务 <名称>（需管理员）", Fn: cmdSvcStart})
	reg.Add(&shell.Command{Name: "svc-stop", Summary: "停止服务 <名称>（需管理员）", Fn: cmdSvcStop})
}

func cmdSvc(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	list, err := sysinfo.ListServices()
	if err != nil {
		return nil, nil, err
	}
	filter := strings.ToLower(first(rest))
	var recs []pipeline.Record
	for _, s := range list {
		if filter != "" && !strings.Contains(strings.ToLower(s.Name), filter) && !strings.Contains(strings.ToLower(s.Display), filter) {
			continue
		}
		recs = append(recs, pipeline.Record{"name": s.Name, "display": s.Display, "state": s.State})
	}
	sc := pipeline.NewSchema("name", "display", "state")
	return recs, sc, nil
}

func cmdSvcStart(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return serviceControl(c, args, "start")
}

func cmdSvcStop(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return serviceControl(c, args, "stop")
}

func serviceControl(c *shell.Ctx, args []string, op string) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr(op, "<服务名>")
	}
	if !winutil.IsElevated() {
		return nil, nil, fmt.Errorf("需要管理员权限（以管理员身份运行 xsh）")
	}
	var errs []string
	for _, name := range rest {
		var err error
		if op == "start" {
			err = sysinfo.ServiceStart(name)
		} else {
			err = sysinfo.ServiceStop(name)
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil, nil, nil
}
