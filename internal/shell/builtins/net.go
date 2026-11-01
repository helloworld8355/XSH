package builtins

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/ui"
)

func registerNet(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "ipconfig", Aliases: []string{"Get-NetIPAddress"}, Summary: "网络接口信息", Fn: cmdIPConfig})
	reg.Add(&shell.Command{Name: "ping", Summary: "Ping <主机> [-n 次数]", Fn: cmdPing})
	reg.Add(&shell.Command{Name: "dns", Aliases: []string{"Resolve-DnsName"}, Summary: "DNS 解析 <名称> [a|cname|ns|mx|txt]", Fn: cmdDNS})
	reg.Add(&shell.Command{Name: "curl", Aliases: []string{"Invoke-WebRequest", "wget"}, Summary: "HTTP 请求 [-X 方法] [-d 数据] [-o 保存] [-k] [-L] <URL>", Fn: cmdCurl})
	reg.Add(&shell.Command{Name: "ports", Aliases: []string{"Get-NetTCPConnection"}, Summary: "网络连接 [PID过滤]", Fn: cmdPorts})
}

func cmdIPConfig(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	ifs, err := gnet.Interfaces()
	if err != nil {
		return nil, nil, err
	}
	var recs []pipeline.Record
	for _, i := range ifs {
		var addrs []string
		for _, a := range i.Addrs {
			addrs = append(addrs, a.Addr)
		}
		state := "down"
		if strings.Contains(strings.ToLower(strings.Join(i.Flags, " ")), "up") {
			state = "up"
		}
		recs = append(recs, pipeline.Record{
			"name":  i.Name,
			"ip":    strings.Join(addrs, ","),
			"mac":   strings.ToUpper(strings.ReplaceAll(i.HardwareAddr, ":", "-")),
			"mtu":   strconv.Itoa(int(i.MTU)),
			"state": state,
		})
	}
	sc := pipeline.NewSchema("name", "ip", "mac", "mtu", "state").WithNumeric("mtu")
	return recs, sc, nil
}

func cmdPing(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"n": true})
	host := first(rest)
	if host == "" {
		return argErr("ping", "<主机> [-n 次数]")
	}
	n := atoi(flags["n"], 4)
	if n < 1 || n > 50 {
		n = 4
	}
	cmd := exec.Command("ping", "-n", strconv.Itoa(n), "-w", "1000", host)
	cmd.Stdout = c.Out
	cmd.Stderr = c.Out
	_ = cmd.Run()
	return nil, nil, nil
}

func cmdDNS(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	name := first(rest)
	if name == "" {
		return argErr("dns", "<名称> [a|cname|ns|mx|txt]")
	}
	typ := "a"
	if len(rest) > 1 {
		typ = strings.ToLower(rest[1])
	}
	var recs []pipeline.Record
	var err error
	switch typ {
	case "a":
		var addrs []net.IP
		addrs, err = net.LookupIP(name)
		if err == nil {
			for _, ip := range addrs {
				recs = append(recs, pipeline.Record{"name": name, "type": "a", "value": ip.String()})
			}
		}
	case "cname":
		var cn string
		cn, err = net.LookupCNAME(name)
		if err == nil {
			recs = append(recs, pipeline.Record{"name": name, "type": "cname", "value": strings.TrimSuffix(cn, ".")})
		}
	case "ns":
		var nss []*net.NS
		nss, err = net.LookupNS(name)
		if err == nil {
			for _, ns := range nss {
				recs = append(recs, pipeline.Record{"name": name, "type": "ns", "value": ns.Host})
			}
		}
	case "mx":
		var mxs []*net.MX
		mxs, err = net.LookupMX(name)
		if err == nil {
			for _, mx := range mxs {
				recs = append(recs, pipeline.Record{"name": name, "type": "mx", "value": fmt.Sprintf("%s (优先级 %d)", mx.Host, mx.Pref)})
			}
		}
	case "txt":
		var txts []string
		txts, err = net.LookupTXT(name)
		if err == nil {
			for _, t := range txts {
				recs = append(recs, pipeline.Record{"name": name, "type": "txt", "value": t})
			}
		}
	default:
		return argErr("dns", "<名称> [a|cname|ns|mx|txt]")
	}
	if err != nil {
		return nil, nil, err
	}
	return recs, pipeline.NewSchema("name", "type", "value"), nil
}

func cmdCurl(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, map[string]bool{"X": true, "d": true, "H": true, "o": true, "m": true, "A": true})
	url := first(rest)
	if url == "" {
		return argErr("curl", "[-X 方法] [-H 头] [-d 数据] [-o 保存] [-k] [-L] <URL>")
	}
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	method := flags["X"]
	var body io.Reader
	if flags["d"] != "" {
		body = strings.NewReader(flags["d"])
		if method == "" {
			method = "POST"
		}
	}
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "xsh-curl")
	if flags["A"] != "" {
		req.Header.Set("User-Agent", flags["A"])
	}
	if flags["H"] != "" {
		if kv := strings.SplitN(flags["H"], ":", 2); len(kv) == 2 {
			req.Header.Set(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
		}
	}
	timeout := 30
	if flags["m"] != "" {
		timeout = atoi(flags["m"], 30)
	}
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: flags["k"] != ""},
	}
	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: tr,
	}
	if flags["L"] == "" {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	ui.Info(c.Out, "HTTP %s", resp.Status)
	if flags["o"] != "" {
		f, err := os.Create(c.Resolve(flags["o"]))
		if err != nil {
			return nil, nil, err
		}
		n, err := io.Copy(f, resp.Body)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, nil, err
		}
		ui.OK(c.Out, "已保存 %s (%s)", flags["o"], ui.HumanBytes(float64(n)))
		return nil, nil, nil
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, nil, err
	}
	text := strings.TrimSuffix(string(b), "\n")
	if text == "" {
		return nil, nil, nil
	}
	var recs []pipeline.Record
	for _, ln := range strings.Split(text, "\n") {
		recs = append(recs, pipeline.Record{"line": strings.TrimSuffix(ln, "\r")})
	}
	return recs, pipeline.LineSchema(), nil
}

func cmdPorts(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	cs, err := gnet.Connections("inet")
	if err != nil {
		return nil, nil, err
	}
	filter := first(rest)
	var recs []pipeline.Record
	pidName := map[int32]string{}
	for _, conn := range cs {
		if conn.Family != 2 { // 仅 IPv4
			continue
		}
		if filter != "" && strconv.Itoa(int(conn.Pid)) != filter {
			continue
		}
		name, ok := pidName[conn.Pid]
		if !ok {
			name = pidLookupName(conn.Pid)
			pidName[conn.Pid] = name
		}
		laddr := net.JoinHostPort(conn.Laddr.IP, strconv.Itoa(int(conn.Laddr.Port)))
		raddr := ""
		if conn.Raddr.IP != "" {
			raddr = net.JoinHostPort(conn.Raddr.IP, strconv.Itoa(int(conn.Raddr.Port)))
		}
		recs = append(recs, pipeline.Record{
			"proto":   protoName(conn.Type),
			"local":   laddr,
			"remote":  raddr,
			"state":   conn.Status,
			"pid":     strconv.Itoa(int(conn.Pid)),
			"process": name,
		})
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i]["local"] < recs[j]["local"]
	})
	sc := pipeline.NewSchema("proto", "local", "remote", "state", "pid", "process").WithNumeric("pid")
	return recs, sc, nil
}

// protoName gopsutil socket 类型 → 可读名称。
func protoName(t uint32) string {
	switch t {
	case 1:
		return "tcp"
	case 2:
		return "udp"
	}
	return strconv.FormatUint(uint64(t), 10)
}

// pidLookupName 查询进程名（无则返回空串）。
func pidLookupName(pid int32) string {
	if pid <= 0 {
		return "System"
	}
	p, err := process.NewProcess(pid)
	if err != nil {
		return ""
	}
	n, err := p.Name()
	if err != nil {
		return ""
	}
	return n
}
