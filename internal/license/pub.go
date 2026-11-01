package license

import (
	"crypto/ed25519"
	"encoding/base64"
)

// embeddedPub 由 tools/keygen 生成后填入（base64 的 Ed25519 公钥）。
const embeddedPub = "8W1FadR2LMGDLh/Guis3o9fxlUnR0WHqVFVq5qeDunU="

// PubKey 内嵌的授权公钥；nil 表示内部构建尚未嵌入。
var PubKey ed25519.PublicKey

func init() {
	if embeddedPub == "REPLACE_WITH_REAL_PUBKEY" {
		return
	}
	b, err := base64.StdEncoding.DecodeString(embeddedPub)
	if err != nil || len(b) != ed25519.PublicKeySize {
		panic("license: 公钥配置无效")
	}
	PubKey = ed25519.PublicKey(b)
}
