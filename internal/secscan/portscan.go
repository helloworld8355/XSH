package secscan

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PortResult 端口扫描结果。
type PortResult struct {
	Port    int
	Open    bool
	Service string
	Banner  string
}

// ScanPorts 并发 TCP connect 扫描（worker 池模型）。
func ScanPorts(host string, ports []PortService, timeout time.Duration, concurrency int, progress func(done, total int)) []PortResult {
	if concurrency < 1 {
		concurrency = 64
	}
	if concurrency > 512 {
		concurrency = 512
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	results := make([]PortResult, len(ports))
	idx := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				ps := ports[i]
				r := PortResult{Port: ps.Port, Service: ps.Service}
				conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(ps.Port)), timeout)
				if err == nil {
					r.Open = true
					r.Banner = grabBanner(conn)
					conn.Close()
				}
				results[i] = r
				mu.Lock()
				done++
				d, t := done, len(ports)
				mu.Unlock()
				if progress != nil {
					progress(d, t)
				}
			}
		}()
	}
	for i := range ports {
		idx <- i
	}
	close(idx)
	wg.Wait()
	return results
}

// grabBanner 尝试读取服务 banner（1.2 秒超时，最多 256 字节，控制字符替换）。
func grabBanner(conn net.Conn) string {
	_ = conn.SetReadDeadline(time.Now().Add(1200 * time.Millisecond))
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, n)
	for _, c := range buf[:n] {
		switch {
		case c == '\r' || c == '\n':
			out = append(out, ' ')
		case c >= 32 && c < 127:
			out = append(out, c)
		}
	}
	return strings.TrimSpace(string(out))
}

// OpenPorts 过滤出开放端口。
func OpenPorts(results []PortResult) []PortResult {
	var out []PortResult
	for _, r := range results {
		if r.Open {
			out = append(out, r)
		}
	}
	return out
}
