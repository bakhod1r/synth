package imagegen

// The four scene recipes. Each is a pure function of (subject, hash, size):
// nothing here reads a clock, a global, or a shared random source.

// avatar draws a person: initials over a coloured field, with a soft off-centre
// disc so a page of avatars does not read as a wall of flat squares.
func avatar(subject string, h uint64, size int) Scene {
	p := pickPalette(h)
	r := newStream(h)
	s := Scene{Size: size, Background: p.bg}
	f := float64(size)

	// A large, mostly off-canvas circle in the accent colour. Its quadrant is
	// the only structural variation, which is enough to break up a grid.
	cx, cy := f*0.15, f*0.15
	if r.chance(1, 2) {
		cx = f * 0.85
	}
	if r.chance(1, 2) {
		cy = f * 0.85
	}
	s.circle(cx, cy, f*0.45, blend(p.bg, p.accent, 0.35))

	s.drawText(initials(subject, 2), f/2, f*0.52, f/20, p.fg)
	return s
}

// monogram draws a company mark: a filled disc on a light field with one or two
// letters. It is deliberately flatter than an avatar — a brand mark that
// carried decorative noise would not look like a logo.
func monogram(subject string, h uint64, size int) Scene {
	p := pickPalette(h)
	s := Scene{Size: size, Background: tint(p.bg, 0.92)}
	f := float64(size)

	s.circle(f/2, f/2, f*0.38, p.bg)
	s.circle(f/2, f/2, f*0.31, p.accent)
	s.drawText(initials(subject, 2), f/2, f/2, f/24, p.fg)
	return s
}

// identicon draws the symmetric 5x5 pixel grid used for accounts with no
// picture. Only the left three columns are decided; the right two mirror them,
// which is what makes the result read as a mark rather than as noise.
func identicon(h uint64, size int) Scene {
	p := pickPalette(h)
	r := newStream(h)
	s := Scene{Size: size, Background: tint(p.bg, 0.93)}

	const grid = 5
	f := float64(size)
	pad := f * 0.12
	cell := (f - 2*pad) / grid

	for col := 0; col < (grid+1)/2; col++ {
		for row := 0; row < grid; row++ {
			// The centre column is sparser so the mark keeps a visible spine
			// instead of filling into a blob.
			on := r.chance(1, 2)
			if col == 2 {
				on = r.chance(2, 5)
			}
			if !on {
				continue
			}
			x := pad + float64(col)*cell
			y := pad + float64(row)*cell
			s.rect(x, y, cell, cell, p.bg)
			if mirror := grid - 1 - col; mirror != col {
				s.rect(pad+float64(mirror)*cell, y, cell, cell, p.bg)
			}
		}
	}
	return s
}

// product draws a thumbnail for a catalogue row: a silhouette chosen from a few
// recipes, plus the product's initials as a label. The recipe is picked from
// the subject hash, so "Espresso Machine" is the same object in every run — the
// point of a product image is that it does not change under the customer.
func product(subject string, h uint64, size int) Scene {
	p := pickPalette(h)
	s := Scene{Size: size, Background: tint(p.bg, 0.90)}
	f := float64(size)

	body := p.accent
	dark := shade(body, 0.30)

	switch h % 5 {
	case 0: // Box, drawn as a lid plus a face so it reads as a solid.
		s.rect(f*0.22, f*0.34, f*0.56, f*0.40, body)
		s.polygon([]Point{
			{X: f * 0.22, Y: f * 0.34}, {X: f * 0.36, Y: f * 0.22},
			{X: f * 0.92, Y: f * 0.22}, {X: f * 0.78, Y: f * 0.34},
		}, tint(body, 0.18))
		s.polygon([]Point{
			{X: f * 0.78, Y: f * 0.34}, {X: f * 0.92, Y: f * 0.22},
			{X: f * 0.92, Y: f * 0.62}, {X: f * 0.78, Y: f * 0.74},
		}, dark)
	case 1: // Bottle.
		s.rect(f*0.44, f*0.14, f*0.12, f*0.14, dark)
		s.roundRect(f*0.32, f*0.26, f*0.36, f*0.52, f*0.08, body)
		s.rect(f*0.32, f*0.44, f*0.36, f*0.12, tint(body, 0.45))
	case 2: // Can.
		s.roundRect(f*0.32, f*0.20, f*0.36, f*0.58, f*0.06, body)
		s.rect(f*0.32, f*0.20, f*0.36, f*0.06, dark)
		s.rect(f*0.32, f*0.72, f*0.36, f*0.06, dark)
	case 3: // Bag with a handle.
		s.rect(f*0.26, f*0.34, f*0.48, f*0.44, body)
		s.rect(f*0.38, f*0.20, f*0.04, f*0.14, dark)
		s.rect(f*0.58, f*0.20, f*0.04, f*0.14, dark)
		s.rect(f*0.38, f*0.20, f*0.24, f*0.04, dark)
	default: // Price tag.
		s.polygon([]Point{
			{X: f * 0.20, Y: f * 0.30}, {X: f * 0.62, Y: f * 0.30},
			{X: f * 0.82, Y: f * 0.50}, {X: f * 0.62, Y: f * 0.70},
			{X: f * 0.20, Y: f * 0.70},
		}, body)
		s.circle(f*0.32, f*0.50, f*0.05, tint(body, 0.75))
	}

	// The label sits on a strip along the bottom so it never lands on top of
	// the silhouette, whichever recipe was drawn.
	s.rect(0, f*0.82, f, f*0.18, p.bg)
	s.drawText(initials(subject, 3), f/2, f*0.905, f/44, p.fg)
	return s
}

// blend mixes a toward b by f in [0,1].
func blend(a, b Color, f float64) Color {
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f) }
	return Color{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B)}
}
