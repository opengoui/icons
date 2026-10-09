package icons_test

import (
	"errors"
	"fmt"
	"slices"

	"github.com/opengoui/icons"
	"github.com/opengoui/icons/name"
)

func Example() {
	svg, err := icons.SVG(name.Search)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(svg[:4]))

	d, err := icons.Draw("arrow-right")
	if err != nil {
		panic(err)
	}
	fmt.Println(d.ViewBox, d.StrokeWidth, len(d.Paths) > 0)

	fmt.Println(slices.Contains(icons.Search("magnifier"), name.Search))
	// Output:
	// <svg
	// 24 2 true
	// true
}

func ExampleSVG_notFound() {
	_, err := icons.SVG("no-such-icon")
	fmt.Println(errors.Is(err, icons.ErrNotFound))
	// Output: true
}

func ExampleResolve() {
	// "alert-circle" is the former name of "circle-alert".
	canon, ok := icons.Resolve("alert-circle")
	fmt.Println(canon, ok)
	// Output: circle-alert true
}

func ExampleLookup() {
	info, ok := icons.Lookup(name.Search)
	fmt.Println(ok, info.Name, info.Categories)
	// Output: true search [text social]
}

func ExampleParse() {
	d, err := icons.Parse([]byte(`<svg viewBox="0 0 24 24" stroke-width="2">
  <line x1="5" y1="12" x2="19" y2="12" />
</svg>`))
	if err != nil {
		panic(err)
	}
	fmt.Println(d.ViewBox, d.StrokeWidth, d.Paths)
	// Output: 24 2 [M5 12L19 12]
}
