package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Entry 审计日志条目（JSONL）。
type Entry struct {
	Time     string `json:"time"`
	Cmd      string `json:"cmd"`
	Target   string `json:"target,omitempty"`
	Args     string `json:"args,omitempty"`
	Result   string `json:"result"`
	Licensee string `json:"licensee,omitempty"`
}

// Log 追加一条审计日志到 %APPDATA%\xsh\audit\YYYYMMDD.jsonl。
func Log(e Entry) error {
	dir, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := time.Now().Format("20060102") + ".jsonl"
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if e.Time == "" {
		e.Time = time.Now().Format(time.RFC3339)
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// Dir 返回审计日志目录。
func Dir() (string, error) {
	return dir()
}

func dir() (string, error) {
	app := os.Getenv("APPDATA")
	if app == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(app, "xsh", "audit"), nil
}
