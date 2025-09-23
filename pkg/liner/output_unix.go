package liner

import (
	"golang.org/x/sys/unix"
	"unsafe"
)

func (s *State) getColumns() bool {
	var ws winSize
	ok, _, _ := unix.Syscall(unix.SYS_IOCTL, uintptr(unix.Stdout),
		unix.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if int(ok) < 0 {
		return false
	}
	s.columns = int(ws.col)
	return true
}
