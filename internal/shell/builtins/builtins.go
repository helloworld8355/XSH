package builtins

import "xsh/internal/shell"

// Register 注册全部内置命令到注册表。
func Register(reg *shell.Registry) {
	registerFS(reg)
	registerText(reg)
	registerVerbs(reg)
	registerProc(reg)
	registerService(reg)
	registerEnv(reg)
	registerNet(reg)
	registerData(reg)
	registerMisc(reg)
	registerTools(reg)
}
