package builtins

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/toolset"
	"xsh/internal/ui"
	"xsh/internal/winutil"
)

func registerTools(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "fetch", Summary: "多线程下载 <URL> [-o 文件] [-t 线程] [-k]", Fn: cmdFetch})
	reg.Add(&shell.Command{Name: "hash", Summary: "哈希 <md5|sha1|sha256|sha512> [文件|文本]", Fn: cmdHash})
	reg.Add(&shell.Command{Name: "enc", Summary: "编码 <base64|base64url|hex|url> <文本>", Fn: cmdEnc})
	reg.Add(&shell.Command{Name: "dec", Summary: "解码 <base64|base64url|hex|url> <文本>", Fn: cmdDec})
	reg.Add(&shell.Command{Name: "clip", Summary: "剪贴板（无参数读取 / 带参数或管道写入）", Fn: cmdClip})
	reg.Add(&shell.Command{Name: "serve", Summary: "静态文件服务 [目录] [-p 端口]", Fn: cmdServe})
	reg.Add(&shell.Command{Name: "wifi", Summary: "Wi-Fi 状态 [profiles 列出配置]", Fn: cmdWifi})
	reg.Add(&shell.Command{Name: "wifi-pw", Summary: "Wi-Fi 密码 [配置名]（需管理员）", Fn: cmdWifiPw})
	reg.Add(&shell.Command{Name: "startup", Summary: "启动项 [add|remove|backup|restore ...]", Fn: cmdStartup})
	reg.Add(&shell.Command{Name: "snapshot", Summary: "系统快照 [-d 保存目录]", Fn: cmdSnapshot})
}

func cmdFetch(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"t": true, "o": true})
	rawURL := first(rest)
	if rawURL == "" {
		return argErr("fetch", "<URL> [-o 文件] [-t 线程] [-k]")
	}
	opt := toolset.FetchOptions{
		URL:      rawURL,
		Dest:     flags["o"],
		Threads:  atoi(flags["t"], 4),
		Insecure: flags["k"] != "",
		Progress: true,
		Out:      c.Out,
	}
	if _, err := toolset.Fetch(opt); err != nil {
		return nil, nil, err
	}
	return nil, nil, nil
}

func cmdHash(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("hash", "<md5|sha1|sha256|sha512> [文件|文本]")
	}
	algo := rest[0]
	if len(rest) > 1 {
		target := c.Resolve(rest[1])
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			sum, err := toolset.HashFile(target, algo)
			if err != nil {
				return nil, nil, err
			}
			return []pipeline.Record{{"algorithm": algo, "hash": sum, "target": rest[1]}},
				pipeline.NewSchema("algorithm", "hash", "target"), nil
		}
		sum, err := toolset.HashString(strings.Join(rest[1:], " "), algo)
		if err != nil {
			return nil, nil, err
		}
		return []pipeline.Record{{"algorithm": algo, "hash": sum}},
			pipeline.NewSchema("algorithm", "hash"), nil
	}
	if len(in) > 0 {
		text, _ := pipedText(in)
		sum, err := toolset.HashString(text, algo)
		if err != nil {
			return nil, nil, err
		}
		return []pipeline.Record{{"algorithm": algo, "hash": sum}},
			pipeline.NewSchema("algorithm", "hash"), nil
	}
	return argErr("hash", "<md5|sha1|sha256|sha512> [文件|文本]")
}

func cmdEnc(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return codecCmd(c, args, in, true)
}

func cmdDec(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return codecCmd(c, args, in, false)
}

// codecCmd enc/dec 共用实现。
func codecCmd(c *shell.Ctx, args []string, in []pipeline.Record, enc bool) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	name := "enc"
	if !enc {
		name = "dec"
	}
	if len(rest) == 0 {
		return argErr(name, "<base64|base64url|hex|url> <文本>")
	}
	algo := rest[0]
	text := strings.Join(rest[1:], " ")
	if text == "" && len(in) > 0 {
		text, _ = pipedText(in)
	}
	var (
		out string
		err error
	)
	if enc {
		out, err = toolset.Encode(text, algo)
	} else {
		out, err = toolset.Decode(text, algo)
	}
	if err != nil {
		return nil, nil, err
	}
	return []pipeline.Record{{"line": out}}, pipeline.LineSchema(), nil
}

func cmdClip(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 && len(in) == 0 {
		text, err := toolset.ClipGet()
		if err != nil {
			return nil, nil, err
		}
		return []pipeline.Record{{"line": text}}, pipeline.LineSchema(), nil
	}
	text := strings.Join(rest, " ")
	if text == "" && len(in) > 0 {
		var sb strings.Builder
		for _, r := range in {
			sb.WriteString(r["line"])
			sb.WriteString("\n")
		}
		text = sb.String()
	}
	if err := toolset.ClipSet(text); err != nil {
		return nil, nil, err
	}
	ui.OK(c.Out, "已复制 %d 字符到剪贴板", len([]rune(text)))
	return nil, nil, nil
}

func cmdServe(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"p": true})
	dir := c.Resolve(first(rest))
	err := toolset.Serve(dir, atoi(flags["p"], 8000), c.Out)
	return nil, nil, err
}

func cmdWifi(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		kvs, err := toolset.WifiInfo()
		if err != nil {
			return nil, nil, err
		}
		var recs []pipeline.Record
		for _, kv := range kvs {
			recs = append(recs, pipeline.Record{"key": kv.K, "value": kv.V})
		}
		return recs, pipeline.NewSchema("key", "value"), nil
	}
	if strings.EqualFold(rest[0], "profiles") || strings.EqualFold(rest[0], "list") {
		return wifiProfileRecords()
	}
	return argErr("wifi", "[profiles|list]")
}

func wifiProfileRecords() ([]pipeline.Record, *pipeline.Schema, error) {
	names, err := toolset.WifiProfiles()
	if err != nil {
		return nil, nil, err
	}
	var recs []pipeline.Record
	for i, n := range names {
		recs = append(recs, pipeline.Record{"idx": strconv.Itoa(i + 1), "profile": n})
	}
	return recs, pipeline.NewSchema("idx", "profile").WithNumeric("idx"), nil
}

func cmdWifiPw(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	name := first(rest)
	if name == "" {
		if !winutil.IsElevated() {
			ui.Warn(c.Out, "提示: 读取明文密钥需要管理员权限")
		}
		return wifiProfileRecords()
	}
	if !winutil.IsElevated() {
		ui.Warn(c.Out, "提示: 通常需要管理员权限才能读取明文密钥")
	}
	auth, pw, err := toolset.WifiPassword(name)
	if err != nil {
		return nil, nil, err
	}
	if pw == "" {
		pw = "(开放网络/无密钥)"
	}
	return []pipeline.Record{{"profile": name, "auth": auth, "password": pw}},
		pipeline.NewSchema("profile", "auth", "password"), nil
}

func cmdStartup(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	switch strings.ToLower(first(rest)) {
	case "", "list":
		items, err := toolset.StartupList()
		if err != nil {
			return nil, nil, err
		}
		var recs []pipeline.Record
		for _, it := range items {
			recs = append(recs, pipeline.Record{"source": it.Source, "name": it.Name, "command": it.Command})
		}
		return recs, pipeline.NewSchema("source", "name", "command"), nil
	case "add":
		if len(rest) < 4 {
			return argErr("startup", "add <hkcu|hklm> <名称> <命令...>")
		}
		if err := toolset.StartupAdd(strings.ToLower(rest[1]), rest[2], strings.Join(rest[3:], " ")); err != nil {
			return nil, nil, err
		}
		ui.OK(c.Out, "已添加启动项 %s", rest[2])
		return nil, nil, nil
	case "remove", "rm":
		if len(rest) < 3 {
			return argErr("startup", "remove <hkcu|hklm> <名称>")
		}
		if err := toolset.StartupRemove(strings.ToLower(rest[1]), rest[2]); err != nil {
			return nil, nil, err
		}
		ui.OK(c.Out, "已删除启动项 %s", rest[2])
		return nil, nil, nil
	case "backup":
		if len(rest) < 2 {
			return argErr("startup", "backup <文件.json>")
		}
		n, err := toolset.StartupBackup(c.Resolve(rest[1]))
		if err != nil {
			return nil, nil, err
		}
		ui.OK(c.Out, "已备份 %d 个启动项 → %s", n, rest[1])
		return nil, nil, nil
	case "restore":
		if len(rest) < 2 {
			return argErr("startup", "restore <文件.json>")
		}
		n, err := toolset.StartupRestore(c.Resolve(rest[1]))
		if err != nil {
			return nil, nil, err
		}
		ui.OK(c.Out, "已恢复 %d 个启动项", n)
		return nil, nil, nil
	}
	return argErr("startup", "[list|add|remove|backup|restore ...]")
}

func cmdSnapshot(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, _ := parseArgs(args, map[string]bool{"d": true})
	dir := flags["d"]
	if dir == "" {
		dir = filepath.Join(os.Getenv("USERPROFILE"), ".xsh", "snapshots")
	}
	snap, err := toolset.TakeSnapshot()
	if err != nil {
		return nil, nil, err
	}
	path, err := toolset.SaveSnapshot(snap, c.Resolve(dir))
	if err != nil {
		return nil, nil, err
	}
	ui.OK(c.Out, "快照已保存 → %s", path)
	ui.Field(c.Out, "主机", snap.Host)
	ui.Field(c.Out, "系统", snap.OS)
	ui.Field(c.Out, "运行时长", snap.Uptime)
	ui.Field(c.Out, "内存", fmt.Sprintf("%s / %s (%.0f%%)",
		ui.HumanBytes(float64(snap.MemUsed)), ui.HumanBytes(float64(snap.MemTotal)), snap.MemPercent))
	for _, d := range snap.Disks {
		ui.Field(c.Out, "磁盘 "+d.Mount, fmt.Sprintf("%.1fGB / %.1fGB (%.0f%%)", d.UsedGB, d.TotalGB, d.Percent))
	}
	return nil, nil, nil
}

// pipedText 将管道输入记录拼为文本（保留行结构，无尾换行）。
func pipedText(in []pipeline.Record) (string, bool) {
	if len(in) == 0 {
		return "", false
	}
	var sb strings.Builder
	for _, r := range in {
		sb.WriteString(r["line"])
		sb.WriteString("\n")
	}
	return strings.TrimSuffix(sb.String(), "\n"), true
}
