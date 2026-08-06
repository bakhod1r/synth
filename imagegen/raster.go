package imagegen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// PNG rasterizes the scene and encodes it as PNG. The result is deterministic:
// the same Scene always produces the same bytes, so generated fixtures can be
// committed and diffed.
func (s Scene) PNG() ([]byte, error) {
	img := s.Image()
	var buf bytes.Buffer
	buf.Grow(1024)
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Image rasterizes the scene into an RGBA image.
//
// Shapes are drawn opaque at a supersampled resolution and then box-filtered
// down. Antialiasing by supersampling rather than by analytic coverage keeps
// the code short and, more importantly, keeps circles and polygons consistent
// with each other — a mismatch would show as a visible seam where a silhouette
// meets its shadow face.
func (s Scene) Image() *image.RGBA {
	size := s.Size
	if size <= 0 {
		size = DefaultSize
	}
	ss := 3
	if size > 256 {
		ss = 2
	}
	big := size * ss

	buf := image.NewRGBA(image.Rect(0, 0, big, big))
	fill(buf, image.Rect(0, 0, big, big), s.Background)

	for _, sh := range s.Shapes {
		drawShape(buf, sh, float64(ss), big)
	}
	return downsample(buf, size, ss)
}

func fill(img *image.RGBA, r image.Rectangle, c Color) {
	rgba := color.RGBA{c.R, c.G, c.B, 0xff}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := img.Pix[img.PixOffset(r.Min.X, y):img.PixOffset(r.Max.X, y)]
		for i := 0; i < len(row); i += 4 {
			row[i], row[i+1], row[i+2], row[i+3] = rgba.R, rgba.G, rgba.B, rgba.A
		}
	}
}

// drawShape paints one shape at scale k into an image of edge length big.
// Pixels are sampled at their centre; only the shape's bounding box is walked.
func drawShape(img *image.RGBA, sh Shape, k float64, big int) {
	var minX, minY, maxX, maxY float64
	switch sh.Type {
	case ShapeRect:
		minX, minY, maxX, maxY = sh.X, sh.Y, sh.X+sh.W, sh.Y+sh.H
	case ShapeCircle:
		minX, minY, maxX, maxY = sh.CX-sh.R, sh.CY-sh.R, sh.CX+sh.R, sh.CY+sh.R
	case ShapePolygon:
		if len(sh.Points) < 3 {
			return
		}
		minX, minY = sh.Points[0].X, sh.Points[0].Y
		maxX, maxY = minX, minY
		for _, p := range sh.Points[1:] {
			minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	default:
		return
	}

	x0 := clampInt(int(math.Floor(minX*k)), 0, big)
	y0 := clampInt(int(math.Floor(minY*k)), 0, big)
	x1 := clampInt(int(math.Ceil(maxX*k))+1, 0, big)
	y1 := clampInt(int(math.Ceil(maxY*k))+1, 0, big)

	rgba := color.RGBA{sh.Fill.R, sh.Fill.G, sh.Fill.B, 0xff}
	for py := y0; py < y1; py++ {
		y := (float64(py) + 0.5) / k
		for px := x0; px < x1; px++ {
			x := (float64(px) + 0.5) / k
			if !inside(sh, x, y) {
				continue
			}
			o := img.PixOffset(px, py)
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = rgba.R, rgba.G, rgba.B, rgba.A
		}
	}
}

func inside(sh Shape, x, y float64) bool {
	switch sh.Type {
	case ShapeRect:
		if x < sh.X || y < sh.Y || x >= sh.X+sh.W || y >= sh.Y+sh.H {
			return false
		}
		r := sh.Radius
		if r <= 0 {
			return true
		}
		if r > sh.W/2 {
			r = sh.W / 2
		}
		if r > sh.H/2 {
			r = sh.H / 2
		}
		// Only the four corner squares can be outside a rounded rect.
		cx, cy := x, y
		switch {
		case x < sh.X+r:
			cx = sh.X + r
		case x > sh.X+sh.W-r:
			cx = sh.X + sh.W - r
		default:
			return true
		}
		switch {
		case y < sh.Y+r:
			cy = sh.Y + r
		case y > sh.Y+sh.H-r:
			cy = sh.Y + sh.H - r
		default:
			return true
		}
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= r*r
	case ShapeCircle:
		dx, dy := x-sh.CX, y-sh.CY
		return dx*dx+dy*dy <= sh.R*sh.R
	case ShapePolygon:
		return inPolygon(sh.Points, x, y)
	}
	return false
}

// inPolygon is the standard even-odd crossing test.
func inPolygon(pts []Point, x, y float64) bool {
	in := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		pi, pj := pts[i], pts[j]
		if (pi.Y > y) == (pj.Y > y) {
			continue
		}
		if x < (pj.X-pi.X)*(y-pi.Y)/(pj.Y-pi.Y)+pi.X {
			in = !in
		}
	}
	return in
}

// downsample box-filters a supersampled buffer down to the target edge length.
func downsample(src *image.RGBA, size, ss int) *image.RGBA {
	if ss == 1 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	n := uint32(ss * ss)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b uint32
			for sy := 0; sy < ss; sy++ {
				o := src.PixOffset(x*ss, y*ss+sy)
				for sx := 0; sx < ss; sx++ {
					r += uint32(src.Pix[o])
					g += uint32(src.Pix[o+1])
					b += uint32(src.Pix[o+2])
					o += 4
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(b / n)
			dst.Pix[o+3] = 0xff
		}
	}
	return dst
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
