package builtins

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
)

func registerData(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "fromjson", Aliases: []string{"ConvertFrom-Json"}, Summary: "解析 JSON [文件]", Fn: cmdFromJSON})
	reg.Add(&shell.Command{Name: "tojson", Aliases: []string{"ConvertTo-Json"}, Summary: "记录转 JSON", Fn: cmdToJSON})
	reg.Add(&shell.Command{Name: "fromcsv", Summary: "解析 CSV（首行为表头）[文件]", Fn: cmdFromCSV})
	reg.Add(&shell.Command{Name: "tocsv", Summary: "记录转 CSV", Fn: cmdToCSV})
}

// jsonValue 把 JSON 值转为字符串（数字保精度，嵌套结构回序列化）。
func jsonValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(b)
	}
}

// anyToRecord 任意 JSON 值 → 记录（对象展开为字段，标量入 value 字段）。
func anyToRecord(v any) pipeline.Record {
	if m, ok := v.(map[string]any); ok {
		rec := pipeline.Record{}
		for k, val := range m {
			rec[k] = jsonValue(val)
		}
		return rec
	}
	return pipeline.Record{"value": jsonValue(v)}
}

// schemaFor 依据记录推导 Schema（全为数字的列标记为数值列）。
func schemaFor(recs []pipeline.Record) *pipeline.Schema {
	fields := pipeline.UnionFields(recs)
	sc := pipeline.NewSchema(fields...)
	for _, f := range fields {
		numeric, seen := true, false
		for _, r := range recs {
			if s := r[f]; s != "" {
				seen = true
				if _, err := strconv.ParseFloat(s, 64); err != nil {
					numeric = false
					break
				}
			}
		}
		if numeric && seen {
			sc = sc.WithNumeric(f)
		}
	}
	return sc
}

func cmdFromJSON(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	var data []byte
	if len(rest) > 0 {
		b, err := os.ReadFile(c.Resolve(rest[0]))
		if err != nil {
			return nil, nil, err
		}
		data = b
	} else if len(in) > 0 {
		var sb strings.Builder
		for _, r := range in {
			sb.WriteString(r["line"])
			sb.WriteString("\n")
		}
		data = []byte(sb.String())
	} else {
		return nil, nil, fmt.Errorf("缺少 JSON 输入（文件或管道）")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	var recs []pipeline.Record
	if arr, ok := v.([]any); ok {
		recs = make([]pipeline.Record, 0, len(arr))
		for _, item := range arr {
			recs = append(recs, anyToRecord(item))
		}
	} else {
		recs = []pipeline.Record{anyToRecord(v)}
	}
	return recs, schemaFor(recs), nil
}

func cmdToJSON(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	if len(in) == 0 {
		return nil, nil, fmt.Errorf("无输入记录")
	}
	fields := pipeline.UnionFields(in)
	numeric := map[string]bool{}
	if inSc != nil {
		for f, ok := range inSc.Numeric {
			if ok {
				numeric[f] = true
			}
		}
	}
	objs := make([]map[string]any, 0, len(in))
	for _, r := range in {
		m := map[string]any{}
		for _, f := range fields {
			if v, ok := r[f]; ok && v != "" {
				if numeric[f] {
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						m[f] = n
						continue
					}
				}
				m[f] = v
			}
		}
		objs = append(objs, m)
	}
	var b []byte
	var err error
	if len(objs) == 1 {
		b, err = json.MarshalIndent(objs[0], "", "  ")
	} else {
		b, err = json.MarshalIndent(objs, "", "  ")
	}
	if err != nil {
		return nil, nil, err
	}
	return []pipeline.Record{{"line": string(b)}}, pipeline.LineSchema(), nil
}

func cmdFromCSV(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	var data string
	if len(rest) > 0 {
		b, err := os.ReadFile(c.Resolve(rest[0]))
		if err != nil {
			return nil, nil, err
		}
		data = string(b)
	} else if len(in) > 0 {
		var sb strings.Builder
		for _, r := range in {
			sb.WriteString(r["line"])
			sb.WriteString("\n")
		}
		data = sb.String()
	} else {
		return nil, nil, fmt.Errorf("缺少 CSV 输入（文件或管道）")
	}
	rd := csv.NewReader(strings.NewReader(data))
	rd.FieldsPerRecord = -1
	rows, err := rd.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("CSV 解析失败: %w", err)
	}
	if len(rows) < 2 {
		return nil, nil, fmt.Errorf("CSV 需要至少表头加一行数据")
	}
	header := rows[0]
	recs := make([]pipeline.Record, 0, len(rows)-1)
	for _, row := range rows[1:] {
		rec := pipeline.Record{}
		for i, h := range header {
			if i < len(row) && h != "" {
				rec[h] = row[i]
			}
		}
		recs = append(recs, rec)
	}
	return recs, schemaFor(recs), nil
}

func cmdToCSV(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	if len(in) == 0 {
		return nil, nil, fmt.Errorf("无输入记录")
	}
	fields := pipeline.UnionFields(in)
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	if err := w.Write(fields); err != nil {
		return nil, nil, err
	}
	for _, r := range in {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = r[f]
		}
		if err := w.Write(row); err != nil {
			return nil, nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, nil, err
	}
	return []pipeline.Record{{"line": strings.TrimSuffix(sb.String(), "\n")}}, pipeline.LineSchema(), nil
}
