//go:build unix

package unix

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/jawharmed/kill-port/internal/proctable"
)

type Table struct{}

func New() *Table {
	return &Table{}
}

func (t *Table) List(port proctable.Port, scope proctable.Scope) ([]proctable.Occupant, error) {
	args := []string{"-nP", "-iTCP:" + strconv.Itoa(int(port)), "-F", "pcT"}
	if scope == proctable.ListenersOnly {
		args = append(args, "-sTCP:LISTEN")
	}
	cmd := exec.Command("lsof", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	return parseLsof(out, port, scope)
}

func (t *Table) SignalPolite(pid int) error {
	return signalPID(pid, syscall.SIGTERM)
}

func (t *Table) SignalForced(pid int) error {
	return signalPID(pid, syscall.SIGKILL)
}

func (t *Table) Alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func signalPID(pid int, sig syscall.Signal) error {
	err := syscall.Kill(pid, sig)
	if err == nil || errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		return proctable.PermissionError{PID: pid}
	}
	return err
}

func parseLsof(out []byte, port proctable.Port, scope proctable.Scope) ([]proctable.Occupant, error) {
	type lsofRow struct {
		pid    int
		name   string
		listen bool
		peer   bool
	}
	byPID := map[int]*lsofRow{}
	order := make([]int, 0)
	var current *lsofRow

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tag, val := line[0], line[1:]
		switch tag {
		case 'p':
			pid, err := strconv.Atoi(val)
			if err != nil {
				return nil, err
			}
			current = byPID[pid]
			if current == nil {
				current = &lsofRow{pid: pid}
				byPID[pid] = current
				order = append(order, pid)
			}
		case 'c':
			if current != nil && current.name == "" {
				current.name = val
			}
		case 'T':
			if current == nil {
				continue
			}
			if strings.HasPrefix(val, "ST=") {
				state := strings.TrimPrefix(val, "ST=")
				if state == "LISTEN" {
					current.listen = true
				} else {
					current.peer = true
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	occupants := make([]proctable.Occupant, 0, len(order))
	for _, pid := range order {
		row := byPID[pid]
		kind := proctable.Peer
		if row.listen {
			kind = proctable.Listener
		}
		if scope == proctable.ListenersOnly && kind != proctable.Listener {
			continue
		}
		if !row.listen && !row.peer {
			continue
		}
		occupants = append(occupants, proctable.Occupant{
			PID:     row.pid,
			Port:    port,
			Name:    row.name,
			Command: commandLine(row.pid),
			Kind:    kind,
		})
	}
	return occupants, nil
}

func commandLine(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
