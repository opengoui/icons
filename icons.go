package icons

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"sync"
)

//go:embed svg/*.svg
var svgFS embed.FS

//go:embed index.json
var indexJSON []byte

// ErrNotFound is returned (wrapped) when a name is neither an icon nor an
// alias of one.
var ErrNotFound = errors.New("icons: icon not found")

// Info describes one icon, as returned by [Lookup]. Its slices are copies the
// caller may modify; they are empty when the icon has no such entries.
type Info struct {
	// Name is the canonical icon name, e.g. "arrow-right".
	Name string
	// Tags are search keywords, e.g. "magnifier", "zoom".
	Tags []string
	// Categories are Lucide's groupings, e.g. "arrows", "text".
	Categories []string
	// Aliases are former names that still resolve to this icon.
	Aliases []string
}

type indexFile struct {
	Version string `json:"version"`
	Icons   map[string]struct {
		Tags       []string `json:"tags"`
		Categories []string `json:"categories"`
		Aliases    []string `json:"aliases"`
	} `json:"icons"`
}

type catalog struct {
	version string
	names   []string        // sorted canonical names
	infos   map[string]Info // canonical name -> info
	aliases map[string]string
}

var load = sync.OnceValue(func() *catalog {
	var f indexFile
	if err := json.Unmarshal(indexJSON, &f); err != nil {
		panic(fmt.Sprintf("icons: corrupt embedded index.json: %v", err))
	}
	c := &catalog{
		version: f.Version,
		infos:   make(map[string]Info, len(f.Icons)),
		aliases: map[string]string{},
	}
	for name, m := range f.Icons {
		c.infos[name] = Info{Name: name, Tags: m.Tags, Categories: m.Categories, Aliases: m.Aliases}
		for _, a := range m.Aliases {
			c.aliases[a] = name
		}
	}
	c.names = slices.Sorted(maps.Keys(c.infos))
	return c
})

// Version returns the Lucide release the embedded icons were synced from,
// e.g. "1.53.0".
func Version() string { return load().version }

// Names returns the canonical names of all icons in ascending order. The
// returned slice is a copy.
func Names() []string { return slices.Clone(load().names) }

// Resolve maps name, which may be an alias, to its canonical icon name.
func Resolve(name string) (string, bool) {
	c := load()
	if _, ok := c.infos[name]; ok {
		return name, true
	}
	canon, ok := c.aliases[name]
	return canon, ok
}

// Has reports whether name is an icon or an alias of one.
func Has(name string) bool {
	_, ok := Resolve(name)
	return ok
}

// Lookup returns the metadata of the icon called name, which may be an alias.
// The slices in the result are copies.
func Lookup(name string) (Info, bool) {
	canon, ok := Resolve(name)
	if !ok {
		return Info{}, false
	}
	in := load().infos[canon]
	in.Tags = slices.Clone(in.Tags)
	in.Categories = slices.Clone(in.Categories)
	in.Aliases = slices.Clone(in.Aliases)
	return in, true
}

// SVG returns the SVG document of the icon called name, which may be an alias.
// The result is a fresh copy the caller may modify. It fails with an error
// wrapping [ErrNotFound] if there is no such icon.
func SVG(name string) ([]byte, error) {
	canon, ok := Resolve(name)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, name)
	}
	return svgFS.ReadFile("svg/" + canon + ".svg")
}

// FS returns a read-only file system holding one "<name>.svg" file per icon at
// its root, for walking or serving the raw files. Aliases are not present.
func FS() fs.FS {
	sub, err := fs.Sub(svgFS, "svg")
	if err != nil {
		panic(err) // unreachable: "svg" is a valid, embedded directory
	}
	return sub
}

// Search returns, in ascending order, the canonical names of the icons that
// match query. The query is split on white space; an icon matches when every
// term is a case-insensitive substring of its name, an alias or one of its
// tags. An empty query matches every icon.
func Search(query string) []string {
	terms := strings.Fields(strings.ToLower(query))
	c := load()
	var out []string
	for _, name := range c.names {
		if matches(c.infos[name], terms) {
			out = append(out, name)
		}
	}
	return out
}

func matches(in Info, terms []string) bool {
	for _, t := range terms {
		if !hasTerm(in, t) {
			return false
		}
	}
	return true
}

func hasTerm(in Info, t string) bool {
	if strings.Contains(in.Name, t) {
		return true
	}
	for _, s := range in.Aliases {
		if strings.Contains(s, t) {
			return true
		}
	}
	for _, s := range in.Tags {
		if strings.Contains(strings.ToLower(s), t) {
			return true
		}
	}
	return false
}
