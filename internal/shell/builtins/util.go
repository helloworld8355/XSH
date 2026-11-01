// Package builtins 提供 shell 全部内置命令。
package builtins

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"xsh/internal/pipeline"
)

// parseArgs 解析简单 flags；valueFlags 中的旗标取下一参数为值。
func parseArgs(args []string, valueFlags map[string]bool) (map[string]string, []string) {
	flags := map[string]string{}
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) >= 2 && a[0] == '-' {
			name := strings.TrimLeft(a, "-")
			if valueFlags[name] && i+1 < len(args) {
				flags[name] = args[i+1]
				i++
			} else {
				flags[name] = ""
			}
		} else {
			rest = append(rest, a)
		}
	}
	return flags, rest
}

func argErr(cmd, usage string) ([]pipeline.Record, *pipeline.Schema, error) {
	return nil, nil, fmt.Errorf("用法: %s %s", cmd, usage)
}

func first(ss []string) string {
	if len(ss) > 0 {
		return ss[0]
	}
	return ""
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func fmtTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

// readFileLines 读取文件行（限制 20MB）。
func readFileLines(path string, max int64) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if max <= 0 {
		max = 20 << 20
	}
	var out []string
	sc := bufio.NewScanner(io.LimitReader(f, max))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out, sc.Err()
}

// isBinary 检查文件是否为二进制（前 2KB 含 NUL）。
func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	return strings.ContainsRune(string(buf[:n]), 0)
}
