package toolset

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"xsh/internal/sysinfo"
	"xsh/internal/ui"
)

// Serve 在 dir 启动静态 HTTP 文件服务（阻塞直至出错）。
func Serve(dir string, port int, out io.Writer) error {
	if port <= 0 || port > 65535 {
		port = 8000
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return fmt.Errorf("目录不存在: %s", abs)
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("端口 %d 不可用: %w", port, err)
	}
	ui.OK(out, "静态服务目录: %s", abs)
	fmt.Fprintf(out, "  本机:   http://127.0.0.1:%d\n", port)
	for _, ip := range sysinfo.LANIPv4s() {
		fmt.Fprintf(out, "  局域网: http://%s:%d\n", ip, port)
	}
	fmt.Fprintln(out, "  Ctrl+C 停止")
	srv := &http.Server{
		Handler: logMiddleware(http.FileServer(http.Dir(abs)), out),
	}
	return srv.Serve(l)
}

// logMiddleware 输出访问日志。
func logMiddleware(next http.Handler, out io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Fprintf(out, "%s %s %s\n", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
