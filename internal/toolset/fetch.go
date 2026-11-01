// Package toolset 提供独立小工具：多线程下载、哈希、编码、剪贴板、静态服务、Wi-Fi、启动项、系统快照。
package toolset

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"xsh/internal/ui"
)

// FetchOptions 下载选项。
type FetchOptions struct {
	URL      string
	Dest     string
	Threads  int
	Insecure bool
	Progress bool
	Out      io.Writer
}

const (
	fetchPartSuffix = ".xshpart"
	fetchMinMulti   = 512 << 10 // 小于 512KB 直接单线程
)

// Fetch 多线程断点续传下载，返回本次实际写入的字节数。
func Fetch(opt FetchOptions) (int64, error) {
	if opt.URL == "" {
		return 0, fmt.Errorf("缺少 URL")
	}
	if !strings.Contains(opt.URL, "://") {
		opt.URL = "http://" + opt.URL
	}
	if opt.Threads < 1 {
		opt.Threads = 4
	}
	if opt.Threads > 16 {
		opt.Threads = 16
	}
	if opt.Dest == "" {
		opt.Dest = fileNameFromURL(opt.URL)
	}
	if opt.Dest == "" || opt.Dest == "." || opt.Dest == "/" || opt.Dest == `\` {
		opt.Dest = "download.bin"
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: opt.Insecure},
			DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
	size, rangeOK, err := fetchProbe(client, opt.URL)
	if err != nil {
		return 0, err
	}
	if size > 0 && rangeOK && opt.Threads > 1 && size >= fetchMinMulti {
		return fetchMulti(client, opt, size)
	}
	return fetchSingle(client, opt, size, rangeOK)
}

// fetchProbe 探测文件大小与 Range 支持。
func fetchProbe(client *http.Client, rawURL string) (size int64, rangeOK bool, err error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return 0, false, err
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}()
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		if i := strings.LastIndex(cr, "/"); i >= 0 {
			if n, e := strconv.ParseInt(strings.TrimSpace(cr[i+1:]), 10, 64); e == nil {
				size = n
			}
		}
		rangeOK = true
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			size, _ = strconv.ParseInt(cl, 10, 64)
		}
	} else {
		return 0, false, fmt.Errorf("HTTP %s", resp.Status)
	}
	if size < 0 {
		size = 0
	}
	return size, rangeOK, nil
}

// fetchSingle 单线程下载；服务器支持 Range 且目标已存在部分内容时续传。
func fetchSingle(client *http.Client, opt FetchOptions, size int64, rangeOK bool) (int64, error) {
	var start int64
	if rangeOK && size > 0 {
		if fi, err := os.Stat(opt.Dest); err == nil && fi.Size() > 0 && fi.Size() < size {
			start = fi.Size()
		}
	}
	req, err := http.NewRequest("GET", opt.URL, nil)
	if err != nil {
		return 0, err
	}
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		return 0, fmt.Errorf("HTTP %s", resp.Status)
	}
	var f *os.File
	if start > 0 && resp.StatusCode == 206 {
		if f, err = os.OpenFile(opt.Dest, os.O_APPEND|os.O_WRONLY, 0o644); err != nil {
			return 0, err
		}
		ui.Info(opt.Out, "断点续传: 从 %s 继续", ui.HumanBytes(float64(start)))
	} else {
		start = 0
		if f, err = os.Create(opt.Dest); err != nil {
			return 0, err
		}
	}
	defer f.Close()
	total := size - start
	if total < 0 {
		total = 0
	}
	p := ui.NewProgress(opt.Out, opt.Progress, "下载", total)
	written, err := copyProgress(f, resp.Body, p)
	p.Done()
	if err != nil {
		return written, err
	}
	ui.OK(opt.Out, "已下载 %s → %s", ui.HumanBytes(float64(start+written)), opt.Dest)
	return written, nil
}

// fetchMulti 分段并发下载；每段独立分片文件，失败后重试自动续传。
func fetchMulti(client *http.Client, opt FetchOptions, size int64) (int64, error) {
	type segment struct {
		idx       int
		start, to int64
	}
	n := opt.Threads
	segs := make([]segment, n)
	step := size / int64(n)
	var s int64
	for i := 0; i < n; i++ {
		to := s + step
		if i == n-1 {
			to = size
		}
		segs[i] = segment{i, s, to}
		s = to
	}
	var done0 int64
	for _, sg := range segs {
		if fi, err := os.Stat(fetchPartName(opt.Dest, sg.idx)); err == nil {
			done0 += fi.Size()
		}
	}
	if done0 > 0 {
		ui.Info(opt.Out, "断点续传: 已完成 %s / %s", ui.HumanBytes(float64(done0)), ui.HumanBytes(float64(size)))
	}
	p := ui.NewProgress(opt.Out, opt.Progress, "下载", size)
	p.Add(done0)
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for _, sg := range segs {
		wg.Add(1)
		go func(sg segment) {
			defer wg.Done()
			if err := fetchSegment(client, opt, sg.start, sg.to, sg.idx, p); err != nil {
				errCh <- fmt.Errorf("分段 %d/%d: %w", sg.idx+1, n, err)
			}
		}(sg)
	}
	wg.Wait()
	close(errCh)
	p.Done()
	if err := <-errCh; err != nil {
		return 0, fmt.Errorf("下载失败（重试将续传）: %w", err)
	}
	out, err := os.Create(opt.Dest)
	if err != nil {
		return 0, err
	}
	var merged int64
	for i := 0; i < n; i++ {
		pf, err := os.Open(fetchPartName(opt.Dest, i))
		if err != nil {
			out.Close()
			return 0, fmt.Errorf("分片缺失: %w", err)
		}
		cnt, err := io.Copy(out, pf)
		pf.Close()
		if err != nil {
			out.Close()
			return 0, err
		}
		merged += cnt
	}
	if err := out.Close(); err != nil {
		return 0, err
	}
	if merged != size {
		return 0, fmt.Errorf("合并后大小 %d 与预期 %d 不符，分片已保留", merged, size)
	}
	for i := 0; i < n; i++ {
		_ = os.Remove(fetchPartName(opt.Dest, i))
	}
	ui.OK(opt.Out, "已下载 %s → %s（%d 线程）", ui.HumanBytes(float64(merged)), opt.Dest, n)
	return merged, nil
}

// fetchSegment 下载一段到独立分片文件（可续传）。
func fetchSegment(client *http.Client, opt FetchOptions, from, to int64, idx int, p *ui.Progress) error {
	part := fetchPartName(opt.Dest, idx)
	cur := from
	if fi, err := os.Stat(part); err == nil {
		if have := fi.Size(); from+have <= to {
			cur = from + have
		} else {
			_ = os.Remove(part)
		}
	}
	if cur >= to {
		f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			f.Close()
		}
		return nil
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	req, err := http.NewRequest("GET", opt.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", cur, to-1))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 206 {
		return fmt.Errorf("服务器不支持分段 (HTTP %s)", resp.Status)
	}
	_, err = copyProgress(f, resp.Body, p)
	return err
}

// copyProgress 拷贝并上报进度。
func copyProgress(dst io.Writer, src io.Reader, p *ui.Progress) (int64, error) {
	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return written, werr
			}
			written += int64(n)
			p.Add(int64(n))
		}
		if err == io.EOF {
			return written, nil
		}
		if err != nil {
			return written, err
		}
	}
}

func fetchPartName(dest string, idx int) string {
	return fmt.Sprintf("%s.%d%s", dest, idx, fetchPartSuffix)
}

// fileNameFromURL 从 URL 提取文件名。
func fileNameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	name := path.Base(u.Path)
	if name == "" || name == "/" || name == "." || name == `\` {
		return ""
	}
	return name
}
