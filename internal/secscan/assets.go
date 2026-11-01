// Package secscan 提供授权评估用的网络扫描：端口/服务/漏洞匹配、弱口令、TLS、Web 与侦察模块。
package secscan

import (
	"embed"
	"encoding/json"
	"strconv"
	"strings"
)

//go:embed assets/top_ports.txt assets/vuln_seed.json assets/dir_wordlist.txt assets/subdomains.txt
var assets embed.FS

// PortService 端口与常见服务名。
type PortService struct {
	Port    int
	Service string
}

var topPortList []PortService

func init() {
	b, err := assets.ReadFile("assets/top_ports.txt")
	if err != nil {
		return
	}
	for _, ln := range strings.Split(string(b), "\n") {
		fields := strings.Fields(strings.TrimSpace(ln))
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		port, err := strconv.Atoi(fields[0])
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		svc := ""
		if len(fields) > 1 {
			svc = fields[1]
		}
		topPortList = append(topPortList, PortService{Port: port, Service: svc})
	}
}

// TopPorts 返回前 n 个常用端口及服务名。
func TopPorts(n int) []PortService {
	if n <= 0 || n > len(topPortList) {
		return topPortList
	}
	return topPortList[:n]
}

// FullPorts 1-65535 全端口列表。
func FullPorts() []PortService {
	out := make([]PortService, 0, 65535)
	for p := 1; p <= 65535; p++ {
		out = append(out, PortService{Port: p})
	}
	return out
}

// VulnRule 漏洞特征种子。
type VulnRule struct {
	ID       string `json:"id"`
	Ports    []int  `json:"ports"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Advice   string `json:"advice"`
}

var vulnRules []VulnRule

func init() {
	b, err := assets.ReadFile("assets/vuln_seed.json")
	if err == nil {
		_ = json.Unmarshal(b, &vulnRules)
	}
}

// VulnRules 返回全部漏洞种子规则。
func VulnRules() []VulnRule {
	return vulnRules
}

// DirWordlist 目录枚举字典。
func DirWordlist() []string {
	return readAssetLines("assets/dir_wordlist.txt")
}

// SubdomainWords 子域枚举字典。
func SubdomainWords() []string {
	return readAssetLines("assets/subdomains.txt")
}

func readAssetLines(name string) []string {
	b, err := assets.ReadFile(name)
	if err != nil {
		return nil
	}
	var out []string
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" && !strings.HasPrefix(ln, "#") {
			out = append(out, ln)
		}
	}
	return out
}
