package imagegen

// palette is a background/foreground pair. The pairs are hand-picked rather
// than generated from a random hue: a random hue produces the occasional
// unreadable combination, and an avatar whose initials are invisible is a bug
// that only shows up in a screenshot.
type palette struct {
	bg, fg, accent Color
}

var palettes = []palette{
	{bg: Color{0x1f, 0x2a, 0x44}, fg: Color{0xe8, 0xed, 0xf7}, accent: Color{0x4c, 0x7d, 0xf0}},
	{bg: Color{0x0f, 0x3d, 0x3e}, fg: Color{0xe6, 0xf4, 0xf1}, accent: Color{0x2e, 0xc4, 0xa6}},
	{bg: Color{0x4a, 0x1d, 0x3f}, fg: Color{0xfb, 0xe9, 0xf4}, accent: Color{0xe0, 0x5b, 0xa8}},
	{bg: Color{0x5c, 0x2b, 0x14}, fg: Color{0xfd, 0xf0, 0xe4}, accent: Color{0xf2, 0x8c, 0x3c}},
	{bg: Color{0x23, 0x3a, 0x1e}, fg: Color{0xed, 0xf6, 0xe6}, accent: Color{0x7c, 0xc4, 0x4a}},
	{bg: Color{0x33, 0x22, 0x54}, fg: Color{0xef, 0xea, 0xfb}, accent: Color{0x9a, 0x74, 0xf0}},
	{bg: Color{0x5b, 0x14, 0x1c}, fg: Color{0xfd, 0xe8, 0xea}, accent: Color{0xe8, 0x4c, 0x5a}},
	{bg: Color{0x14, 0x3b, 0x5c}, fg: Color{0xe4, 0xf1, 0xfb}, accent: Color{0x3d, 0xa5, 0xe8}},
	{bg: Color{0x3f, 0x3a, 0x11}, fg: Color{0xf9, 0xf6, 0xe3}, accent: Color{0xd8, 0xb4, 0x2f}},
	{bg: Color{0x2b, 0x2b, 0x2b}, fg: Color{0xf0, 0xf0, 0xf0}, accent: Color{0x9e, 0x9e, 0x9e}},
	{bg: Color{0x11, 0x2e, 0x2b}, fg: Color{0xe2, 0xf3, 0xf0}, accent: Color{0x40, 0xb3, 0x8f}},
	{bg: Color{0x45, 0x18, 0x2e}, fg: Color{0xfa, 0xe9, 0xf1}, accent: Color{0xd0, 0x4f, 0x88}},
}

// pickPalette selects a pair from the subject hash. Using the hash directly
// (not a stream draw) keeps the colour stable even if the shape recipes below
// change how many random values they consume.
func pickPalette(h uint64) palette { return palettes[h%uint64(len(palettes))] }

// tint blends c toward white by f in [0,1]. Product cards use it for the paper
// behind the object, so the card stays in the same colour family as the object
// without swallowing it.
func tint(c Color, f float64) Color {
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	mix := func(v uint8) uint8 { return uint8(float64(v) + (255-float64(v))*f) }
	return Color{mix(c.R), mix(c.G), mix(c.B)}
}

// shade blends c toward black by f in [0,1]; used for the shadow face of a
// three-dimensional silhouette.
func shade(c Color, f float64) Color {
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	mix := func(v uint8) uint8 { return uint8(float64(v) * (1 - f)) }
	return Color{mix(c.R), mix(c.G), mix(c.B)}
}
