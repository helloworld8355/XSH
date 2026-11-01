package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Version 为 XSH 版本号。
const Version = "1.0.0"

func SetNoColor(v bool) { color.NoColor = v }

func OK(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, color.GreenString("[+] ")+format+"\n", a...)
}

func Warn(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, color.YellowString("[!] ")+format+"\n", a...)
}

func Err(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, color.RedString("[x] ")+format+"\n", a...)
}

func Info(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, color.CyanString("[*] ")+format+"\n", a...)
}

func Title(w io.Writer, s string) {
	fmt.Fprintf(w, "\n%s\n", color.New(color.Bold, color.FgCyan).Sprint("== "+s))
}

func Field(w io.Writer, k, v string) {
	fmt.Fprintf(w, "  %s %s\n", color.CyanString("%-14s", k), v)
}

func Hdr(w io.Writer, s string) {
	fmt.Fprintln(w, color.New(color.Bold, color.FgWhite).Sprint(s))
}

func Sep(w io.Writer, n int) {
	if n < 4 {
		n = 4
	}
	fmt.Fprintln(w, color.HiBlackString(strings.Repeat("-", n)))
}

// Dim 输出暗色文本。
func Dim(w io.Writer, s string) {
	fmt.Fprintln(w, color.HiBlackString(s))
}
