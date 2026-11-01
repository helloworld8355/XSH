package toolset

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

// Codecs 支持的编解码算法。
var Codecs = []string{"base64", "base64url", "hex", "url"}

// Encode 编码字符串。
func Encode(s, algo string) (string, error) {
	switch strings.ToLower(algo) {
	case "base64", "b64":
		return base64.StdEncoding.EncodeToString([]byte(s)), nil
	case "base64url", "b64url":
		return base64.URLEncoding.EncodeToString([]byte(s)), nil
	case "hex":
		return hex.EncodeToString([]byte(s)), nil
	case "url":
		return url.QueryEscape(s), nil
	}
	return "", unsupportedCodec(algo)
}

// Decode 解码字符串。
func Decode(s, algo string) (string, error) {
	switch strings.ToLower(algo) {
	case "base64", "b64":
		b, err := base64.StdEncoding.DecodeString(s)
		return string(b), err
	case "base64url", "b64url":
		b, err := base64.URLEncoding.DecodeString(s)
		return string(b), err
	case "hex":
		b, err := hex.DecodeString(s)
		return string(b), err
	case "url":
		return url.QueryUnescape(s)
	}
	return "", unsupportedCodec(algo)
}

func unsupportedCodec(algo string) error {
	return fmt.Errorf("不支持的算法 %q（支持 %s）", algo, strings.Join(Codecs, "/"))
}
