package ui

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

const xshLogo = `__  __  ____   _   _
\ \/ / / ___| | | | |
 \  /  \___ \ | |_| |
 /  \   ___) ||  _  |
/_/\_\ |____/ |_| |_|`

// LogoLines 返回 ASCII 横幅的行。
func LogoLines() []string {
	return []string{
		"__  __  ____   _   _ ",
		"\\ \\/ / / ___| | | | |",
		" \\  /  \\___ \\ | |_| |",
		" /  \\   ___) ||  _  |",
		"/_/\\_\\ |____/ |_| |_|",
	}
}

// Banner 输出玄枢 XSH 的 ASCII 横幅与品牌名。
func Banner(w io.Writer) {
	fmt.Fprintln(w)
	for _, ln := range LogoLines() {
		fmt.Fprintln(w, color.CyanString(ln))
	}
	fmt.Fprintf(w, "  玄枢 XSH v%s — 多合一 CLI 工具\n\n", Version)
}
