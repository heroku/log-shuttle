// +build !windows,!linux,!cgo

package util

import "os"

/*
Examples:

    IsTerminal(os.Stdin)
    IsTerminal(os.Stdout)
    IsTerminal(os.Stderr)
*/
func IsTerminal(file *os.File) bool {
	panic("Not implemented")
}
