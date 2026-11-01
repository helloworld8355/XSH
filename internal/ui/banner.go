package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// 横幅样式。
const (
	BannerSimple = "simple"
	BannerNormal = "normal"
	BannerCool   = "cool"
)

const xshLogo = `__  __  ____   _   _
\ \/ / / ___| | | | |
 \  /  \___ \ | |_| |
 /  \   ___) ||  _  |
/_/\_\ |____/ |_| |_|`

// LogoLines 返回小号 ASCII 横幅的行。
func LogoLines() []string {
	return []string{
		"__  __  ____   _   _ ",
		"\\ \\/ / / ___| | | | |",
		" \\  /  \\___ \\ | |_| |",
		" /  \\   ___) ||  _  |",
		"/_/\\_\\ |____/ |_| |_|",
	}
}

// blockLogoLines 大号块状 ASCII 艺术字 XSH（6 行）。
var blockLogoLines = []string{
	" ██╗  ██╗  ███████╗  ██╗  ██╗",
	" ╚██╗██╔╝  ██╔════╝  ██║  ██║",
	"  ╚███╔╝   ███████╗  ███████║",
	"  ██╔██╗   ╚════██║  ╚════██║",
	" ██╔╝ ██╗  ███████╔╝ ██╗  ██║",
	" ╚═╝  ╚═╝  ╚══════╝  ╚═╝  ╚═╝",
}

// coolGradient 炫酷横幅的逐行渐变色。
var coolGradient = []*color.Color{
	color.New(color.FgCyan),
	color.New(color.FgHiCyan),
	color.New(color.FgBlue),
	color.New(color.FgMagenta),
	color.New(color.FgHiMagenta),
	color.New(color.FgYellow),
}

// ParseBannerStyle 解析横幅样式（支持中英文别名，空值默认普通）。
func ParseBannerStyle(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", BannerNormal, "n", "普通", "标准":
		return BannerNormal, nil
	case BannerSimple, "min", "s", "简洁", "极简":
		return BannerSimple, nil
	case BannerCool, "fancy", "c", "炫酷", "豪华":
		return BannerCool, nil
	default:
		return "", fmt.Errorf("未知横幅样式 %q（可选: simple 简洁 / normal 普通 / cool 炫酷）", s)
	}
}

// Banner 按样式输出玄枢 XSH 启动横幅。
func Banner(w io.Writer, style string) {
	switch style {
	case BannerSimple:
		bannerSimple(w)
	case BannerCool:
		bannerCool(w)
	default:
		bannerNormal(w)
	}
}

// bannerSimple 简洁：单行品牌标识。
func bannerSimple(w io.Writer) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s %s\n\n",
		color.New(color.Bold, color.FgCyan).Sprint("玄枢 XSH"),
		color.HiBlackString("v"+Version+" — 多合一 CLI 工具"))
}

// bannerNormal 普通：大号块状艺术字 + 品牌行（默认样式）。
func bannerNormal(w io.Writer) {
	fmt.Fprintln(w)
	boldCyan := color.New(color.Bold, color.FgCyan)
	for _, ln := range blockLogoLines {
		fmt.Fprintln(w, boldCyan.Sprint(ln))
	}
	fmt.Fprintf(w, "\n  %s %s\n\n",
		boldCyan.Sprint("玄枢 XSH"),
		color.HiBlackString("v"+Version+" — 多合一 CLI 工具"))
}

// bannerCool 炫酷：渐变色艺术字 + 装饰边线 + 标语。
func bannerCool(w io.Writer) {
	const rule = 34
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.CyanString(strings.Repeat("▄", rule)))
	fmt.Fprintln(w)
	for i, ln := range blockLogoLines {
		fmt.Fprintln(w, coolGradient[i%len(coolGradient)].Sprint(ln))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s\n",
		color.New(color.Bold, color.FgHiMagenta).Sprint("◆ 玄枢 XSH"),
		color.New(color.Bold, color.FgHiWhite).Sprint("v"+Version))
	fmt.Fprintf(w, "  %s\n", color.HiBlackString("系统 · 诊断 · 开发 · 安全 · 工具集"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, color.MagentaString(strings.Repeat("▀", rule)))
	fmt.Fprintln(w)
}
