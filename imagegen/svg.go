package imagegen

import (
	"fmt"
	"strconv"
	"strings"
)

// SVG renders the scene as SVG markup. The output contains no text elements,
// no external references and no scripts: it is safe to inline in a page and it
// renders identically wherever it is opened.
func (s Scene) SVG() string {
	var b strings.Builder
	b.Grow(256 + 96*len(s.Shapes))

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		s.Size, s.Size, s.Size, s.Size)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, s.Size, s.Size, hex(s.Background))

	for _, sh := range s.Shapes {
		switch sh.Type {
		case ShapeRect:
			b.WriteString(`<rect x="` + num(sh.X) + `" y="` + num(sh.Y) +
				`" width="` + num(sh.W) + `" height="` + num(sh.H) + `"`)
			if sh.Radius > 0 {
				b.WriteString(` rx="` + num(sh.Radius) + `"`)
			}
			b.WriteString(` fill="` + hex(sh.Fill) + `"/>`)
		case ShapeCircle:
			b.WriteString(`<circle cx="` + num(sh.CX) + `" cy="` + num(sh.CY) +
				`" r="` + num(sh.R) + `" fill="` + hex(sh.Fill) + `"/>`)
		case ShapePolygon:
			b.WriteString(`<polygon points="`)
			for i, p := range sh.Points {
				if i > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(num(p.X) + "," + num(p.Y))
			}
			b.WriteString(`" fill="` + hex(sh.Fill) + `"/>`)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// num formats a coordinate with at most two decimals and no trailing zeros, so
// the markup stays small — these strings are often base64'd into a column and
// every byte is paid for per row.
func num(v float64) string {
	return strconv.FormatFloat(round2(v), 'f', -1, 64)
}

func round2(v float64) float64 {
	r := float64(int64(v*100+copysign(0.5, v))) / 100
	if r == 0 {
		// Avoid "-0", which is valid SVG but noisy in golden files.
		return 0
	}
	return r
}

func copysign(v, sign float64) float64 {
	if sign < 0 {
		return -v
	}
	return v
}

const hexDigits = "0123456789abcdef"

func hex(c Color) string {
	b := [7]byte{'#'}
	for i, v := range [3]uint8{c.R, c.G, c.B} {
		b[1+i*2] = hexDigits[v>>4]
		b[2+i*2] = hexDigits[v&0x0f]
	}
	return string(b[:])
}
