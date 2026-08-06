// Package benchmarks holds cross-package performance measurements that do not
// belong to any one package's own test suite.
//
// They are informational. Nothing here asserts a timing threshold: a shared CI
// runner's numbers are too noisy to gate a merge on, and a benchmark that fails
// randomly gets ignored, which is worse than no benchmark.
package benchmarks

import (
	"testing"

	"github.com/bakhod1r/synth"
	"github.com/bakhod1r/synth/imagegen"
	"github.com/google/uuid"
)

var sink []byte

func BenchmarkImageSVG(b *testing.B) {
	for _, k := range imagegen.Kinds() {
		b.Run(string(k), func(b *testing.B) {
			o := imagegen.Options{Kind: k, Subject: "Ada Lovelace", Size: 128}
			b.ReportAllocs()
			for b.Loop() {
				out, err := imagegen.Encode(o, imagegen.FormatSVG)
				if err != nil {
					b.Fatal(err)
				}
				sink = out
			}
		})
	}
}

// PNG is the expensive path — it rasterizes and compresses — so it is measured
// separately rather than averaged into the SVG numbers.
func BenchmarkImagePNG(b *testing.B) {
	for _, size := range []int{32, 128, 512} {
		b.Run(sizeName(size), func(b *testing.B) {
			o := imagegen.Options{Kind: imagegen.KindProduct, Subject: "Espresso Machine", Size: size}
			b.ReportAllocs()
			for b.Loop() {
				out, err := imagegen.Encode(o, imagegen.FormatPNG)
				if err != nil {
					b.Fatal(err)
				}
				sink = out
			}
		})
	}
}

func BenchmarkImageDataURL(b *testing.B) {
	o := imagegen.Options{Kind: imagegen.KindAvatar, Subject: "Ada Lovelace"}
	b.ReportAllocs()
	for b.Loop() {
		out, err := imagegen.Encode(o, imagegen.FormatDataURL)
		if err != nil {
			b.Fatal(err)
		}
		sink = out
	}
}

// The number that matters in practice: what an image column costs per row of a
// real generation, next to the same row without one.
type plainUser struct {
	ID   uuid.UUID `synth:"pk"`
	Name string
}

type avatarUser struct {
	ID     uuid.UUID `synth:"pk"`
	Name   string
	Avatar string `synth:"avatar,from=Name,format=svg,size=64"`
}

func BenchmarkRowsWithoutImage(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = synth.Make[plainUser](100, synth.WithSeed(1))
	}
}

func BenchmarkRowsWithImage(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = synth.Make[avatarUser](100, synth.WithSeed(1))
	}
}

func sizeName(n int) string {
	switch n {
	case 32:
		return "32px"
	case 128:
		return "128px"
	default:
		return "512px"
	}
}
