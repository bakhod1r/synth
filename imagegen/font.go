package imagegen

import (
	"strings"
	"unicode"
)

// A 5x7 bitmap font, written as art so a wrong pixel is visible in review
// rather than hidden in a hex table. Only the glyphs a label can contain are
// present; anything else falls back to '?'.
//
// Drawing text as rectangles rather than as an SVG <text> element is deliberate:
// an SVG that names a font renders differently on every machine, and would not
// match the PNG at all. Rectangles render identically everywhere.
const (
	glyphW = 5
	glyphH = 7
)

var glyphs = map[rune][glyphH]string{
	'A': {" ### ", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'B': {"#### ", "#   #", "#   #", "#### ", "#   #", "#   #", "#### "},
	'C': {" ####", "#    ", "#    ", "#    ", "#    ", "#    ", " ####"},
	'D': {"#### ", "#   #", "#   #", "#   #", "#   #", "#   #", "#### "},
	'E': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#####"},
	'F': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#    "},
	'G': {" ####", "#    ", "#    ", "#  ##", "#   #", "#   #", " ####"},
	'H': {"#   #", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'I': {"#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "#####"},
	'J': {"#####", "    #", "    #", "    #", "    #", "#   #", " ### "},
	'K': {"#   #", "#  # ", "# #  ", "##   ", "# #  ", "#  # ", "#   #"},
	'L': {"#    ", "#    ", "#    ", "#    ", "#    ", "#    ", "#####"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #", "#   #", "#   #"},
	'N': {"#   #", "##  #", "# # #", "#  ##", "#   #", "#   #", "#   #"},
	'O': {" ### ", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'P': {"#### ", "#   #", "#   #", "#### ", "#    ", "#    ", "#    "},
	'Q': {" ### ", "#   #", "#   #", "#   #", "# # #", "#  # ", " ## #"},
	'R': {"#### ", "#   #", "#   #", "#### ", "# #  ", "#  # ", "#   #"},
	'S': {" ####", "#    ", "#    ", " ### ", "    #", "    #", "#### "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "},
	'U': {"#   #", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'V': {"#   #", "#   #", "#   #", "#   #", "#   #", " # # ", "  #  "},
	'W': {"#   #", "#   #", "#   #", "#   #", "# # #", "## ##", "#   #"},
	'X': {"#   #", "#   #", " # # ", "  #  ", " # # ", "#   #", "#   #"},
	'Y': {"#   #", "#   #", " # # ", "  #  ", "  #  ", "  #  ", "  #  "},
	'Z': {"#####", "    #", "   # ", "  #  ", " #   ", "#    ", "#####"},
	'0': {" ### ", "#   #", "#  ##", "# # #", "##  #", "#   #", " ### "},
	'1': {"  #  ", " ##  ", "  #  ", "  #  ", "  #  ", "  #  ", "#####"},
	'2': {" ### ", "#   #", "    #", "   # ", "  #  ", " #   ", "#####"},
	'3': {"#####", "   # ", "  #  ", "   # ", "    #", "#   #", " ### "},
	'4': {"   # ", "  ## ", " # # ", "#  # ", "#####", "   # ", "   # "},
	'5': {"#####", "#    ", "#### ", "    #", "    #", "#   #", " ### "},
	'6': {"  ## ", " #   ", "#    ", "#### ", "#   #", "#   #", " ### "},
	'7': {"#####", "    #", "   # ", "  #  ", " #   ", " #   ", " #   "},
	'8': {" ### ", "#   #", "#   #", " ### ", "#   #", "#   #", " ### "},
	'9': {" ### ", "#   #", "#   #", " ####", "    #", "   # ", " ##  "},
	'&': {" ##  ", "#  # ", " ##  ", " ##  ", "#  ##", "#  # ", " ## #"},
	'-': {"     ", "     ", "     ", "#####", "     ", "     ", "     "},
	'.': {"     ", "     ", "     ", "     ", "     ", " ##  ", " ##  "},
	'?': {" ### ", "#   #", "    #", "   # ", "  #  ", "     ", "  #  "},
}

// drawText paints s centred on (cx,cy) at the given pixel scale, one rect per
// lit font pixel. Glyphs are separated by one blank column.
func (s *Scene) drawText(text string, cx, cy, scale float64, fill Color) {
	runes := []rune(text)
	if len(runes) == 0 {
		return
	}
	totalW := float64(len(runes)*(glyphW+1)-1) * scale
	totalH := glyphH * scale
	x0 := cx - totalW/2
	y0 := cy - totalH/2

	for i, r := range runes {
		g, ok := glyphs[r]
		if !ok {
			g = glyphs['?']
		}
		gx := x0 + float64(i*(glyphW+1))*scale
		for row := 0; row < glyphH; row++ {
			line := g[row]
			for col := 0; col < glyphW && col < len(line); col++ {
				if line[col] != '#' {
					continue
				}
				s.rect(gx+float64(col)*scale, y0+float64(row)*scale, scale, scale, fill)
			}
		}
	}
}

// initials extracts up to max leading letters from a subject: "Ada Lovelace"
// gives "AL", "Nordwind Kaffee GmbH" gives "NK". A subject with no word
// boundary ("thermosflask") falls back to its own leading characters, and an
// empty subject to "?" — an image is still produced, which matters because a
// nullable column must not fail generation.
func initials(subject string, max int) string {
	words := strings.FieldsFunc(subject, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var b strings.Builder
	for _, w := range words {
		if b.Len() >= max {
			break
		}
		b.WriteRune(fold(([]rune(w))[0]))
	}
	if b.Len() == 0 {
		return "?"
	}
	if b.Len() == 1 && len(words) == 1 {
		// One word: take a second letter so a single-word subject still fills
		// the frame the way a two-word one does.
		if r := []rune(words[0]); len(r) > 1 && max > 1 {
			b.WriteRune(fold(r[1]))
		}
	}
	return b.String()
}

// fold maps a rune onto the font's alphabet. Accented Latin letters lose their
// accent rather than becoming '?', so "Émile" reads as E and not as a hole.
func fold(r rune) rune {
	r = unicode.ToUpper(r)
	if _, ok := glyphs[r]; ok {
		return r
	}
	const (
		src = "ÀÁÂÃÄÅĀĂĄÇĆČÈÉÊËĒĖĘĚÌÍÎÏĪĮŁÑŃŇÒÓÔÕÖŌŐØÙÚÛÜŪŮŰŞŚŠȘŢŤȚÝŸŹŻŽÐÞ"
		dst = "AAAAAAAAACCCEEEEEEEEIIIIIILNNNOOOOOOOOUUUUUUUSSSSTTTYYZZZDT"
	)
	sr, dr := []rune(src), []rune(dst)
	for i, c := range sr {
		if c == r {
			return dr[i]
		}
	}
	return '?'
}
