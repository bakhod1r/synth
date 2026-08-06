// Command images demonstrates the drawn-image kinds: an avatar that belongs to
// the person in the same row, a thumbnail that belongs to the product, and a
// company mark that belongs to the company.
//
// Nothing is downloaded. Every image is drawn from the row's own text, so the
// output is identical on every machine and on every run — which is what makes
// it usable as a test fixture rather than as a demo that only works online.
//
// Run:
//
//	go run ./examples/images
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bakhod1r/synth"
	"github.com/google/uuid"
)

// User shows the point of from=: it ties an image to its row. Without it the
// avatar would depict a different, unrelated person.
type User struct {
	ID     uuid.UUID `synth:"pk"`
	Name   string
	Avatar string `synth:"avatar,from=Name,format=svg,size=96"`
	// An identicon needs no subject of its own; keying it off the ID gives the
	// same account the same mark forever.
	Icon string `synth:"identicon,from=ID,size=48"`
}

type Product struct {
	ID    uuid.UUID `synth:"pk"`
	Title string    `synth:"product"`
	Image string    `synth:"productimage,from=Title,size=96"`
}

type Company struct {
	Name string `synth:"company"`
	Logo string `synth:"logo,from=Name,size=96"`
}

func main() {
	users := synth.Make[User](3, synth.WithSeed(42))
	products := synth.Make[Product](3, synth.WithSeed(42))
	companies := synth.Make[Company](3, synth.WithSeed(42))

	for _, u := range users {
		fmt.Printf("%-22s avatar %d bytes of SVG, icon %s\n",
			u.Name, len(u.Avatar), truncate(u.Icon))
	}
	for _, p := range products {
		fmt.Printf("%-22s %s\n", p.Title, truncate(p.Image))
	}
	for _, c := range companies {
		fmt.Printf("%-22s %s\n", c.Name, truncate(c.Logo))
	}

	// The same row generated twice is the same image: regenerate the fixture
	// and the diff is empty.
	again := synth.Make[User](3, synth.WithSeed(42))
	fmt.Println("stable across runs:", again[0].Avatar == users[0].Avatar)

	// dir=<path> in the tag writes PNG or SVG files and puts the path in the
	// column instead of the image. A struct tag is a compile-time constant, so
	// for a directory chosen at runtime write the value out yourself:
	dir, err := os.MkdirTemp("", "synth-images-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	for _, u := range users {
		name := filepath.Join(dir, strings.ReplaceAll(u.Name, " ", "-")+".svg")
		if err := os.WriteFile(name, []byte(u.Avatar), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	fmt.Println("wrote avatars to", dir)
}

func truncate(s string) string {
	if len(s) <= 48 {
		return s
	}
	return s[:48] + "…"
}
