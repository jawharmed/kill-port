package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jawharmed/kill-port/internal/memory"
	"github.com/jawharmed/kill-port/internal/proctable"
)

func TestHelp_PrintsEnglishUsageFlagsAndExamplesWithoutPort(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		t.Run(flag, func(t *testing.T) {
			stdout, stderr, exit := run(t, memory.New(), flag)

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Usage: killport") {
				t.Fatalf("stdout = %q, want English usage", stdout)
			}
			for _, want := range []string{"-h, --help", "-V, --version", "-n, --dry-run", "-s, --soft", "-w, --wait", "-v, --verbose", "-a, --all", "-l, --listen", "-i, --interactive"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout = %q, want flag %s", stdout, want)
				}
			}
			if !strings.Contains(stdout, "Listeners and Peers") {
				t.Fatalf("stdout = %q, want --all spelled out as Listeners and Peers", stdout)
			}
			if !strings.Contains(stdout, "killport 8080") {
				t.Fatalf("stdout = %q, want a usage example", stdout)
			}
			if !strings.Contains(stdout, "killport -s -w 8080") {
				t.Fatalf("stdout = %q, want a combined-shorts example", stdout)
			}
			if !strings.Contains(stdout, "Port") || !strings.Contains(stdout, "Occupant") {
				t.Fatalf("stdout = %q, want Port and Occupant in English help", stdout)
			}
		})
	}
}

func TestVersion_PrintsVersionWithoutPortAndExitsZero(t *testing.T) {
	for _, flag := range []string{"-V", "--version"} {
		t.Run(flag, func(t *testing.T) {
			stdout, stderr, exit := run(t, memory.New(), flag)

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "killport version 0.0.0-dev") {
				t.Fatalf("stdout = %q, want version line", stdout)
			}
		})
	}
}

func TestHelpAndVersion_IgnorePortArgumentsAndDoNotKill(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "help", args: []string{"--help", "8080"}, want: "Usage: killport"},
		{name: "version", args: []string{"--version", "8080"}, want: "killport version 0.0.0-dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := run(t, table, tc.args...)
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("stdout = %q, want %q", stdout, tc.want)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestUnknownFlag_PrintsEnglishErrorPointingAtHelpAndExitsNonZero(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	stdout, stderr, exit := run(t, table, "--nope", "8080")

	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Unknown flag: --nope") {
		t.Fatalf("stderr = %q, want unknown-flag error", stderr)
	}
	if !strings.Contains(stderr, "--help") {
		t.Fatalf("stderr = %q, want a pointer to --help", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none", got)
	}
	if got := table.PolitePIDs(); len(got) != 0 {
		t.Fatalf("polite PIDs = %v, want none", got)
	}
}

func TestDryRun_ListsOccupantsWithoutSignalling(t *testing.T) {
	for _, flag := range []string{"-n", "--dry-run"} {
		t.Run(flag, func(t *testing.T) {
			table := memory.New(
				memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
				memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
			)

			stdout, stderr, exit := run(t, table, flag, "8080")

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
				t.Fatalf("stdout = %q, want Listener listed", stdout)
			}
			if strings.Contains(stdout, "curl") {
				t.Fatalf("stdout = %q, want Peers omitted by default", stdout)
			}
			if strings.Contains(stdout, "Killed") {
				t.Fatalf("stdout = %q, want no kill summary", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestDryRun_EmptyPort_PrintsEnglishMessageAndExitsZero(t *testing.T) {
	stdout, stderr, exit := run(t, memory.New(), "--dry-run", "8080")

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

func TestDefaultOutput_ShowsProcessNameNotCommandLine(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:     4242,
		Port:    8080,
		Name:    "node",
		Command: "node server.js --watch",
		Kind:    proctable.Listener,
	})

	stdout, _, exit := run(t, table, "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "Port 8080  PID 4242  node\n") {
		t.Fatalf("stdout = %q, want process name only", stdout)
	}
	if strings.Contains(stdout, "node server.js --watch") {
		t.Fatalf("stdout = %q, want command line hidden without --verbose", stdout)
	}
}

func TestVerbose_ShowsFullCommandLineAndStillKills(t *testing.T) {
	for _, flag := range []string{"-v", "--verbose"} {
		t.Run(flag, func(t *testing.T) {
			table := memory.New(memory.Occupant{
				PID:     4242,
				Port:    8080,
				Name:    "node",
				Command: "node server.js --watch",
				Kind:    proctable.Listener,
			})

			stdout, stderr, exit := run(t, table, flag, "8080")
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node server.js --watch") {
				t.Fatalf("stdout = %q, want full command line", stdout)
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
		})
	}
}

func TestVerboseDryRun_InspectsCommandLineWithoutKilling(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "long", args: []string{"--verbose", "--dry-run", "8080"}},
		{name: "short", args: []string{"-n", "-v", "8080"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table := memory.New(memory.Occupant{
				PID:     4242,
				Port:    8080,
				Name:    "node",
				Command: "node server.js --watch",
				Kind:    proctable.Listener,
			})

			stdout, stderr, exit := run(t, table, tc.args...)
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node server.js --watch") {
				t.Fatalf("stdout = %q, want full command line", stdout)
			}
			if strings.Contains(stdout, "Killed") {
				t.Fatalf("stdout = %q, want no kill summary", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

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

	stdout, _, exit := run(t, table, "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if strings.Contains(stdout, "curl") {
		t.Fatalf("stdout = %q, want Peers omitted by default", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want Listener [4242] only", got)
	}
}

func TestAll_TargetsListenersAndPeers(t *testing.T) {
	for _, flag := range []string{"-a", "--all"} {
		t.Run(flag, func(t *testing.T) {
			table := memory.New(
				memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
				memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
			)

			stdout, stderr, exit := run(t, table, flag, "8080")
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
				t.Fatalf("stdout = %q, want Listener listed", stdout)
			}
			if !strings.Contains(stdout, "Port 8080  PID 99  curl") {
				t.Fatalf("stdout = %q, want Peer listed", stdout)
			}
			if !strings.Contains(stdout, "Killed 2 Occupants with SIGKILL.") {
				t.Fatalf("stdout = %q, want both Occupants killed", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 2 || got[0] != 4242 || got[1] != 99 {
				t.Fatalf("forced PIDs = %v, want [4242 99]", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestListen_TargetsListenersOnly(t *testing.T) {
	for _, flag := range []string{"-l", "--listen"} {
		t.Run(flag, func(t *testing.T) {
			table := memory.New(
				memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
				memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
			)

			stdout, stderr, exit := run(t, table, flag, "8080")
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
				t.Fatalf("stdout = %q, want Listener listed", stdout)
			}
			if strings.Contains(stdout, "curl") {
				t.Fatalf("stdout = %q, want Peers omitted", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
				t.Fatalf("forced PIDs = %v, want Listener [4242] only", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestScopeFlags_LastFlagWins(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantPIDs []int
		wantPeer bool
	}{
		{name: "--all then --listen", args: []string{"--all", "--listen", "8080"}, wantPIDs: []int{4242}, wantPeer: false},
		{name: "--listen then --all", args: []string{"--listen", "--all", "8080"}, wantPIDs: []int{4242, 99}, wantPeer: true},
		{name: "-a then -l", args: []string{"-a", "-l", "8080"}, wantPIDs: []int{4242}, wantPeer: false},
		{name: "-l then -a", args: []string{"-l", "-a", "8080"}, wantPIDs: []int{4242, 99}, wantPeer: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table := memory.New(
				memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
				memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
			)

			stdout, stderr, exit := run(t, table, tc.args...)
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if strings.Contains(stdout, "curl") != tc.wantPeer {
				t.Fatalf("stdout = %q, wantPeer=%v", stdout, tc.wantPeer)
			}
			got := table.ForcedPIDs()
			if len(got) != len(tc.wantPIDs) {
				t.Fatalf("forced PIDs = %v, want %v", got, tc.wantPIDs)
			}
			for i, pid := range tc.wantPIDs {
				if got[i] != pid {
					t.Fatalf("forced PIDs = %v, want %v", got, tc.wantPIDs)
				}
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestDryRun_RespectsResolvedScopeWithoutSignalling(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantPeer bool
	}{
		{name: "dry-run --all", args: []string{"--dry-run", "--all", "8080"}, wantPeer: true},
		{name: "dry-run -a", args: []string{"-n", "-a", "8080"}, wantPeer: true},
		{name: "dry-run --listen", args: []string{"--dry-run", "--listen", "8080"}, wantPeer: false},
		{name: "dry-run --all then --listen", args: []string{"-n", "--all", "--listen", "8080"}, wantPeer: false},
		{name: "dry-run --listen then --all", args: []string{"-n", "--listen", "--all", "8080"}, wantPeer: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table := memory.New(
				memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
				memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer},
			)

			stdout, stderr, exit := run(t, table, tc.args...)
			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Port 8080  PID 4242  node") {
				t.Fatalf("stdout = %q, want Listener listed", stdout)
			}
			if strings.Contains(stdout, "curl") != tc.wantPeer {
				t.Fatalf("stdout = %q, wantPeer=%v", stdout, tc.wantPeer)
			}
			if strings.Contains(stdout, "Killed") {
				t.Fatalf("stdout = %q, want no kill summary", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestInteractive_MultiplePIDsDeclined_KillsNothingAndExitsNonZero(t *testing.T) {
	cases := []struct {
		name  string
		flag  string
		stdin string
	}{
		{name: "n", flag: "--interactive", stdin: "n\n"},
		{name: "empty enter", flag: "-i", stdin: "\n"},
		{name: "no", flag: "--interactive", stdin: "no\n"},
		{name: "garbage", flag: "-i", stdin: "maybe\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			table := twoListeners(t)
			stdout, stderr, exit := runStdin(t, table, tc.stdin, tc.flag, "8080")

			if exit == 0 {
				t.Fatalf("exit code = 0, want non-zero")
			}
			if !strings.Contains(stdout, "Port 8080  PID 11  node") {
				t.Fatalf("stdout = %q, want Occupants listed before the prompt", stdout)
			}
			if !strings.Contains(stdout, "Port 8080  PID 22  node6") {
				t.Fatalf("stdout = %q, want Occupants listed before the prompt", stdout)
			}
			if strings.Contains(stdout, "Killed") {
				t.Fatalf("stdout = %q, want no kill summary after cancel", stdout)
			}
			if !strings.Contains(stderr, "Kill 2 Occupants?") {
				t.Fatalf("stderr = %q, want confirmation prompt", stderr)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestInteractive_MultiplePIDsAccepted_ProceedsToKill(t *testing.T) {
	for _, answer := range []string{"y\n", "Y\n", "yes\n", "YES\n", "oui\n", " Oui \n"} {
		t.Run(strings.TrimSpace(answer), func(t *testing.T) {
			table := twoListeners(t)
			stdout, stderr, exit := runStdin(t, table, answer, "--interactive", "8080")

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if !strings.Contains(stderr, "Kill 2 Occupants?") {
				t.Fatalf("stderr = %q, want confirmation prompt", stderr)
			}
			if !strings.Contains(stdout, "Killed 2 Occupants with SIGKILL.") {
				t.Fatalf("stdout = %q, want SIGKILL success line", stdout)
			}
			if got := table.ForcedPIDs(); len(got) != 2 || got[0] != 11 || got[1] != 22 {
				t.Fatalf("forced PIDs = %v, want [11 22]", got)
			}
			if got := table.PolitePIDs(); len(got) != 0 {
				t.Fatalf("polite PIDs = %v, want none", got)
			}
		})
	}
}

func TestInteractive_SinglePID_NeverPromptsAndStillKills(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})

	stdout, stderr, exit := runStdin(t, table, "n\n", "--interactive", "8080")

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if strings.Contains(stderr, "Kill") {
		t.Fatalf("stderr = %q, want no prompt for a single PID", stderr)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGKILL.") {
		t.Fatalf("stdout = %q, want SIGKILL success line", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242]", got)
	}
}

func TestInteractive_SamePIDTwice_NeverPrompts(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
	)

	_, stderr, exit := runStdin(t, table, "n\n", "-i", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if strings.Contains(stderr, "Kill") {
		t.Fatalf("stderr = %q, want no prompt for one distinct PID", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242] once", got)
	}
}

func TestInteractive_Cancel_DoesNotWaitOrSignal(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 11, Port: 8080, Name: "node", Kind: proctable.Listener, StayAliveAfterForced: true},
		memory.Occupant{PID: 22, Port: 8080, Name: "node6", Kind: proctable.Listener, StayAliveAfterForced: true},
	)
	clk := newFakeClock()

	stdout, stderr, exit := runClockStdin(t, table, clk, "\n", "-i", "-w", "8080")
	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if strings.Contains(stdout, "Killed") {
		t.Fatalf("stdout = %q, want no kill summary after cancel", stdout)
	}
	if !strings.Contains(stderr, "Kill 2 Occupants?") {
		t.Fatalf("stderr = %q, want confirmation prompt", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none", got)
	}
	if got := table.PolitePIDs(); len(got) != 0 {
		t.Fatalf("polite PIDs = %v, want none", got)
	}
	if clk.elapsed() != 0 {
		t.Fatalf("elapsed = %s, want no wait after cancel", clk.elapsed())
	}
}

func TestSoft_OccupantsDieAfterPolite_SkipsForce(t *testing.T) {
	for _, flag := range []string{"-s", "--soft"} {
		t.Run(flag, func(t *testing.T) {
			table := memory.New(memory.Occupant{
				PID:         4242,
				Port:        8080,
				Name:        "node",
				Kind:        proctable.Listener,
				DieOnPolite: true,
			})
			clk := newFakeClock()

			stdout, stderr, exit := runClock(t, table, clk, flag, "8080")

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Killed 1 Occupant with SIGTERM.") {
				t.Fatalf("stdout = %q, want SIGTERM-only success line", stdout)
			}
			if got := table.PolitePIDs(); len(got) != 1 || got[0] != 4242 {
				t.Fatalf("polite PIDs = %v, want [4242]", got)
			}
			if got := table.ForcedPIDs(); len(got) != 0 {
				t.Fatalf("forced PIDs = %v, want none", got)
			}
			if clk.elapsed() != 0 {
				t.Fatalf("elapsed = %s, want no grace wait when Occupants already died", clk.elapsed())
			}
		})
	}
}

func TestSoft_SurvivorsAreForcedAfterGrace(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})
	clk := newFakeClock()

	stdout, stderr, exit := runClock(t, table, clk, "--soft", "8080")

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGTERM then SIGKILL.") {
		t.Fatalf("stdout = %q, want SIGTERM-then-SIGKILL success line", stdout)
	}
	if got := table.PolitePIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("polite PIDs = %v, want [4242]", got)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242]", got)
	}
	if clk.elapsed() < 3*time.Second {
		t.Fatalf("elapsed = %s, want 3s grace before force", clk.elapsed())
	}
}

func TestSoft_ForcesOnlySurvivors(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 11, Port: 8080, Name: "node", Kind: proctable.Listener, DieOnPolite: true},
		memory.Occupant{PID: 22, Port: 8080, Name: "wedged", Kind: proctable.Listener},
	)
	clk := newFakeClock()

	stdout, _, exit := runClock(t, table, clk, "-s", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "Killed 2 Occupants with SIGTERM then SIGKILL.") {
		t.Fatalf("stdout = %q, want mixed outcome line", stdout)
	}
	if got := table.PolitePIDs(); len(got) != 2 || got[0] != 11 || got[1] != 22 {
		t.Fatalf("polite PIDs = %v, want [11 22]", got)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 22 {
		t.Fatalf("forced PIDs = %v, want survivor [22] only", got)
	}
}

func TestWait_AlreadyFree_SucceedsImmediately(t *testing.T) {
	for _, flag := range []string{"-w", "--wait"} {
		t.Run(flag, func(t *testing.T) {
			clk := newFakeClock()
			stdout, stderr, exit := runClock(t, memory.New(), clk, flag, "8080")

			if exit != 0 {
				t.Fatalf("exit code = %d, want 0", exit)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if !strings.Contains(stdout, "Nothing listening on Port 8080.") {
				t.Fatalf("stdout = %q, want empty-Port message", stdout)
			}
			if clk.elapsed() != 0 {
				t.Fatalf("elapsed = %s, want immediate success on a free Port", clk.elapsed())
			}
		})
	}
}

func TestWait_AfterKill_ReturnsWhenOccupantsAreGone(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:  4242,
		Port: 8080,
		Name: "node",
		Kind: proctable.Listener,
	})
	clk := newFakeClock()

	stdout, stderr, exit := runClock(t, table, clk, "--wait", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGKILL.") {
		t.Fatalf("stdout = %q, want SIGKILL success line", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want [4242]", got)
	}
	if clk.elapsed() != 0 {
		t.Fatalf("elapsed = %s, want no extra wait once Occupants are gone", clk.elapsed())
	}
}

func TestWait_Timeout_NamesOccupiedPortsAndExitsNonZero(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 1, Port: 3000, Name: "vite", Kind: proctable.Listener, StayAliveAfterForced: true},
		memory.Occupant{PID: 2, Port: 8080, Name: "node", Kind: proctable.Listener, StayAliveAfterForced: true},
	)
	clk := newFakeClock()

	stdout, stderr, exit := runClock(t, table, clk, "--wait", "3000", "8080")
	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stdout, "Killed 2 Occupants with SIGKILL.") {
		t.Fatalf("stdout = %q, want kill summary before the timeout", stdout)
	}
	if !strings.Contains(stderr, "Ports 3000, 8080 are still occupied.") {
		t.Fatalf("stderr = %q, want named Ports still occupied", stderr)
	}
	if clk.elapsed() < 10*time.Second {
		t.Fatalf("elapsed = %s, want 10s wait timeout", clk.elapsed())
	}
}

func TestWait_Timeout_SinglePort(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:                  4242,
		Port:                 8080,
		Name:                 "node",
		Kind:                 proctable.Listener,
		StayAliveAfterForced: true,
	})
	clk := newFakeClock()

	_, stderr, exit := runClock(t, table, clk, "-w", "8080")
	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr, "Port 8080 is still occupied.") {
		t.Fatalf("stderr = %q, want named Port still occupied", stderr)
	}
}

func TestWait_RespectsListenScope_PeerMayRemain(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer, StayAliveAfterForced: true},
	)
	clk := newFakeClock()

	_, stderr, exit := runClock(t, table, clk, "--wait", "--listen", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0 (Peer is out of listen scope)", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("forced PIDs = %v, want Listener [4242] only", got)
	}
}

func TestWait_All_WaitsForPeersToo(t *testing.T) {
	table := memory.New(
		memory.Occupant{PID: 4242, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 99, Port: 8080, Name: "curl", Kind: proctable.Peer, StayAliveAfterForced: true},
	)
	clk := newFakeClock()

	_, stderr, exit := runClock(t, table, clk, "--all", "--wait", "8080")
	if exit == 0 {
		t.Fatalf("exit code = 0, want non-zero while a Peer remains")
	}
	if !strings.Contains(stderr, "Port 8080 is still occupied.") {
		t.Fatalf("stderr = %q, want Port still occupied by Peer", stderr)
	}
	if got := table.ForcedPIDs(); len(got) != 2 || got[0] != 4242 || got[1] != 99 {
		t.Fatalf("forced PIDs = %v, want [4242 99]", got)
	}
}

func TestCombinedShorts_SoftAndWait(t *testing.T) {
	table := memory.New(memory.Occupant{
		PID:         4242,
		Port:        8080,
		Name:        "node",
		Kind:        proctable.Listener,
		DieOnPolite: true,
	})
	clk := newFakeClock()

	stdout, stderr, exit := runClock(t, table, clk, "-s", "-w", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Killed 1 Occupant with SIGTERM.") {
		t.Fatalf("stdout = %q, want SIGTERM-only success line", stdout)
	}
	if got := table.PolitePIDs(); len(got) != 1 || got[0] != 4242 {
		t.Fatalf("polite PIDs = %v, want [4242]", got)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none", got)
	}
	if clk.elapsed() != 0 {
		t.Fatalf("elapsed = %s, want no wait once Occupants died on SIGTERM", clk.elapsed())
	}
}

func TestDryRun_WaitAndInteractive_ListsOnly(t *testing.T) {
	table := twoListeners(t)
	clk := newFakeClock()

	stdout, stderr, exit := runClockStdin(t, table, clk, "n\n", "--dry-run", "--wait", "--interactive", "8080")
	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}
	if strings.Contains(stderr, "Kill") {
		t.Fatalf("stderr = %q, want no prompt on dry-run", stderr)
	}
	if strings.Contains(stdout, "Killed") {
		t.Fatalf("stdout = %q, want no kill summary", stdout)
	}
	if !strings.Contains(stdout, "PID 11") || !strings.Contains(stdout, "PID 22") {
		t.Fatalf("stdout = %q, want Occupants listed", stdout)
	}
	if got := table.ForcedPIDs(); len(got) != 0 {
		t.Fatalf("forced PIDs = %v, want none", got)
	}
	if got := table.PolitePIDs(); len(got) != 0 {
		t.Fatalf("polite PIDs = %v, want none", got)
	}
	if clk.elapsed() != 0 {
		t.Fatalf("elapsed = %s, want no wait-on-kill during dry-run", clk.elapsed())
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
	return runStdin(t, table, "", args...)
}

func runStdin(t *testing.T, table *memory.Table, stdin string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	exit = Run(args, table, strings.NewReader(stdin), &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), exit
}

func twoListeners(t *testing.T) *memory.Table {
	t.Helper()
	return memory.New(
		memory.Occupant{PID: 11, Port: 8080, Name: "node", Kind: proctable.Listener},
		memory.Occupant{PID: 22, Port: 8080, Name: "node6", Kind: proctable.Listener},
	)
}

type fakeClock struct {
	current time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{current: time.Unix(1_700_000_000, 0).UTC()}
}

func (c *fakeClock) clock() clock {
	return clock{
		now:   func() time.Time { return c.current },
		sleep: func(d time.Duration) { c.current = c.current.Add(d) },
	}
}

func (c *fakeClock) elapsed() time.Duration {
	return c.current.Sub(time.Unix(1_700_000_000, 0).UTC())
}

func runClock(t *testing.T, table *memory.Table, clk *fakeClock, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	return runClockStdin(t, table, clk, "", args...)
}

func runClockStdin(t *testing.T, table *memory.Table, clk *fakeClock, stdin string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	exit = runCLI(args, table, strings.NewReader(stdin), &outBuf, &errBuf, clk.clock())
	return outBuf.String(), errBuf.String(), exit
}
