package secscan

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WebReport Web 服务检查结果。
type WebReport struct {
	URL            string
	Status         string
	Server         string
	Title          string
	MissingHeaders []string
}

// requiredSecurityHeaders 安全响应头基线。
var requiredSecurityHeaders = []string{
	"Strict-Transport-Security",
	"Content-Security-Policy",
	"X-Frame-Options",
	"X-Content-Type-Options",
	"Referrer-Policy",
}

// CheckWeb 抓取页面并分析安全响应头。
func CheckWeb(rawURL string, timeout time.Duration) (*WebReport, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	r := &WebReport{URL: rawURL, Status: resp.Status, Server: resp.Header.Get("Server")}
	for _, h := range requiredSecurityHeaders {
		if resp.Header.Get(h) == "" {
			r.MissingHeaders = append(r.MissingHeaders, h)
		}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	r.Title = extractTitle(string(body))
	return r, nil
}

func extractTitle(body string) string {
	lower := strings.ToLower(body)
	i := strings.Index(lower, "<title")
	if i < 0 {
		return ""
	}
	j := strings.Index(lower[i:], ">")
	if j < 0 {
		return ""
	}
	k := strings.Index(lower[i+j:], "</title>")
	if k < 0 {
		return ""
	}
	return strings.TrimSpace(body[i+j+1 : i+j+k])
}

// CORSResult CORS 策略检测结果。
type CORSResult struct {
	URL      string
	Wildcard bool
	Reflect  bool
}

// CheckCORS 检测 CORS：通配符或反射任意 Origin 均为宽松策略。
func CheckCORS(rawURL string, timeout time.Duration) (*CORSResult, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	probe := "https://xsh-cors-probe.example.com"
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", probe)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	acao := resp.Header.Get("Access-Control-Allow-Origin")
	return &CORSResult{
		URL:      rawURL,
		Wildcard: acao == "*",
		Reflect:  acao == probe,
	}, nil
}

// MethodsResult HTTP 方法检测结果。
type MethodsResult struct {
	URL       string
	Allowed   []string
	Dangerous []string
}

// CheckMethods 检测 OPTIONS 允许的方法与危险方法（PUT/DELETE/TRACE）可用性。
func CheckMethods(rawURL string, timeout time.Duration) (*MethodsResult, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	r := &MethodsResult{URL: rawURL}
	req, err := http.NewRequest("OPTIONS", rawURL, nil)
	if err == nil {
		if resp, err := client.Do(req); err == nil {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			r.Allowed = parseCSV(resp.Header.Get("Allow"))
			if len(r.Allowed) == 0 {
				r.Allowed = parseCSV(resp.Header.Get("Access-Control-Allow-Methods"))
			}
		}
	}
	for _, m := range []string{"PUT", "DELETE", "TRACE"} {
		req, err := http.NewRequest(m, rawURL, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		code := resp.StatusCode
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if code >= 200 && code < 300 {
			r.Dangerous = append(r.Dangerous, fmt.Sprintf("%s (HTTP %d)", m, code))
		}
	}
	return r, nil
}

func parseCSV(s string) []string {
	var out []string
	for _, f := range strings.Split(s, ",") {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
