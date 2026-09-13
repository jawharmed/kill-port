# killport

Free a TCP Port by terminating its Occupant.

`killport` is a local Developer tool. Name one or more Ports; by default it finds the Listeners (TCP LISTEN) and terminates them immediately (SIGKILL on Unix, force-terminate on Windows). Pass `--all` to target Listeners and Peers. The same `killport` command works on macOS, Linux, and Windows (the Windows file is `killport.exe`).

## Install

Each version tag (`v1.0.0`, and so on) publishes **one** GitHub Release with `killport` binaries for macOS, Linux, and Windows. Homebrew and Scoop install those binaries onto PATH. Homebrew and Scoop need a published Release: the first `v*` tag fills the formula and Scoop manifest checksums.

### GitHub Releases

Download the archive for your OS from [GitHub Releases](https://github.com/jawharmed/kill-port/releases), extract `killport` (`killport.exe` on Windows), and put it on your PATH.

| OS | Architecture | Archive |
| --- | --- | --- |
| macOS | Intel (amd64) | `killport_Darwin_x86_64.tar.gz` |
| macOS | Apple Silicon (arm64) | `killport_Darwin_arm64.tar.gz` |
| Linux | amd64 | `killport_Linux_x86_64.tar.gz` |
| Windows | amd64 | `killport_Windows_x86_64.zip` |

### Homebrew

A project tap (not homebrew-core):

```
brew tap jawharmed/kill-port https://github.com/jawharmed/kill-port
brew install killport
```

### Scoop

Add this repository as a Scoop bucket:

```
scoop bucket add kill-port https://github.com/jawharmed/kill-port
scoop install killport
```

## Syntax

```
killport [flags] <port> [port...]
```

`-h` / `--help` and `-V` / `--version` work without a Port.

## Flags

| Short | Long | Meaning |
| --- | --- | --- |
| `-h` | `--help` | Show this help and exit; no Port required |
| `-V` | `--version` | Print the version and exit; no Port required |
| `-n` | `--dry-run` | List Occupants without signalling |
| `-s` | `--soft` | SIGTERM first; SIGKILL after 3s if still alive |
| `-w` | `--wait` | Wait until Occupants are gone, or 10s |
| `-v` | `--verbose` | Show each Occupant's full command line |
| `-a` | `--all` | Target Listeners and Peers on the named Ports |
| `-l` | `--listen` | Target Listeners only (default) |
| `-i` | `--interactive` | Confirm when more than one PID would be signalled |

`--all` means Listeners and Peers (non-LISTEN TCP) on the named Ports. `--listen` forces Listener-only targeting, including to undo `--all` in the same command. If both appear, the last one wins.

On Windows, `--soft` is a polite terminate then force after 3 seconds; the default is force immediately.

Combined shorts work: `killport -s -w 8080`.

## Examples

```
killport 8080
killport -n 3000 8080
killport 3000 8080 4200
killport --all 8080
killport -s -w 8080
```

Default output shows each Occupant's Port, PID, and process name, then a short success line:

```
Port 8080  PID 4242  node
Killed 1 Occupant with SIGKILL.
```

`-n` / `--dry-run` lists Occupants the same way and exits 0 without signalling.

An already-free Port is success: `killport` prints that nothing is listening and exits 0.

IPv4 and IPv6 Listeners on the same Port number are both targeted. A dual-stack Occupant, or a PID that occupies two named Ports, is signalled only once.

By default, connected Peers are left alone.

## Errors

Missing or invalid Ports (not an integer in 1–65535) print an English error on stderr and exit non-zero.

If the Developer cannot kill an Occupant, `killport` names it and prints the next step. It never elevates itself. On Unix that is `sudo`; on Windows, re-run from an elevated terminal:

```
Permission denied for nginx (PID 1) on Port 80.
Re-run with: sudo killport 80
```

```
Permission denied for nginx (PID 1) on Port 80.
Re-run from an elevated terminal: killport 80
```

## Licence

MIT. See `LICENSE`.
