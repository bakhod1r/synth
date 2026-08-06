package imagegen

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"strings"
	"testing"
)

func allKinds(t *testing.T) []Kind {
	t.Helper()
	k := Kinds()
	if len(k) == 0 {
		t.Fatal("Kinds() is empty")
	}
	return k
}

// The contract the whole package exists for: same input, same bytes. A dataset
// regenerated in CI has to diff clean.
func TestDeterministic(t *testing.T) {
	for _, k := range allKinds(t) {
		for _, f := range []Format{FormatSVG, FormatPNG, FormatDataURL} {
			o := Options{Kind: k, Subject: "Ada Lovelace", Size: 64}
			a, err := Encode(o, f)
			if err != nil {
				t.Fatalf("%s/%s: %v", k, f, err)
			}
			b, err := Encode(o, f)
			if err != nil {
				t.Fatalf("%s/%s: %v", k, f, err)
			}
			if !bytes.Equal(a, b) {
				t.Errorf("%s/%s: repeated encode differed", k, f)
			}
		}
	}
}

// A different subject must give a different image, or the "matching" part of
// the feature is a lie.
func TestSubjectChangesImage(t *testing.T) {
	for _, k := range allKinds(t) {
		a, err := Encode(Options{Kind: k, Subject: "Ada Lovelace"}, FormatSVG)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Encode(Options{Kind: k, Subject: "Grace Hopper"}, FormatSVG)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(a, b) {
			t.Errorf("%s: two subjects produced the same image", k)
		}
	}
}

// Near-identical subjects must not produce near-identical images: "user1" and
// "user2" are the common case in a generated dataset.
func TestNeighbouringSubjectsDiverge(t *testing.T) {
	seen := map[string]string{}
	for _, s := range []string{"user1", "user2", "user3", "user4", "user5", "user6"} {
		b, err := Encode(Options{Kind: KindIdenticon, Subject: s}, FormatSVG)
		if err != nil {
			t.Fatal(err)
		}
		if prev, dup := seen[string(b)]; dup {
			t.Fatalf("%q and %q produced identical identicons", prev, s)
		}
		seen[string(b)] = s
	}
}

func TestSeedVariesImage(t *testing.T) {
	a, _ := Encode(Options{Kind: KindAvatar, Subject: "Ada"}, FormatSVG)
	b, _ := Encode(Options{Kind: KindAvatar, Subject: "Ada", Seed: 99}, FormatSVG)
	if bytes.Equal(a, b) {
		t.Fatal("seed had no effect")
	}
	c, _ := Encode(Options{Kind: KindAvatar, Subject: "Ada", Seed: 99}, FormatSVG)
	if !bytes.Equal(b, c) {
		t.Fatal("seeded output is not stable")
	}
}

func TestUnknownKindAndFormat(t *testing.T) {
	if _, err := Render(Options{Kind: "hologram"}); err == nil {
		t.Fatal("unknown kind accepted")
	}
	if _, err := Encode(Options{Kind: KindAvatar}, "tiff"); err == nil {
		t.Fatal("unknown format accepted")
	}
}

func TestSizeDefaultAndClamp(t *testing.T) {
	s, err := Render(Options{Kind: KindAvatar, Subject: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Size != DefaultSize {
		t.Fatalf("Size=%d, want default %d", s.Size, DefaultSize)
	}
	if s, _ = Render(Options{Kind: KindAvatar, Subject: "x", Size: -5}); s.Size != DefaultSize {
		t.Fatalf("negative size gave %d", s.Size)
	}
	if s, _ = Render(Options{Kind: KindAvatar, Subject: "x", Size: 99999}); s.Size != maxSize {
		t.Fatalf("size not clamped: %d", s.Size)
	}
}

func TestSVGShape(t *testing.T) {
	for _, k := range allKinds(t) {
		s, err := Render(Options{Kind: k, Subject: "Olive Oil", Size: 48})
		if err != nil {
			t.Fatal(err)
		}
		svg := s.SVG()
		if !strings.HasPrefix(svg, `<svg xmlns="http://www.w3.org/2000/svg"`) {
			t.Errorf("%s: bad SVG prefix: %.60s", k, svg)
		}
		if !strings.HasSuffix(svg, "</svg>") {
			t.Errorf("%s: unterminated SVG", k)
		}
		// A font reference or a script would break the two guarantees the
		// package makes: identical rendering everywhere, and safe to inline.
		for _, bad := range []string{"<text", "font", "<script", "http://www.w3.org/1999/xlink"} {
			if strings.Contains(svg, bad) {
				t.Errorf("%s: SVG contains %q", k, bad)
			}
		}
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Errorf("%s: SVG contains a non-finite coordinate", k)
		}
	}
}

func TestPNGDecodes(t *testing.T) {
	for _, k := range allKinds(t) {
		b, err := Encode(Options{Kind: k, Subject: "Trail Backpack", Size: 40}, FormatPNG)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if got := img.Bounds().Dx(); got != 40 {
			t.Errorf("%s: width %d, want 40", k, got)
		}
		if got := img.Bounds().Dy(); got != 40 {
			t.Errorf("%s: height %d, want 40", k, got)
		}
	}
}

// Above 256 the rasterizer switches supersampling factor; the switch must not
// change the size contract.
func TestPNGLargeSize(t *testing.T) {
	b, err := Encode(Options{Kind: KindProduct, Subject: "Desk Lamp", Size: 300}, FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 300 {
		t.Fatalf("width %d, want 300", img.Bounds().Dx())
	}
}

func TestDataURLDecodesToSVG(t *testing.T) {
	b, err := Encode(Options{Kind: KindMonogram, Subject: "Acme"}, FormatDataURL)
	if err != nil {
		t.Fatal(err)
	}
	const prefix = "data:image/svg+xml;base64,"
	s := string(b)
	if !strings.HasPrefix(s, prefix) {
		t.Fatalf("bad prefix: %.40s", s)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte("<svg")) {
		t.Fatalf("payload is not SVG: %.40s", raw)
	}
}

// The empty subject is the nullable-column case: it must still produce an
// image rather than failing the row.
func TestEmptySubject(t *testing.T) {
	for _, k := range allKinds(t) {
		b, err := Encode(Options{Kind: k}, FormatSVG)
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if len(b) == 0 {
			t.Fatalf("%s: empty output", k)
		}
	}
}

func TestInitials(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ada Lovelace", "AL"},
		{"ada lovelace", "AL"},
		{"Émile Zola", "EZ"},
		{"Nordwind Kaffee GmbH", "NK"},
		{"thermos", "TH"},
		{"", "?"},
		{"   ", "?"},
		{"日本", "??"},
		// An apostrophe is a word boundary, so O'Brien contributes O and B.
		{"o'brien mcdonald", "OB"},
	}
	for _, c := range cases {
		if got := initials(c.in, 2); got != c.want {
			t.Errorf("initials(%q)=%q, want %q", c.in, got, c.want)
		}
	}
	if got := initials("Bureau of Labor Statistics", 3); got != "BOL" {
		t.Errorf("three-letter initials = %q, want BOL", got)
	}
}

// Every glyph must be exactly 5 columns wide, or text drifts out of its box.
func TestGlyphGeometry(t *testing.T) {
	for r, g := range glyphs {
		for i, row := range g {
			if len(row) != glyphW {
				t.Errorf("glyph %q row %d is %d wide, want %d", r, i, len(row), glyphW)
			}
			for _, c := range row {
				if c != '#' && c != ' ' {
					t.Errorf("glyph %q row %d has %q", r, i, c)
				}
			}
		}
	}
	if _, ok := glyphs['?']; !ok {
		t.Fatal("the fallback glyph is missing")
	}
}

func TestFoldCoversAccents(t *testing.T) {
	for _, c := range []struct{ in, want rune }{
		{'é', 'E'}, {'Ø', 'O'}, {'ş', 'S'}, {'ü', 'U'}, {'a', 'A'}, {'7', '7'},
		{'漢', '?'},
	} {
		if got := fold(c.in); got != c.want {
			t.Errorf("fold(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

// The identicon's whole point is left-right symmetry.
func TestIdenticonIsSymmetric(t *testing.T) {
	s := identicon(hash("someone"), 100)
	mirrored := map[Point]bool{}
	for _, sh := range s.Shapes {
		mirrored[Point{X: round2(float64(s.Size) - sh.X - sh.W), Y: sh.Y}] = true
	}
	for _, sh := range s.Shapes {
		if !mirrored[Point{X: round2(sh.X), Y: sh.Y}] {
			t.Fatalf("cell at (%v,%v) has no mirror", sh.X, sh.Y)
		}
	}
}

func TestFingerprint(t *testing.T) {
	o := Options{Kind: KindAvatar, Subject: "Ada", Size: 64}
	f := Fingerprint(o)
	if len(f) != 16 {
		t.Fatalf("Fingerprint length %d, want 16", len(f))
	}
	if f != Fingerprint(o) {
		t.Fatal("Fingerprint is not stable")
	}
	for _, other := range []Options{
		{Kind: KindAvatar, Subject: "Grace", Size: 64},
		{Kind: KindAvatar, Subject: "Ada", Size: 65},
		{Kind: KindMonogram, Subject: "Ada", Size: 64},
		{Kind: KindAvatar, Subject: "Ada", Size: 64, Seed: 1},
	} {
		if Fingerprint(other) == f {
			t.Errorf("Fingerprint collided with %+v", other)
		}
	}
}

func TestExt(t *testing.T) {
	if got := Ext(FormatPNG); got != "png" {
		t.Errorf("Ext(png)=%q", got)
	}
	for _, f := range []Format{FormatSVG, FormatDataURL, "whatever"} {
		if got := Ext(f); got != "svg" {
			t.Errorf("Ext(%q)=%q, want svg", f, got)
		}
	}
}

func TestColorHelpers(t *testing.T) {
	c := Color{100, 100, 100}
	if got := tint(c, 1); got != (Color{255, 255, 255}) {
		t.Errorf("tint to white gave %+v", got)
	}
	if got := shade(c, 1); got != (Color{0, 0, 0}) {
		t.Errorf("shade to black gave %+v", got)
	}
	// Out-of-range factors are clamped, not wrapped.
	if got := tint(c, 5); got != (Color{255, 255, 255}) {
		t.Errorf("tint(5) gave %+v", got)
	}
	if got := shade(c, -5); got != c {
		t.Errorf("shade(-5) gave %+v", got)
	}
	if got := blend(Color{0, 0, 0}, Color{10, 20, 30}, 2); got != (Color{10, 20, 30}) {
		t.Errorf("blend(2) gave %+v", got)
	}
	if got := blend(Color{0, 0, 0}, Color{10, 20, 30}, -1); got != (Color{0, 0, 0}) {
		t.Errorf("blend(-1) gave %+v", got)
	}
}

func TestHexAndNum(t *testing.T) {
	if got := hex(Color{0x0a, 0xff, 0x00}); got != "#0aff00" {
		t.Fatalf("hex = %q", got)
	}
	for _, c := range []struct {
		in   float64
		want string
	}{{1, "1"}, {1.126, "1.13"}, {0.004, "0"}, {-2.5, "-2.5"}, {12.3456, "12.35"}} {
		if got := num(c.in); got != c.want {
			t.Errorf("num(%v)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestInPolygon(t *testing.T) {
	square := []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if !inPolygon(square, 5, 5) {
		t.Error("centre reported outside")
	}
	if inPolygon(square, 15, 5) {
		t.Error("point to the right reported inside")
	}
	if inPolygon(square, 5, -1) {
		t.Error("point above reported inside")
	}
}

// A polygon with too few points is skipped rather than panicking, because a
// scene recipe is data and a typo in one must not take the process down.
func TestDegeneratePolygon(t *testing.T) {
	s := Scene{Size: 8, Background: Color{255, 255, 255}}
	s.polygon([]Point{{1, 1}, {2, 2}}, Color{0, 0, 0})
	s.Shapes = append(s.Shapes, Shape{Type: ShapeType(200)})
	if img := s.Image(); img.Bounds().Dx() != 8 {
		t.Fatal("degenerate shapes changed the image size")
	}
	if !strings.Contains(s.SVG(), "<polygon") {
		t.Fatal("polygon dropped from SVG")
	}
}

func TestRoundRectCorners(t *testing.T) {
	sh := Shape{Type: ShapeRect, X: 0, Y: 0, W: 10, H: 10, Radius: 20}
	if inside(sh, 0.1, 0.1) {
		t.Error("corner of a fully rounded rect reported inside")
	}
	if !inside(sh, 5, 5) {
		t.Error("centre reported outside")
	}
	if !inside(sh, 5, 0.1) {
		t.Error("top edge midpoint reported outside")
	}
	if inside(sh, 11, 5) {
		t.Error("point beyond the right edge reported inside")
	}
}

func TestSceneImageZeroSize(t *testing.T) {
	if got := (Scene{}).Image().Bounds().Dx(); got != DefaultSize {
		t.Fatalf("zero-size scene rendered %d wide, want %d", got, DefaultSize)
	}
}
