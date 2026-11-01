package builtins

import (
	"fmt"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/ui"
)

func registerVerbs(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "where", Aliases: []string{"Where-Object"}, Summary: "过滤 <字段> <eq|ne|gt|lt|ge|le|like|match|in> <值>", Fn: cmdWhere})
	reg.Add(&shell.Command{Name: "select", Aliases: []string{"Select-Object"}, Summary: "投影 <字段,...> 或 -first/-last N", Fn: cmdSelect})
	reg.Add(&shell.Command{Name: "sort", Aliases: []string{"Sort-Object"}, Summary: "排序 <字段> [desc]", Fn: cmdSort})
	reg.Add(&shell.Command{Name: "measure", Aliases: []string{"Measure-Object"}, Summary: "统计 [字段] → count/sum/avg/min/max", Fn: cmdMeasure})
	reg.Add(&shell.Command{Name: "group", Aliases: []string{"Group-Object"}, Summary: "分组计数 <字段>", Fn: cmdGroup})
	reg.Add(&shell.Command{Name: "ft", Aliases: []string{"Format-Table"}, Summary: "表格输出 [字段,...]", Fn: cmdFT})
	reg.Add(&shell.Command{Name: "fl", Aliases: []string{"Format-List"}, Summary: "列表输出 [字段,...]", Fn: cmdFL})
	reg.Add(&shell.Command{Name: "first", Summary: "前 N 条 <N>", Fn: cmdFirst})
	reg.Add(&shell.Command{Name: "last", Summary: "后 N 条 <N>", Fn: cmdLast})
}

func cmdWhere(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	if len(args) < 3 {
		return argErr("where", "<字段> <eq|ne|gt|lt|ge|le|like|match|in> <值>")
	}
	out := pipeline.Filter(in, args[0], args[1], strings.Join(args[2:], " "))
	return out, inSc, nil
}

func cmdSelect(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"first": true, "last": true})
	if flags["first"] != "" {
		return pipeline.FirstN(in, atoi(flags["first"], 10)), inSc, nil
	}
	if flags["last"] != "" {
		return pipeline.LastN(in, atoi(flags["last"], 10)), inSc, nil
	}
	if len(rest) == 0 {
		return argErr("select", "<字段,...> | -first N | -last N")
	}
	var fields []string
	for _, f := range rest {
		fields = append(fields, strings.Split(f, ",")...)
	}
	out := pipeline.Project(in, fields)
	sc := pipeline.NewSchema(fields...)
	if inSc != nil {
		for _, f := range fields {
			if inSc.Numeric[f] {
				sc.Numeric[f] = true
			}
		}
	}
	return out, sc, nil
}

func cmdSort(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("sort", "<字段> [desc]")
	}
	desc := len(rest) > 1 && strings.EqualFold(rest[1], "desc")
	numHint := inSc != nil && inSc.Numeric[rest[0]]
	pipeline.SortRecords(in, rest[0], desc, numHint)
	return in, inSc, nil
}

func cmdMeasure(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	r := pipeline.Measure(in, first(rest))
	sc := pipeline.NewSchema("count").WithNumeric("count")
	if len(r) > 1 {
		sc = sc.WithNumeric("sum", "avg", "min", "max")
	}
	return []pipeline.Record{r}, sc, nil
}

func cmdGroup(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("group", "<字段>")
	}
	out := pipeline.Group(in, rest[0])
	sc := pipeline.NewSchema("value", "count").WithNumeric("count")
	return out, sc, nil
}

func resolveFields(args []string, in []pipeline.Record, inSc *pipeline.Schema) []string {
	_, rest := parseArgs(args, nil)
	if len(rest) > 0 {
		var fields []string
		for _, f := range rest {
			fields = append(fields, strings.Split(f, ",")...)
		}
		return fields
	}
	if inSc != nil {
		return inSc.Fields
	}
	return pipeline.UnionFields(in)
}

func cmdFT(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	fields := resolveFields(args, in, inSc)
	rows := make([][]string, 0, len(in))
	for _, rec := range in {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = rec[f]
		}
		rows = append(rows, row)
	}
	ui.PrintTable(c.Out, fields, rows)
	return nil, nil, nil
}

func cmdFL(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	fields := resolveFields(args, in, inSc)
	for _, rec := range in {
		for _, f := range fields {
			fmt.Fprintf(c.Out, "%-16s %s\n", f, rec[f])
		}
		fmt.Fprintln(c.Out)
	}
	return nil, nil, nil
}

func cmdFirst(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("first", "<N>")
	}
	return pipeline.FirstN(in, atoi(rest[0], 10)), inSc, nil
}

func cmdLast(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("last", "<N>")
	}
	return pipeline.LastN(in, atoi(rest[0], 10)), inSc, nil
}
