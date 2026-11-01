package toolset

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"

	"xsh/internal/winutil"
)

// StartupItem 启动项。
type StartupItem struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// StartupList 列出启动项（HKCU/HKLM Run 键 + 启动文件夹）。
func StartupList() ([]StartupItem, error) {
	var items []StartupItem
	hives := []struct {
		source string
		hive   registry.Key
	}{
		{"hkcu", registry.CURRENT_USER},
		{"hklm", registry.LOCAL_MACHINE},
	}
	for _, h := range hives {
		k, err := registry.OpenKey(h.hive, runKeyPath, registry.QUERY_VALUE|registry.WOW64_64KEY)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		for _, n := range names {
			if v, _, err := k.GetStringValue(n); err == nil {
				items = append(items, StartupItem{Source: h.source, Name: n, Command: v})
			}
		}
		k.Close()
	}
	for _, f := range startupFolders() {
		ents, err := os.ReadDir(f.dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			items = append(items, StartupItem{Source: f.source, Name: e.Name(), Command: filepath.Join(f.dir, e.Name())})
		}
	}
	return items, nil
}

type startupFolder struct{ dir, source string }

// startupFolders 用户与公共启动文件夹。
func startupFolders() []startupFolder {
	var out []startupFolder
	if d := os.Getenv("APPDATA"); d != "" {
		out = append(out, startupFolder{
			filepath.Join(d, `Microsoft\Windows\Start Menu\Programs\Startup`),
			"user-folder",
		})
	}
	if d := os.Getenv("ProgramData"); d != "" {
		out = append(out, startupFolder{
			filepath.Join(d, `Microsoft\Windows\Start Menu\Programs\StartUp`),
			"common-folder",
		})
	}
	return out
}

// StartupAdd 添加注册表启动项（hkcu/hklm）。
func StartupAdd(source, name, command string) error {
	hive, ok := hiveBySource(source)
	if !ok {
		return fmt.Errorf("source 必须是 hkcu 或 hklm")
	}
	k, err := registry.OpenKey(hive, runKeyPath, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return regErr(err)
	}
	defer k.Close()
	if err := k.SetStringValue(name, command); err != nil {
		return regErr(err)
	}
	return nil
}

// StartupRemove 删除注册表启动项。
func StartupRemove(source, name string) error {
	hive, ok := hiveBySource(source)
	if !ok {
		return fmt.Errorf("source 必须是 hkcu 或 hklm")
	}
	k, err := registry.OpenKey(hive, runKeyPath, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return regErr(err)
	}
	defer k.Close()
	if err := k.DeleteValue(name); err != nil {
		return regErr(err)
	}
	return nil
}

// StartupBackup 备份全部启动项到 JSON 文件。
func StartupBackup(path string) (int, error) {
	items, err := StartupList()
	if err != nil {
		return 0, err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return 0, err
	}
	return len(items), nil
}

// StartupRestore 从备份恢复注册表启动项（仅 hkcu/hklm），返回恢复数量。
func StartupRestore(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var items []StartupItem
	if err := json.Unmarshal(b, &items); err != nil {
		return 0, err
	}
	restored := 0
	for _, it := range items {
		if it.Source != "hkcu" && it.Source != "hklm" {
			continue
		}
		if err := StartupAdd(it.Source, it.Name, it.Command); err == nil {
			restored++
		}
	}
	return restored, nil
}

func hiveBySource(source string) (registry.Key, bool) {
	switch strings.ToLower(source) {
	case "hkcu":
		return registry.CURRENT_USER, true
	case "hklm":
		return registry.LOCAL_MACHINE, true
	}
	return 0, false
}

// regErr 为注册表权限错误补充提权提示。
func regErr(err error) error {
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) && !winutil.IsElevated() {
		return fmt.Errorf("权限不足（请以管理员运行）: %w", err)
	}
	return err
}
