// Package imagegen renders small, deterministic images from a name and a seed.
//
// The images are not photographs and are not fetched from anywhere: they are
// drawn from a handful of primitives, so a dataset that needs an avatar or a
// product thumbnail gets one offline, reproducibly, with no network call and no
// third-party placeholder service in the loop.
//
// Two properties matter and are tested:
//
//   - Determinism. The same Options always produce byte-identical output. A
//     generated dataset can be regenerated in CI and diffed.
//   - Subject fidelity. The image is a function of Subject, so the row for
//     "Ada Lovelace" carries the same avatar in every run, in every format, and
//     in every dataset that mentions her — the way a real avatar behaves.
//
// Text is rasterized from a built-in 5x7 bitmap font into plain rectangles, so
// the SVG output needs no font on the viewer's machine and the PNG output shows
// exactly the same glyphs as the SVG.
package imagegen

import (
	"encoding/base64"
	"fmt"
)

// Kind selects what the image depicts.
type Kind string

const (
	// KindAvatar is a person avatar: initials on a coloured field.
	KindAvatar Kind = "avatar"
	// KindProduct is a product thumbnail: a geometric silhouette plus a label.
	KindProduct Kind = "product"
	// KindIdenticon is a symmetric pixel grid derived from the subject, in the
	// style GitHub uses for users without a picture.
	KindIdenticon Kind = "identicon"
	// KindMonogram is a company mark: one or two letters in a filled circle.
	KindMonogram Kind = "monogram"
)

// Kinds lists every supported kind, in a stable order.
func Kinds() []Kind { return []Kind{KindAvatar, KindProduct, KindIdenticon, KindMonogram} }

// Format is an encoding of a rendered scene.
type Format string

const (
	// FormatSVG is SVG markup.
	FormatSVG Format = "svg"
	// FormatPNG is a PNG image.
	FormatPNG Format = "png"
	// FormatDataURL is SVG markup wrapped in a base64 data URL, which is what a
	// database column or a JSON field usually wants: it renders in a browser
	// with no file to ship alongside it.
	FormatDataURL Format = "dataurl"
)

// DefaultSize is the edge length used when Options.Size is zero.
const DefaultSize = 128

// maxSize caps the edge length. The renderer supersamples 3x, so an unbounded
// size would let one field allocate gigabytes.
const maxSize = 1024

// Options describes one image.
type Options struct {
	// Kind selects what to draw. Required.
	Kind Kind
	// Subject is the text the image must match — a person's name, a product
	// name, a company. It drives every colour and shape choice, so the same
	// subject always yields the same image.
	Subject string
	// Seed is extra entropy mixed into the subject hash. Zero (the default)
	// makes the image a pure function of Subject, which is normally what you
	// want; a non-zero seed gives a different but equally stable image per run
	// of a generator.
	Seed uint64
	// Size is the edge length in pixels. Zero means DefaultSize. Values above
	// 1024 are clamped.
	Size int
}

// Render draws the scene described by o. It returns an error only for an
// unknown Kind; every other field has a defined fallback.
func Render(o Options) (Scene, error) {
	size := o.Size
	switch {
	case size <= 0:
		size = DefaultSize
	case size > maxSize:
		size = maxSize
	}
	h := hash(o.Subject) ^ o.Seed
	switch o.Kind {
	case KindAvatar:
		return avatar(o.Subject, h, size), nil
	case KindProduct:
		return product(o.Subject, h, size), nil
	case KindIdenticon:
		return identicon(h, size), nil
	case KindMonogram:
		return monogram(o.Subject, h, size), nil
	default:
		return Scene{}, fmt.Errorf("imagegen: unknown kind %q", o.Kind)
	}
}

// Encode renders o and encodes it in the requested format.
func Encode(o Options, f Format) ([]byte, error) {
	s, err := Render(o)
	if err != nil {
		return nil, err
	}
	switch f {
	case FormatSVG:
		return []byte(s.SVG()), nil
	case FormatPNG:
		return s.PNG()
	case FormatDataURL:
		svg := s.SVG()
		out := make([]byte, 0, len("data:image/svg+xml;base64,")+base64.StdEncoding.EncodedLen(len(svg)))
		out = append(out, "data:image/svg+xml;base64,"...)
		return base64.StdEncoding.AppendEncode(out, []byte(svg)), nil
	default:
		return nil, fmt.Errorf("imagegen: unknown format %q", f)
	}
}

// Fingerprint is a short, stable hex identifier for the image o describes. It
// is the natural filename when images are written to disk: two rows with the
// same subject share one file instead of writing the same bytes twice.
func Fingerprint(o Options) string {
	h := hash(string(o.Kind)+"\x00"+o.Subject) ^ o.Seed ^ uint64(o.Size)*0x100000001b3
	const digits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[h&0xf]
		h >>= 4
	}
	return string(b[:])
}

// Ext is the file extension for a format, without the dot. A data URL is SVG
// text, so it shares the SVG extension when written to disk.
func Ext(f Format) string {
	if f == FormatPNG {
		return "png"
	}
	return "svg"
}

// hash is FNV-1a over the subject. It is written out rather than imported so
// the value is pinned: a stored dataset's images must not change because a
// dependency changed its mixing.
func hash(s string) uint64 {
	const (
		offset = 14695981039346656037
		prime  = 1099511628211
	)
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	// Subject-free callers (an identicon with no subject) would otherwise all
	// share the offset basis; splitmix the result so even the empty string
	// lands somewhere unremarkable.
	return splitmix(h)
}

// splitmix is the SplitMix64 finalizer: a cheap bijection that decorrelates
// neighbouring hashes, so "user1" and "user2" do not get near-identical images.
func splitmix(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// stream turns a single hash into a repeatable sequence of choices. It is a
// local PRNG rather than internal/rng because imagegen must stay usable on its
// own and must never consume a record's random stream: two fields drawing the
// same subject have to agree.
type stream struct{ s uint64 }

func newStream(seed uint64) *stream { return &stream{s: seed} }

func (r *stream) next() uint64 {
	r.s = splitmix(r.s)
	return r.s
}

// intn returns a value in [0,n). n<=0 yields 0.
func (r *stream) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

// chance reports true with probability num/den.
func (r *stream) chance(num, den int) bool { return r.intn(den) < num }
