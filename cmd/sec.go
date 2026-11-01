package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"xsh/internal/config"
	"xsh/internal/secscan"
	"xsh/internal/ui"
)

var (
	secLicense    string
	secAuthorized bool
	secJSON       bool
	secThreads    int
	secTimeout    int
	secScanFull   bool
	secDirQPS     float64
	wpUsers       []string
	wpPasses      []string
)

var secCmd = &cobra.Command{
	Use:   "sec <模块> <目标>",
	Short: "授权安全评估（需 license 与 --authorized）",
	Long: `授权安全评估（仅限已获得书面授权的目标，全程审计）:

  scan <host>                          端口扫描 + 服务识别 + 漏洞匹配（--full 全端口）
  hosts <cidr>                         网段主机发现（TCP 探针，限 /16）
  dir <url>                            目录/路径枚举（限速）
  axfr <domain>                        DNS 区域传送检测
  subdomain <domain>                   子域枚举
  cors <url>                           CORS 策略检测
  methods <url>                        HTTP 方法与危险方法检测
  tls <host[:port]>                    证书与 TLS 版本检查
  web <url>                            Web 安全响应头检测
  weakpass <host> <ssh|smb|mysql> [端口]  弱口令检测（默认字典，命中即停）

通用参数: --license <file> --authorized --json --threads N --timeout 秒`,
	SilenceUsage: true,
}

func init() {
	secCmd.PersistentFlags().StringVar(&secLicense, "license", "", "license 文件路径")
	secCmd.PersistentFlags().BoolVar(&secAuthorized, "authorized", false, "声明已获得目标系统的书面授权")
	secCmd.PersistentFlags().BoolVar(&secJSON, "json", false, "以 JSON 输出结果")
	secCmd.PersistentFlags().IntVar(&secThreads, "threads", 128, "并发数（端口扫描上限 512）")
	secCmd.PersistentFlags().IntVar(&secTimeout, "timeout", 3, "单次连接超时（秒）")
	secScanCmd.Flags().BoolVar(&secScanFull, "full", false, "全端口 1-65535 扫描")
	secDirCmd.Flags().Float64Var(&secDirQPS, "qps", 0, "枚举限速（默认取配置 dirscan_rate_per_sec=50）")
	secWeakpassCmd.Flags().StringSliceVar(&wpUsers, "users", nil, "自定义用户名字典（逗号分隔）")
	secWeakpassCmd.Flags().StringSliceVar(&wpPasses, "passes", nil, "自定义弱口令字典（逗号分隔）")
	secCmd.AddCommand(secScanCmd, secHostsCmd, secDirCmd, secAxfrCmd, secSubCmd,
		secCorsCmd, secMethodsCmd, secTlsCmd, secWebCmd, secWeakpassCmd)
}

// secGuard 校验授权门禁（license + 显式声明 + 审计）。
func secGuard() (*secscan.Guard, error) {
	path := secLicense
	if path == "" {
		p, err := defaultLicensePath()
		if err != nil {
			return nil, fmt.Errorf("%w（sec 模块需要 --license）", err)
		}
		path = p
	}
	return secscan.NewGuard(path, "cli", secAuthorized)
}

func secTimeoutDur() time.Duration {
	if secTimeout <= 0 {
		return 3 * time.Second
	}
	return time.Duration(secTimeout) * time.Second
}

func secOutJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func intJoin(xs []int) string {
	ss := make([]string, len(xs))
	for i, x := range xs {
		ss[i] = strconv.Itoa(x)
	}
	return strings.Join(ss, "/")
}

// ---------------- scan ----------------

var secScanCmd = &cobra.Command{
	Use:   "scan <host>",
	Short: "端口扫描 + 服务识别 + 漏洞匹配",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		host := args[0]
		g.Audit("sec.scan", host, "start")
		ui.InitConsole()
		var ports []secscan.PortService
		if secScanFull {
			ports = secscan.FullPorts()
		} else {
			ports = secscan.TopPorts(0)
		}
		ui.Title(os.Stdout, "端口扫描: "+host)
		ui.Dim(os.Stdout, fmt.Sprintf("请确认已获得目标书面授权 · 探测 %d 个端口", len(ports)))
		p := ui.NewProgressCount(os.Stdout, true, "扫描", int64(len(ports)))
		results := secscan.ScanPorts(host, ports, secTimeoutDur(), secThreads, func(done, total int) {
			p.Add(1)
		})
		p.Done()
		open := secscan.OpenPorts(results)
		findings := secscan.MatchVulns(results)
		sort.Slice(findings, func(i, j int) bool {
			return secscan.SeverityRank(findings[i].Severity) > secscan.SeverityRank(findings[j].Severity)
		})
		g.Audit("sec.scan", host, fmt.Sprintf("done open=%d findings=%d", len(open), len(findings)))
		if secJSON {
			return secOutJSON(map[string]any{"host": host, "open_ports": open, "findings": findings})
		}
		if len(open) == 0 {
			ui.Warn(os.Stdout, "未发现开放端口（主机可能不在线或被防火墙过滤）")
			return nil
		}
		rows := make([][]string, 0, len(open))
		for _, o := range open {
			rows = append(rows, []string{strconv.Itoa(o.Port), o.Service, o.Banner})
		}
		ui.Hdr(os.Stdout, fmt.Sprintf("开放端口 (%d)", len(open)))
		ui.PrintTable(os.Stdout, []string{"端口", "服务", "Banner"}, rows)
		if len(findings) > 0 {
			ui.Hdr(os.Stdout, "风险提示")
			for _, f := range findings {
				ui.Warn(os.Stdout, "[%s] %s（端口 %s）", strings.ToUpper(f.Severity), f.Title, intJoin(f.Ports))
				ui.Dim(os.Stdout, "  "+f.Advice)
			}
		} else {
			ui.OK(os.Stdout, "未匹配到已知风险规则")
		}
		return nil
	},
}

// ---------------- hosts ----------------

var secHostsCmd = &cobra.Command{
	Use:   "hosts <cidr>",
	Short: "网段主机发现（TCP 探针）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		cidr := args[0]
		total := secscan.CIDRSize(cidr)
		if total == 0 {
			return fmt.Errorf("无效 CIDR 或超出 /16 规模: %s", cidr)
		}
		g.Audit("sec.hosts", cidr, "start")
		ui.InitConsole()
		ui.Title(os.Stdout, "主机发现: "+cidr)
		ui.Dim(os.Stdout, fmt.Sprintf("TCP 探针方式 · 待探测 %d 个地址", total))
		p := ui.NewProgressCount(os.Stdout, true, "探测", int64(total))
		alive := secscan.DiscoverHosts(cidr, secTimeoutDur(), secThreads, func(done, t int) {
			p.Add(1)
		})
		p.Done()
		g.Audit("sec.hosts", cidr, fmt.Sprintf("done alive=%d", len(alive)))
		if secJSON {
			return secOutJSON(map[string]any{"cidr": cidr, "alive": alive})
		}
		if len(alive) == 0 {
			ui.Warn(os.Stdout, "网段内未发现存活主机")
			return nil
		}
		rows := make([][]string, 0, len(alive))
		for _, h := range alive {
			ports := make([]string, len(h.Ports))
			for i, pt := range h.Ports {
				ports[i] = strconv.Itoa(pt)
			}
			rows = append(rows, []string{h.IP, strings.Join(ports, ", ")})
		}
		ui.Hdr(os.Stdout, fmt.Sprintf("存活主机 (%d)", len(alive)))
		ui.PrintTable(os.Stdout, []string{"IP", "开放探针端口"}, rows)
		return nil
	},
}

// ---------------- dir ----------------

var secDirCmd = &cobra.Command{
	Use:   "dir <url>",
	Short: "目录/路径枚举（限速）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		target := args[0]
		g.Audit("sec.dir", target, "start")
		ui.InitConsole()
		words := secscan.DirWordlist()
		qps := secDirQPS
		if qps <= 0 {
			qps = float64(config.Load().DirScanRate)
		}
		ui.Title(os.Stdout, "目录枚举: "+target)
		ui.Dim(os.Stdout, fmt.Sprintf("内置字典 %d 项 · 限速 %.0f QPS", len(words), qps))
		p := ui.NewProgressCount(os.Stdout, true, "枚举", int64(len(words)))
		results := secscan.DirBrute(target, words, secTimeoutDur(), qps, secThreads, func(done, t int) {
			p.Add(1)
		})
		p.Done()
		g.Audit("sec.dir", target, fmt.Sprintf("done hits=%d", len(results)))
		if secJSON {
			return secOutJSON(map[string]any{"url": target, "found": results})
		}
		if len(results) == 0 {
			ui.Warn(os.Stdout, "未发现可访问路径")
			return nil
		}
		sort.Slice(results, func(i, j int) bool { return results[i].Status < results[j].Status })
		rows := make([][]string, 0, len(results))
		for _, r := range results {
			rows = append(rows, []string{r.Path, strconv.Itoa(r.Status), ui.HumanBytes(float64(r.Size))})
		}
		ui.Hdr(os.Stdout, fmt.Sprintf("发现路径 (%d)", len(results)))
		ui.PrintTable(os.Stdout, []string{"路径", "状态码", "大小"}, rows)
		return nil
	},
}

// ---------------- axfr ----------------

var secAxfrCmd = &cobra.Command{
	Use:   "axfr <domain>",
	Short: "DNS 区域传送检测",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		domain := args[0]
		g.Audit("sec.axfr", domain, "start")
		r, err := secscan.CheckAXFR(domain)
		if err != nil {
			return err
		}
		g.Audit("sec.axfr", domain, fmt.Sprintf("done vulnerable=%v", r.Vulnerable))
		ui.InitConsole()
		if secJSON {
			return secOutJSON(r)
		}
		ui.Title(os.Stdout, "AXFR 检测: "+domain)
		if len(r.NS) == 0 {
			ui.Warn(os.Stdout, "未查询到权威 NS 记录")
			return nil
		}
		ui.Field(os.Stdout, "权威 NS", strings.Join(r.NS, ", "))
		if r.Vulnerable {
			ui.Err(os.Stdout, "存在风险: %s 允许区域传送（AXFR）", r.NS[0])
			ui.Dim(os.Stdout, "  泄露全部内部解析记录，应在权威 DNS 上限制 AXFR 来源")
			ui.Hdr(os.Stdout, fmt.Sprintf("泄露记录（前 %d 条）", len(r.Records)))
			for _, rec := range r.Records {
				ui.Dim(os.Stdout, "  "+rec)
			}
		} else {
			ui.OK(os.Stdout, "各权威 NS 均拒绝区域传送")
		}
		return nil
	},
}

// ---------------- subdomain ----------------

var secSubCmd = &cobra.Command{
	Use:   "subdomain <domain>",
	Short: "子域枚举（内置字典）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		domain := args[0]
		g.Audit("sec.subdomain", domain, "start")
		ui.InitConsole()
		words := secscan.SubdomainWords()
		ui.Title(os.Stdout, "子域枚举: "+domain)
		ui.Dim(os.Stdout, fmt.Sprintf("内置字典 %d 项", len(words)))
		p := ui.NewProgressCount(os.Stdout, true, "解析", int64(len(words)))
		results := secscan.SubdomainEnum(domain, words, secThreads, func(done, t int) {
			p.Add(1)
		})
		p.Done()
		g.Audit("sec.subdomain", domain, fmt.Sprintf("done found=%d", len(results)))
		if secJSON {
			return secOutJSON(map[string]any{"domain": domain, "found": results})
		}
		if len(results) == 0 {
			ui.Warn(os.Stdout, "未发现子域")
			return nil
		}
		rows := make([][]string, 0, len(results))
		for _, s := range results {
			rows = append(rows, []string{s.Name, s.IP})
		}
		ui.Hdr(os.Stdout, fmt.Sprintf("发现子域 (%d)", len(results)))
		ui.PrintTable(os.Stdout, []string{"子域", "IP"}, rows)
		return nil
	},
}

// ---------------- cors ----------------

var secCorsCmd = &cobra.Command{
	Use:   "cors <url>",
	Short: "CORS 策略检测",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		target := args[0]
		g.Audit("sec.cors", target, "start")
		r, err := secscan.CheckCORS(target, secTimeoutDur())
		if err != nil {
			return err
		}
		g.Audit("sec.cors", target, fmt.Sprintf("done wildcard=%v reflect=%v", r.Wildcard, r.Reflect))
		ui.InitConsole()
		if secJSON {
			return secOutJSON(r)
		}
		ui.Title(os.Stdout, "CORS 检测: "+r.URL)
		switch {
		case r.Wildcard:
			ui.Err(os.Stdout, "风险: Access-Control-Allow-Origin: * （任意来源可跨域读取响应）")
			ui.Dim(os.Stdout, "  若接口返回敏感数据，应改为白名单 Origin")
		case r.Reflect:
			ui.Err(os.Stdout, "风险: 服务端反射任意 Origin（等于对所有来源开放）")
			ui.Dim(os.Stdout, "  应使用固定的 Origin 白名单而非原样回显")
		default:
			ui.OK(os.Stdout, "未发现宽松 CORS 策略")
		}
		return nil
	},
}

// ---------------- methods ----------------

var secMethodsCmd = &cobra.Command{
	Use:   "methods <url>",
	Short: "HTTP 方法与危险方法检测",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		target := args[0]
		g.Audit("sec.methods", target, "start")
		r, err := secscan.CheckMethods(target, secTimeoutDur())
		if err != nil {
			return err
		}
		g.Audit("sec.methods", target, fmt.Sprintf("done dangerous=%d", len(r.Dangerous)))
		ui.InitConsole()
		if secJSON {
			return secOutJSON(r)
		}
		ui.Title(os.Stdout, "HTTP 方法检测: "+r.URL)
		if len(r.Allowed) > 0 {
			ui.Field(os.Stdout, "允许方法", strings.Join(r.Allowed, ", "))
		} else {
			ui.Dim(os.Stdout, "OPTIONS 未返回 Allow/A-C-Allow-Methods")
		}
		if len(r.Dangerous) > 0 {
			ui.Err(os.Stdout, "风险: 危险方法可用 %s", strings.Join(r.Dangerous, ", "))
			ui.Dim(os.Stdout, "  PUT/DELETE 可写删内容、TRACE 可致 XST，应在网关/服务端禁用")
		} else {
			ui.OK(os.Stdout, "PUT/DELETE/TRACE 均不可用")
		}
		return nil
	},
}

// ---------------- tls ----------------

var secTlsCmd = &cobra.Command{
	Use:   "tls <host[:port]>",
	Short: "证书与 TLS 版本检查",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		target := args[0]
		g.Audit("sec.tls", target, "start")
		r, err := secscan.CheckTLS(target, secTimeoutDur())
		if err != nil {
			return err
		}
		g.Audit("sec.tls", target, fmt.Sprintf("done version=%s expired=%v", r.Version, r.Expired))
		ui.InitConsole()
		if secJSON {
			return secOutJSON(r)
		}
		ui.Title(os.Stdout, "TLS 检查: "+r.Host)
		ui.Field(os.Stdout, "协议版本", r.Version)
		ui.Field(os.Stdout, "加密套件", r.Cipher)
		ui.Field(os.Stdout, "颁发者", r.Issuer)
		ui.Field(os.Stdout, "主题", r.Subject)
		ui.Field(os.Stdout, "到期时间", r.NotAfter.Format("2006-01-02 15:04:05"))
		switch {
		case r.Expired:
			ui.Err(os.Stdout, "风险: 证书已过期 %d 天", -r.DaysLeft)
		case r.DaysLeft < 30:
			ui.Warn(os.Stdout, "证书剩余 %d 天有效期，请及时续期", r.DaysLeft)
		default:
			ui.OK(os.Stdout, "证书有效期剩余 %d 天", r.DaysLeft)
		}
		if r.WeakVersion {
			ui.Err(os.Stdout, "风险: 使用弱协议 %s（应至少启用 TLS 1.2）", r.Version)
		}
		if r.SelfSigned {
			ui.Warn(os.Stdout, "证书为自签名（内部系统常见，公网服务应使用受信 CA）")
		}
		return nil
	},
}

// ---------------- web ----------------

var secWebCmd = &cobra.Command{
	Use:   "web <url>",
	Short: "Web 安全响应头检测",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		target := args[0]
		g.Audit("sec.web", target, "start")
		r, err := secscan.CheckWeb(target, secTimeoutDur())
		if err != nil {
			return err
		}
		g.Audit("sec.web", target, fmt.Sprintf("done missing=%d", len(r.MissingHeaders)))
		ui.InitConsole()
		if secJSON {
			return secOutJSON(r)
		}
		ui.Title(os.Stdout, "Web 检测: "+r.URL)
		ui.Field(os.Stdout, "状态", r.Status)
		if r.Server != "" {
			ui.Field(os.Stdout, "Server", r.Server)
		}
		if r.Title != "" {
			ui.Field(os.Stdout, "页面标题", r.Title)
		}
		if len(r.MissingHeaders) > 0 {
			ui.Warn(os.Stdout, "缺少安全响应头:")
			for _, h := range r.MissingHeaders {
				ui.Dim(os.Stdout, "  - "+h)
			}
		} else {
			ui.OK(os.Stdout, "安全响应头基线完整")
		}
		return nil
	},
}

// ---------------- weakpass ----------------

var secWeakpassCmd = &cobra.Command{
	Use:   "weakpass <host> <ssh|smb|mysql> [端口]",
	Short: "弱口令检测（默认字典，命中即停）",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		g, err := secGuard()
		if err != nil {
			return err
		}
		host, service := args[0], strings.ToLower(args[1])
		defPort := 0
		switch service {
		case "ssh":
			defPort = 22
		case "smb":
			defPort = 445
		case "mysql":
			defPort = 3306
		default:
			return fmt.Errorf("不支持的服务 %q（支持 ssh/smb/mysql）", service)
		}
		port := defPort
		if len(args) == 3 {
			p, err := strconv.Atoi(args[2])
			if err != nil || p <= 0 || p > 65535 {
				return fmt.Errorf("无效端口 %q", args[2])
			}
			port = p
		}
		g.Audit("sec.weakpass", fmt.Sprintf("%s:%d/%s", host, port, service), "start")
		ui.InitConsole()
		users, passes := wpUsers, wpPasses
		nu, np := len(users), len(passes)
		if nu == 0 {
			nu = len(secscan.DefaultUsers)
		}
		if np == 0 {
			np = len(secscan.DefaultPasses)
		}
		ui.Title(os.Stdout, fmt.Sprintf("弱口令检测: %s:%d (%s)", host, port, service))
		ui.Dim(os.Stdout, fmt.Sprintf("最多尝试 %d 组凭据（命中即停，受控串行）", nu*np))
		results := secscan.TryCreds(host, port, service, users, passes, secTimeoutDur())
		var hit *secscan.CredResult
		for i := range results {
			if results[i].OK {
				hit = &results[i]
				break
			}
		}
		g.Audit("sec.weakpass", host, fmt.Sprintf("done tried=%d hit=%v", len(results), hit != nil))
		if secJSON {
			return secOutJSON(map[string]any{
				"host": host, "port": port, "service": service,
				"tried": len(results), "hit": hit,
			})
		}
		ui.Info(os.Stdout, "已尝试 %d 组凭据", len(results))
		if hit != nil {
			ui.Err(os.Stdout, "发现弱口令: %s / %s（%s）", hit.User, hit.Pass, hit.Note)
			ui.Dim(os.Stdout, "  请立即修改为长随机口令并启用失败锁定")
		} else {
			ui.OK(os.Stdout, "默认字典未命中弱口令")
		}
		return nil
	},
}
