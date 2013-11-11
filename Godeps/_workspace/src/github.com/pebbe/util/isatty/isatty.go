package main

import (
	"fmt"
	"github.com/pebbe/util"
	"os"
)

func main() {
	fmt.Println("stdin: ", util.IsTerminal(os.Stdin))
	fmt.Println("stdout:", util.IsTerminal(os.Stdout))
	fmt.Println("stderr:", util.IsTerminal(os.Stderr))
}
