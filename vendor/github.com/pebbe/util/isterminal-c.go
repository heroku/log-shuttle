// +build !windows,!linux,cgo

package util

/*
#include <unistd.h>
*/
import "C"

import "os"

/*
Examples:

    IsTerminal(os.Stdin)
    IsTerminal(os.Stdout)
    IsTerminal(os.Stderr)
*/
func IsTerminal(file *os.File) bool {
	return int(C.isatty(C.int(file.Fd()))) != 0
}
