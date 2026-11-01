package diag

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"

	"xsh/internal/config"
	"xsh/internal/sysinfo"
	"xsh/internal/ui"
)

// Net 网络诊断：接口 → 网关 → DNS → HTTP 探测。
func (r *Runner) Net() error {
	ui.Title(r.Out, "网络诊断")
	cfg := config.Load()
	var problems []string

	ifs, _ := gnet.Interfaces()
	var upLines []string
	for _, i := range ifs {
		if !strings.Contains(strings.ToLower(strings.Join(i.Flags, " ")), "up") || len(i.Addrs) == 0 {
			continue
		}
		var addrs []string
		for _, a := range i.Addrs {
			addrs = append(addrs, a.Addr)
		}
		upLines = append(upLines, fmt.Sprintf("%s (%s)", i.Name, strings.Join(addrs, ",")))
	}
	if len(upLines) == 0 {
		ui.Err(r.Out, "没有处于连接状态的网络接口")
		problems = append(problems, "网络接口未连接（检查网线/Wi-Fi 开关）")
	} else {
		ui.OK(r.Out, "活动接口: %s", strings.Join(upLines, " | "))
	}

	gw := sysinfo.DefaultGateway()
	if gw == "" {
		ui.Err(r.Out, "未获取到默认网关（DHCP 配置异常或未连网）")
		problems = append(problems, "无默认网关")
	} else {
		out, _ := exec.Command("ping", "-n", "3", "-w", "1000", gw).Output()
		s := string(out)
		switch {
		case strings.Contains(s, "TTL=") || strings.Contains(s, "时间="):
			if strings.Contains(s, "丢失 = 0") || strings.Contains(s, "Lost = 0") {
				ui.OK(r.Out, "网关 %s 可达（无丢包）", gw)
			} else {
				ui.Warn(r.Out, "网关 %s 可达但有丢包", gw)
				problems = append(problems, "到网关存在丢包（检查 Wi-Fi 信号或网线）")
			}
		default:
			ui.Err(r.Out, "网关 %s 不可达", gw)
			problems = append(problems, "网关不可达（重启路由器/光猫）")
		}
	}

	domain := cfg.ProbeDomain
	if domain == "" {
		domain = "www.msftconnecttest.com"
	}
	t0 := time.Now()
	ips, err := net.LookupIP(domain)
	dnsMs := time.Since(t0).Milliseconds()
	if err != nil {
		ui.Err(r.Out, "DNS 解析 %s 失败: %v", domain, err)
		problems = append(problems, "DNS 解析失败（`ipconfig /flushdns` 或更换 DNS 为 223.5.5.5/119.29.29.29）")
	} else {
		ui.OK(r.Out, "DNS 解析 %s → %s (%dms)", domain, firstIP(ips), dnsMs)
	}

	probeURL := cfg.ProbeURL
	if probeURL == "" {
		probeURL = "https://www.msftconnecttest.com/connecttest.txt"
	}
	t0 = time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(probeURL)
	totalMs := time.Since(t0).Milliseconds()
	if err != nil {
		ui.Err(r.Out, "HTTP 探测失败: %v", err)
		problems = append(problems, "无法访问外网（若 DNS 正常，检查代理/防火墙）")
	} else {
		resp.Body.Close()
		ui.OK(r.Out, "HTTP 探测 %s → %s (%dms)", probeURL, resp.Status, totalMs)
		if totalMs > 2000 {
			ui.Warn(r.Out, "访问延迟偏高")
			problems = append(problems, "外网延迟偏高（联系运营商或更换线路）")
		}
	}

	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if v := os.Getenv(k); v != "" {
			ui.Info(r.Out, "检测到代理 %s=%s", k, v)
		}
	}

	ui.Sep(r.Out, 60)
	if len(problems) == 0 {
		ui.OK(r.Out, "网络一切正常")
	} else {
		ui.Warn(r.Out, "发现 %d 个问题:", len(problems))
		for i, p := range problems {
			fmt.Fprintf(r.Out, "  %d. %s\n", i+1, p)
		}
	}
	return nil
}

func firstIP(ips []net.IP) string {
	if len(ips) == 0 {
		return "?"
	}
	return ips[0].String()
}
