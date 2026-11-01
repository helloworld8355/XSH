package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 持久化配置（%APPDATA%\xsh\config.json）。
type Config struct {
	ScanRatePerSec int    `json:"scan_rate_per_sec"`
	WeakPassRate   int    `json:"weakpass_rate_per_sec"`
	DirScanRate    int    `json:"dirscan_rate_per_sec"`
	VulnDBMirror   string `json:"vuln_db_mirror"`
	ProbeDomain    string `json:"probe_domain"`
	ProbeURL       string `json:"probe_url"`
	DNSResolver    string `json:"dns_resolver"`
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		ScanRatePerSec: 500,
		WeakPassRate:   5,
		DirScanRate:    50,
		ProbeDomain:    "www.msftconnecttest.com",
		ProbeURL:       "https://www.msftconnecttest.com/connecttest.txt",
	}
}

// Dir 返回配置目录 %APPDATA%\xsh。
func Dir() (string, error) {
	app := os.Getenv("APPDATA")
	if app == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(app, "xsh"), nil
}

// Path 返回配置文件路径。
func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load 加载配置；不存在时写入默认配置。
func Load() *Config {
	p, err := Path()
	if err != nil {
		return Default()
	}
	b, err := os.ReadFile(p)
	if err != nil {
		c := Default()
		_ = c.Save()
		return c
	}
	c := Default()
	if json.Unmarshal(b, c) != nil {
		return Default()
	}
	return c
}

// Save 保存配置。
func (c *Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
