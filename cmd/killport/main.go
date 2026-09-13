//go:build unix

package main

import (
	"os"

	"github.com/jawharmed/kill-port/internal/cli"
	"github.com/jawharmed/kill-port/internal/unix"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], unix.New(), os.Stdin, os.Stdout, os.Stderr))
}
