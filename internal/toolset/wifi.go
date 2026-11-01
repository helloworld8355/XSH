package toolset

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KV 通用键值对（netsh 本地化输出）。
type KV struct{ K, V string }

// WifiProfiles 列出已保存的 Wi-Fi 配置名（中英文 netsh 输出兼容）。
func WifiProfiles() ([]string, error) {
	out, err := exec.Command("netsh", "wlan", "show", "profiles").Output()
	if err != nil {
		return nil, fmt.Errorf("netsh 执行失败: %w", err)
	}
	var names []string
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSuffix(ln, "\r")
		if i := strings.Index(ln, ": "); i >= 0 {
			if v := strings.TrimSpace(ln[i+2:]); v != "" && !strings.Contains(v, "%") {
				names = append(names, v)
			}
		}
	}
	return names, nil
}

// WifiInfo 解析 `netsh wlan show interfaces` 的键值对。
func WifiInfo() ([]KV, error) {
	out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return nil, fmt.Errorf("netsh 执行失败: %w", err)
	}
	var kvs []KV
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSuffix(ln, "\r")
		t := strings.TrimSpace(ln)
		if i := strings.Index(t, ":"); i > 0 && i < len(t)-1 {
			k := strings.TrimSpace(t[:i])
			v := strings.TrimSpace(t[i+1:])
			if k != "" && v != "" {
				kvs = append(kvs, KV{k, v})
			}
		}
	}
	return kvs, nil
}

// wlanProfileXML netsh 导出的配置文件结构。
type wlanProfileXML struct {
	Name string `xml:"name"`
	MSM  struct {
		Security struct {
			AuthEncryption struct {
				Authentication string `xml:"authentication"`
			} `xml:"authEncryption"`
			SharedKey struct {
				KeyMaterial string `xml:"keyMaterial"`
			} `xml:"sharedKey"`
		} `xml:"security"`
	} `xml:"MSM"`
}

// WifiPassword 读取配置的明文密钥（需要管理员权限），返回 (认证方式, 密钥)。
func WifiPassword(name string) (auth, pw string, err error) {
	tmp, err := os.MkdirTemp("", "xsh-wifi-*")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmp)
	cmd := exec.Command("netsh", "wlan", "export", "profile", "name="+name, "key=clear", "folder="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("导出失败（通常需要管理员）: %s", strings.TrimSpace(string(out)))
	}
	ents, _ := os.ReadDir(tmp)
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(tmp, e.Name()))
		if err != nil {
			continue
		}
		var p wlanProfileXML
		if xml.Unmarshal(b, &p) == nil {
			return p.MSM.Security.AuthEncryption.Authentication, p.MSM.Security.SharedKey.KeyMaterial, nil
		}
	}
	return "", "", fmt.Errorf("未找到密钥（开放网络或需管理员权限）")
}
