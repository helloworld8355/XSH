package diag

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"xsh/internal/ui"
)

// evtRecord PowerShell Get-WinEvent 转出的 JSON 记录。
type evtRecord struct {
	TimeCreated  string `json:"TimeCreated"`
	ProviderName string `json:"ProviderName"`
	Id           int    `json:"Id"`
	LevelName    string `json:"LevelDisplayName"`
	Message      string `json:"Message"`
}

// Event 最近错误/警告事件诊断。
func (r *Runner) Event() error {
	ui.Title(r.Out, "事件日志（最近 48 小时 错误/警告）")
	ps := `Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2; StartTime=(Get-Date).AddHours(-48)} -MaxEvents 30 | Select-Object TimeCreated, ProviderName, Id, LevelDisplayName, Message | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).Output()
	if err != nil {
		return fmt.Errorf("读取事件日志失败（尝试以管理员运行）: %w", err)
	}
	recs, err := parseEvents(out)
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		ui.OK(r.Out, "最近 48 小时没有错误/警告事件")
		return nil
	}
	rows := make([][]string, 0, len(recs))
	for _, e := range recs {
		rows = append(rows, []string{
			evtTime(e.TimeCreated),
			e.LevelName,
			fmt.Sprintf("%s (%d)", e.ProviderName, e.Id),
			truncate(strings.TrimSpace(e.Message), 64),
		})
	}
	ui.PrintTable(r.Out, []string{"时间", "级别", "来源", "消息"}, rows)
	ui.Info(r.Out, "提示: 频繁出现的错误可用 `xsh diag driver` 与 `xsh diag bsod` 交叉定位")
	return nil
}

func parseEvents(out []byte) ([]evtRecord, error) {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	var many []evtRecord
	if err := json.Unmarshal([]byte(s), &many); err == nil {
		return many, nil
	}
	var one evtRecord
	if err := json.Unmarshal([]byte(s), &one); err != nil {
		return nil, fmt.Errorf("事件解析失败: %w", err)
	}
	return []evtRecord{one}, nil
}

// evtTime 解析 ConvertTo-Json 的 /Date(ms)/ 时间。
func evtTime(s string) string {
	s = strings.ReplaceAll(s, "/Date(", "")
	s = strings.ReplaceAll(s, ")/", "")
	var ms int64
	if _, err := fmt.Sscanf(s, "%d", &ms); err != nil {
		return s
	}
	return time.UnixMilli(ms).Format("01-02 15:04")
}

func truncate(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return strings.TrimSpace(string(rs[:n])) + "…"
}
