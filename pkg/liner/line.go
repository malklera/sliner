package liner

import (
	"bufio"
	"container/ring"
	"fmt"
	"io"
	"os"
	"unicode"
)

type action int

const (
	left action = iota
	right
	up
	down
	home
	end
	insert
	del
	pageUp
	pageDown
	f1
	f2
	f3
	f4
	f5
	f6
	f7
	f8
	f9
	f10
	f11
	f12
	altB
	altBs // Alt+Backspace
	altD
	altF
	altY
	shiftTab
	wordLeft
	wordRight
	winch
	unknown
)

const (
	ctrlA = 1
	ctrlB = 2
	ctrlC = 3
	ctrlD = 4
	ctrlE = 5
	ctrlF = 6
	ctrlG = 7
	ctrlH = 8
	tab   = 9
	lf    = 10
	ctrlK = 11
	ctrlL = 12
	cr    = 13
	ctrlN = 14
	ctrlO = 15
	ctrlP = 16
	ctrlQ = 17
	ctrlR = 18
	ctrlS = 19
	ctrlT = 20
	ctrlU = 21
	ctrlV = 22
	ctrlW = 23
	ctrlX = 24
	ctrlY = 25
	ctrlZ = 26
	esc   = 27
	bs    = 127
)

const (
	beep = "\a"
)

//WARN: the prompt string cant have \n has to fix that
// is a valid reason, maybe discard the prompt part and only dealth with the input field

// PromptWithSuggestion displays prompt and an editable text with cursor at
// given position. The cursor will be set to the end of the line if given position
// is negative or greater than length of text (in runes). Returns a line of user input, not
// including a trailing newline character.
func (s *State) PromptWithSuggestion(prompt string, text string, pos int) (string, error) {
	if s.outputRedirected {
		return "", ErrNotTerminalOutput
	}

	for _, r := range prompt {
		if unicode.Is(unicode.C, r) {
			return "", ErrInvalidPrompt
		}
	}

	// WARN: check this, i do not understand why is here, what it do
	if s.inputRedirected || !s.terminalSupported {
		return s.promptUnsupported(prompt)
	}

	p := []rune(prompt)
	// TODO: why do i have this here?
	const minWorkingSpace = 10
	if s.columns < countGlyphs(p)+minWorkingSpace {
		return s.tooNarrow(prompt)
	}

	// TODO: once it works, get ride of the part that shows the prompt
	fmt.Print(prompt)

	var line = []rune(text)
	// NOTE: do i use this?
	killAction := 0 // used to mark kill related actions

	defer s.stopPrompt()

	if pos < 0 || len(line) < pos {
		pos = len(line)
	}
	if len(line) > 0 {
		err := s.refresh(p, line, pos)
		if err != nil {
			return "", err
		}
	}

restart:
	s.startPrompt()
	s.getColumns()

mainLoop:
	for {
		next, err := s.readNext()
	haveNext:
		if err != nil {
			if s.shouldRestart != nil && s.shouldRestart(err) {
				goto restart
			}
			return "", err
		}

		switch v := next.(type) {
		case rune:
			switch v {
			case cr, lf:
				if s.needRefresh {
					err := s.refresh(p, line, pos)
					if err != nil {
						return "", err
					}
				}
				fmt.Println()
				break mainLoop
			case ctrlA: // Start of line
				pos = 0
				s.needRefresh = true
			case ctrlE: // End of line
				pos = len(line)
				s.needRefresh = true
			case ctrlB: // left
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
					s.needRefresh = true
				} else {
					s.doBeep()
				}
			case ctrlF: // right
				if pos < len(line) {
					pos += len(getPrefixGlyphs(line[pos:], 1))
					s.needRefresh = true
				} else {
					s.doBeep()
				}
			case ctrlD: // del
				if pos == 0 && len(line) == 0 {
					// exit
					return "", io.EOF
				}

				// ctrlD is a potential EOF, so the rune reader shuts down.
				// Therefore, if it isn't actually an EOF, we must re-startPrompt.
				s.restartPrompt()

				if pos >= len(line) {
					s.doBeep()
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
					s.needRefresh = true
				}
			case ctrlK: // delete remainder of line
				if pos >= len(line) {
					s.doBeep()
				} else {
					if killAction > 0 {
						s.addToKillRing(line[pos:], 1) // Add in apend mode
					} else {
						s.addToKillRing(line[pos:], 0) // Add in normal mode
					}

					killAction = 2 // Mark that there was a kill action
					line = line[:pos]
					s.needRefresh = true
				}
			case ctrlT: // transpose prev glyph with glyph under cursor
				if len(line) < 2 || pos < 1 {
					s.doBeep()
				} else {
					if pos == len(line) {
						pos -= len(getSuffixGlyphs(line, 1))
					}
					prev := getSuffixGlyphs(line[:pos], 1)
					next := getPrefixGlyphs(line[pos:], 1)
					scratch := make([]rune, len(prev))
					copy(scratch, prev)
					copy(line[pos-len(prev):], next)
					copy(line[pos-len(prev)+len(next):], scratch)
					pos += len(next)
					s.needRefresh = true
				}
			case ctrlL: // clear screen
				s.eraseScreen()
				s.needRefresh = true
			case ctrlC: // reset
				fmt.Println("^C")
				if s.ctrlCAborts {
					return "", ErrPromptAborted
				}
				line = line[:0]
				pos = 0
				fmt.Print(prompt)
				s.restartPrompt()
			case ctrlH, bs: // Backspace
				if pos <= 0 {
					s.doBeep()
				} else {
					n := len(getSuffixGlyphs(line[:pos], 1))
					line = append(line[:pos-n], line[pos:]...)
					pos -= n
					s.needRefresh = true
				}
			case ctrlU: // Erase line before cursor
				if killAction > 0 {
					s.addToKillRing(line[:pos], 2) // Add in prepend mode
				} else {
					s.addToKillRing(line[:pos], 0) // Add in normal mode
				}

				killAction = 2 // Mark that there was some killing
				line = line[pos:]
				pos = 0
				s.needRefresh = true
			case ctrlW: // Erase word
				pos, line, killAction = s.eraseWord(pos, line, killAction)
			case ctrlY: // Paste from Yank buffer
				line, pos, next, err = s.yank(p, line, pos)
				goto haveNext
			// Catch keys that do nothing, but you don't want them to beep
			case esc:
				// DO NOTHING
			// Unused keys
			case ctrlG, ctrlO, ctrlQ, ctrlS, ctrlV, ctrlX, ctrlZ:
				fallthrough
			// Catch unhandled control codes (anything <= 31)
			case 0, 28, 29, 30, 31:
				s.doBeep()
			default:
				if pos == len(line) &&
					len(p)+len(line) < s.columns*4 && // Avoid countGlyphs on large lines
					countGlyphs(p)+countGlyphs(line) < s.columns-1 {
					line = append(line, v)
					fmt.Printf("%c", v)
					pos++
				} else {
					line = append(line[:pos], append([]rune{v}, line[pos:]...)...)
					pos++
					s.needRefresh = true
				}
			}
		case action:
			switch v {
			case del:
				if pos >= len(line) {
					s.doBeep()
				} else {
					n := len(getPrefixGlyphs(line[pos:], 1))
					line = append(line[:pos], line[pos+n:]...)
				}
			case left:
				if pos > 0 {
					pos -= len(getSuffixGlyphs(line[:pos], 1))
				} else {
					s.doBeep()
				}
			case wordLeft, altB:
				if pos > 0 {
					var spaceHere, spaceLeft, leftKnown bool
					for {
						pos--
						if pos == 0 {
							break
						}
						if leftKnown {
							spaceHere = spaceLeft
						} else {
							spaceHere = unicode.IsSpace(line[pos])
						}
						spaceLeft, leftKnown = unicode.IsSpace(line[pos-1]), true
						if !spaceHere && spaceLeft {
							break
						}
					}
				} else {
					s.doBeep()
				}
			case right:
				if pos < len(line) {
					pos += len(getPrefixGlyphs(line[pos:], 1))
				} else {
					s.doBeep()
				}
			case wordRight, altF:
				if pos < len(line) {
					var spaceHere, spaceLeft, hereKnown bool
					for {
						pos++
						if pos == len(line) {
							break
						}
						if hereKnown {
							spaceLeft = spaceHere
						} else {
							spaceLeft = unicode.IsSpace(line[pos-1])
						}
						spaceHere, hereKnown = unicode.IsSpace(line[pos]), true
						if spaceHere && !spaceLeft {
							break
						}
					}
				} else {
					s.doBeep()
				}
			case home: // Start of line
				pos = 0
			case end: // End of line
				pos = len(line)
			case altD: // Delete next word
				if pos == len(line) {
					s.doBeep()
					break
				}
				// Remove whitespace to the right
				var buf []rune // Store the deleted chars in a buffer
				for {
					if pos == len(line) || !unicode.IsSpace(line[pos]) {
						break
					}
					buf = append(buf, line[pos])
					line = append(line[:pos], line[pos+1:]...)
				}
				// Remove non-whitespace to the right
				for {
					if pos == len(line) || unicode.IsSpace(line[pos]) {
						break
					}
					buf = append(buf, line[pos])
					line = append(line[:pos], line[pos+1:]...)
				}
				// Save the result on the killRing
				if killAction > 0 {
					s.addToKillRing(buf, 2) // Add in prepend mode
				} else {
					s.addToKillRing(buf, 0) // Add in normal mode
				}
				killAction = 2 // Mark that there was some killing
			case altBs: // Erase word
				pos, line, killAction = s.eraseWord(pos, line, killAction)
			}
			s.needRefresh = true
		}
		if s.needRefresh && len(s.next) == 0 {
			err := s.refresh(p, line, pos)
			if err != nil {
				// TODO: why return empty string instead of nil?
				return "", err
			}
		}
		if killAction > 0 {
			killAction--
		}
	}
	return string(line), nil
}

func (s *State) tooNarrow(prompt string) (string, error) {
	// Docker and OpenWRT and etc sometimes return 0 column width
	// Reset mode temporarily. Restore baked mode in case the terminal
	// is wide enough for the next Prompt attempt.
	m, merr := TerminalMode()
	s.origMode.ApplyMode()
	if merr == nil {
		defer m.ApplyMode()
	}
	if s.r == nil {
		// Windows does not always set s.r
		s.r = bufio.NewReader(os.Stdin)
		defer func() { s.r = nil }()
	}
	return s.promptUnsupported(prompt)
}

func (s *State) refresh(prompt []rune, buf []rune, pos int) error {
	if s.columns == 0 {
		return ErrZeroColums
	}

	s.needRefresh = false

	s.cursorPos(0)
	_, err := fmt.Print(string(prompt))
	if err != nil {
		return err
	}

	pLen := countGlyphs(prompt)
	bLen := countGlyphs(buf)
	// on some OS / terminals extra column is needed to place the cursor char
	if cursorColumn {
		bLen++
	}
	pos = countGlyphs(buf[:pos])
	if pLen+bLen < s.columns {
		_, err = fmt.Print(string(buf))
		s.eraseLine()
		s.cursorPos(pLen + pos)
	} else {
		// Find space available
		space := s.columns - pLen
		space-- // space for cursor
		start := pos - space/2
		end := start + space
		if end > bLen {
			end = bLen
			start = end - space
		}
		if start < 0 {
			start = 0
			end = space
		}
		pos -= start

		// Leave space for markers
		if start > 0 {
			start++
		}
		if end < bLen {
			end--
		}
		startRune := len(getPrefixGlyphs(buf, start))
		line := getPrefixGlyphs(buf[startRune:], end-start)

		// Output
		if start > 0 {
			fmt.Print("{")
		}
		fmt.Print(string(line))
		if end < bLen {
			fmt.Print("}")
		}

		// Set cursor position
		s.eraseLine()
		s.cursorPos(pLen + pos)
	}
	return err
}

func (s *State) doBeep() {
	if !s.noBeep {
		fmt.Print(beep)
	}
}

// addToKillRing adds some text to the kill ring. If mode is 0 it adds it to a
// new node in the end of the kill ring, and move the current pointer to the new
// node. If mode is 1 or 2 it appends or prepends the text to the current entry
// of the killRing.
func (s *State) addToKillRing(text []rune, mode int) {
	// Don't use the same underlying array as text
	killLine := make([]rune, len(text))
	copy(killLine, text)

	// Point killRing to a newNode, procedure depends on the killring state and
	// append mode.
	if mode == 0 { // Add new node to killRing
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
		} else if s.killRing.Len() >= KillRingMax { // if killring is "full"
			s.killRing = s.killRing.Next()
		} else { // Normal case
			s.killRing.Link(ring.New(1))
			s.killRing = s.killRing.Next()
		}
	} else {
		if s.killRing == nil { // if killring is empty, create a new one
			s.killRing = ring.New(1)
			s.killRing.Value = []rune{}
		}
		if mode == 1 { // Append to last entry
			killLine = append(s.killRing.Value.([]rune), killLine...)
		} else if mode == 2 { // Prepend to last entry
			killLine = append(killLine, s.killRing.Value.([]rune)...)
		}
	}

	// Save text in the current killring node
	s.killRing.Value = killLine
}

func (s *State) eraseWord(pos int, line []rune, killAction int) (int, []rune, int) {
	if pos == 0 {
		s.doBeep()
		return pos, line, killAction
	}
	// Remove whitespace to the left
	var buf []rune // Store the deleted chars in a buffer
	for {
		if pos == 0 || !unicode.IsSpace(line[pos-1]) {
			break
		}
		buf = append(buf, line[pos-1])
		line = append(line[:pos-1], line[pos:]...)
		pos--
	}
	// Remove non-whitespace to the left
	for {
		if pos == 0 || unicode.IsSpace(line[pos-1]) {
			break
		}
		buf = append(buf, line[pos-1])
		line = append(line[:pos-1], line[pos:]...)
		pos--
	}
	// Invert the buffer and save the result on the killRing
	var newBuf []rune
	for i := len(buf) - 1; i >= 0; i-- {
		newBuf = append(newBuf, buf[i])
	}
	if killAction > 0 {
		s.addToKillRing(newBuf, 2) // Add in prepend mode
	} else {
		s.addToKillRing(newBuf, 0) // Add in normal mode
	}
	killAction = 2 // Mark that there was some killing

	s.needRefresh = true
	return pos, line, killAction
}

func (s *State) yank(p []rune, text []rune, pos int) ([]rune, int, interface{}, error) {
	if s.killRing == nil {
		return text, pos, rune(esc), nil
	}

	lineStart := text[:pos]
	lineEnd := text[pos:]
	var line []rune

	for {
		value := s.killRing.Value.([]rune)
		line = make([]rune, 0)
		line = append(line, lineStart...)
		line = append(line, value...)
		line = append(line, lineEnd...)

		pos = len(lineStart) + len(value)
		err := s.refresh(p, line, pos)
		if err != nil {
			return line, pos, 0, err
		}

		next, err := s.readNext()
		if err != nil {
			return line, pos, next, err
		}

		switch v := next.(type) {
		case rune:
			return line, pos, next, nil
		case action:
			switch v {
			case altY:
				s.killRing = s.killRing.Prev()
			default:
				return line, pos, next, nil
			}
		}
	}
}
