// Package syscheck 系统健康检查：CPU/内存/磁盘/服务/启动项/事件日志等检查项与综合评分。
package syscheck

import (
	"os"
	"time"
)

// Severity 严重级别。
type Severity int

// 严重级别定义。
const (
	SevOK Severity = iota
	SevInfo
	SevWarn
	SevCrit
)

// String 返回级别文本。
func (s Severity) String() string {
	switch s {
	case SevOK:
		return "通过"
	case SevInfo:
		return "提示"
	case SevWarn:
		return "警告"
	case SevCrit:
		return "严重"
	}
	return "?"
}

// Check 单项检查结果。
type Check struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Sev    Severity `json:"sev"`
	Score  int      `json:"score"`
	Detail string   `json:"detail"`
	Advice string   `json:"advice,omitempty"`
}

// Report 检查报告。
type Report struct {
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Checks   []Check   `json:"checks"`
	Total    int       `json:"total"`
	Grade    string    `json:"grade"`
}

type checker struct {
	id string
	fn func() Check
}

var allCheckers []checker

func registerChecker(id string, fn func() Check) {
	allCheckers = append(allCheckers, checker{id: id, fn: fn})
}

// Run 执行全量检查（阻塞约 3-5 秒）。
func Run() (*Report, error) {
	hn, _ := os.Hostname()
	r := &Report{Time: time.Now(), Hostname: hn}
	var sum, n int
	for _, c := range allCheckers {
		ch := c.fn()
		if ch.ID == "" {
			ch.ID = c.id
		}
		if ch.Name == "" {
			ch.Name = c.id
		}
		r.Checks = append(r.Checks, ch)
		sum += ch.Score
		n++
	}
	if n > 0 {
		r.Total = sum * 100 / (n * 10)
	}
	r.Grade = gradeOf(r.Total)
	return r, nil
}

func gradeOf(total int) string {
	switch {
	case total >= 90:
		return "A"
	case total >= 75:
		return "B"
	case total >= 60:
		return "C"
	}
	return "D"
}
