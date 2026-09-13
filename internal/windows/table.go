//go:build windows

package windows

import (
	"errors"
	"syscall"
	"unsafe"

	"github.com/jawharmed/kill-port/internal/proctable"
)

const (
	errorInvalidHandle             syscall.Errno = 6
	errorInvalidParameter          syscall.Errno = 87
	stillActive                    uint32        = 259
	processQueryLimitedInformation               = 0x1000
	tcpStateListen                               = 2
	tcpTableOwnerPIDAll                          = 5
	wmClose                                      = 0x0010
)

var (
	modIphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modNtdll    = syscall.NewLazyDLL("ntdll.dll")

	procGetExtendedTcpTable        = modIphlpapi.NewProc("GetExtendedTcpTable")
	procEnumWindows                = modUser32.NewProc("EnumWindows")
	procGetWindowThreadProcessId   = modUser32.NewProc("GetWindowThreadProcessId")
	procPostMessageW               = modUser32.NewProc("PostMessageW")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
	procNtQueryInformationProcess  = modNtdll.NewProc("NtQueryInformationProcess")

	enumCloseWindowCallback = syscall.NewCallback(enumCloseWindow)
)

type Table struct{}

func New() *Table {
	return &Table{}
}

func (t *Table) List(port proctable.Port, scope proctable.Scope) ([]proctable.Occupant, error) {
	rows, err := tcpRows()
	if err != nil {
		return nil, err
	}
	return occupantsFor(rows, port, scope), nil
}

func (t *Table) SignalPolite(pid int) error {
	h, err := openTerminate(pid)
	if err != nil {
		return err
	}
	syscall.CloseHandle(h)
	closeWindowsOf(uint32(pid))
	return nil
}

func (t *Table) SignalForced(pid int) error {
	h, err := openTerminate(pid)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	if err := syscall.TerminateProcess(h, 1); err != nil {
		return mapProcErr(pid, err)
	}
	return nil
}

func (t *Table) Alive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}

func openTerminate(pid int) (syscall.Handle, error) {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return 0, mapProcErr(pid, err)
	}
	return h, nil
}

func mapProcErr(pid int, err error) error {
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return proctable.PermissionError{PID: pid}
	}
	if errors.Is(err, errorInvalidParameter) || errors.Is(err, errorInvalidHandle) {
		return nil
	}
	return err
}

func closeWindowsOf(pid uint32) {
	_, _, _ = procEnumWindows.Call(enumCloseWindowCallback, uintptr(pid))
}

func enumCloseWindow(hwnd uintptr, lparam uintptr) uintptr {
	var owner uint32
	_, _, _ = procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
	if uintptr(owner) == lparam {
		_, _, _ = procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
	return 1
}
