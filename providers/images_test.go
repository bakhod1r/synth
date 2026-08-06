package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth/imagegen"
	"github.com/bakhod1r/synth/internal/rng"
	"github.com/bakhod1r/synth/locale"
	"github.com/bakhod1r/synth/schema"
	"github.com/google/uuid"
)

func imgCtx(t *testing.T, params map[string]string) Ctx {
	t.Helper()
	loc := locale.Get("en")
	return Ctx{
		Rand:   rng.New(7),
		Locale: loc,
		Params: params,
		Field:  &schema.Field{Name: "image", Params: params},
		Place:  &loc.Places[0],
		Gender: "female",
	}
}

var imageKinds = []schema.Kind{
	schema.KindAvatar, schema.KindProductImage, schema.KindIdenticon, schema.KindLogo,
}

func TestImageKindsRegistered(t *testing.T) {
	for _, k := range imageKinds {
		if Get(k) == nil {
			t.Fatalf("%s has no provider", k)
		}
	}
}

func TestImageDefaultIsDataURL(t *testing.T) {
	for _, k := range imageKinds {
		v, ok := Get(k)(imgCtx(t, map[string]string{})).(string)
		if !ok {
			t.Fatalf("%s: provider did not return a string", k)
		}
		if !strings.HasPrefix(v, "data:image/svg+xml;base64,") {
			t.Errorf("%s: %.40s is not a data URL", k, v)
		}
	}
}

func TestImageFormats(t *testing.T) {
	svg := Get(schema.KindAvatar)(imgCtx(t, map[string]string{"format": "svg"})).(string)
	if !strings.HasPrefix(svg, "<svg") {
		t.Errorf("format=svg gave %.30s", svg)
	}
	png := Get(schema.KindAvatar)(imgCtx(t, map[string]string{"format": "PNG"})).(string)
	if !strings.HasPrefix(png, "\x89PNG") {
		t.Errorf("format=png did not produce PNG bytes")
	}
	bad := Get(schema.KindAvatar)(imgCtx(t, map[string]string{"format": "tiff"})).(string)
	if !strings.Contains(bad, "unknown image format") {
		t.Errorf("bad format reported as %q", bad)
	}
}

func TestImageSize(t *testing.T) {
	svg := Get(schema.KindAvatar)(imgCtx(t, map[string]string{"format": "svg", "size": "40"})).(string)
	if !strings.Contains(svg, `width="40"`) {
		t.Errorf("size=40 not honoured: %.80s", svg)
	}
}

// from= is the point of the feature: the avatar has to belong to the name in
// the row next to it.
func TestImageFromSibling(t *testing.T) {
	c := imgCtx(t, map[string]string{"format": "svg"})
	c.Field.From = "name"
	c.Sibling = func(string) any { return "Ada Lovelace" }
	got := Get(schema.KindAvatar)(c).(string)

	want, err := imagegen.Encode(imagegen.Options{
		Kind: imagegen.KindAvatar, Subject: "Ada Lovelace", Size: imagegen.DefaultSize,
	}, imagegen.FormatSVG)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatal("avatar does not match the sibling name")
	}
}

// A missing sibling must fall back rather than fail the row.
func TestImageFromMissingSibling(t *testing.T) {
	c := imgCtx(t, map[string]string{"format": "svg"})
	c.Field.From = "name"
	c.Sibling = func(string) any { return nil }
	if v := Get(schema.KindAvatar)(c).(string); !strings.HasPrefix(v, "<svg") {
		t.Fatalf("fallback failed: %.40s", v)
	}
}

// Keying an identicon off a uuid or an int id is the common case, and those
// columns are not strings.
func TestImageFromNonStringSibling(t *testing.T) {
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	c := imgCtx(t, map[string]string{"format": "svg"})
	c.Field.From = "id"
	c.Sibling = func(string) any { return id }
	got := Get(schema.KindIdenticon)(c).(string)

	want, err := imagegen.Encode(imagegen.Options{
		Kind: imagegen.KindIdenticon, Subject: id.String(), Size: imagegen.DefaultSize,
	}, imagegen.FormatSVG)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatal("identicon is not keyed off the uuid sibling")
	}
}

func TestImageSubjectParam(t *testing.T) {
	a := Get(schema.KindLogo)(imgCtx(t, map[string]string{"format": "svg", "subject": "Acme"})).(string)
	b := Get(schema.KindLogo)(imgCtx(t, map[string]string{"format": "svg", "subject": "Acme"})).(string)
	if a != b {
		t.Fatal("subject= is not stable")
	}
}

// Without vary=, the same subject must give the same image every time; with
// vary=true it must not.
func TestImageVaryAndSeed(t *testing.T) {
	stable := map[string]string{"format": "svg", "subject": "Same Person"}
	if Get(schema.KindAvatar)(imgCtx(t, stable)) != Get(schema.KindAvatar)(imgCtx(t, stable)) {
		t.Fatal("repeated subject changed image")
	}

	seeded := map[string]string{"format": "svg", "subject": "Same Person", "seed": "12345"}
	if Get(schema.KindAvatar)(imgCtx(t, seeded)) == Get(schema.KindAvatar)(imgCtx(t, stable)) {
		t.Fatal("seed= had no effect")
	}
	// A malformed seed is ignored rather than fatal.
	junk := map[string]string{"format": "svg", "subject": "Same Person", "seed": "abc"}
	if Get(schema.KindAvatar)(imgCtx(t, junk)) != Get(schema.KindAvatar)(imgCtx(t, stable)) {
		t.Fatal("unparsable seed was not ignored")
	}

	vary := map[string]string{"format": "svg", "subject": "Same Person", "vary": "true"}
	c1, c2 := imgCtx(t, vary), imgCtx(t, vary)
	c2.Rand = rng.New(99)
	if Get(schema.KindAvatar)(c1) == Get(schema.KindAvatar)(c2) {
		t.Fatal("vary=true did not vary")
	}
}

func TestImageWritesToDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "images")
	params := map[string]string{"format": "png", "dir": dir, "subject": "Olive Oil", "size": "32"}
	path := Get(schema.KindProductImage)(imgCtx(t, params)).(string)

	if filepath.Dir(path) != dir {
		t.Fatalf("path %q is not under %q", path, dir)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("path %q does not end in .png", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "\x89PNG") {
		t.Fatal("file is not a PNG")
	}
	// Same subject, same file: rows must share, not accumulate.
	if again := Get(schema.KindProductImage)(imgCtx(t, params)).(string); again != path {
		t.Fatalf("second call wrote %q, want %q", again, path)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("%d files written, want 1", len(entries))
	}
}

// An unwritable directory must surface in the cell, not abort generation.
func TestImageDirError(t *testing.T) {
	f := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := Get(schema.KindAvatar)(imgCtx(t, map[string]string{"dir": filepath.Join(f, "sub")})).(string)
	if !strings.HasPrefix(v, "ERROR: ") {
		t.Fatalf("unwritable dir gave %q", v)
	}
}

// Each kind draws its own default subject when no name is supplied, so a lone
// column still looks like what it claims to be.
func TestImageDefaultSubjects(t *testing.T) {
	for _, k := range imageKinds {
		a := Get(k)(imgCtx(t, map[string]string{"format": "svg"})).(string)
		c := imgCtx(t, map[string]string{"format": "svg"})
		c.Rand = rng.New(1234)
		if b := Get(k)(c).(string); a == b {
			t.Errorf("%s: default subject did not vary with the row", k)
		}
	}
}
