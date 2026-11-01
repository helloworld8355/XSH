package toolset

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pOpenClipboard              = user32.NewProc("OpenClipboard")
	pCloseClipboard             = user32.NewProc("CloseClipboard")
	pEmptyClipboard             = user32.NewProc("EmptyClipboard")
	pSetClipboardData           = user32.NewProc("SetClipboardData")
	pGetClipboardData           = user32.NewProc("GetClipboardData")
	pIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")

	pGlobalAlloc   = kernel32.NewProc("GlobalAlloc")
	pGlobalLock    = kernel32.NewProc("GlobalLock")
	pGlobalUnlock  = kernel32.NewProc("GlobalUnlock")
	pGlobalSize    = kernel32.NewProc("GlobalSize")
	pRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")
)

var clipMu sync.Mutex

// ClipSet 写入剪贴板文本（UTF-16）。
func ClipSet(text string) error {
	clipMu.Lock()
	defer clipMu.Unlock()
	if r, _, _ := pOpenClipboard.Call(0); r == 0 {
		return fmt.Errorf("无法打开剪贴板")
	}
	defer pCloseClipboard.Call()
	if r, _, _ := pEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("无法清空剪贴板")
	}
	ws, err := windows.UTF16FromString(text)
	if err != nil {
		return err
	}
	h, _, _ := pGlobalAlloc.Call(gmemMoveable, uintptr(len(ws))*2)
	if h == 0 {
		return fmt.Errorf("GlobalAlloc 失败")
	}
	ptr, _, _ := pGlobalLock.Call(h)
	if ptr == 0 {
		return fmt.Errorf("GlobalLock 失败")
	}
	pRtlMoveMemory.Call(ptr, uintptr(unsafe.Pointer(&ws[0])), uintptr(len(ws))*2)
	pGlobalUnlock.Call(h)
	if r, _, _ := pSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		return fmt.Errorf("SetClipboardData 失败")
	}
	return nil
}

// ClipGet 读取剪贴板文本。
func ClipGet() (string, error) {
	clipMu.Lock()
	defer clipMu.Unlock()
	if r, _, _ := pOpenClipboard.Call(0); r == 0 {
		return "", fmt.Errorf("无法打开剪贴板")
	}
	defer pCloseClipboard.Call()
	if r, _, _ := pIsClipboardFormatAvailable.Call(cfUnicodeText); r == 0 {
		return "", fmt.Errorf("剪贴板中没有文本")
	}
	h, _, _ := pGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", fmt.Errorf("GetClipboardData 失败")
	}
	ptr, _, _ := pGlobalLock.Call(h)
	if ptr == 0 {
		return "", fmt.Errorf("GlobalLock 失败")
	}
	defer pGlobalUnlock.Call(h)
	size, _, _ := pGlobalSize.Call(h)
	if size < 2 {
		return "", nil
	}
	buf := make([]uint16, int(size)/2)
	pRtlMoveMemory.Call(uintptr(unsafe.Pointer(&buf[0])), ptr, size)
	return windows.UTF16ToString(buf), nil
}
