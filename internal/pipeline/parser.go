package pipeline

import (
	"fmt"
	"strings"
)

// Stage 管道中的一个命令阶段。
type Stage struct {
	Name string
	Args []string
}

// Parse 将输入行解析为命令管道，支持单/双引号、| 分隔与 # 注释。
func Parse(line string) ([]Stage, error) {
	toks, err := tokenize(line)
	if err != nil {
		return nil, err
	}
	var stages []Stage
	var cur []string
	for _, t := range toks {
		if t == "|" {
			if len(cur) == 0 {
				return nil, fmt.Errorf("管道位置错误")
			}
			stages = append(stages, Stage{Name: cur[0], Args: cur[1:]})
			cur = nil
			continue
		}
		cur = append(cur, t)
	}
	if len(cur) == 0 {
		if len(stages) > 0 {
			return nil, fmt.Errorf("管道位置错误")
		}
		return nil, nil
	}
	stages = append(stages, Stage{Name: cur[0], Args: cur[1:]})
	return stages, nil
}

func tokenize(s string) ([]string, error) {
	var toks []string
	var b strings.Builder
	inS, inD := false, false
	flush := func() {
		if b.Len() > 0 {
			toks = append(toks, b.String())
			b.Reset()
		}
	}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case inS:
			if r == '\'' {
				inS = false
			} else {
				b.WriteRune(r)
			}
		case inD:
			switch {
			case r == '"':
				inD = false
			case r == '\\' && i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\\'):
				i++
				b.WriteRune(runes[i])
			default:
				b.WriteRune(r)
			}
		case r == '\'':
			inS = true
		case r == '"':
			inD = true
		case r == '|':
			flush()
			toks = append(toks, "|")
		case r == '#':
			if b.Len() == 0 {
				return toks, nil
			}
			b.WriteRune(r)
		case r == ' ' || r == '\t':
			flush()
		default:
			b.WriteRune(r)
		}
	}
	if inS || inD {
		return nil, fmt.Errorf("引号未闭合")
	}
	flush()
	return toks, nil
}
