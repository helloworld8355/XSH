package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"xsh/internal/syscheck"
	"xsh/internal/ui"
)

var (
	checkJSON bool
	checkHTML string
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "系统健康体检（14 项检查 + 综合评分）",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.InitConsole()
		report, err := syscheck.Run()
		if err != nil {
			return err
		}
		if checkHTML != "" {
			f, err := os.Create(checkHTML)
			if err != nil {
				return err
			}
			err = report.RenderHTML(f)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
			ui.OK(os.Stdout, "HTML 报告已保存: %s", checkHTML)
		}
		if checkJSON {
			b, err := report.JSON()
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		}
		report.RenderText(os.Stdout)
		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "以 JSON 输出")
	checkCmd.Flags().StringVar(&checkHTML, "html", "", "保存 HTML 报告到指定文件")
}
