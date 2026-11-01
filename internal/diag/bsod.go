package diag

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"xsh/internal/ui"
)

//go:embed assets/bsod_codes.json
var diagAssets embed.FS

// bsodInfo 蓝屏代码知识条目。
type bsodInfo struct {
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	Advice string `json:"advice"`
}

var bugcheckRe = regexp.MustCompile(`0x[0-9A-Fa-f]{6,8}`)

// BSOD 蓝屏诊断：minidump 列表 + BugCheck 事件 + 代码解释。
func (r *Runner) BSOD() error {
	ui.Title(r.Out, "蓝屏诊断")
	var problems []string

	win := os.Getenv("WINDIR")
	if win == "" {
		win = `C:\Windows`
	}
	if ents, err := os.ReadDir(filepath.Join(win, "Minidump")); err == nil {
		var rows [][]string
		for _, e := range ents {
			if !strings.HasSuffix(e.Name(), ".dmp") {
				continue
			}
			if fi, err := e.Info(); err == nil {
				rows = append(rows, []string{e.Name(), fi.ModTime().Format("2006-01-02 15:04"), ui.HumanBytes(float64(fi.Size()))})
			}
		}
		if len(rows) > 0 {
			ui.Hdr(r.Out, "Minidump 文件")
			ui.PrintTable(r.Out, []string{"文件", "时间", "大小"}, rows)
		} else {
			ui.OK(r.Out, "无 minidump 记录")
		}
	} else {
		ui.OK(r.Out, "无 minidump 目录（未发生过蓝屏或未配置小转储）")
	}
	if fi, err := os.Stat(filepath.Join(win, "MEMORY.DMP")); err == nil {
		ui.Warn(r.Out, "存在完整内存转储 MEMORY.DMP (%s, %s)",
			fi.ModTime().Format("2006-01-02 15:04"), ui.HumanBytes(float64(fi.Size())))
	}

	ps := `Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-WER-SystemErrorReporting'} -MaxEvents 5 | Select-Object TimeCreated, Message | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		type werEvt struct {
			TimeCreated string `json:"TimeCreated"`
			Message     string `json:"Message"`
		}
		var evts []werEvt
		s := strings.TrimSpace(string(out))
		if json.Unmarshal([]byte(s), &evts) != nil {
			var one werEvt
			if json.Unmarshal([]byte(s), &one) == nil {
				evts = []werEvt{one}
			}
		}
		if len(evts) > 0 {
			ui.Hdr(r.Out, "BugCheck 记录")
			for _, e := range evts {
				code := bugcheckRe.FindString(e.Message)
				fmt.Fprintf(r.Out, "  %s  代码 %s\n", evtTime(e.TimeCreated), code)
				if code != "" {
					key := maskCode(code)
					if info, ok := lookupBSOD(key); ok {
						ui.Field(r.Out, "  名称", info.Name)
						ui.Field(r.Out, "  说明", info.Desc)
						ui.Field(r.Out, "  建议", info.Advice)
					} else {
						ui.Dim(r.Out, "  知识库中无此条目，可检索微软文档: "+code)
					}
				}
				problems = append(problems, "存在蓝屏记录 "+code)
			}
		} else {
			ui.OK(r.Out, "无 BugCheck 事件记录")
		}
	} else {
		ui.OK(r.Out, "无 BugCheck 事件记录")
	}

	ui.Sep(r.Out, 60)
	if len(problems) == 0 {
		ui.OK(r.Out, "未发现蓝屏记录")
		return nil
	}
	ui.Warn(r.Out, "发现 %d 条蓝屏线索:", len(problems))
	for i, p := range problems {
		fmt.Fprintf(r.Out, "  %d. %s\n", i+1, p)
	}
	ui.Info(r.Out, "常见原因: 驱动缺陷 / 内存故障 / 超频不稳；建议运行 `xsh diag driver` 排查驱动")
	return nil
}

// lookupBSOD 按归一化代码（如 0x7B）查询知识库。
func lookupBSOD(key string) (bsodInfo, bool) {
	b, err := diagAssets.ReadFile("assets/bsod_codes.json")
	if err != nil {
		return bsodInfo{}, false
	}
	var db map[string]bsodInfo
	if json.Unmarshal(b, &db) != nil {
		return bsodInfo{}, false
	}
	info, ok := db[key]
	return info, ok
}

// maskCode 将 0x0000007B 归一化为 0x7B 键。
func maskCode(code string) string {
	hex := strings.TrimPrefix(strings.ToLower(code), "0x")
	hex = strings.TrimLeft(hex, "0")
	if hex == "" {
		hex = "0"
	}
	return "0x" + hex
}
