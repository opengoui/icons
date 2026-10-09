package icons

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Drawing is an icon reduced to what a vector renderer needs: a square view
// box and SVG path data, all to be stroked with round caps and joins. It
// spares consumers an XML parser and the shape elements (circle, rect, ...) of
// the source SVG.
type Drawing struct {
	// ViewBox is the edge of the square coordinate space, e.g. 24.
	ViewBox float64
	// StrokeWidth is the line width in view-box units, e.g. 2.
	StrokeWidth float64
	// Paths are SVG path-data strings (commands M L H V C S Q T A Z, absolute
	// and relative). Every shape of the icon is one entry.
	Paths []string
}

// Draw returns the [Drawing] of the icon called name, which may be an alias.
// It fails with an error wrapping [ErrNotFound] if there is no such icon.
func Draw(name string) (Drawing, error) {
	data, err := SVG(name)
	if err != nil {
		return Drawing{}, err
	}
	d, err := Parse(data)
	if err != nil {
		return Drawing{}, fmt.Errorf("icons: %s: %w", name, err)
	}
	return d, nil
}

// Parse reduces a Lucide-style SVG document to a [Drawing]. It understands a
// root <svg> with a square viewBox at the origin and a stroke-width, holding
// flat path, circle, ellipse, rect, line, polyline and polygon elements. Fills
// are ignored: Lucide draws everything as a stroke, and the few elements that
// also set fill="currentColor" are discs that the stroke already covers. Any
// other element is an error rather than silently dropped.
func Parse(svg []byte) (Drawing, error) {
	dec := xml.NewDecoder(bytes.NewReader(svg))
	var d Drawing
	sawRoot := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Drawing{}, fmt.Errorf("parse svg: %w", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		a := attrs(el)
		if !sawRoot {
			if el.Name.Local != "svg" {
				return Drawing{}, fmt.Errorf("root element is <%s>, want <svg>", el.Name.Local)
			}
			sawRoot = true
			if err := d.parseRoot(a); err != nil {
				return Drawing{}, err
			}
			continue
		}
		p, err := elementPath(el.Name.Local, a)
		if err != nil {
			return Drawing{}, err
		}
		d.Paths = append(d.Paths, p)
	}
	if !sawRoot {
		return Drawing{}, errors.New("empty svg document")
	}
	return d, nil
}

func (d *Drawing) parseRoot(a map[string]string) error {
	vb := strings.Fields(strings.ReplaceAll(a["viewBox"], ",", " "))
	if len(vb) != 4 {
		return fmt.Errorf("viewBox %q: want 4 numbers", a["viewBox"])
	}
	var n [4]float64
	for i, s := range vb {
		v, err := num(s)
		if err != nil {
			return fmt.Errorf("viewBox %q: %w", a["viewBox"], err)
		}
		n[i] = v
	}
	if n[0] != 0 || n[1] != 0 || n[2] != n[3] || !(n[2] > 0) {
		return fmt.Errorf("viewBox %q: want a square at the origin", a["viewBox"])
	}
	d.ViewBox = n[2]
	d.StrokeWidth = 1 // the SVG default
	if s, ok := a["stroke-width"]; ok {
		w, err := num(s)
		if err != nil || w < 0 {
			return fmt.Errorf("stroke-width %q is not a non-negative number", s)
		}
		d.StrokeWidth = w
	}
	return nil
}

func attrs(el xml.StartElement) map[string]string {
	m := make(map[string]string, len(el.Attr))
	for _, at := range el.Attr {
		m[at.Name.Local] = at.Value
	}
	return m
}

func num(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%q is not a finite number", s)
	}
	return v, nil
}

// elementPath converts one shape element to path data.
func elementPath(tag string, a map[string]string) (string, error) {
	// get reads a numeric attribute, 0 when absent.
	var err error
	get := func(k string) float64 {
		s, ok := a[k]
		if !ok || err != nil {
			return 0
		}
		v, e := num(s)
		if e != nil {
			err = fmt.Errorf("<%s> %s: %w", tag, k, e)
		}
		return v
	}
	var p string
	switch tag {
	case "path":
		p = a["d"]
		if strings.TrimSpace(p) == "" {
			return "", errors.New("<path> without d")
		}
	case "circle":
		r := get("r")
		p = ellipse(get("cx"), get("cy"), r, r)
	case "ellipse":
		p = ellipse(get("cx"), get("cy"), get("rx"), get("ry"))
	case "line":
		p = "M" + f(get("x1")) + " " + f(get("y1")) + "L" + f(get("x2")) + " " + f(get("y2"))
	case "rect":
		x, y, w, h := get("x"), get("y"), get("width"), get("height")
		rx, ry := get("rx"), get("ry")
		if _, ok := a["rx"]; !ok {
			rx = ry
		}
		if _, ok := a["ry"]; !ok {
			ry = rx
		}
		p = rect(x, y, w, h, math.Min(rx, w/2), math.Min(ry, h/2))
	case "polyline", "polygon":
		pts := strings.Fields(strings.ReplaceAll(a["points"], ",", " "))
		if len(pts) < 4 || len(pts)%2 != 0 {
			return "", fmt.Errorf("<%s> points %q: want x y pairs", tag, a["points"])
		}
		for i := 0; i < len(pts); i += 2 {
			x, e1 := num(pts[i])
			y, e2 := num(pts[i+1])
			if e := errors.Join(e1, e2); e != nil {
				return "", fmt.Errorf("<%s> points: %w", tag, e)
			}
			cmd := "L"
			if i == 0 {
				cmd = "M"
			}
			p += cmd + f(x) + " " + f(y)
		}
		if tag == "polygon" {
			p += "Z"
		}
	default:
		return "", fmt.Errorf("unsupported element <%s>", tag)
	}
	if err != nil {
		return "", err
	}
	return p, nil
}

func f(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

func ellipse(cx, cy, rx, ry float64) string {
	return "M" + f(cx-rx) + " " + f(cy) +
		"a" + f(rx) + " " + f(ry) + " 0 1 0 " + f(2*rx) + " 0" +
		"a" + f(rx) + " " + f(ry) + " 0 1 0 " + f(-2*rx) + " 0Z"
}

func rect(x, y, w, h, rx, ry float64) string {
	if !(rx > 0) || !(ry > 0) {
		return "M" + f(x) + " " + f(y) + "H" + f(x+w) + "V" + f(y+h) + "H" + f(x) + "Z"
	}
	arc := func(dx, dy float64) string {
		return "a" + f(rx) + " " + f(ry) + " 0 0 1 " + f(dx) + " " + f(dy)
	}
	return "M" + f(x+rx) + " " + f(y) +
		"H" + f(x+w-rx) + arc(rx, ry) +
		"V" + f(y+h-ry) + arc(-rx, ry) +
		"H" + f(x+rx) + arc(-rx, -ry) +
		"V" + f(y+ry) + arc(rx, -ry) + "Z"
}
