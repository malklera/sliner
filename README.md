# sLiner


Allows for a pre-filled input filed of a single line.

This is a copy of [liner](https://github.com/peterh/liner), all credits to the
original creator, I only strip his project of the thing I do not need.

~~I want something even simpler than that, but I do not want it to be a fork, so
I copy it.~~

Change my mind, too clompex, I just want a pre-filled input prompt that works
on Linux.


Line Editing
------------

The following line editing commands are supported on platforms and terminals
that Liner supports:

Keystroke    | Action
---------    | ------
Ctrl-A, Home | Move cursor to beginning of line
Ctrl-E, End  | Move cursor to end of line
Ctrl-B, Left | Move cursor one character left
Ctrl-F, Right| Move cursor one character right
Ctrl-Left, Alt-B    | Move cursor to previous word
Ctrl-Right, Alt-F   | Move cursor to next word
Ctrl-D, Del  | (if line is *not* empty) Delete character under cursor
Ctrl-D       | (if line *is* empty) End of File - usually quits application
Ctrl-C       | Reset input (create new empty prompt)
Ctrl-L       | Clear screen (line is unmodified)
Ctrl-T       | Transpose previous character with current character
Ctrl-H, BackSpace | Delete character before cursor
Ctrl-W, Alt-BackSpace | Delete word leading up to cursor
Alt-D        | Delete word following cursor
Ctrl-K       | Delete from cursor to end of line
Ctrl-U       | Delete from start of line to cursor
Ctrl-Y       | Paste from Yank buffer (Alt-Y to paste next yank instead)


Getting started
-----------------

```go
package main

import (
	"github.com/malklera/sliner/pkg/liner"
	"log"
)

func main() {
	line := liner.NewLiner()
	defer line.Close()

	log.Println("What is the best language?")

	input, err := line.PrefilledInput("Go", -1)

	if err != nil {
		log.Fatalf("Error with the input: %v", err)
	}

	log.Printf("The best language is '%s'", input)
}
```

For documentation, see the [wiki](https://github.com/malklera/sliner/wiki).
