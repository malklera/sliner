package liner

import (

	// WARN: check if i can get ride of this package
	"github.com/mattn/go-runewidth"
)

// countGlyphs considers zero-width characters to be zero glyphs wide,
// and members of Chinese, Japanese, and Korean scripts to be 2 glyphs wide.
func countGlyphs(s []rune) int {
	n := 0
	for _, r := range s {
		// speed up the common case
		if r < 127 {
			n++
			continue
		}

		n += runewidth.RuneWidth(r)
	}
	return n
}
