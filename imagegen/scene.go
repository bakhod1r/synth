package imagegen

// Color is an opaque 8-bit RGB colour.
type Color struct{ R, G, B uint8 }

// ShapeType discriminates the Shape union.
type ShapeType uint8

const (
	// ShapeRect is an axis-aligned rectangle, optionally with rounded corners.
	ShapeRect ShapeType = iota
	// ShapeCircle is a circle centred on (CX,CY).
	ShapeCircle
	// ShapePolygon is a closed polygon through Points.
	ShapePolygon
)

// Point is a coordinate in scene space (pixels, origin top-left).
type Point struct{ X, Y float64 }

// Shape is one drawing primitive. Only the fields relevant to Type are read,
// which keeps the scene a flat slice with no interface dispatch — the renderer
// walks it twice (SVG and raster) and both must agree exactly.
type Shape struct {
	Type ShapeType
	Fill Color

	// Rect.
	X, Y, W, H float64
	// Radius is the corner radius of a rect. Ignored by other types.
	Radius float64

	// Circle.
	CX, CY, R float64

	// Polygon.
	Points []Point
}

// Scene is a finished, resolution-independent image: a background plus shapes
// painted in order. Text has already been expanded into rectangles by the time
// a Scene exists, so nothing downstream needs a font.
type Scene struct {
	Size       int
	Background Color
	Shapes     []Shape
}

func (s *Scene) rect(x, y, w, h float64, fill Color) {
	s.Shapes = append(s.Shapes, Shape{Type: ShapeRect, X: x, Y: y, W: w, H: h, Fill: fill})
}

func (s *Scene) roundRect(x, y, w, h, r float64, fill Color) {
	s.Shapes = append(s.Shapes, Shape{Type: ShapeRect, X: x, Y: y, W: w, H: h, Radius: r, Fill: fill})
}

func (s *Scene) circle(cx, cy, r float64, fill Color) {
	s.Shapes = append(s.Shapes, Shape{Type: ShapeCircle, CX: cx, CY: cy, R: r, Fill: fill})
}

func (s *Scene) polygon(pts []Point, fill Color) {
	s.Shapes = append(s.Shapes, Shape{Type: ShapePolygon, Points: pts, Fill: fill})
}
