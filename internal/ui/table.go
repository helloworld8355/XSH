package ui

import (
	"fmt"
	"io"
	"strings"
)

// PrintTable 输出对齐表格（支持 CJK 宽度对齐）。
func PrintTable(w io.Writer, header []string, rows [][]string) {
	if len(header) == 0 {
		return
	}
	ncol := len(header)
	widths := make([]int, ncol)
	for i, h := range header {
		widths[i] = StringWidth(h)
	}
	for _, row := range rows {
		for i := 0; i < ncol && i < len(row); i++ {
			if v := StringWidth(row[i]); v > widths[i] {
				widths[i] = v
			}
		}
	}
	total := 1
	for _, wd := range widths {
		total += wd + 3
	}
	var b strings.Builder
	for i, h := range header {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(PadRight(h, widths[i]))
	}
	Hdr(w, b.String())
	Sep(w, total)
	for _, row := range rows {
		var rb strings.Builder
		for i := 0; i < ncol; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if i > 0 {
				rb.WriteString("  ")
			}
			rb.WriteString(PadRight(cell, widths[i]))
		}
		fmt.Fprintln(w, rb.String())
	}
	if len(rows) > 0 {
		fmt.Fprintln(w)
	}
}
