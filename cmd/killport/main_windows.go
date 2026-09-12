//go:build windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Windows is not supported yet.")
	os.Exit(1)
}
