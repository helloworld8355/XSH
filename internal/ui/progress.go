package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Progress 单行进度显示。
type Progress struct {
	w       io.Writer
	enabled bool
	msg     string
	unit    string // "B" 为字节型；其他为计数型单位
	total   int64
	mu      sync.Mutex
	done    int64
	last    int64
	rate    float64
	lastAt  time.Time
	stop    chan struct{}
	stopped sync.WaitGroup
}

// NewProgress 创建字节型进度显示；enabled 通常表示 w 为终端。
func NewProgress(w io.Writer, enabled bool, msg string, total int64) *Progress {
	return newProgress(w, enabled, msg, total, "B")
}

// NewProgressCount 创建计数型进度显示（端口数/条目数等非字节场景）。
func NewProgressCount(w io.Writer, enabled bool, msg string, total int64) *Progress {
	return newProgress(w, enabled, msg, total, "项")
}

func newProgress(w io.Writer, enabled bool, msg string, total int64, unit string) *Progress {
	p := &Progress{
		w: w, enabled: enabled, msg: msg, unit: unit, total: total,
		stop: make(chan struct{}), lastAt: time.Now(),
	}
	if enabled {
		p.stopped.Add(1)
		go p.loop()
	}
	return p
}

func (p *Progress) loop() {
	defer p.stopped.Done()
	t := time.NewTicker(400 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-t.C:
			p.render()
		}
	}
}

// Add 累加已完成字节数。
func (p *Progress) Add(n int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	p.done += n
	p.mu.Unlock()
}

// SetTotal 更新总字节数。
func (p *Progress) SetTotal(n int64) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	p.total = n
	p.mu.Unlock()
}

func (p *Progress) render() {
	p.mu.Lock()
	now := time.Now()
	dt := now.Sub(p.lastAt).Seconds()
	if dt > 0.2 {
		p.rate = float64(p.done-p.last) / dt
		p.last = p.done
		p.lastAt = now
	}
	done, total, rate := p.done, p.total, p.rate
	p.mu.Unlock()
	fmt.Fprintf(p.w, "\r  %s %s%s  %s   ", p.msg, p.fmtN(done), p.suffix(total), p.speedStr(rate))
}

func (p *Progress) fmtN(n int64) string {
	if p.unit == "B" {
		return HumanBytes(float64(n))
	}
	return fmt.Sprintf("%d%s", n, p.unit)
}

func (p *Progress) suffix(total int64) string {
	if total <= 0 {
		return ""
	}
	return "/" + p.fmtN(total)
}

// Done 结束进度显示。
func (p *Progress) Done() {
	close(p.stop)
	p.stopped.Wait()
	if p.enabled {
		p.render()
		fmt.Fprint(p.w, "\n")
	}
}

func (p *Progress) speedStr(rate float64) string {
	if rate <= 0 {
		return ""
	}
	if p.unit == "B" {
		return HumanBytes(rate) + "/s"
	}
	return fmt.Sprintf("%.1f/s", rate)
}

// HumanBytes 人类可读的字节数。
func HumanBytes(n float64) string {
	if n < 1024 {
		return fmt.Sprintf("%.0fB", n)
	}
	for _, u := range []string{"KB", "MB", "GB", "TB", "PB"} {
		n /= 1024
		if n < 1024 {
			return fmt.Sprintf("%.1f%s", n, u)
		}
	}
	return fmt.Sprintf("%.1fEB", n/1024)
}
