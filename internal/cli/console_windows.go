//go:build windows

package cli

import (
	"syscall"
	"unsafe"
)

const enableVirtualTerminalProcessing = 0x0004

func init() {
	var mode uint32
	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	kern32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kern32.NewProc("GetConsoleMode")
	setMode := kern32.NewProc("SetConsoleMode")

	r, _, _ := getMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return
	}
	setMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
}
