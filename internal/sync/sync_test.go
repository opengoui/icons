package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const svgDoc = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0" /></svg>`

// archive builds a gzipped tarball shaped like a Lucide source release.
func archive(t *testing.T, members map[string]string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for k := range members {
			if !yield(k) {
				return
			}
		}
	}) {
		body := members[name]
		if err := tw.WriteHeader(&tar.Header{Name: "lucide-9.9.9/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func goodMembers() map[string]string {
	return map[string]string{
		"LICENSE":                 "ISC",
		"README.md":               "ignored",
		"icons/arrow-up-0-1.svg":  svgDoc,
		"icons/arrow-up-0-1.json": `{"tags":["sort"],"categories":["arrows"],"aliases":[{"name":"old-sort","deprecated":true}]}`,
		"icons/x.svg":             svgDoc,
		"icons/x.json":            `{"tags":[],"categories":[]}`,
		"packages/other/y.svg":    svgDoc,
	}
}

func TestReadArchiveAndRender(t *testing.T) {
	b, err := readArchive("9.9.9", archive(t, goodMembers()))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.svgs) != 2 {
		t.Fatalf("svgs = %d, want 2 (files outside icons/ must be ignored)", len(b.svgs))
	}
	if got := b.meta["arrow-up-0-1"]; !slices.Equal(got.Aliases, []string{"old-sort"}) || !slices.Equal(got.Tags, []string{"sort"}) {
		t.Fatalf("meta = %+v", got)
	}

	files, err := b.render()
	if err != nil {
		t.Fatal(err)
	}
	gen := string(files["name/name_gen.go"])
	for _, want := range []string{"package name", `ArrowUp01 = "arrow-up-0-1"`, `X = "x"`, "Lucide 9.9.9"} {
		if !strings.Contains(gen, want) {
			t.Errorf("name_gen.go lacks %q:\n%s", want, gen)
		}
	}
	for _, key := range []string{"LICENSE.lucide", "index.json", "svg/x.svg", "svg/arrow-up-0-1.svg"} {
		if _, ok := files[key]; !ok {
			t.Errorf("missing output %s", key)
		}
	}
}

func TestWriteRemovesStaleSVG(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "svg"), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, "svg", "gone.svg")
	keepOther := filepath.Join(root, "README.md")
	for _, p := range []string{stale, keepOther} {
		if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := readArchive("9.9.9", archive(t, goodMembers()))
	if err != nil {
		t.Fatal(err)
	}
	files, err := b.render()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "LICENSE"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := write(root, files); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "LICENSE")); string(got) != "mine" {
		t.Errorf("module LICENSE overwritten: %q", got)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale svg survived: %v", err)
	}
	if _, err := os.Stat(keepOther); err != nil {
		t.Errorf("hand-written file removed: %v", err)
	}
	if got := recordedVersion(root); got != "9.9.9" {
		t.Errorf("recordedVersion = %q, want 9.9.9", got)
	}
}

func TestReadArchiveRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(m map[string]string)
		want   string
	}{
		{"svg without json", func(m map[string]string) { delete(m, "icons/x.json") }, "no x.json"},
		{"json without svg", func(m map[string]string) { delete(m, "icons/x.svg") }, "x.json has no x.svg"},
		{"not svg", func(m map[string]string) { m["icons/x.svg"] = "<html>" }, "not an <svg>"},
		{"bad name", func(m map[string]string) { m["icons/Bad_Name.svg"] = svgDoc; m["icons/Bad_Name.json"] = "{}" }, "invalid icon name"},
		{"alias is icon", func(m map[string]string) { m["icons/x.json"] = `{"aliases":[{"name":"arrow-up-0-1"}]}` }, "also an icon"},
		{"duplicate alias", func(m map[string]string) { m["icons/x.json"] = `{"aliases":[{"name":"old-sort"}]}` }, "claimed by both"},
		{"no license", func(m map[string]string) { delete(m, "LICENSE") }, "no LICENSE"},
		{"empty", func(m map[string]string) { clear(m) }, "no icons"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := goodMembers()
			tt.mutate(m)
			_, err := readArchive("9.9.9", archive(t, m))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRenderIdentifierCollision(t *testing.T) {
	m := goodMembers()
	m["icons/arrow-up-01.svg"] = svgDoc
	m["icons/arrow-up-01.json"] = "{}"
	b, err := readArchive("9.9.9", archive(t, m))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.render(); err == nil || !strings.Contains(err.Error(), "ArrowUp01") {
		t.Fatalf("err = %v, want identifier collision", err)
	}
}

func TestIdent(t *testing.T) {
	for in, want := range map[string]string{
		"search":       "Search",
		"a-arrow-down": "AArrowDown",
		"arrow-up-0-1": "ArrowUp01",
		"2fa":          "Icon2fa",
		"building-2":   "Building2",
		"wifi":         "Wifi",
	} {
		if got := ident(in); got != want {
			t.Errorf("ident(%q) = %q, want %q", in, got, want)
		}
	}
}
