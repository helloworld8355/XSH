package secscan

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// AXFRResult DNS 区域传送测试结果。
type AXFRResult struct {
	Domain     string
	NS         []string
	Vulnerable bool
	Records    []string
}

// CheckAXFR 检测域名的权威 NS 是否允许区域传送（AXFR）。
func CheckAXFR(domain string) (*AXFRResult, error) {
	r := &AXFRResult{Domain: domain}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeNS)
	resp, err := dnsExchange(m, "")
	if err != nil {
		return nil, err
	}
	for _, rr := range append(resp.Answer, resp.Ns...) {
		if ns, ok := rr.(*dns.NS); ok {
			r.NS = append(r.NS, strings.TrimSuffix(ns.Ns, "."))
		}
	}
	for _, ns := range r.NS {
		rrs := tryAXFR(domain, ns)
		if len(rrs) > 0 {
			r.Vulnerable = true
			for _, rr := range rrs {
				r.Records = append(r.Records, rr.String())
				if len(r.Records) >= 100 {
					break
				}
			}
			break
		}
	}
	return r, nil
}

// dnsExchange 向公共解析器（或指定服务器）发起 UDP 查询，重试一次。
func dnsExchange(m *dns.Msg, server string) (*dns.Msg, error) {
	if server == "" {
		server = "8.8.8.8"
	}
	if !strings.Contains(server, ":") {
		server += ":53"
	}
	client := &dns.Client{Timeout: 6 * time.Second, Net: "udp"}
	var lastErr error
	for i := 0; i < 2; i++ {
		resp, _, err := client.Exchange(m, server)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("DNS 查询失败: %w", lastErr)
}

// tryAXFR 尝试对 NS 发起 TCP 区域传送。
func tryAXFR(domain, ns string) []dns.RR {
	server := ns
	if !strings.Contains(server, ":") {
		server += ":53"
	}
	t := new(dns.Transfer)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeAXFR)
	env, err := t.In(m, server)
	if err != nil {
		return nil
	}
	var rrs []dns.RR
	for e := range env {
		if e.Error != nil {
			return nil
		}
		rrs = append(rrs, e.RR...)
		if len(rrs) >= 200 {
			break
		}
	}
	return rrs
}
