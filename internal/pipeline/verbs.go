package pipeline

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Filter 按 where 语义过滤记录。op: eq ne gt lt ge le like match in。
func Filter(in []Record, field, op, value string) []Record {
	var out []Record
	for _, r := range in {
		if matchOp(r[field], op, value) {
			out = append(out, r)
		}
	}
	return out
}

func matchOp(a, op, b string) bool {
	switch op {
	case "eq":
		return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	case "ne":
		return !strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
	case "like":
		return strings.Contains(strings.ToLower(a), strings.ToLower(b))
	case "match":
		ok, err := regexp.MatchString(b, a)
		return err == nil && ok
	case "in":
		for _, part := range strings.Split(b, ",") {
			if strings.EqualFold(strings.TrimSpace(part), strings.TrimSpace(a)) {
				return true
			}
		}
		return false
	case "gt", "lt", "ge", "le":
		fa, ea := strconv.ParseFloat(strings.TrimSpace(a), 64)
		fb, eb := strconv.ParseFloat(strings.TrimSpace(b), 64)
		if ea == nil && eb == nil {
			return cmpFloat(op, fa, fb)
		}
		return cmpString(op, a, b)
	}
	return false
}

func cmpFloat(op string, a, b float64) bool {
	switch op {
	case "gt":
		return a > b
	case "lt":
		return a < b
	case "ge":
		return a >= b
	case "le":
		return a <= b
	}
	return false
}

func cmpString(op string, a, b string) bool {
	switch op {
	case "gt":
		return a > b
	case "lt":
		return a < b
	case "ge":
		return a >= b
	case "le":
		return a <= b
	}
	return false
}

// Project 投影字段（保留记录中存在的字段）。
func Project(in []Record, fields []string) []Record {
	out := make([]Record, 0, len(in))
	for _, r := range in {
		nr := Record{}
		for _, f := range fields {
			if v, ok := r[f]; ok {
				nr[f] = v
			}
		}
		out = append(out, nr)
	}
	return out
}

// SortRecords 就地排序；numericHint 提示字段为数值。
func SortRecords(in []Record, field string, desc, numericHint bool) {
	numeric := numericHint
	if !numeric {
		numeric = true
		for _, r := range in {
			if _, err := strconv.ParseFloat(strings.TrimSpace(r[field]), 64); err != nil {
				numeric = false
				break
			}
		}
	}
	sort.SliceStable(in, func(i, j int) bool {
		a, b := in[i][field], in[j][field]
		less := false
		if numeric {
			fa, _ := strconv.ParseFloat(strings.TrimSpace(a), 64)
			fb, _ := strconv.ParseFloat(strings.TrimSpace(b), 64)
			less = fa < fb
		} else {
			less = a < b
		}
		if desc {
			return !less && a != b
		}
		return less
	})
}

// Measure 统计：count 及可选字段的 sum/avg/min/max。
func Measure(in []Record, field string) Record {
	r := Record{"count": strconv.Itoa(len(in))}
	if field == "" || len(in) == 0 {
		return r
	}
	sum := 0.0
	min, max := math.Inf(1), math.Inf(-1)
	numeric := true
	for _, rec := range in {
		f, err := strconv.ParseFloat(strings.TrimSpace(rec[field]), 64)
		if err != nil {
			numeric = false
			break
		}
		sum += f
		if f < min {
			min = f
		}
		if f > max {
			max = f
		}
	}
	if numeric {
		r["sum"] = TrimF(sum)
		r["avg"] = TrimF(sum / float64(len(in)))
		r["min"] = TrimF(min)
		r["max"] = TrimF(max)
	}
	return r
}

// TrimF 格式化浮点并去掉多余的 0。
func TrimF(f float64) string {
	if math.IsInf(f, 0) {
		return "0"
	}
	s := strconv.FormatFloat(f, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Group 按字段分组计数（保持出现顺序）。
func Group(in []Record, field string) []Record {
	var order []string
	counts := map[string]int{}
	for _, r := range in {
		k := r[field]
		if _, ok := counts[k]; !ok {
			order = append(order, k)
		}
		counts[k]++
	}
	out := make([]Record, 0, len(order))
	for _, k := range order {
		out = append(out, Record{"value": k, "count": strconv.Itoa(counts[k])})
	}
	return out
}

// UnionFields 汇总所有记录的字段（按字母序）。
func UnionFields(in []Record) []string {
	set := map[string]bool{}
	for _, r := range in {
		for k := range r {
			set[k] = true
		}
	}
	fields := make([]string, 0, len(set))
	for k := range set {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return fields
}

// FirstN 取前 N 条。
func FirstN(in []Record, n int) []Record {
	if n >= len(in) {
		return in
	}
	return in[:n]
}

// LastN 取后 N 条。
func LastN(in []Record, n int) []Record {
	if n >= len(in) {
		return in
	}
	return in[len(in)-n:]
}
