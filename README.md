# killport

Free a TCP Port by terminating its Occupant.

`killport` is a local Developer tool. Name one or more Ports; by default it finds the Listeners (TCP LISTEN) and terminates them immediately with SIGKILL. Pass `--all` to also terminate Peers.

## Usage

```
killport 8080
killport 3000 8080 4200
killport -n 3000 8080
killport --all 8080
killport -s -w 8080
```

Default output shows each Occupant's Port, PID, and process name, then a short success line:

```
Port 8080  PID 4242  node
Killed 1 Occupant with SIGKILL.
```

`-n` / `--dry-run` lists Occupants the same way and exits 0 without signalling. `-v` / `--verbose` shows each Occupant's full command line. `-h` / `--help` and `-V` / `--version` work without a Port.

`-a` / `--all` targets Listeners and Peers (non-LISTEN TCP) on the named Ports. `-l` / `--listen` forces Listener-only targeting (the default), including to undo `--all` in the same command. If both appear, the last one wins.

`-s` / `--soft` sends SIGTERM first, then SIGKILL after 3 seconds only if the Occupant is still alive. `-w` / `--wait` does not return until matching Occupants are gone or 10 seconds pass. `-i` / `--interactive` asks for confirmation only when more than one PID would be signalled. Combined shorts work: `killport -s -w 8080`.

An already-free Port is success: `killport` prints that nothing is listening and exits 0.

IPv4 and IPv6 Listeners on the same Port number are both targeted. A dual-stack Occupant, or a PID that occupies two named Ports, is signalled only once.

By default, connected Peers are left alone.

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
