package secscan

import (
	"context"
	"database/sql"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	smb2 "github.com/hirochachacha/go-smb2"
	"golang.org/x/crypto/ssh"
)

// CredResult 弱口令尝试结果。
type CredResult struct {
	Service string
	User    string
	Pass    string
	OK      bool
	Note    string
}

// DefaultUsers 常见用户名字典。
var DefaultUsers = []string{"admin", "administrator", "root", "test", "guest", "sa", "oracle"}

// DefaultPasses 常见弱口令字典。
var DefaultPasses = []string{
	"123456", "12345678", "123456789", "password", "Password1", "P@ssw0rd",
	"admin123", "admin@123", "root123", "abc123", "111111", "000000",
	"admin", "password123", "1qaz2wsx", "Aa123456", "qwerty", "letmein",
}

// TryCreds 对目标做受控弱口令验证（ssh/smb/mysql），命中即停。
func TryCreds(host string, port int, service string, users, passes []string, timeout time.Duration) []CredResult {
	if len(users) == 0 {
		users = DefaultUsers
	}
	if len(passes) == 0 {
		passes = DefaultPasses
	}
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	var out []CredResult
	for _, u := range users {
		for _, p := range passes {
			ok, note := tryOne(host, port, strings.ToLower(service), u, p, timeout)
			out = append(out, CredResult{Service: service, User: u, Pass: p, OK: ok, Note: note})
			if ok {
				return out
			}
		}
	}
	return out
}

func tryOne(host string, port int, service, user, pass string, timeout time.Duration) (bool, string) {
	switch service {
	case "ssh":
		return trySSH(host, port, user, pass, timeout)
	case "smb":
		return trySMB(host, port, user, pass, timeout)
	case "mysql":
		return tryMySQL(host, port, user, pass, timeout)
	}
	return false, "不支持的服务（支持 ssh/smb/mysql）"
}

func trySSH(host string, port int, user, pass string, timeout time.Duration) (bool, string) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), cfg)
	if err != nil {
		return false, ""
	}
	conn.Close()
	return true, "SSH 登录成功"
}

func trySMB(host string, port int, user, pass string, timeout time.Duration) (bool, string) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return false, ""
	}
	defer conn.Close()
	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{User: user, Password: pass},
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	s, err := d.Dial(conn)
	if err != nil {
		return false, ""
	}
	_ = s.Logoff()
	return true, "SMB 登录成功"
}

func tryMySQL(host string, port int, user, pass string, timeout time.Duration) (bool, string) {
	cfg := mysql.Config{
		User:                 user,
		Passwd:               pass,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(host, strconv.Itoa(port)),
		Timeout:              timeout,
		ReadTimeout:          timeout,
		AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return false, ""
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return false, ""
	}
	return true, "MySQL 登录成功"
}
