//go:build linux

package liner

import "golang.org/x/sys/unix"

const (
	getTermios = unix.TCGETS
	setTermios = unix.TCSETS

	icrnl  = unix.ICRNL
	inpck  = unix.INPCK
	istrip = unix.ISTRIP
	ixon   = unix.IXON
	cs8    = unix.CS8
	isig   = unix.ISIG
	icanon = unix.ICANON
	iexten = unix.IEXTEN

	cursorColumn = false
)

type termios struct {
	unix.Termios
}
