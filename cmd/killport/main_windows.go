//go:build windows

package main

import (
	"os"

	"github.com/jawharmed/kill-port/internal/cli"
	"github.com/jawharmed/kill-port/internal/windows"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], windows.New(), os.Stdin, os.Stdout, os.Stderr))
}
