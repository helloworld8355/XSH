package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"xsh/internal/toolset"
	"xsh/internal/ui"
)

// printJSON 以缩进 JSON 输出。
func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

var (
	fetchThreads  int
	fetchOut      string
	fetchInsecure bool
	servePort     int
	snapJSON      bool
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <URL>",
	Short: "多线程断点续传下载",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		_, err := toolset.Fetch(toolset.FetchOptions{
			URL:      args[0],
			Dest:     fetchOut,
			Threads:  fetchThreads,
			Insecure: fetchInsecure,
			Progress: true,
			Out:      os.Stdout,
		})
		return err
	},
}

var hashCmd = &cobra.Command{
	Use:   "hash <md5|sha1|sha256|sha512> <文件>",
	Short: "计算文件哈希（md5/sha1/sha256/sha512）",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sum, err := toolset.HashFile(args[1], args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%s  %s\n", sum, args[1])
		return nil
	},
}

var encCmd = &cobra.Command{
	Use:   "enc <base64|base64url|hex|url> <文本>",
	Short: "文本编码",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := toolset.Encode(args[1], args[0])
		if err != nil {
			return err
		}
		fmt.Println(s)
		return nil
	},
}

var decCmd = &cobra.Command{
	Use:   "dec <base64|base64url|hex|url> <文本>",
	Short: "文本解码",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := toolset.Decode(args[1], args[0])
		if err != nil {
			return err
		}
		fmt.Println(s)
		return nil
	},
}

var clipCmd = &cobra.Command{
	Use:   "clip [文本...]",
	Short: "剪贴板读写（无参数=读取，有参数=写入）",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		if len(args) == 0 {
			s, err := toolset.ClipGet()
			if err != nil {
				return err
			}
			fmt.Println(s)
			return nil
		}
		if err := toolset.ClipSet(strings.Join(args, " ")); err != nil {
			return err
		}
		ui.OK(os.Stdout, "已复制到剪贴板")
		return nil
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve [目录]",
	Short: "启动静态 HTTP 服务（Ctrl+C 停止）",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		return toolset.Serve(dir, servePort, os.Stdout)
	},
}

var wifiCmd = &cobra.Command{
	Use:   "wifi [profile]",
	Short: "Wi-Fi 信息（无参数=概况，指定 profile=显示密码）",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		if len(args) == 0 {
			kvs, err := toolset.WifiInfo()
			if err != nil {
				return err
			}
			ui.Title(os.Stdout, "Wi-Fi")
			rows := make([][]string, 0, len(kvs))
			for _, kv := range kvs {
				rows = append(rows, []string{kv.K, kv.V})
			}
			ui.PrintTable(os.Stdout, []string{"项", "值"}, rows)
			return nil
		}
		auth, pw, err := toolset.WifiPassword(args[0])
		if err != nil {
			return err
		}
		ui.Title(os.Stdout, "Wi-Fi: "+args[0])
		ui.Field(os.Stdout, "认证方式", auth)
		ui.Field(os.Stdout, "密码", pw)
		return nil
	},
}

// ---------------- startup ----------------

var startupCmd = &cobra.Command{
	Use:   "startup",
	Short: "启动项管理（list/add/remove/backup/restore）",
}

var startupListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出启动项（HKCU/HKLM Run + 启动文件夹）",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		items, err := toolset.StartupList()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(items))
		for _, it := range items {
			rows = append(rows, []string{it.Source, it.Name, it.Command})
		}
		ui.Hdr(os.Stdout, fmt.Sprintf("启动项 (%d)", len(items)))
		ui.PrintTable(os.Stdout, []string{"来源", "名称", "命令"}, rows)
		return nil
	},
}

var startupAddCmd = &cobra.Command{
	Use:   "add <hkcu|hklm> <名称> <命令>",
	Short: "添加注册表启动项",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		if err := toolset.StartupAdd(strings.ToLower(args[0]), args[1], args[2]); err != nil {
			return err
		}
		ui.OK(os.Stdout, "已添加启动项: %s", args[1])
		return nil
	},
}

var startupRemoveCmd = &cobra.Command{
	Use:   "remove <hkcu|hklm> <名称>",
	Short: "删除注册表启动项",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		if err := toolset.StartupRemove(strings.ToLower(args[0]), args[1]); err != nil {
			return err
		}
		ui.OK(os.Stdout, "已删除启动项: %s", args[1])
		return nil
	},
}

var startupBackupCmd = &cobra.Command{
	Use:   "backup <文件>",
	Short: "备份全部启动项到 JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		n, err := toolset.StartupBackup(args[0])
		if err != nil {
			return err
		}
		ui.OK(os.Stdout, "已备份 %d 项 → %s", n, args[0])
		return nil
	},
}

var startupRestoreCmd = &cobra.Command{
	Use:   "restore <文件>",
	Short: "从 JSON 备份恢复启动项",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		n, err := toolset.StartupRestore(args[0])
		if err != nil {
			return err
		}
		ui.OK(os.Stdout, "已恢复 %d 项启动项", n)
		return nil
	},
}

// ---------------- snapshot ----------------

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "系统快照（save/list/show）",
}

var snapshotSaveCmd = &cobra.Command{
	Use:   "save [目录]",
	Short: "采集并保存系统快照",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		s, err := toolset.TakeSnapshot()
		if err != nil {
			return err
		}
		path, err := toolset.SaveSnapshot(s, dir)
		if err != nil {
			return err
		}
		ui.OK(os.Stdout, "快照已保存: %s", path)
		return nil
	},
}

var snapshotListCmd = &cobra.Command{
	Use:   "list [目录]",
	Short: "列出已保存的快照",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		names, err := toolset.ListSnapshots(dir)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			ui.Warn(os.Stdout, "目录下没有快照文件")
			return nil
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	},
}

var snapshotShowCmd = &cobra.Command{
	Use:   "show <文件>",
	Short: "查看快照内容",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		s, err := toolset.LoadSnapshot(args[0])
		if err != nil {
			return err
		}
		if snapJSON {
			return printJSON(s)
		}
		ui.Title(os.Stdout, "系统快照")
		ui.Field(os.Stdout, "时间", s.Time)
		ui.Field(os.Stdout, "主机", s.Host)
		ui.Field(os.Stdout, "系统", s.OS)
		ui.Field(os.Stdout, "运行时长", s.Uptime)
		ui.Field(os.Stdout, "处理器", fmt.Sprintf("%s ×%d", s.CPU, s.Cores))
		ui.Field(os.Stdout, "内存", fmt.Sprintf("%s / %s (%.1f%%)",
			ui.HumanBytes(float64(s.MemUsed)), ui.HumanBytes(float64(s.MemTotal)), s.MemPercent))
		if s.Gateway != "" {
			ui.Field(os.Stdout, "网关", s.Gateway)
		}
		if len(s.LANIPs) > 0 {
			ui.Field(os.Stdout, "内网地址", strings.Join(s.LANIPs, ", "))
		}
		if len(s.Disks) > 0 {
			ui.Hdr(os.Stdout, "磁盘")
			rows := make([][]string, 0, len(s.Disks))
			for _, d := range s.Disks {
				rows = append(rows, []string{d.Mount,
					fmt.Sprintf("%.1fG / %.1fG", d.UsedGB, d.TotalGB),
					strconv.Itoa(int(d.Percent)) + "%"})
			}
			ui.PrintTable(os.Stdout, []string{"挂载点", "已用/总量", "使用率"}, rows)
		}
		if len(s.TopProcs) > 0 {
			ui.Hdr(os.Stdout, "热点进程")
			rows := make([][]string, 0, len(s.TopProcs))
			for _, p := range s.TopProcs {
				rows = append(rows, []string{strconv.Itoa(int(p.PID)), p.Name,
					fmt.Sprintf("%.1f%%", p.CPU), fmt.Sprintf("%.0fMB", p.MemMB)})
			}
			ui.PrintTable(os.Stdout, []string{"PID", "名称", "CPU", "内存"}, rows)
		}
		return nil
	},
}

var toolCmds = []*cobra.Command{fetchCmd, hashCmd, encCmd, decCmd, clipCmd, serveCmd, wifiCmd, startupCmd, snapshotCmd}

func init() {
	fetchCmd.Flags().IntVar(&fetchThreads, "threads", 4, "下载线程数（1-16）")
	fetchCmd.Flags().StringVarP(&fetchOut, "out", "o", "", "保存路径（默认取 URL 文件名）")
	fetchCmd.Flags().BoolVar(&fetchInsecure, "insecure", false, "跳过 TLS 证书校验")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8000, "监听端口")
	snapshotShowCmd.Flags().BoolVar(&snapJSON, "json", false, "以 JSON 输出")
	startupCmd.AddCommand(startupListCmd, startupAddCmd, startupRemoveCmd, startupBackupCmd, startupRestoreCmd)
	snapshotCmd.AddCommand(snapshotSaveCmd, snapshotListCmd, snapshotShowCmd)
}
