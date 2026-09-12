package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jawharmed/kill-port/internal/proctable"
)

func Run(args []string, table proctable.ProcessTable, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: killport <port> [port...]")
		return 1
	}
	ports, err := parsePorts(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	signalled := map[int]struct{}{}
	killed := 0
	for _, port := range ports {
		occupants, err := table.List(port, proctable.ListenersOnly)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if len(occupants) == 0 {
			fmt.Fprintf(stdout, "Nothing listening on Port %d.\n", port)
			continue
		}
		for _, occupant := range occupants {
			fmt.Fprintf(stdout, "Port %d  PID %d  %s\n", occupant.Port, occupant.PID, occupant.Name)
			if _, already := signalled[occupant.PID]; already {
				continue
			}
			if err := table.SignalForced(occupant.PID); err != nil {
				writeSignalError(stderr, args, occupant, err)
				return 1
			}
			signalled[occupant.PID] = struct{}{}
			killed++
		}
	}
	if killed > 0 {
		writeKilled(stdout, killed)
	}
	return 0
}

func writeSignalError(stderr io.Writer, args []string, occupant proctable.Occupant, err error) {
	var denied proctable.PermissionError
	if errors.As(err, &denied) {
		fmt.Fprintf(stderr, "Permission denied for %s (PID %d) on Port %d.\n", occupant.Name, occupant.PID, occupant.Port)
		fmt.Fprintf(stderr, "Re-run with: sudo killport %s\n", strings.Join(args, " "))
		return
	}
	fmt.Fprintln(stderr, err.Error())
}

func writeKilled(stdout io.Writer, killed int) {
	noun := "Occupant"
	if killed != 1 {
		noun = "Occupants"
	}
	fmt.Fprintf(stdout, "Killed %d %s with SIGKILL.\n", killed, noun)
}

func parsePorts(args []string) ([]proctable.Port, error) {
	ports := make([]proctable.Port, 0, len(args))
	for _, arg := range args {
		n, err := strconv.Atoi(arg)
		if err != nil || n < 1 || n > 65535 {
			return nil, fmt.Errorf("Invalid Port: %s (expected 1–65535)", arg)
		}
		ports = append(ports, proctable.Port(n))
	}
	return ports, nil
}
