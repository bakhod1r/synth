package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bakhod1r/synth/imagegen"
	"github.com/bakhod1r/synth/schema"
)

// Drawn images: avatars, product thumbnails, identicons and company marks.
//
// These differ from KindImageURL in the one way that matters for a test
// fixture: nothing is fetched. The image is rendered from the row's own text,
// so it is stable across runs, works offline, and does not quietly depend on a
// placeholder service staying up.

func init() {
	registry[schema.KindAvatar] = imageProvider(imagegen.KindAvatar)
	registry[schema.KindProductImage] = imageProvider(imagegen.KindProduct)
	registry[schema.KindIdenticon] = imageProvider(imagegen.KindIdenticon)
	registry[schema.KindLogo] = imageProvider(imagegen.KindMonogram)
}

// imageProvider builds the provider for one image kind.
//
// Params:
//
//	from=<field>  take the subject from a sibling column (avatar from=name)
//	format=       dataurl (default) | svg | png
//	size=         edge length in pixels, default 128
//	dir=          write the image to this directory and return the file path
//	              instead of the image itself
//	seed=         extra entropy; two columns with the same subject and
//	              different seeds get different images
//	vary=true     mix the row's own randomness in, so repeated subjects still
//	              differ. Off by default: an avatar that changes for the same
//	              person on every row is not an avatar.
func imageProvider(kind imagegen.Kind) Provider {
	return func(c Ctx) any {
		opts := imagegen.Options{
			Kind:    kind,
			Subject: imageSubject(c, kind),
			Size:    imageSize(c),
		}
		if v, ok := c.Params["seed"]; ok {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				opts.Seed = n
			}
		}
		if c.Params["vary"] == "true" {
			opts.Seed ^= c.Rand.Uint64()
		}

		format, err := imageFormat(c.Params["format"])
		if err != nil {
			return err.Error()
		}
		data, err := imagegen.Encode(opts, format)
		if err != nil {
			return err.Error()
		}
		if dir := c.Params["dir"]; dir != "" {
			return writeImage(dir, imagegen.Fingerprint(opts)+"."+imagegen.Ext(format), data)
		}
		return string(data)
	}
}

// imageSubject resolves the text the image must match: an explicit from=
// sibling when there is one, otherwise a fresh value of the kind the image
// depicts, so a lone avatar column still looks like a person's avatar rather
// than an arbitrary blob.
func imageSubject(c Ctx, kind imagegen.Kind) string {
	if c.Field != nil && c.Field.From != "" && c.Sibling != nil {
		// Any sibling will do, not just a string one: keying an identicon off a
		// uuid or an integer id is the normal way to give an account a stable
		// mark, and those columns are not strings.
		if v := c.Sibling(c.Field.From); v != nil {
			if s := fmt.Sprint(v); s != "" {
				return s
			}
		}
	}
	if v, ok := c.Params["subject"]; ok && v != "" {
		return v
	}
	switch kind {
	case imagegen.KindProduct:
		return pick(c.Rand, c.Locale.Products)
	case imagegen.KindMonogram:
		return pick(c.Rand, c.Locale.Companies)
	case imagegen.KindIdenticon:
		return strconv.FormatUint(c.Rand.Uint64(), 36)
	default:
		return pick(c.Rand, c.Locale.FirstNamesFor(c.Gender)) + " " +
			pick(c.Rand, c.Locale.LastNamesFor(c.Gender))
	}
}

func imageSize(c Ctx) int {
	if n, ok := intParam(c.Params, "size"); ok {
		return n
	}
	return imagegen.DefaultSize
}

func imageFormat(s string) (imagegen.Format, error) {
	switch strings.ToLower(s) {
	case "", "dataurl", "datauri":
		return imagegen.FormatDataURL, nil
	case "svg":
		return imagegen.FormatSVG, nil
	case "png":
		return imagegen.FormatPNG, nil
	default:
		return "", fmt.Errorf("synth: unknown image format %q (want dataurl, svg or png)", s)
	}
}

// writeImage puts the image on disk and returns the path that belongs in the
// column. The filename is the image's fingerprint, so rows that share a subject
// share one file and a rerun overwrites rather than accumulates.
//
// A failure is returned as the cell value rather than panicking: one unwritable
// directory should not abort a million-row generation, and the bad value is
// visible in the output where it will be noticed.
func writeImage(dir, name string, data []byte) string {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "ERROR: " + err.Error()
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "ERROR: " + err.Error()
	}
	return path
}
