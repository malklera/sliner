// Package liner allows you to have a pre-filled prompt for input
package liner

import (
	"bufio"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
	"strings"
)

type nexter struct {
	r   rune
	err error
}

// State represents an open terminal
type State struct {
	commonState
	origMode    termios
	defaultMode termios
	next        <-chan nexter
	winch       chan os.Signal
	pending     []rune
	useCHA      bool
}

// NewLiner initializes a new *State, and sets the terminal into raw mode. To
// restore the terminal to its previous state, call State.Close().
func NewLiner() *State {
	var s State
	s.r = bufio.NewReader(os.Stdin)

	s.terminalSupported = TerminalSupported()
	if m, err := TerminalMode(); err == nil {
		s.origMode = *m.(*termios)
	} else {
		s.inputRedirected = true
	}
	if _, err := getMode(unix.Stdout); err != nil {
		s.outputRedirected = true
	}
	if s.inputRedirected && s.outputRedirected {
		s.terminalSupported = false
	}
	if s.terminalSupported && !s.inputRedirected && !s.outputRedirected {
		mode := s.origMode
		mode.Iflag &^= icrnl | inpck | istrip | ixon
		mode.Cflag |= cs8
		mode.Lflag &^= unix.ECHO | icanon | iexten
		mode.Cc[unix.VMIN] = 1
		mode.Cc[unix.VTIME] = 0
		mode.ApplyMode()

		winch := make(chan os.Signal, 1)
		signal.Notify(winch, unix.SIGWINCH)
		s.winch = winch

		s.checkOutput()
	}

	if !s.outputRedirected {
		s.outputRedirected = !s.getColumns()
	}

	return &s
}

// TerminalSupported returns true if the current terminal supports
// line editing features, and false if liner will use the 'dumb'
// fallback for input.
// Note that TerminalSupported does not check all factors that may
// cause liner to not fully support the terminal (such as stdin redirection)
func TerminalSupported() bool {
	bad := map[string]bool{"": true, "dumb": true, "cons25": true}
	return !bad[strings.ToLower(os.Getenv("TERM"))]
}

// Close returns the terminal to its previous mode
func (s *State) Close() error {
	signal.Stop(s.winch)
	if !s.inputRedirected {
		s.origMode.ApplyMode()
	}
	return nil
}
