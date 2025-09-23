//go:build linux

package liner

import (
	"golang.org/x/sys/unix"
	"unsafe"
)

func (mode *termios) ApplyMode() error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(unix.Stdin), setTermios, uintptr(unsafe.Pointer(mode)))

	if errno != 0 {
		return errno
	}
	return nil
}

// TerminalMode returns the current terminal input mode as an InputModeSetter.
//
// This function is provided for convenience, and should
// not be necessary for most users of liner.
func TerminalMode() (ModeApplier, error) {
	return getMode(unix.Stdin)
}

func getMode(handle int) (*termios, error) {
	var mode termios
	var err error
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(handle), getTermios, uintptr(unsafe.Pointer(&mode)))
	if errno != 0 {
		err = errno
	}

	return &mode, err
}
