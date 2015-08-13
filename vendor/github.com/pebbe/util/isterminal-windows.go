// +build windows

// with code copied from github.com/kless/termina by Jonas mg

package util

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32        = syscall.MustLoadDLL("kernel32.dll")
	procGetConsoleMode = modkernel32.MustFindProc("GetConsoleMode")
)

/*
Examples:

    IsTerminal(os.Stdin)
    IsTerminal(os.Stdout)
    IsTerminal(os.Stderr)
*/
func IsTerminal(file *os.File) bool {
	var st uint32
	return getConsoleMode(syscall.Handle(file.Fd()), &st) == nil
}

func getConsoleMode(hConsoleHandle syscall.Handle, lpMode *uint32) (err error) {
	r1, _, e1 := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(hConsoleHandle), uintptr(unsafe.Pointer(lpMode)), 0)
	if int(r1) == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
