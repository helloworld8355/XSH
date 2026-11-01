package secscan

import "strings"

// VulnFinding 漏洞匹配结果。
type VulnFinding struct {
	ID       string
	Title    string
	Severity string
	Advice   string
	Ports    []int
}

// MatchVulns 根据开放端口匹配漏洞种子。
func MatchVulns(open []PortResult) []VulnFinding {
	var openPorts []int
	for _, r := range open {
		if r.Open {
			openPorts = append(openPorts, r.Port)
		}
	}
	var out []VulnFinding
	for _, rule := range vulnRules {
		for _, p := range rule.Ports {
			if containsInt(openPorts, p) {
				out = append(out, VulnFinding{
					ID: rule.ID, Title: rule.Title, Severity: rule.Severity,
					Advice: rule.Advice, Ports: rule.Ports,
				})
				break
			}
		}
	}
	return out
}

// SeverityRank 严重程度权重（critical>high>medium>low）。
func SeverityRank(s string) int {
	switch strings.ToLower(s) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	}
	return 0
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
