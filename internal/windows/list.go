//go:build windows

package windows

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/jawharmed/kill-port/internal/proctable"
)

type tcpRow struct {
	pid        uint32
	localPort  uint16
	remotePort uint16
	listen     bool
}

type mibTCPRowOwnerPID struct {
	state      uint32
	localAddr  uint32
	localPort  uint32
	remoteAddr uint32
	remotePort uint32
	owningPID  uint32
}

type mibTCP6RowOwnerPID struct {
	localAddr     [16]byte
	localScopeID  uint32
	localPort     uint32
	remoteAddr    [16]byte
	remoteScopeID uint32
	remotePort    uint32
	state         uint32
	owningPID     uint32
}

func tcpRows() ([]tcpRow, error) {
	v4, err := tcpTable(syscall.AF_INET)
	if err != nil {
		return nil, err
	}
	v6, err := tcpTable(syscall.AF_INET6)
	if err != nil {
		return nil, err
	}
	return append(parseIPv4(v4), parseIPv6(v6)...), nil
}

func tcpTable(family uint32) ([]byte, error) {
	var size uint32
	var buf []byte
	for try := 0; try < 8; try++ {
		var ptr uintptr
		if len(buf) > 0 {
			ptr = uintptr(unsafe.Pointer(&buf[0]))
		}
		r1, _, _ := procGetExtendedTcpTable.Call(
			ptr,
			uintptr(unsafe.Pointer(&size)),
			1,
			uintptr(family),
			tcpTableOwnerPIDAll,
			0,
		)
		if r1 == 0 {
			return buf, nil
		}
		if syscall.Errno(r1) != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, syscall.Errno(r1)
		}
		buf = make([]byte, size)
	}
	return nil, syscall.ERROR_INSUFFICIENT_BUFFER
}

func parseIPv4(buf []byte) []tcpRow {
	return parseTCP(buf, unsafe.Sizeof(mibTCPRowOwnerPID{}), func(p unsafe.Pointer) tcpRow {
		row := (*mibTCPRowOwnerPID)(p)
		return tcpRow{
			pid:        row.owningPID,
			localPort:  syscall.Ntohs(uint16(row.localPort)),
			remotePort: syscall.Ntohs(uint16(row.remotePort)),
			listen:     row.state == tcpStateListen,
		}
	})
}

func parseIPv6(buf []byte) []tcpRow {
	return parseTCP(buf, unsafe.Sizeof(mibTCP6RowOwnerPID{}), func(p unsafe.Pointer) tcpRow {
		row := (*mibTCP6RowOwnerPID)(p)
		return tcpRow{
			pid:        row.owningPID,
			localPort:  syscall.Ntohs(uint16(row.localPort)),
			remotePort: syscall.Ntohs(uint16(row.remotePort)),
			listen:     row.state == tcpStateListen,
		}
	})
}

func parseTCP(buf []byte, rowSize uintptr, read func(unsafe.Pointer) tcpRow) []tcpRow {
	if len(buf) < 4 {
		return nil
	}
	n := int(*(*uint32)(unsafe.Pointer(&buf[0])))
	need := 4 + n*int(rowSize)
	if n <= 0 || len(buf) < need {
		return nil
	}
	out := make([]tcpRow, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, read(unsafe.Pointer(&buf[4+i*int(rowSize)])))
	}
	return out
}

func occupantsFor(rows []tcpRow, port proctable.Port, scope proctable.Scope) []proctable.Occupant {
	type occupantRow struct {
		pid    int
		listen bool
	}
	byPID := map[int]*occupantRow{}
	order := make([]int, 0)
	for _, row := range rows {
		if row.pid == 0 || !rowMatches(row, port, scope) {
			continue
		}
		pid := int(row.pid)
		existing := byPID[pid]
		if existing == nil {
			byPID[pid] = &occupantRow{pid: pid, listen: row.listen}
			order = append(order, pid)
			continue
		}
		if row.listen {
			existing.listen = true
		}
	}

	names := exeNames()
	occupants := make([]proctable.Occupant, 0, len(order))
	for _, pid := range order {
		kind := proctable.Peer
		if byPID[pid].listen {
			kind = proctable.Listener
		}
		occupants = append(occupants, proctable.Occupant{
			PID:     pid,
			Port:    port,
			Name:    names[uint32(pid)],
			Command: commandLine(uint32(pid)),
			Kind:    kind,
		})
	}
	return occupants
}

func rowMatches(row tcpRow, port proctable.Port, scope proctable.Scope) bool {
	want := uint16(port)
	if row.listen {
		return row.localPort == want
	}
	if scope == proctable.ListenersOnly {
		return false
	}
	return row.localPort == want || row.remotePort == want
}

func exeNames() map[uint32]string {
	names := map[uint32]string{}
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return names
	}
	defer syscall.CloseHandle(snap)
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := syscall.Process32First(snap, &entry); err != nil {
		return names
	}
	for {
		names[entry.ProcessID] = syscall.UTF16ToString(entry.ExeFile[:])
		if err := syscall.Process32Next(snap, &entry); err != nil {
			return names
		}
	}
}

func commandLine(pid uint32) string {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)
	if line := queryCommandLine(h); line != "" {
		return line
	}
	return queryImagePath(h)
}

func queryCommandLine(h syscall.Handle) string {
	const processCommandLineInformation = 60
	var needed uint32
	_ = ntQuery(h, processCommandLineInformation, nil, 0, &needed)
	if needed == 0 {
		return ""
	}
	buf := make([]byte, needed)
	if err := ntQuery(h, processCommandLineInformation, unsafe.Pointer(&buf[0]), needed, &needed); err != nil {
		return ""
	}
	type unicodeString struct {
		Length        uint16
		MaximumLength uint16
		Buffer        *uint16
	}
	u := (*unicodeString)(unsafe.Pointer(&buf[0]))
	if u.Length == 0 || u.Buffer == nil {
		return ""
	}
	line := syscall.UTF16ToString(unsafe.Slice(u.Buffer, int(u.Length)/2))
	runtime.KeepAlive(buf)
	return line
}

func queryImagePath(h syscall.Handle) string {
	n := uint32(syscall.MAX_PATH)
	buf := make([]uint16, n)
	r1, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(h),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&n)),
	)
	if r1 == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func ntQuery(h syscall.Handle, class uint32, info unsafe.Pointer, length uint32, needed *uint32) error {
	r1, _, _ := procNtQueryInformationProcess.Call(
		uintptr(h),
		uintptr(class),
		uintptr(info),
		uintptr(length),
		uintptr(unsafe.Pointer(needed)),
	)
	if r1 != 0 {
		return syscall.Errno(r1)
	}
	return nil
}
