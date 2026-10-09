package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// maxFileSize bounds a single archive member; real Lucide files are a few KB.
const maxFileSize = 1 << 20

var validName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// meta is the per-icon record stored in index.json.
type meta struct {
	Tags       []string `json:"tags,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// index is the content of index.json.
type index struct {
	Version string          `json:"version"`
	Icons   map[string]meta `json:"icons"`
}

// bundle is everything extracted from one Lucide release.
type bundle struct {
	version string
	svgs    map[string][]byte
	meta    map[string]meta
	license []byte
}

// lucideJSON is the subset of Lucide's per-icon JSON that we keep.
type lucideJSON struct {
	Tags       []string `json:"tags"`
	Categories []string `json:"categories"`
	Aliases    []struct {
		Name string `json:"name"`
	} `json:"aliases"`
}

// readArchive extracts icons/*.svg, icons/*.json and LICENSE from a gzipped
// Lucide source tarball. The archive's single top-level directory is ignored.
func readArchive(version string, r io.Reader) (*bundle, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	b := &bundle{version: version, svgs: map[string][]byte{}, meta: map[string]meta{}}
	jsons := map[string]lucideJSON{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok {
			continue
		}
		dir, file := path.Split(rel)
		switch {
		case rel == "LICENSE":
			if b.license, err = readLimited(tr); err != nil {
				return nil, err
			}
		case dir == "icons/" && strings.HasSuffix(file, ".svg"):
			data, err := readLimited(tr)
			if err != nil {
				return nil, err
			}
			b.svgs[strings.TrimSuffix(file, ".svg")] = data
		case dir == "icons/" && strings.HasSuffix(file, ".json"):
			data, err := readLimited(tr)
			if err != nil {
				return nil, err
			}
			var j lucideJSON
			if err := json.Unmarshal(data, &j); err != nil {
				return nil, fmt.Errorf("parse %s: %w", rel, err)
			}
			jsons[strings.TrimSuffix(file, ".json")] = j
		}
	}
	if err := b.finish(jsons); err != nil {
		return nil, err
	}
	return b, nil
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileSize {
		return nil, fmt.Errorf("archive member exceeds %d bytes", maxFileSize)
	}
	return data, nil
}

// finish validates the extracted data and fills b.meta.
func (b *bundle) finish(jsons map[string]lucideJSON) error {
	if len(b.svgs) == 0 {
		return errors.New("archive has no icons/*.svg")
	}
	if len(b.license) == 0 {
		return errors.New("archive has no LICENSE")
	}
	taken := map[string]string{} // alias -> owning icon
	for name, data := range b.svgs {
		if !validName.MatchString(name) {
			return fmt.Errorf("invalid icon name %q", name)
		}
		if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("<svg")) {
			return fmt.Errorf("%s.svg is not an <svg> document", name)
		}
		j, ok := jsons[name]
		if !ok {
			return fmt.Errorf("%s.svg has no %s.json", name, name)
		}
		m := meta{Tags: j.Tags, Categories: j.Categories}
		for _, a := range j.Aliases {
			if !validName.MatchString(a.Name) {
				return fmt.Errorf("%s: invalid alias %q", name, a.Name)
			}
			if _, clash := b.svgs[a.Name]; clash {
				return fmt.Errorf("%s: alias %q is also an icon", name, a.Name)
			}
			if owner, dup := taken[a.Name]; dup {
				return fmt.Errorf("alias %q claimed by both %s and %s", a.Name, owner, name)
			}
			taken[a.Name] = name
			m.Aliases = append(m.Aliases, a.Name)
		}
		slices.Sort(m.Aliases)
		b.meta[name] = m
	}
	for name := range jsons {
		if _, ok := b.svgs[name]; !ok {
			return fmt.Errorf("%s.json has no %s.svg", name, name)
		}
	}
	return nil
}

// ident converts an icon name such as "arrow-up-0-1" to an exported Go
// identifier such as "ArrowUp01".
func ident(name string) string {
	var sb strings.Builder
	for part := range strings.SplitSeq(name, "-") {
		sb.WriteString(upperFirst(part))
	}
	s := sb.String()
	if s != "" && unicode.IsDigit(rune(s[0])) {
		s = "Icon" + s
	}
	return s
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// render returns the files to write, keyed by slash path relative to the module
// root: svg/*.svg, index.json, name/name_gen.go and LICENSE.lucide (Lucide's
// license; the module's own LICENSE is never touched).
func (b *bundle) render() (map[string][]byte, error) {
	files := map[string][]byte{"LICENSE.lucide": b.license}
	for name, data := range b.svgs {
		files["svg/"+name+".svg"] = data
	}
	idx, err := json.MarshalIndent(index{Version: b.version, Icons: b.meta}, "", "\t")
	if err != nil {
		return nil, err
	}
	files["index.json"] = append(idx, '\n')

	var src bytes.Buffer
	fmt.Fprintf(&src, "// Code generated by internal/sync from Lucide %s; DO NOT EDIT.\n\n", b.version)
	src.WriteString("package name\n\nconst (\n")
	seen := map[string]string{}
	for _, n := range slices.Sorted(maps.Keys(b.svgs)) {
		id := ident(n)
		if prev, dup := seen[id]; dup {
			return nil, fmt.Errorf("icons %q and %q both map to identifier %s", prev, n, id)
		}
		seen[id] = n
		fmt.Fprintf(&src, "\t// %s is the %q icon.\n\t%s = %q\n", id, n, id, n)
	}
	src.WriteString(")\n")
	formatted, err := format.Source(src.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format name_gen.go: %w", err)
	}
	files["name/name_gen.go"] = formatted
	return files, nil
}

// write replaces the generated files under root with files. Stale svg files
// are removed; hand-written files are never touched.
func write(root string, files map[string][]byte) error {
	svgDir := filepath.Join(root, "svg")
	if err := os.MkdirAll(svgDir, 0o755); err != nil {
		return err
	}
	old, err := filepath.Glob(filepath.Join(svgDir, "*.svg"))
	if err != nil {
		return err
	}
	for _, p := range old {
		if _, keep := files["svg/"+filepath.Base(p)]; !keep {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "name"), 0o755); err != nil {
		return err
	}
	for rel, data := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
