// Package tests holds end-to-end checks that cross package boundaries: struct
// tags through the generator, the YAML frontend through the encoders, files on
// disk. Unit tests live next to the code they cover; what lands here is what
// only breaks when the pieces are wired together.
package tests

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/imagegen"
	"github.com/google/uuid"
)

// generateCSV runs a YAML spec through the generator and encodes the rows as
// CSV, which is how the CLI writes them. Going through CSV is not incidental:
// an image column is long and full of characters a naive encoder mangles, and
// that has to be exercised rather than assumed.
func generateCSV(t *testing.T, spec string) ([][]string, error) {
	t.Helper()
	y, err := synth.YAMLBytes([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := y.Generate()
	if err != nil {
		t.Fatal(err)
	}

	cols := y.Columns()
	var buf strings.Builder
	w := csv.NewWriter(&buf)
	if err := w.Write(cols); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			rec[i] = fmt.Sprint(row[c])
		}
		if err := w.Write(rec); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		t.Fatal(err)
	}
	return csv.NewReader(strings.NewReader(buf.String())).ReadAll()
}

type user struct {
	ID     uuid.UUID `synth:"pk"`
	Name   string
	Avatar string `synth:"avatar,from=Name,format=svg,size=64"`
	Icon   string `synth:"identicon,from=ID,format=svg,size=32"`
}

// The end-to-end version of the package's central promise: the same seed gives
// the same rows and the same pictures.
func TestGenerationIsReproducible(t *testing.T) {
	a := synth.Make[user](25, synth.WithSeed(2024))
	b := synth.Make[user](25, synth.WithSeed(2024))
	if len(a) != 25 {
		t.Fatalf("got %d rows", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs between runs", i)
		}
	}
}

// The avatar must depict the name in its own row, not some other row's name.
func TestAvatarMatchesItsOwnRow(t *testing.T) {
	rows := synth.Make[user](30, synth.WithSeed(7))
	for _, r := range rows {
		want, err := imagegen.Encode(imagegen.Options{
			Kind: imagegen.KindAvatar, Subject: r.Name, Size: 64,
		}, imagegen.FormatSVG)
		if err != nil {
			t.Fatal(err)
		}
		if r.Avatar != string(want) {
			t.Fatalf("avatar for %q does not match the name", r.Name)
		}
	}
}

// Two rows that happen to share a name must share an avatar: that is what makes
// it an identity and not decoration.
func TestSameNameSameAvatar(t *testing.T) {
	rows := synth.Make[user](500, synth.WithSeed(11))
	byName := map[string]string{}
	shared := 0
	for _, r := range rows {
		if prev, ok := byName[r.Name]; ok {
			shared++
			if prev != r.Avatar {
				t.Fatalf("%q got two different avatars", r.Name)
			}
			continue
		}
		byName[r.Name] = r.Avatar
	}
	if shared == 0 {
		t.Skip("no repeated names in this sample")
	}
}

// An avatar is a depiction, not an identifier: two people with the same
// initials can legitimately draw the same picture, exactly as they do in every
// initials-based avatar service. What must never happen is two *unrelated*
// names — different initials — sharing an image, and the overall table must not
// collapse onto a handful of pictures.
func TestAvatarsAreVaried(t *testing.T) {
	rows := synth.Make[user](200, synth.WithSeed(3))

	byImage := map[string]string{}
	names, images := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		names[r.Name] = true
		images[r.Avatar] = true
		if prev, ok := byImage[r.Avatar]; ok && prev != r.Name {
			if firstTwo(prev) != firstTwo(r.Name) {
				t.Fatalf("%q and %q share an avatar despite different initials", prev, r.Name)
			}
		}
		byImage[r.Avatar] = r.Name
	}
	// Collisions are allowed but must stay rare; 90% is far above what a
	// broken hash or a stuck palette would produce.
	if min := len(names) * 9 / 10; len(images) < min {
		t.Fatalf("%d distinct names gave only %d images, want at least %d", len(names), len(images), min)
	}
	// An identicon is keyed off the primary key, so it has no excuse to repeat.
	icons := map[string]bool{}
	for _, r := range rows {
		icons[r.Icon] = true
	}
	if len(icons) != len(rows) {
		t.Fatalf("%d rows produced %d identicons", len(rows), len(icons))
	}
}

// firstTwo is the initials rule the avatar uses, repeated here so the test does
// not depend on an unexported helper in another package.
func firstTwo(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		out = append(out, []rune(strings.ToUpper(w))[0])
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

// The YAML frontend is a separate path into the same providers, so it gets its
// own end-to-end check — including that a data URL survives CSV quoting.
func TestYAMLCatalogueThroughCSV(t *testing.T) {
	const spec = `
name: catalog
count: 5
seed: 42
fields:
  id:        { kind: uuid, pk: true }
  title:     { kind: product }
  thumbnail: { kind: productimage, from: title, size: 48 }
`
	recs, err := generateCSV(t, spec)
	if err != nil {
		t.Fatalf("image column broke CSV quoting: %v", err)
	}
	if len(recs) != 6 {
		t.Fatalf("got %d lines including the header, want 6", len(recs))
	}
	const prefix = "data:image/svg+xml;base64,"
	for _, rec := range recs[1:] {
		cell := rec[2]
		if !strings.HasPrefix(cell, prefix) {
			t.Fatalf("cell is not a data URL: %.40s", cell)
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(cell, prefix))
		if err != nil {
			t.Fatalf("payload did not survive CSV: %v", err)
		}
		if !strings.HasPrefix(string(raw), "<svg") {
			t.Fatal("payload is not SVG")
		}
	}
}

// A data URL is long and full of characters JSON escapes; encoding it must not
// corrupt it.
func TestImageSurvivesJSON(t *testing.T) {
	rows := synth.Make[user](3, synth.WithSeed(5))
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var back []user
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if rows[i].Avatar != back[i].Avatar {
			t.Fatalf("row %d avatar changed through JSON", i)
		}
	}
}

// dir= is the only part of the feature that touches the filesystem, so the
// sharing behaviour is verified against a real directory rather than a mock.
func TestImagesWrittenToDisk(t *testing.T) {
	dir := t.TempDir()
	spec := `
name: gallery
count: 40
seed: 1
fields:
  title: { kind: product }
  path:  { kind: productimage, from: title, format: png, size: 32, dir: ` + dir + ` }
`
	recs, err := generateCSV(t, spec)
	if err != nil {
		t.Fatal(err)
	}

	titles := map[string]bool{}
	for _, rec := range recs[1:] {
		titles[rec[0]] = true
		if !strings.HasSuffix(rec[1], ".png") {
			t.Fatalf("column holds %q, not a png path", rec[1])
		}
		b, err := os.ReadFile(rec[1])
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "\x89PNG") {
			t.Fatalf("%s is not a PNG", rec[1])
		}
	}

	// One file per distinct product, not one per row.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(titles) {
		t.Fatalf("%d files for %d distinct titles", len(entries), len(titles))
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".png" {
			t.Fatalf("unexpected file %q", e.Name())
		}
	}
}
