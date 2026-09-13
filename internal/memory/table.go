package memory

import "github.com/jawharmed/kill-port/internal/proctable"

// Occupant is a fake Occupant for CLI tests, including a permission bit
// and liveness knobs for --soft / --wait.
type Occupant struct {
	PID                  int
	Port                 proctable.Port
	Name                 string
	Command              string
	Kind                 proctable.Kind
	DenyKill             bool
	DieOnPolite          bool
	StayAliveAfterForced bool
}

// Table is an in-memory process-table adapter. Tests observe which PIDs
// were signalled through ForcedPIDs and PolitePIDs.
type Table struct {
	occupants []Occupant
	polite    []int
	forced    []int
}

func New(occupants ...Occupant) *Table {
	return &Table{occupants: occupants}
}

func (t *Table) List(port proctable.Port, scope proctable.Scope) ([]proctable.Occupant, error) {
	found := make([]proctable.Occupant, 0)
	for _, occupant := range t.occupants {
		if occupant.Port != port {
			continue
		}
		if occupant.Kind == proctable.Peer && scope == proctable.ListenersOnly {
			continue
		}
		if t.gone(occupant) {
			continue
		}
		found = append(found, proctable.Occupant{
			PID:     occupant.PID,
			Port:    occupant.Port,
			Name:    occupant.Name,
			Command: occupant.Command,
			Kind:    occupant.Kind,
		})
	}
	return found, nil
}

func (t *Table) SignalPolite(pid int) error {
	return t.signal(pid, &t.polite)
}

func (t *Table) SignalForced(pid int) error {
	return t.signal(pid, &t.forced)
}

func (t *Table) Alive(pid int) bool {
	for _, occupant := range t.occupants {
		if occupant.PID == pid && !t.gone(occupant) {
			return true
		}
	}
	return false
}

func (t *Table) gone(occupant Occupant) bool {
	if occupant.StayAliveAfterForced {
		return false
	}
	if t.wasSignalled(t.forced, occupant.PID) {
		return true
	}
	return occupant.DieOnPolite && t.wasSignalled(t.polite, occupant.PID)
}

func (t *Table) wasSignalled(signalled []int, pid int) bool {
	for _, signalledPID := range signalled {
		if signalledPID == pid {
			return true
		}
	}
	return false
}

func (t *Table) ForcedPIDs() []int {
	return append([]int(nil), t.forced...)
}

func (t *Table) PolitePIDs() []int {
	return append([]int(nil), t.polite...)
}

func (t *Table) signal(pid int, dest *[]int) error {
	for _, occupant := range t.occupants {
		if occupant.PID == pid && occupant.DenyKill {
			return proctable.PermissionError{PID: pid}
		}
	}
	*dest = append(*dest, pid)
	return nil
}
