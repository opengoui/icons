package icons_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"slices"
	"strconv"
	"testing"

	"github.com/opengoui/icons"
	"github.com/opengoui/icons/name"
)

func TestSVG(t *testing.T) {
	data, err := icons.SVG(name.Search)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("<svg")) || !bytes.Contains(data, []byte(`stroke="currentColor"`)) {
		t.Fatalf("unexpected document:\n%s", data)
	}
	data[0] = 'X' // the result is a private copy
	again, _ := icons.SVG(name.Search)
	if again[0] != '<' {
		t.Fatal("SVG returned shared storage")
	}
}

func TestSVGNotFound(t *testing.T) {
	_, err := icons.SVG("no-such-icon")
	if !errors.Is(err, icons.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAliasResolves(t *testing.T) {
	// "alert-circle" was renamed to "circle-alert" in Lucide.
	canon, ok := icons.Resolve("alert-circle")
	if !ok || canon != name.CircleAlert {
		t.Fatalf("Resolve(alert-circle) = %q, %v; want %q", canon, ok, name.CircleAlert)
	}
	want, _ := icons.SVG(name.CircleAlert)
	got, err := icons.SVG("alert-circle")
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("alias SVG differs from canonical: err=%v", err)
	}
	if icons.Has("alert-circle") != true || !icons.Has(name.CircleAlert) || icons.Has("nope") {
		t.Fatal("Has disagrees with Resolve")
	}
}

func TestLookup(t *testing.T) {
	in, ok := icons.Lookup(name.Search)
	if !ok || in.Name != "search" || !slices.Contains(in.Tags, "magnifier") {
		t.Fatalf("Lookup(search) = %+v, %v", in, ok)
	}
	in.Tags[0] = "mutated"
	again, _ := icons.Lookup(name.Search)
	if again.Tags[0] == "mutated" {
		t.Fatal("Lookup returned shared slices")
	}
	if _, ok := icons.Lookup("nope"); ok {
		t.Fatal("Lookup(nope) succeeded")
	}
}

func TestNames(t *testing.T) {
	names := icons.Names()
	if len(names) < 1000 || !slices.IsSorted(names) {
		t.Fatalf("Names: %d entries, sorted=%v", len(names), slices.IsSorted(names))
	}
	names[0] = "mutated"
	if icons.Names()[0] == "mutated" {
		t.Fatal("Names returned shared storage")
	}
}

func TestSearch(t *testing.T) {
	if got := icons.Search("magnifier"); !slices.Contains(got, name.Search) {
		t.Fatalf(`Search("magnifier") = %v, want it to contain "search"`, got)
	}
	if got := icons.Search("ARROW right"); !slices.Contains(got, name.ArrowRight) {
		t.Fatalf(`Search("ARROW right") misses arrow-right: %v`, got)
	}
	if got := icons.Search("zzzz-no-match"); len(got) != 0 {
		t.Fatalf("Search(nonsense) = %v", got)
	}
	if got := icons.Search(""); len(got) != len(icons.Names()) {
		t.Fatalf(`Search("") = %d icons, want all %d`, len(got), len(icons.Names()))
	}
}

// TestFSMatchesCatalog checks that the embedded files and the index describe
// exactly the same set of icons.
func TestFSMatchesCatalog(t *testing.T) {
	entries, err := fs.ReadDir(icons.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		files = append(files, e.Name())
	}
	var want []string
	for _, n := range icons.Names() {
		want = append(want, n+".svg")
	}
	slices.Sort(want) // "a-b.svg" sorts before "a.svg", unlike the bare names
	if !slices.Equal(files, want) {
		t.Fatalf("svg files and index.json disagree: %d files, %d indexed", len(files), len(want))
	}
}

// TestNameConstantsMatchCatalog checks that the generated name constants are
// exactly the icon set: none stale, none missing.
func TestNameConstantsMatchCatalog(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "name/name_gen.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, s := range gd.Specs {
			lit := s.(*ast.ValueSpec).Values[0].(*ast.BasicLit)
			v, err := strconv.Unquote(lit.Value)
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, v)
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, icons.Names()) {
		t.Fatalf("name constants (%d) differ from icons (%d); run go generate", len(got), len(icons.Names()))
	}
}

func TestVersion(t *testing.T) {
	if icons.Version() == "" {
		t.Fatal("empty version")
	}
}

func TestParseShapes(t *testing.T) {
	const doc = `<svg viewBox="0 0 24 24" stroke-width="2">
  <path d="m21 21-4.34-4.34" />
  <circle cx="11" cy="11" r="8" />
  <rect x="3" y="4" width="10" height="6" rx="2" />
  <rect x="1" y="1" width="4" height="4" />
  <line x1="1" x2="2" y1="3" y2="4" />
  <polygon points="0 0 1 1 2 0" />
  <polyline points="0,0 1,1" />
  <ellipse cx="5" cy="5" rx="3" ry="2" />
</svg>`
	d, err := icons.Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"m21 21-4.34-4.34",
		"M3 11a8 8 0 1 0 16 0a8 8 0 1 0 -16 0Z",
		"M5 4H11a2 2 0 0 1 2 2V8a2 2 0 0 1 -2 2H5a2 2 0 0 1 -2 -2V6a2 2 0 0 1 2 -2Z",
		"M1 1H5V5H1Z",
		"M1 3L2 4",
		"M0 0L1 1L2 0Z",
		"M0 0L1 1",
		"M2 5a3 2 0 1 0 6 0a3 2 0 1 0 -6 0Z",
	}
	if d.ViewBox != 24 || d.StrokeWidth != 2 || !slices.Equal(d.Paths, want) {
		t.Fatalf("Parse = %+v\nwant paths %q", d, want)
	}
}

func TestParseRejects(t *testing.T) {
	for name, doc := range map[string]string{
		"group":      `<svg viewBox="0 0 24 24"><g><path d="M0 0"/></g></svg>`,
		"non-square": `<svg viewBox="0 0 24 12"><path d="M0 0"/></svg>`,
		"offset":     `<svg viewBox="1 0 24 24"><path d="M0 0"/></svg>`,
		"no viewBox": `<svg><path d="M0 0"/></svg>`,
		"bad number": `<svg viewBox="0 0 24 24"><circle cx="x" cy="1" r="1"/></svg>`,
		"empty path": `<svg viewBox="0 0 24 24"><path d=""/></svg>`,
		"not svg":    `<html/>`,
		"not xml":    `<svg`,
	} {
		if _, err := icons.Parse([]byte(doc)); err == nil {
			t.Errorf("%s: Parse succeeded", name)
		}
	}
}

// TestDrawAll guarantees every embedded icon reduces to a drawing, so a Lucide
// upgrade that introduces an unsupported element fails here, not in an app.
func TestDrawAll(t *testing.T) {
	for _, n := range icons.Names() {
		d, err := icons.Draw(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if d.ViewBox != 24 || d.StrokeWidth != 2 || len(d.Paths) == 0 {
			t.Fatalf("%s: unexpected drawing %+v", n, d)
		}
	}
	if _, err := icons.Draw("nope"); !errors.Is(err, icons.ErrNotFound) {
		t.Fatalf("Draw(nope) err = %v", err)
	}
}
