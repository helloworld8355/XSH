package winutil

import (
	"golang.org/x/sys/windows"
)

// IsElevated 当前进程是否以管理员权限运行。
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
