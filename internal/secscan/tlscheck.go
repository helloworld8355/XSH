package secscan

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// TLSReport TLS 证书与协议检查结果。
type TLSReport struct {
	Host        string
	Version     string
	Cipher      string
	Issuer      string
	Subject     string
	NotAfter    time.Time
	DaysLeft    int
	SelfSigned  bool
	Expired     bool
	WeakVersion bool
}

// CheckTLS 检查目标 TLS 配置（不校验证书链，仅读取信息）。
func CheckTLS(hostport string, timeout time.Duration) (*TLSReport, error) {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	if !strings.Contains(hostport, ":") {
		hostport += ":443"
	}
	d := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(d, "tcp", hostport, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, fmt.Errorf("TLS 握手失败: %w", err)
	}
	defer conn.Close()
	st := conn.ConnectionState()
	if len(st.PeerCertificates) == 0 {
		return nil, fmt.Errorf("未获取到证书")
	}
	cert := st.PeerCertificates[0]
	return &TLSReport{
		Host:        hostport,
		Version:     tlsVersionName(st.Version),
		Cipher:      tls.CipherSuiteName(st.CipherSuite),
		Issuer:      cert.Issuer.String(),
		Subject:     cert.Subject.String(),
		NotAfter:    cert.NotAfter,
		DaysLeft:    int(time.Until(cert.NotAfter).Hours() / 24),
		SelfSigned:  cert.CheckSignatureFrom(cert) == nil,
		Expired:     time.Now().After(cert.NotAfter),
		WeakVersion: st.Version == tls.VersionTLS10 || st.Version == tls.VersionTLS11,
	}, nil
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionSSL30:
		return "SSL 3.0"
	}
	return fmt.Sprintf("0x%04X", v)
}
