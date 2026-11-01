package syscheck

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/fatih/color"

	"xsh/internal/ui"
)

// sevColor 级别着色文本。
func sevColor(s Severity) string {
	switch s {
	case SevCrit:
		return color.RedString(s.String())
	case SevWarn:
		return color.YellowString(s.String())
	case SevInfo:
		return color.CyanString(s.String())
	}
	return color.GreenString(s.String())
}

// RenderText 控制台输出报告。
func (r *Report) RenderText(w io.Writer) {
	ui.Title(w, "系统体检报告")
	ui.Field(w, "主机", r.Hostname)
	ui.Field(w, "时间", r.Time.Format("2006-01-02 15:04:05"))
	ui.Sep(w, 72)
	rows := make([][]string, 0, len(r.Checks))
	for _, c := range r.Checks {
		rows = append(rows, []string{c.Name, sevColor(c.Sev), c.Detail, c.Advice})
	}
	ui.PrintTable(w, []string{"检查项", "结果", "详情", "建议"}, rows)
	ui.Sep(w, 72)
	total := r.Total
	switch {
	case total >= 90:
		fmt.Fprintf(w, "综合评分: %s  等级: %s\n", color.GreenString("%d/100", total), color.GreenString(r.Grade))
	case total >= 75:
		fmt.Fprintf(w, "综合评分: %s  等级: %s\n", color.YellowString("%d/100", total), color.YellowString(r.Grade))
	default:
		fmt.Fprintf(w, "综合评分: %s  等级: %s\n", color.RedString("%d/100", total), color.RedString(r.Grade))
	}
}

// JSON 序列化为 JSON 字节。
func (r *Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// RenderHTML 输出 HTML 报告。
func (r *Report) RenderHTML(w io.Writer) error {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<title>XSH 系统体检报告</title>
<style>
body{font-family:"Segoe UI",system-ui,sans-serif;margin:24px;background:#f6f8fa;color:#1f2328}
h1{font-size:20px} table{border-collapse:collapse;width:100%;background:#fff;font-size:14px}
th,td{border:1px solid #d0d7de;padding:6px 10px;text-align:left;vertical-align:top}
th{background:#eef1f4} .crit{color:#cf222e;font-weight:600} .warn{color:#9a6700;font-weight:600}
.info{color:#0969da} .ok{color:#1a7f37}
.total{margin-top:12px;font-size:16px;font-weight:600}
</style>
</head>
<body>
<h1>玄枢 XSH 系统体检报告</h1>
<p>`)
	fmt.Fprintf(&b, "主机: %s &nbsp; 时间: %s</p>\n", html.EscapeString(r.Hostname), r.Time.Format("2006-01-02 15:04:05"))
	b.WriteString("<table><tr><th>检查项</th><th>结果</th><th>详情</th><th>建议</th></tr>\n")
	for _, c := range r.Checks {
		cls := ""
		switch c.Sev {
		case SevCrit:
			cls = "crit"
		case SevWarn:
			cls = "warn"
		case SevInfo:
			cls = "info"
		default:
			cls = "ok"
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td>%s</td></tr>\n",
			html.EscapeString(c.Name), cls, html.EscapeString(c.Sev.String()),
			html.EscapeString(c.Detail), html.EscapeString(c.Advice))
	}
	b.WriteString("</table>\n")
	fmt.Fprintf(&b, "<p class=\"total\">综合评分: %d/100 &nbsp; 等级: %s</p>\n", r.Total, r.Grade)
	b.WriteString("</body>\n</html>\n")
	_, err := io.WriteString(w, b.String())
	return err
}
