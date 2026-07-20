package gen

import (
	"math"
	"strings"

	"github.com/bakhodir/synth/internal/rng"
)

// nastyStrings are values that routinely break parsers, validators, encoders
// and UIs: empty, whitespace, unicode/emoji, RTL, quotes, SQL/HTML fragments,
// and pathologically long input.
var nastyStrings = []string{
	"",
	" ",
	"\t\n",
	"😀🔥💥",                       // emoji
	"مرحبا",                     // RTL (Arabic)
	"Ω≈ç√∫˜µ",                   // math/unicode
	"'; DROP TABLE users;--",    // SQL-ish
	"<script>alert(1)</script>", // HTML/JS
	"null",                      // stringly null
	"a\x00b",                    // embedded NUL byte
	"Robert');",                 // quote break
	strings.Repeat("A", 10000),  // very long
	"\"quoted\",comma",          // CSV break
	"line1\nline2",              // embedded newline
}

// chaosValue returns an edge-case replacement matched to v's dynamic type.
// Non string/int/float values pass through unchanged so struct assignment and
// referential fields stay valid.
func chaosValue(r *rng.Rand, v any) any {
	switch v.(type) {
	case string:
		return nastyStrings[r.Pick(len(nastyStrings))]
	case int:
		return nastyInts[r.Pick(len(nastyInts))]
	case float64:
		return nastyFloats[r.Pick(len(nastyFloats))]
	default:
		return v
	}
}

var nastyInts = []int{0, -1, 1, math.MaxInt32, math.MinInt32, math.MaxInt64, math.MinInt64}

var nastyFloats = []float64{0, math.Copysign(0, -1), 1e308, -1e308, math.SmallestNonzeroFloat64, 0.1 + 0.2}
