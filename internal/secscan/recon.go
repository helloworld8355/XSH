package secscan

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HostAlive 主机存活探测结果。
type HostAlive struct {
	IP    string
	Ports []int
}

// probePorts 主机发现使用的 TCP 探针端口（无原始套接字、无伪造包）。
var probePorts = []int{445, 139, 135, 22, 80, 443, 3389}

// DiscoverHosts TCP 探针方式探测网段存活主机。
func DiscoverHosts(cidr string, timeout time.Duration, concurrency int, progress func(done, total int)) []HostAlive {
	if concurrency < 1 {
		concurrency = 32
	}
	if timeout <= 0 {
		timeout = 800 * time.Millisecond
	}
	ips := expandCIDR(cidr)
	if len(ips) == 0 {
		return nil
	}
	results := make([]HostAlive, len(ips))
	idx := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				ip := ips[i]
				var hit []int
				for _, p := range probePorts {
					conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(p)), timeout)
					if err == nil {
						conn.Close()
						hit = append(hit, p)
						break
					}
				}
				results[i] = HostAlive{IP: ip, Ports: hit}
				mu.Lock()
				done++
				d, t := done, len(ips)
				mu.Unlock()
				if progress != nil {
					progress(d, t)
				}
			}
		}()
	}
	for i := range ips {
		idx <- i
	}
	close(idx)
	wg.Wait()
	var out []HostAlive
	for _, r := range results {
		if len(r.Ports) > 0 {
			out = append(out, r)
		}
	}
	return out
}

// expandCIDR 展开 CIDR 为 IP 列表（限 /16 规模；单 IP 原样返回）。
func expandCIDR(cidr string) []string {
	if ip := net.ParseIP(cidr); ip != nil {
		return []string{cidr}
	}
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	ones, bits := ipnet.Mask.Size()
	if bits-ones > 16 {
		return nil
	}
	var out []string
	for cur := ip.Mask(ipnet.Mask); ipnet.Contains(cur); incIP(cur) {
		out = append(out, cur.String())
		if len(out) >= 65534 {
			break
		}
	}
	if len(out) > 2 {
		return out[1 : len(out)-1]
	}
	return out
}

// CIDRSize 返回网段可探测 IP 数（0 表示无效或超出 /16 规模）。
func CIDRSize(cidr string) int {
	return len(expandCIDR(cidr))
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// DirResult 目录枚举结果。
type DirResult struct {
	Path   string
	Status int
	Size   int64
}

// DirBrute 目录枚举（默认内置字典，令牌桶限速），返回非 404 的路径。
func DirBrute(baseURL string, words []string, timeout time.Duration, qps float64, concurrency int, progress func(done, total int)) []DirResult {
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if len(words) == 0 {
		words = DirWordlist()
	}
	if qps <= 0 {
		qps = 20
	}
	if concurrency < 1 {
		concurrency = 8
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	limiter := rate.NewLimiter(rate.Limit(qps), int(qps)+1)
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	results := make([]DirResult, len(words))
	idx := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				_ = limiter.Wait(context.Background())
				resp, err := client.Get(baseURL + "/" + words[i])
				if err == nil {
					size := resp.ContentLength
					code := resp.StatusCode
					_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
					_ = resp.Body.Close()
					if code != 404 && code > 0 {
						results[i] = DirResult{Path: "/" + words[i], Status: code, Size: size}
					}
				}
				mu.Lock()
				done++
				d, t := done, len(words)
				mu.Unlock()
				if progress != nil {
					progress(d, t)
				}
			}
		}()
	}
	for i := range words {
		idx <- i
	}
	close(idx)
	wg.Wait()
	var out []DirResult
	for _, r := range results {
		if r.Status != 0 {
			out = append(out, r)
		}
	}
	return out
}

// SubResult 子域枚举结果。
type SubResult struct {
	Name string
	IP   string
}

// SubdomainEnum 用内置字典枚举子域。
func SubdomainEnum(domain string, words []string, concurrency int, progress func(done, total int)) []SubResult {
	if len(words) == 0 {
		words = SubdomainWords()
	}
	if concurrency < 1 {
		concurrency = 32
	}
	results := make([]SubResult, len(words))
	idx := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				name := words[i] + "." + domain
				if ips, err := net.LookupIP(name); err == nil && len(ips) > 0 {
					results[i] = SubResult{Name: name, IP: ips[0].String()}
				}
				mu.Lock()
				done++
				d, t := done, len(words)
				mu.Unlock()
				if progress != nil {
					progress(d, t)
				}
			}
		}()
	}
	for i := range words {
		idx <- i
	}
	close(idx)
	wg.Wait()
	var out []SubResult
	for _, r := range results {
		if r.IP != "" {
			out = append(out, r)
		}
	}
	return out
}
