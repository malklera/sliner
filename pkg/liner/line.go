package liner

import (
	"bufio"
	"fmt"
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
	esc   = 27
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

		historyAction = false
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
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
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
			case ctrlP: // up
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos > 0 {
					if historyPos == len(historyPrefix) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(historyPrefix[historyPos])
					pos = len(line)
					s.needRefresh = true
				} else {
					s.doBeep()
				}
			case ctrlN: // down
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos < len(historyPrefix) {
					historyPos++
					if historyPos == len(historyPrefix) {
						line = []rune(historyEnd)
					} else {
						line = []rune(historyPrefix[historyPos])
					}
					pos = len(line)
					s.needRefresh = true
				} else {
					s.doBeep()
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
				if s.multiLineMode {
					s.resetMultiLine(p, line, pos)
				}
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
			case ctrlR: // Reverse Search
				line, pos, next, err = s.reverseISearch(line, pos)
				s.needRefresh = true
				goto haveNext
			case tab: // Tab completion
				line, pos, next, err = s.tabComplete(p, line, pos)
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
				if pos == len(line) && !s.multiLineMode &&
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
			case up:
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos > 0 {
					if historyPos == len(historyPrefix) {
						historyEnd = string(line)
					}
					historyPos--
					line = []rune(historyPrefix[historyPos])
					pos = len(line)
				} else {
					s.doBeep()
				}
			case down:
				historyAction = true
				if historyStale {
					historyPrefix = s.getHistoryByPrefix(string(line))
					historyPos = len(historyPrefix)
					historyStale = false
				}
				if historyPos < len(historyPrefix) {
					historyPos++
					if historyPos == len(historyPrefix) {
						line = []rune(historyEnd)
					} else {
						line = []rune(historyPrefix[historyPos])
					}
					pos = len(line)
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
			case winch: // Window change
				if s.multiLineMode {
					if s.maxRows-s.cursorRows > 0 {
						s.moveDown(s.maxRows - s.cursorRows)
					}
					for i := 0; i < s.maxRows-1; i++ {
						s.cursorPos(0)
						s.eraseLine()
						s.moveUp(1)
					}
					s.maxRows = 1
					s.cursorRows = 1
				}
			}
			s.needRefresh = true
		}
		if s.needRefresh && !s.inputWaiting() {
			err := s.refresh(p, line, pos)
			if err != nil {
				return "", err
			}
		}
		if !historyAction {
			historyStale = true
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
