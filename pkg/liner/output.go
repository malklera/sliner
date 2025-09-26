package liner

import (
	"fmt"
	"os"
	"strings"
)

type winSize struct {
	row    uint16
	col    uint16
	xpixel uint16
	ypixel uint16
}

func (s *State) checkOutput() {
	// xterm is known to support CHA
	if strings.Contains(strings.ToLower(os.Getenv("TERM")), "xterm") {
		s.useCHA = true
		return
	}

	// The test for functional ANSI CHA is unreliable (eg the Windows
	// telnet command does not support reading the cursor position with
	// an ANSI DSR request, despite setting TERM=ansi)

	// Assume CHA isn't supported (which should be safe, although it
	// does result in occasional visible cursor jitter)
	s.useCHA = false
}

func (s *State) cursorPos(x int) {
	if s.useCHA {
		// 'G' is "Cursor Character Absolute (CHA)"
		fmt.Printf("\x1b[%dG", x+1)
	} else {
		// 'C' is "Cursor Forward (CUF)"
		fmt.Print("\r")
		if x > 0 {
			fmt.Printf("\x1b[%dC", x)
		}
	}
}

func (s *State) eraseLine() {
	fmt.Print("\x1b[0K")
}
