package imagegen

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGallery renders one image of every kind for a spread of subjects. It
// asserts only that nothing fails, because what is being checked here is not a
// value — it is whether the pictures look right, and no assertion catches an
// avatar whose initials sit off the edge of the frame.
//
// Point SYNTH_GALLERY_DIR at a directory to keep the PNGs and look at them:
//
//	SYNTH_GALLERY_DIR=/tmp/gallery go test ./imagegen -run TestGallery
//
// Without it the files go to a temp directory the framework cleans up, so the
// rendering path still runs on every CI run.
func TestGallery(t *testing.T) {
	dir := os.Getenv("SYNTH_GALLERY_DIR")
	if dir == "" {
		dir = t.TempDir()
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	subjects := map[Kind][]string{
		// Two words, one word, an accent, and a name long enough to overflow
		// its box if the text were not being measured.
		KindAvatar:    {"Ada Lovelace", "cher", "Émile Zola", "Bartholomew Featherstonehaugh"},
		KindProduct:   {"Espresso Machine", "Wool Scarf", "Olive Oil", "Trail Backpack", "Desk Lamp"},
		KindIdenticon: {"user-1", "user-2", "user-3"},
		KindMonogram:  {"Acme", "Nordwind Kaffee GmbH"},
	}

	for _, k := range Kinds() {
		for _, s := range subjects[k] {
			b, err := Encode(Options{Kind: k, Subject: s, Size: 128}, FormatPNG)
			if err != nil {
				t.Fatalf("%s/%s: %v", k, s, err)
			}
			name := filepath.Join(dir, string(k)+"-"+safeName(s)+".png")
			if err := os.WriteFile(name, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("gallery written to %s", dir)
}

// safeName keeps a subject to characters every filesystem accepts.
func safeName(s string) string {
	out := []rune(s)
	for i, r := range out {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
		default:
			out[i] = '_'
		}
	}
	return string(out)
}
