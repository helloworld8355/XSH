package shell

import (
	"os"
	"strings"

	"github.com/chzyer/readline"
)

// completer 实现命令名与文件名补全。
type completer struct {
	reg *Registry
	ctx *Ctx
}

func newCompleter(reg *Registry, ctx *Ctx) readline.AutoCompleter {
	return &completer{reg: reg, ctx: ctx}
}

func (c *completer) Do(line []rune, pos int) ([][]rune, int) {
	if pos > len(line) {
		pos = len(line)
	}
	prefix := string(line[:pos])
	fields := strings.Fields(prefix)

	// 命令名补全
	if len(fields) <= 1 && !strings.HasSuffix(prefix, " ") {
		start := ""
		if len(fields) == 1 {
			start = strings.ToLower(fields[0])
		}
		var cands [][]rune
		for _, n := range c.reg.AllNames() {
			if len(n) > len(start) && strings.HasPrefix(strings.ToLower(n), start) {
				cands = append(cands, []rune(n))
			}
		}
		return cands, len(start)
	}

	// 文件名补全
	lastStart := strings.LastIndexAny(prefix, " \t") + 1
	word := prefix[lastStart:]
	sepIdx := strings.LastIndexAny(word, `/\`)
	dirPart, base := "", word
	if sepIdx >= 0 {
		dirPart = word[:sepIdx+1]
		base = word[sepIdx+1:]
	}
	var absDir string
	if dirPart == "" {
		absDir = c.ctx.Cwd
	} else {
		absDir = c.ctx.Resolve(strings.TrimSuffix(strings.TrimSuffix(dirPart, "\\"), "/"))
	}
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, 0
	}
	var cands [][]rune
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
			continue
		}
		suffix := ""
		if e.IsDir() {
			suffix = string(os.PathSeparator)
		}
		cands = append(cands, []rune(dirPart+name+suffix))
	}
	return cands, len(word)
}
