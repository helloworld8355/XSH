// keygen 生成 Ed25519 授权密钥对并签发 license。
//
// 用法:
//
//	keygen <secretDir>                          生成密钥对（私钥写入 secretDir，公钥打印）
//	keygen <secretDir> issue <payload.json> <out.license>
//	                                            用已有私钥签发 license
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"xsh/internal/license"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	secretDir := os.Args[1]
	keyPath := filepath.Join(secretDir, "xsh_signing_key.txt")

	if len(os.Args) >= 5 && os.Args[2] == "issue" {
		if err := issue(keyPath, os.Args[3], os.Args[4]); err != nil {
			fail(err)
		}
		return
	}
	gen(secretDir, keyPath)
}

func usage() {
	fmt.Fprintln(os.Stderr, "用法: keygen <secretDir> [issue <payload.json> <out.license>]")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}

func gen(secretDir, keyPath string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(secretDir, 0o700); err != nil {
		fail(err)
	}
	privB64 := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(keyPath, []byte(privB64), 0o600); err != nil {
		fail(err)
	}
	fmt.Println("私钥已写入:", keyPath)
	fmt.Println("公钥(PUB):", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("提示: 将公钥填入 internal/license/pub.go 的 embeddedPub")
}

func issue(keyPath, payloadPath, outPath string) error {
	b, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("读取私钥失败: %w", err)
	}
	raw, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return fmt.Errorf("私钥文件无效")
	}
	pb, err := os.ReadFile(payloadPath)
	if err != nil {
		return fmt.Errorf("读取 payload 失败: %w", err)
	}
	var p license.Payload
	if err := json.Unmarshal(pb, &p); err != nil {
		return fmt.Errorf("payload 解析失败: %w", err)
	}
	lic, err := license.Issue(ed25519.PrivateKey(raw), p)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, []byte(lic+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Println("license 已签发 →", outPath)
	return nil
}
