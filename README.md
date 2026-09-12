# killport

Free a TCP Port by terminating its Listener.

`killport` is a local Developer tool. Name one or more Ports; it finds the Listeners (TCP LISTEN) and terminates them immediately with SIGKILL.

## Usage

```
killport 8080
killport 3000 8080 4200
```

Default output shows each Occupant's Port, PID, and process name, then a short success line:

```
Port 8080  PID 4242  node
Killed 1 Occupant with SIGKILL.
```

An already-free Port is success: `killport` prints that nothing is listening and exits 0.

IPv4 and IPv6 Listeners on the same Port number are both targeted. A dual-stack Occupant, or a PID that occupies two named Ports, is signalled only once.

Connected Peers (non-LISTEN TCP) are left alone.

## Errors

Missing or invalid Ports (not an integer in 1–65535) print an English error on stderr and exit non-zero.

If the Developer cannot kill an Occupant, `killport` names it and prints the next command to run. It never elevates itself:

```
Permission denied for nginx (PID 1) on Port 80.
Re-run with: sudo killport 80
```

## Install

Build the `killport` binary with Go:

```
go build -o killport ./cmd/killport
```

## Licence

MIT. See `LICENSE`.
