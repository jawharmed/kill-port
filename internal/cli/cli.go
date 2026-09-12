package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jawharmed/kill-port/internal/proctable"
)

// Version is the killport build identity. Release builds override it via ldflags.
var Version = "0.0.0-dev"

const helpText = `Usage: killport [flags] <port> [port...]

Free a TCP Port by terminating its Listener (TCP LISTEN Occupant).

Flags:
  -h, --help       Show this help and exit; no Port required
  -V, --version    Print the version and exit; no Port required
  -n, --dry-run    List Occupants without signalling
  -v, --verbose    Show each Occupant's full command line

Examples:
  killport 8080
  killport 3000 8080
  killport -n 3000 8080
  killport -n -v 8080
`

func Run(args []string, table proctable.ProcessTable, stdout, stderr io.Writer) int {
	parsed, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if parsed.help {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	if parsed.version {
		fmt.Fprintf(stdout, "killport version %s\n", Version)
		return 0
	}
	if len(parsed.ports) == 0 {
		fmt.Fprintln(stderr, "Usage: killport <port> [port...]")
		return 1
	}
	ports, err := parsePorts(parsed.ports)
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
			label := occupant.Name
			if parsed.verbose && occupant.Command != "" {
				label = occupant.Command
			}
			fmt.Fprintf(stdout, "Port %d  PID %d  %s\n", occupant.Port, occupant.PID, label)
			if parsed.dryRun {
				continue
			}
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

type parsedArgs struct {
	help    bool
	version bool
	dryRun  bool
	verbose bool
	ports   []string
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{ports: make([]string, 0, len(args))}
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			parsed.help = true
		case "-V", "--version":
			parsed.version = true
		case "-n", "--dry-run":
			parsed.dryRun = true
		case "-v", "--verbose":
			parsed.verbose = true
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return parsedArgs{}, fmt.Errorf("Unknown flag: %s\nSee killport --help", arg)
			}
			parsed.ports = append(parsed.ports, arg)
		}
	}
	return parsed, nil
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
