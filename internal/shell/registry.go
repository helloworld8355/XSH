// Package shell 实现命令注册表、管道执行引擎与交互式 REPL。
package shell

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/ui"
)

// Ctx 执行上下文。
type Ctx struct {
	Cwd         string
	Out         io.Writer
	Registry    *Registry
	Interactive bool
	HistFile    string
	RunDepth    int
}

// CmdFn 命令实现签名：输入记录 + 输入 schema → 输出记录 + 输出 schema。
type CmdFn func(c *Ctx, args []string, in []pipeline.Record, inSchema *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error)

// Command 内置命令。
type Command struct {
	Name    string
	Aliases []string
	Summary string
	Fn      CmdFn
}

// Registry 命令注册表（交互模式与单命令模式共用）。
type Registry struct {
	m    map[string]*Command
	name []string
}

// NewRegistry 创建注册表。
func NewRegistry() *Registry {
	return &Registry{m: map[string]*Command{}}
}

// Add 注册命令及其别名。
func (r *Registry) Add(cmd *Command) {
	for _, n := range append([]string{cmd.Name}, cmd.Aliases...) {
		r.m[strings.ToLower(n)] = cmd
	}
	r.name = append(r.name, cmd.Name)
}

// Lookup 按名称（忽略大小写）查找命令。
func (r *Registry) Lookup(name string) (*Command, bool) {
	c, ok := r.m[strings.ToLower(name)]
	return c, ok
}

// Commands 按注册顺序返回命令。
func (r *Registry) Commands() []*Command {
	out := make([]*Command, 0, len(r.name))
	seen := map[*Command]bool{}
	for _, n := range r.name {
		if c, ok := r.Lookup(n); ok && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out
}

// AllNames 返回全部命令名与别名（排序，补全用）。
func (r *Registry) AllNames() []string {
	seen := map[string]bool{}
	var out []string
	for _, cmd := range r.Commands() {
		for _, n := range append([]string{cmd.Name}, cmd.Aliases...) {
			ln := strings.ToLower(n)
			if !seen[ln] {
				seen[ln] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Execute 解析并执行一行管道命令。
func (r *Registry) Execute(line string, c *Ctx) error {
	stages, err := pipeline.Parse(line)
	if err != nil {
		ui.Err(c.Out, "%v", err)
		return err
	}
	if len(stages) == 0 {
		return nil
	}
	var in []pipeline.Record
	var sc *pipeline.Schema
	for _, st := range stages {
		if cmd, ok := r.Lookup(st.Name); ok {
			out, outSc, err := cmd.Fn(c, st.Args, in, sc)
			if err != nil {
				ui.Err(c.Out, "%s: %v", st.Name, err)
				return err
			}
			in, sc = out, outSc
			continue
		}
		out, outSc, err := r.runExternal(c, st.Name, st.Args, in)
		if err != nil {
			ui.Err(c.Out, "%s: %v", st.Name, err)
			return err
		}
		in, sc = out, outSc
	}
	if len(in) > 0 {
		AutoFormat(c.Out, in, sc)
	}
	return nil
}

// AutoFormat 管道末端隐式输出。
func AutoFormat(w io.Writer, in []pipeline.Record, sc *pipeline.Schema) {
	if sc != nil && len(sc.Fields) == 1 && sc.Fields[0] == "line" {
		for _, rec := range in {
			fmt.Fprintln(w, rec["line"])
		}
		return
	}
	var fields []string
	if sc != nil {
		fields = sc.Fields
	} else {
		fields = pipeline.UnionFields(in)
	}
	rows := make([][]string, 0, len(in))
	for _, rec := range in {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = rec[f]
		}
		rows = append(rows, row)
	}
	ui.PrintTable(w, fields, rows)
}

// runExternal 将未注册命令作为外部程序执行。
func (r *Registry) runExternal(c *Ctx, name string, args []string, in []pipeline.Record) ([]pipeline.Record, *pipeline.Schema, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil, nil, fmt.Errorf("未知命令 %q", name)
	}
	cmd := exec.Command(path, args...)
	cmd.Dir = c.Cwd
	if len(in) > 0 {
		var sb strings.Builder
		for _, rec := range in {
			sb.WriteString(rec["line"])
			sb.WriteString("\n")
		}
		cmd.Stdin = strings.NewReader(sb.String())
	}
	cmd.Stderr = c.Out
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, nil, err
		}
	}
	recs := make([]pipeline.Record, 0, 64)
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSuffix(ln, "\r")
		recs = append(recs, pipeline.Record{"line": ln})
	}
	if n := len(recs); n > 0 && recs[n-1]["line"] == "" {
		recs = recs[:n-1]
	}
	return recs, pipeline.LineSchema(), nil
}

// Resolve 基于当前工作目录解析相对路径。
func (c *Ctx) Resolve(p string) string {
	if p == "" {
		return c.Cwd
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(c.Cwd, p))
}
