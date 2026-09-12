# Kill Port

A local-developer tool invoked as `killport`: name a TCP Port, terminate its Occupant, in one step.

## Language

**Developer**:
The person running the tool on their own machine while working.
_Avoid_: user, operator, admin

**Port**:
A TCP port identified by a number from 1 to 65535 that the Developer wants freed.
_Avoid_: socket, endpoint

**Occupant**:
The process the Developer wants gone so the Port is free. By default that is a Listener; the Developer can widen the set to include Peers.
_Avoid_: holder, server, connection, “process occupied by the port”

**Listener**:
A process with a TCP socket in LISTEN state on the Port — the server bound to it.
_Avoid_: client, connection

**Peer**:
A process with a non-LISTEN TCP socket on the Port (a connected client). Not targeted unless the Developer widens the set.
_Avoid_: connection, socket, “all processes”
