package toolset

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// HashAlgos 支持的哈希算法。
var HashAlgos = []string{"md5", "sha1", "sha256", "sha512"}

// HashFile 计算文件哈希（流式读取）。
func HashFile(path, algo string) (string, error) {
	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// HashString 计算字符串哈希。
func HashString(s, algo string) (string, error) {
	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}
	_, _ = io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func newHasher(algo string) (hash.Hash, error) {
	switch strings.ToLower(algo) {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	}
	return nil, fmt.Errorf("不支持的算法 %q（支持 %s）", algo, strings.Join(HashAlgos, "/"))
}
