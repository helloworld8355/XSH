package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"xsh/internal/license"
	"xsh/internal/ui"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		style, err := ui.ParseBannerStyle(bannerStyle)
		if err != nil {
			return err
		}
		ui.Banner(os.Stdout, style)
		ui.Field(os.Stdout, "版本", ui.Version)
		ui.Field(os.Stdout, "Go", runtime.Version())
		ui.Field(os.Stdout, "平台", runtime.GOOS+"/"+runtime.GOARCH)
		return nil
	},
}

var licenseCmd = &cobra.Command{
	Use:   "license [文件]",
	Short: "校验并显示 license 授权信息",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) > 0 {
			path = args[0]
		} else {
			p, err := defaultLicensePath()
			if err != nil {
				return err
			}
			path = p
		}
		p, err := license.LoadVerify(path)
		if err != nil {
			return err
		}
		ui.InitConsole()
		ui.Title(os.Stdout, "授权信息")
		ui.Field(os.Stdout, "文件", path)
		ui.Field(os.Stdout, "被授权方", p.Licensee)
		ui.Field(os.Stdout, "有效期至", p.Expiry)
		if len(p.Features) > 0 {
			ui.Field(os.Stdout, "授权功能", strings.Join(p.Features, ", "))
		}
		ui.OK(os.Stdout, "签名校验通过")
		return nil
	},
}

// licenseDefaultCandidate 默认 license 存放位置。
func licenseDefaultCandidate() string {
	if home := os.Getenv("USERPROFILE"); home != "" {
		return filepath.Join(home, ".xsh", "license")
	}
	return "license"
}

// defaultLicensePath 在默认位置查找 license 文件。
func defaultLicensePath() (string, error) {
	var cands []string
	if c := licenseDefaultCandidate(); c != "" {
		cands = append(cands, c)
	}
	if app := os.Getenv("APPDATA"); app != "" {
		cands = append(cands, filepath.Join(app, "xsh", "license"))
	}
	cands = append(cands, "license")
	for _, c := range cands {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("未找到 license 文件（默认位置 %s），请通过参数或 --license 指定", licenseDefaultCandidate())
}
