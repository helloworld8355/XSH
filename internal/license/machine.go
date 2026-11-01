package license

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"syscall"
	"unsafe"
)

// MachineFingerprint 机器指纹：计算机名 + C 盘卷序列号的 SHA256 前 16 位。
func MachineFingerprint() string {
	h := sha256.New()
	fmt.Fprintln(h, computerName())
	fmt.Fprintln(h, volumeSerial())
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func computerName() string {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	p := k32.NewProc("GetComputerNameW")
	buf := make([]uint16, 256)
	size := uint32(len(buf))
	if r, _, _ := p.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func volumeSerial() string {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	p := k32.NewProc("GetVolumeInformationW")
	root, err := syscall.UTF16PtrFromString(`C:\`)
	if err != nil {
		return ""
	}
	var serial uint32
	p.Call(uintptr(unsafe.Pointer(root)), 0, 0, uintptr(unsafe.Pointer(&serial)), 0, 0, 0, 0)
	return fmt.Sprintf("%08X", serial)
}
