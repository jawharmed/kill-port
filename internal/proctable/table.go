package proctable

// Port is a TCP port number the Developer wants freed (1–65535).
type Port uint16

// Kind is the Occupant's role on the Port.
type Kind int

const (
	Listener Kind = iota
	Peer
)

// Scope selects which Occupants List returns for a Port.
type Scope int

const (
	ListenersOnly Scope = iota
	ListenersAndPeers
)

// Occupant is a process the Developer may want gone so a Port is free.
type Occupant struct {
	PID     int
	Port    Port
	Name    string
	Command string
	Kind    Kind
}

// PermissionError means the Developer cannot signal that Occupant.
type PermissionError struct {
	PID int
}

func (e PermissionError) Error() string {
	return "permission denied"
}

// ProcessTable lists Occupants and signals them. Later tickets wire flags
// onto this seam; they must not need to redesign it.
type ProcessTable interface {
	List(port Port, scope Scope) ([]Occupant, error)
	SignalPolite(pid int) error
	SignalForced(pid int) error
	Alive(pid int) bool
}
