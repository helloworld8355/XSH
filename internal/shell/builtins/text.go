package builtins

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
)

func registerText(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "grep", Aliases: []string{"Select-String"}, Summary: "正则过滤 [-i忽略大小写] [-v取反] <模式> [文件...]", Fn: cmdGrep})
	reg.Add(&shell.Command{Name: "head", Summary: "前 N 行 [-n N=10] [文件]", Fn: cmdHead})
	reg.Add(&shell.Command{Name: "tail", Summary: "后 N 行 [-n N=10] [文件]", Fn: cmdTail})
	reg.Add(&shell.Command{Name: "wc", Summary: "行/词/字符统计 [文件...]", Fn: cmdWc})
	reg.Add(&shell.Command{Name: "diff", Aliases: []string{"Compare-Object"}, Summary: "对比两个文件 <a> <b>", Fn: cmdDiff})
	reg.Add(&shell.Command{Name: "uniq", Summary: "相邻重复行去重", Fn: cmdUniq})
	reg.Add(&shell.Command{Name: "cut", Summary: "按分隔符取字段 -d <分隔符> -f <序号|N-M>", Fn: cmdCut})
}

func cmdGrep(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, nil)
	pat := first(rest)
	if pat == "" {
		return argErr("grep", "[-i] [-v] <模式> [文件...]")
	}
	expr := pat
	if flags["i"] != "" {
		expr = "(?i)" + pat
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, nil, err
	}
	invert := flags["v"] != ""
	var recs []pipeline.Record
	sc := pipeline.LineSchema()
	if len(rest) > 1 {
		sc = pipeline.NewSchema("file", "line_no", "line").WithNumeric("line_no")
		for _, f := range rest[1:] {
			lines, err := readFileLines(c.Resolve(f), 0)
			if err != nil {
				return nil, nil, err
			}
			for i, ln := range lines {
				if re.MatchString(ln) != invert {
					recs = append(recs, pipeline.Record{"file": f, "line_no": strconv.Itoa(i + 1), "line": ln})
				}
			}
		}
		return recs, sc, nil
	}
	for _, rec := range in {
		if re.MatchString(rec["line"]) != invert {
			recs = append(recs, pipeline.Record{"line": rec["line"]})
		}
	}
	return recs, sc, nil
}

func cmdHead(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"n": true})
	n := atoi(flags["n"], 10)
	var lines []pipeline.Record
	var sc *pipeline.Schema
	if len(rest) > 0 {
		ls, err := readFileLines(c.Resolve(rest[0]), 0)
		if err != nil {
			return nil, nil, err
		}
		lines = make([]pipeline.Record, 0, len(ls))
		for _, ln := range ls {
			lines = append(lines, pipeline.Record{"line": ln})
		}
		sc = pipeline.LineSchema()
	} else {
		lines, sc = in, pipeline.LineSchema()
	}
	if n >= len(lines) {
		return lines, sc, nil
	}
	return lines[:n], sc, nil
}

func cmdTail(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"n": true})
	n := atoi(flags["n"], 10)
	var lines []pipeline.Record
	var sc *pipeline.Schema
	if len(rest) > 0 {
		ls, err := readFileLines(c.Resolve(rest[0]), 0)
		if err != nil {
			return nil, nil, err
		}
		lines = make([]pipeline.Record, 0, len(ls))
		for _, ln := range ls {
			lines = append(lines, pipeline.Record{"line": ln})
		}
		sc = pipeline.LineSchema()
	} else {
		lines, sc = in, pipeline.LineSchema()
	}
	if n >= len(lines) {
		return lines, sc, nil
	}
	return lines[len(lines)-n:], sc, nil
}

func cmdWc(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	count := func(lines []string) (int, int, int) {
		words, chars := 0, 0
		for _, ln := range lines {
			words += len(strings.Fields(ln))
			chars += len([]rune(ln))
		}
		return len(lines), words, chars
	}
	var recs []pipeline.Record
	if len(rest) > 0 {
		for _, f := range rest {
			lines, err := readFileLines(c.Resolve(f), 0)
			if err != nil {
				return nil, nil, err
			}
			l, w, ch := count(lines)
			recs = append(recs, pipeline.Record{"file": f, "lines": strconv.Itoa(l), "words": strconv.Itoa(w), "chars": strconv.Itoa(ch)})
		}
	} else {
		var lines []string
		for _, r := range in {
			lines = append(lines, r["line"])
		}
		l, w, ch := count(lines)
		recs = append(recs, pipeline.Record{"file": "-", "lines": strconv.Itoa(l), "words": strconv.Itoa(w), "chars": strconv.Itoa(ch)})
	}
	sc := pipeline.NewSchema("file", "lines", "words", "chars").WithNumeric("lines", "words", "chars")
	return recs, sc, nil
}

func cmdDiff(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) < 2 {
		return argErr("diff", "<文件A> <文件B>")
	}
	a, err := readFileLines(c.Resolve(rest[0]), 0)
	if err != nil {
		return nil, nil, err
	}
	b, err := readFileLines(c.Resolve(rest[1]), 0)
	if err != nil {
		return nil, nil, err
	}
	if len(a) > 1000 {
		a = a[:1000]
	}
	if len(b) > 1000 {
		b = b[:1000]
	}
	recs := diffRecords(a, b)
	return recs, pipeline.NewSchema("op", "line"), nil
}

func diffRecords(a, b []string) []pipeline.Record {
	n, m := len(a), len(b)
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []pipeline.Record
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, pipeline.Record{"op": " ", "line": a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			out = append(out, pipeline.Record{"op": "-", "line": a[i]})
			i++
		default:
			out = append(out, pipeline.Record{"op": "+", "line": b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, pipeline.Record{"op": "-", "line": a[i]})
	}
	for ; j < m; j++ {
		out = append(out, pipeline.Record{"op": "+", "line": b[j]})
	}
	return out
}

func cmdUniq(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	var recs []pipeline.Record
	prev := "\x00"
	for _, rec := range in {
		if rec["line"] != prev {
			recs = append(recs, pipeline.Record{"line": rec["line"]})
			prev = rec["line"]
		}
	}
	return recs, pipeline.LineSchema(), nil
}

func cmdCut(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"d": true, "f": true})
	if len(rest) == 0 && len(in) == 0 {
		return argErr("cut", "[-d 分隔符] -f <序号|N-M> [输入]")
	}
	delim := flags["d"]
	if delim == "" {
		delim = "\t"
	}
	fields := flags["f"]
	if fields == "" {
		fields = "1"
	}
	lo, hi := 1, 1
	if parts := strings.SplitN(fields, "-", 2); len(parts) == 2 {
		lo, hi = atoi(parts[0], 1), atoi(parts[1], 1)
	} else {
		lo, hi = atoi(fields, 1), atoi(fields, 1)
	}
	var src []string
	if len(rest) > 0 {
		lines, err := readFileLines(c.Resolve(rest[0]), 0)
		if err != nil {
			return nil, nil, err
		}
		src = lines
	} else {
		for _, r := range in {
			src = append(src, r["line"])
		}
	}
	var recs []pipeline.Record
	for _, ln := range src {
		parts := strings.Split(ln, delim)
		if lo < 1 || lo > len(parts) {
			recs = append(recs, pipeline.Record{"line": ""})
			continue
		}
		end := hi
		if end > len(parts) {
			end = len(parts)
		}
		recs = append(recs, pipeline.Record{"line": strings.Join(parts[lo-1:end], delim)})
	}
	return recs, pipeline.LineSchema(), nil
}

var _ = fmt.Sprintf
