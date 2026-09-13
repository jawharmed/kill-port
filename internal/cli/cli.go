package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jawharmed/kill-port/internal/proctable"
)

// Version is the killport build identity. Release builds override it via ldflags.
var Version = "0.0.0-dev"

type clock struct {
	now   func() time.Time
	sleep func(time.Duration)
}

func realClock() clock {
	return clock{now: time.Now, sleep: time.Sleep}
}

const helpText = `Usage: killport [flags] <port> [port...]

Free a TCP Port by terminating its Occupant. By default that is a Listener
(TCP LISTEN); --all also includes Peers.

Flags:
  -h, --help         Show this help and exit; no Port required
  -V, --version      Print the version and exit; no Port required
  -n, --dry-run      List Occupants without signalling
  -s, --soft         SIGTERM first; SIGKILL after 3s if still alive
  -w, --wait         Wait until Occupants are gone, or 10s
  -v, --verbose      Show each Occupant's full command line
  -a, --all          Target Listeners and Peers on the named Ports
  -l, --listen       Target Listeners only (default)
  -i, --interactive  Confirm when more than one PID would be signalled

Examples:
  killport 8080
  killport 3000 8080
  killport -n 3000 8080
  killport -n -v 8080
  killport --all 8080
  killport -s -w 8080
`

func Run(args []string, table proctable.ProcessTable, stdin io.Reader, stdout, stderr io.Writer) int {
	return runCLI(args, table, stdin, stdout, stderr, realClock())
}

func runCLI(args []string, table proctable.ProcessTable, stdin io.Reader, stdout, stderr io.Writer, clk clock) int {
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

	targets, err := listTargets(table, ports, parsed, stdout)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if parsed.dryRun {
		return 0
	}
	if parsed.interactive && len(targets) > 1 && !confirmKill(stdin, stderr, len(targets)) {
		return 1
	}

	var result killResult
	var occupant proctable.Occupant
	if parsed.soft {
		result, occupant, err = signalSoft(table, targets, clk)
	} else {
		result, occupant, err = signalForced(table, targets)
	}
	if err != nil {
		writeSignalError(stderr, args, occupant, err)
		return 1
	}
	if result.count > 0 {
		writeKilled(stdout, result)
	}
	if parsed.wait {
		return waitUntilFree(table, ports, parsed.scope, clk, stderr)
	}
	return 0
}

const (
	softGrace = 3 * time.Second
	waitLimit = 10 * time.Second
	pollEvery = 50 * time.Millisecond
)

type killResult struct {
	count  int
	polite bool
	forced bool
}

func signalForced(table proctable.ProcessTable, targets []proctable.Occupant) (killResult, proctable.Occupant, error) {
	result := killResult{}
	for _, occupant := range targets {
		if err := table.SignalForced(occupant.PID); err != nil {
			return result, occupant, err
		}
		result.forced = true
		result.count++
	}
	return result, proctable.Occupant{}, nil
}

func signalSoft(table proctable.ProcessTable, targets []proctable.Occupant, clk clock) (killResult, proctable.Occupant, error) {
	result := killResult{}
	for _, occupant := range targets {
		if err := table.SignalPolite(occupant.PID); err != nil {
			return result, occupant, err
		}
		result.polite = true
		result.count++
	}
	waitWhileAlive(table, targets, clk)
	for _, occupant := range targets {
		if !table.Alive(occupant.PID) {
			continue
		}
		if err := table.SignalForced(occupant.PID); err != nil {
			return result, occupant, err
		}
		result.forced = true
	}
	return result, proctable.Occupant{}, nil
}

func waitWhileAlive(table proctable.ProcessTable, targets []proctable.Occupant, clk clock) {
	deadline := clk.now().Add(softGrace)
	for clk.now().Before(deadline) {
		if !anyAlive(table, targets) {
			return
		}
		clk.sleep(pollEvery)
	}
}

func anyAlive(table proctable.ProcessTable, targets []proctable.Occupant) bool {
	for _, occupant := range targets {
		if table.Alive(occupant.PID) {
			return true
		}
	}
	return false
}

func waitUntilFree(table proctable.ProcessTable, ports []proctable.Port, scope proctable.Scope, clk clock, stderr io.Writer) int {
	deadline := clk.now().Add(waitLimit)
	for {
		occupied, err := occupiedPorts(table, ports, scope)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if len(occupied) == 0 {
			return 0
		}
		if !clk.now().Before(deadline) {
			writeStillOccupied(stderr, occupied)
			return 1
		}
		clk.sleep(pollEvery)
	}
}

func occupiedPorts(table proctable.ProcessTable, ports []proctable.Port, scope proctable.Scope) ([]proctable.Port, error) {
	occupied := make([]proctable.Port, 0)
	for _, port := range ports {
		occupants, err := table.List(port, scope)
		if err != nil {
			return nil, err
		}
		if len(occupants) > 0 {
			occupied = append(occupied, port)
		}
	}
	return occupied, nil
}

func writeStillOccupied(stderr io.Writer, ports []proctable.Port) {
	if len(ports) == 1 {
		fmt.Fprintf(stderr, "Port %d is still occupied.\n", ports[0])
		return
	}
	labels := make([]string, len(ports))
	for i, port := range ports {
		labels[i] = strconv.Itoa(int(port))
	}
	fmt.Fprintf(stderr, "Ports %s are still occupied.\n", strings.Join(labels, ", "))
}

func listTargets(table proctable.ProcessTable, ports []proctable.Port, parsed parsedArgs, stdout io.Writer) ([]proctable.Occupant, error) {
	seen := map[int]struct{}{}
	targets := make([]proctable.Occupant, 0)
	for _, port := range ports {
		occupants, err := table.List(port, parsed.scope)
		if err != nil {
			return nil, err
		}
		if len(occupants) == 0 {
			fmt.Fprintf(stdout, "Nothing listening on Port %d.\n", port)
			continue
		}
		for _, occupant := range occupants {
			writeOccupant(stdout, occupant, parsed.verbose)
			if _, already := seen[occupant.PID]; already {
				continue
			}
			seen[occupant.PID] = struct{}{}
			targets = append(targets, occupant)
		}
	}
	return targets, nil
}

func writeOccupant(stdout io.Writer, occupant proctable.Occupant, verbose bool) {
	label := occupant.Name
	if verbose && occupant.Command != "" {
		label = occupant.Command
	}
	fmt.Fprintf(stdout, "Port %d  PID %d  %s\n", occupant.Port, occupant.PID, label)
}

func confirmKill(stdin io.Reader, stderr io.Writer, pidCount int) bool {
	fmt.Fprintf(stderr, "Kill %d Occupants? [y/N] ", pidCount)
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	return isYes(line)
}

func isYes(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes", "oui":
		return true
	default:
		return false
	}
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

func writeKilled(stdout io.Writer, result killResult) {
	noun := "Occupant"
	if result.count != 1 {
		noun = "Occupants"
	}
	how := "SIGKILL"
	if result.polite && result.forced {
		how = "SIGTERM then SIGKILL"
	} else if result.polite {
		how = "SIGTERM"
	}
	fmt.Fprintf(stdout, "Killed %d %s with %s.\n", result.count, noun, how)
}

type parsedArgs struct {
	help        bool
	version     bool
	dryRun      bool
	verbose     bool
	soft        bool
	wait        bool
	interactive bool
	scope       proctable.Scope
	ports       []string
}

func parseArgs(args []string) (parsedArgs, error) {
	parsed := parsedArgs{
		ports: make([]string, 0, len(args)),
		scope: proctable.ListenersOnly,
	}
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			parsed.help = true
		case "-V", "--version":
			parsed.version = true
		case "-n", "--dry-run":
			parsed.dryRun = true
		case "-s", "--soft":
			parsed.soft = true
		case "-w", "--wait":
			parsed.wait = true
		case "-v", "--verbose":
			parsed.verbose = true
		case "-a", "--all":
			parsed.scope = proctable.ListenersAndPeers
		case "-l", "--listen":
			parsed.scope = proctable.ListenersOnly
		case "-i", "--interactive":
			parsed.interactive = true
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
