// Package devenv 开发环境体检：50+ 工具链探测与 PATH 审计。
package devenv

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Tool 工具探测结果。
type Tool struct {
	Name    string
	Found   bool
	Path    string
	Version string
	Env     string
}

// defaultVerRe 兜底版本号匹配。
var defaultVerRe = regexp.MustCompile(`(\d+(\.\d+)+[\w.\-+]*)`)

var verTimeout = 10 * time.Second

// Detect 并发探测全部工具链。
func Detect() []Tool {
	tools := make([]Tool, len(detectors))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, d := range detectors {
		wg.Add(1)
		go func(i int, d detector) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			tools[i] = detectOne(d)
		}(i, d)
	}
	wg.Wait()
	return tools
}

func detectOne(d detector) Tool {
	t := Tool{Name: d.name, Env: envSummary(d.env)}
	path, err := exec.LookPath(d.name)
	if err != nil {
		return t
	}
	t.Found = true
	t.Path = path
	t.Version = probeVersion(d, path)
	return t
}

// probeVersion 执行版本命令并提取版本号（stdout+stderr）。
func probeVersion(d detector, path string) string {
	if len(d.args) == 0 {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), verTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, d.args...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if s == "" && err != nil {
		return ""
	}
	re := d.re
	if re == nil {
		re = defaultVerRe
	}
	if m := re.FindStringSubmatch(s); m != nil {
		if len(m) > 1 {
			return m[1]
		}
		return truncateRunes(m[0], 40)
	}
	return truncateRunes(strings.SplitN(s, "\n", 2)[0], 40)
}

// envSummary 汇总相关环境变量（VAR=value 空格分隔，值截断）。
func envSummary(vars []string) string {
	if len(vars) == 0 {
		return ""
	}
	var parts []string
	for _, k := range vars {
		if v := os.Getenv(k); v != "" {
			parts = append(parts, k+"="+truncateRunes(v, 36))
		}
	}
	return strings.Join(parts, " ")
}

func truncateRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return strings.TrimSpace(string(rs[:n])) + "…"
}
