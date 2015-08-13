// +build linux

package util

import (
	"os"
	"syscall"
	"unsafe"
)

/*
Examples:

    IsTerminal(os.Stdin)
    IsTerminal(os.Stdout)
    IsTerminal(os.Stderr)
*/
func IsTerminal(file *os.File) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, file.Fd(),
		uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}
