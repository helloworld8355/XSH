// Package license 实现 Ed25519 离线授权：license 文件 = base64(payload) + "." + base64(signature)。
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Payload 授权内容。
type Payload struct {
	Licensee string   `json:"licensee"`
	Expiry   string   `json:"expiry"` // YYYY-MM-DD
	Features []string `json:"features,omitempty"`
	Machine  string   `json:"machine,omitempty"` // 绑定机器指纹，空为不绑定
}

// Verify 验证 license 内容（签名 + 有效期 + 机器绑定）。
func Verify(content string) (*Payload, error) {
	if PubKey == nil {
		return nil, fmt.Errorf("授权公钥未嵌入（内部构建）")
	}
	content = strings.TrimSpace(content)
	dot := strings.LastIndex(content, ".")
	if dot <= 0 || dot == len(content)-1 {
		return nil, fmt.Errorf("license 格式无效")
	}
	msg, err := base64.StdEncoding.DecodeString(content[:dot])
	if err != nil {
		return nil, fmt.Errorf("payload 解码失败: %w", err)
	}
	sig, err := base64.StdEncoding.DecodeString(content[dot+1:])
	if err != nil {
		return nil, fmt.Errorf("签名解码失败: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return nil, fmt.Errorf("签名长度无效")
	}
	if !ed25519.Verify(PubKey, msg, sig) {
		return nil, fmt.Errorf("签名校验失败")
	}
	var p Payload
	if err := json.Unmarshal(msg, &p); err != nil {
		return nil, fmt.Errorf("payload 解析失败: %w", err)
	}
	if p.Licensee == "" {
		return nil, fmt.Errorf("license 缺少被授权方")
	}
	if p.Expiry != "" {
		exp, err := time.Parse("2006-01-02", p.Expiry)
		if err != nil {
			return nil, fmt.Errorf("有效期格式错误: %w", err)
		}
		if time.Now().After(exp.Add(24 * time.Hour)) {
			return nil, fmt.Errorf("授权已于 %s 过期", p.Expiry)
		}
	}
	if p.Machine != "" && p.Machine != MachineFingerprint() {
		return nil, fmt.Errorf("授权与当前设备不匹配")
	}
	return &p, nil
}

// LoadVerify 从文件读取并验证。
func LoadVerify(path string) (*Payload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 license 失败: %w", err)
	}
	return Verify(string(b))
}

// Issue 用私钥签发 license（供 tools/keygen 使用）。
func Issue(priv ed25519.PrivateKey, p Payload) (string, error) {
	msg, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, msg)
	return base64.StdEncoding.EncodeToString(msg) + "." + base64.StdEncoding.EncodeToString(sig), nil
}
