//go:build windows

package windows

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/jawharmed/kill-port/internal/proctable"
)

func TestMain(m *testing.M) {
	switch os.Getenv("KILLPORT_FIXTURE") {
	case "listen":
		listenFixture("tcp", "127.0.0.1:0")
		os.Exit(0)
	case "listen6":
		listenFixture("tcp", "[::1]:0")
		os.Exit(0)
	case "udp":
		udpFixture()
		os.Exit(0)
	case "peer":
		peerFixture()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestList_FindsTCPListener(t *testing.T) {
	cmd, port, _ := startFixture(t, "listen")
	table := New()

	occupants, err := table.List(port, proctable.ListenersOnly)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !hasPID(occupants, cmd.Process.Pid) {
		t.Fatalf("occupants = %v, want Listener PID %d", occupants, cmd.Process.Pid)
	}
	for _, occupant := range occupants {
		if occupant.PID == cmd.Process.Pid && occupant.Kind != proctable.Listener {
			t.Fatalf("Kind = %v, want Listener", occupant.Kind)
		}
		if occupant.PID == cmd.Process.Pid && occupant.Port != port {
			t.Fatalf("Port = %d, want %d", occupant.Port, port)
		}
		if occupant.PID == cmd.Process.Pid && occupant.Name == "" {
			t.Fatalf("Name is empty, want a process name")
		}
	}
}

func TestList_FindsIPv6TCPListener(t *testing.T) {
	if _, err := net.Listen("tcp", "[::1]:0"); err != nil {
		t.Skipf("IPv6 not available: %v", err)
	}
	cmd, port, _ := startFixture(t, "listen6")
	table := New()

	occupants, err := table.List(port, proctable.ListenersOnly)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !hasPID(occupants, cmd.Process.Pid) {
		t.Fatalf("occupants = %v, want IPv6 Listener PID %d", occupants, cmd.Process.Pid)
	}
	for _, occupant := range occupants {
		if occupant.PID == cmd.Process.Pid && occupant.Kind != proctable.Listener {
			t.Fatalf("Kind = %v, want Listener", occupant.Kind)
		}
	}
}

func TestList_IgnoresUDP(t *testing.T) {
	_, port, _ := startFixture(t, "udp")
	occupants, err := New().List(port, proctable.ListenersOnly)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(occupants) != 0 {
		t.Fatalf("occupants = %v, want none (UDP is not TCP)", occupants)
	}
}

func TestForcedSignal_KillsListener(t *testing.T) {
	cmd, _, exited := startFixture(t, "listen")
	table := New()
	pid := cmd.Process.Pid

	if !table.Alive(pid) {
		t.Fatalf("fixture PID %d not alive before kill", pid)
	}
	if err := table.SignalForced(pid); err != nil {
		t.Fatalf("SignalForced: %v", err)
	}
	waitExited(t, exited)
	if table.Alive(pid) {
		t.Fatalf("fixture PID %d still alive after force-terminate", pid)
	}
}

func TestPoliteSignal_DoesNotForceKill(t *testing.T) {
	cmd, _, exited := startFixture(t, "listen")
	table := New()
	pid := cmd.Process.Pid

	if err := table.SignalPolite(pid); err != nil {
		t.Fatalf("SignalPolite: %v", err)
	}
	select {
	case <-exited:
		t.Fatal("Occupant died after polite signal; want it still alive")
	case <-time.After(150 * time.Millisecond):
	}
	if !table.Alive(pid) {
		t.Fatalf("fixture PID %d died after polite terminate", pid)
	}
	if err := table.SignalForced(pid); err != nil {
		t.Fatalf("SignalForced: %v", err)
	}
	waitExited(t, exited)
}

func TestList_ListenersOnlyOmitsPeers(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := proctable.Port(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
	}()

	peer, _, _ := startFixture(t, "peer", "KILLPORT_ADDR="+ln.Addr().String())
	table := New()

	waitUntil(t, 2*time.Second, func() bool {
		occupants, err := table.List(port, proctable.ListenersAndPeers)
		return err == nil && hasPID(occupants, peer.Process.Pid)
	})

	listeners, err := table.List(port, proctable.ListenersOnly)
	if err != nil {
		t.Fatalf("List ListenersOnly: %v", err)
	}
	if hasPID(listeners, peer.Process.Pid) {
		t.Fatalf("Peer PID %d listed as Listener: %v", peer.Process.Pid, listeners)
	}

	all, err := table.List(port, proctable.ListenersAndPeers)
	if err != nil {
		t.Fatalf("List ListenersAndPeers: %v", err)
	}
	if !hasPID(all, peer.Process.Pid) {
		t.Fatalf("occupants = %v, want Peer PID %d", all, peer.Process.Pid)
	}
}

func startFixture(t *testing.T, fixture string, extraEnv ...string) (*exec.Cmd, proctable.Port, <-chan struct{}) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "KILLPORT_FIXTURE="+fixture)
	cmd.Env = append(cmd.Env, extraEnv...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-exited
	})

	port, err := readPort(stdout)
	if err != nil {
		t.Fatalf("fixture %s: %v", fixture, err)
	}
	return cmd, port, exited
}

func waitExited(t *testing.T, exited <-chan struct{}) {
	t.Helper()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("Occupant still running after forced kill")
	}
}

func readPort(r io.Reader) (proctable.Port, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(trimNewline(line))
	if err != nil {
		return 0, fmt.Errorf("fixture port %q: %w", line, err)
	}
	return proctable.Port(n), nil
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}

func hasPID(occupants []proctable.Occupant, pid int) bool {
	for _, occupant := range occupants {
		if occupant.PID == pid {
			return true
		}
	}
	return false
}

func listenFixture(network, address string) {
	ln, err := net.Listen(network, address)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(ln.Addr().(*net.TCPAddr).Port)
	_ = os.Stdout.Sync()
	block()
}

func udpFixture() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(conn.LocalAddr().(*net.UDPAddr).Port)
	_ = os.Stdout.Sync()
	block()
}

func peerFixture() {
	conn, err := net.Dial("tcp", os.Getenv("KILLPORT_ADDR"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(conn.LocalAddr().(*net.TCPAddr).Port)
	_ = os.Stdout.Sync()
	block()
}

func block() {
	for {
		time.Sleep(time.Hour)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out after %s", timeout)
}
