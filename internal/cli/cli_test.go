package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jawharmed/kill-port/internal/memory"
	"github.com/jawharmed/kill-port/internal/proctable"
)

func TestNoPorts_PrintsUsageOnStderrAndExitsNonZero(t *testing.T) {
	stdout, stderr, exit := run(t, memory.New())

	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage: killport <port> [port...]") {
		t.Fatalf("stderr = %q, want usage line", stderr)
	}
}

func TestInvalidPort_PrintsErrorOnStderrAndExitsNonZero(t *testing.T) {
	cases := []string{"0", "65536", "abc", "1.5"}
	for _, arg := range cases {
		t.Run(arg, func(t *testing.T) {
			stdout, stderr, exit := run(t, memory.New(), arg)
			if exit == 0 {
				t.Fatalf("exit code = 0, want non-zero")
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			want := "Invalid Port: " + arg + " (expected 1–65535)"
			if !strings.Contains(stderr, want) {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}

func TestEmptyPort_PrintsEnglishMessageAndExitsZero(t *testing.T) {
	stdout, stderr, exit := run(t, memory.New(), "8080")

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Nothing listening on Port 8080.") {
		t.Fatalf("stdout = %q, want empty-Port message", stdout)
	}
}

func TestListener_IsTerminatedWithForcedKillAndReported(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	stdout, stderr, exit := run(t, table, "8080")

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
		t.Fatalf("stdout = %q, want Port, PID, and process name", stdout)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGKILL.") {
		t.Fatalf("stdout = %q, want SIGKILL success line", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242]", got)
	}
	if got := table.PolitePIDs(); len(got) != 0 {
		t.Fatalf("polite PIDs = %v, want none", got)
	}
}

func TestPermissionDenied_NamesOccupantShowsSudoAndExitsNonZero(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:      1,
		Port:     80,
		Name:     "nginx",
		Kind:     proctable.Listener,
		DenyKill: true,
	})

	stdout, stderr, exit := run(t, table, "80")

	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stdout, "Port 80  PID 1  nginx") {
		t.Fatalf("stdout = %q, want Occupant listed before the failure", stdout)
	}
	if !strings.Contains(stderr, "Permission denied for nginx (PID 1) on Port 80.") {
		t.Fatalf("stderr = %q, want Occupant named", stderr)
	}
	if !strings.Contains(stderr, "Re-run with: sudo killport 80") {
		t.Fatalf("stderr = %q, want next sudo command", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none (signal was denied)", got)
	}
	if got := table.PolitePIDs(); len(got) != 0 {
		t.Fatalf("polite PIDs = %v, want none", got)
	}
}

func TestIPv4AndIPv6Listeners_BothTargetedAsDistinctOccupants(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 11, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 22, Port: 8080, Name: "node6", Kind: proctable.Listener},
	)

	stdout, _, exit := run(t, table, "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "Port 8080  PID 11  node") {
		t.Fatalf("stdout = %q, want IPv4 Occupant", stdout)
	}
	if !strings.Contains(stdout, "Port 8080  PID 22  node6") {
		t.Fatalf("stdout = %q, want IPv6 Occupant", stdout)
	}
	if !strings.Contains(stdout, "Killed 2 Occupants with SIGKILL.") {
		t.Fatalf("stdout = %q, want two Occupants killed", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 2 || got[0] != 11 || got[1] != 22 {
		t.Fatalf("forced PIDs = %v, want [11 22]", got)
	}
}

func TestDualStackOccupant_IsSignalledOnce(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
	)

	_, _, exit := run(t, table, "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242] once", got)
	}
}

func TestPIDOnTwoNamedPorts_IsSignalledOnce(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 4242, Port: 3000, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
	)

	stdout, _, exit := run(t, table, "3000", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "Port 3000  PID 4242  node") {
		t.Fatalf("stdout = %q, want Port 3000 listed", stdout)
	}
	if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
		t.Fatalf("stdout = %q, want Port 8080 listed", stdout)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGKILL.") {
		t.Fatalf("stdout = %q, want one Occupant killed", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242] once", got)
	}
}

func TestEmptyPortThenListener_ContinuesAndExitsZero(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	stdout, stderr, exit := run(t, table, "3000", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Nothing listening on Port 3000.") {
		t.Fatalf("stdout = %q, want empty-Port message", stdout)
	}
	if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
		t.Fatalf("stdout = %q, want remaining Listener reported", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242]", got)
	}
}

func TestPeer_IsNotTargetedByDefault(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
	)

	_, _, exit := run(t, table, "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want Listener [4242] only", got)
	}
}

func TestInvalidPortAmongValid_DoesNotKillAndExitsNonZero(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	stdout, stderr, exit := run(t, table, "8080", "0")
	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Invalid Port: 0 (expected 1–65535)") {
		t.Fatalf("stderr = %q, want invalid Port error", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none", got)
	}
}

func run(t *testing.T, table *memory.Table, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	exit = Run(args, table, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), exit
}
