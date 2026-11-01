package ui

import (
	"syscall"
	"unsafe"
)

var (
	k32          = syscall.NewLazyDLL("kernel32.dll")
	procSetOutCP = k32.NewProc("SetConsoleOutputCP")
	procGMH      = k32.NewProc("GetStdHandle")
	procGCM      = k32.NewProc("GetConsoleMode")
	procSCM      = k32.NewProc("SetConsoleMode")
)

// InitConsole 初始化控制台：UTF-8 代码页 + ANSI 转义序列支持。
func InitConsole() {
	procSetOutCP.Call(uintptr(65001))
	h, _, _ := procGMH.Call(uintptr(^uintptr(10))) // STD_OUTPUT_HANDLE(-11)
	var mode uint32
	if r, _, _ := procGCM.Call(h, uintptr(unsafe.Pointer(&mode))); r != 0 {
		procSCM.Call(h, uintptr(mode|0x4)) // ENABLE_VIRTUAL_TERMINAL_PROCESSING
	}
}
