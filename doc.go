// Package icons embeds the Lucide icon set (https://lucide.dev) as SVG files
// and exposes them by name. It has no dependencies beyond the standard library
// and knows nothing about any UI toolkit: it hands out SVG documents and
// metadata, and callers decide how to render them.
//
// Every icon is a 24x24 SVG that paints with stroke="currentColor", so the
// stroke color is whatever the consumer substitutes for "currentColor".
//
// Look an icon up by its Lucide name, or by one of its former names:
//
//	data, err := icons.SVG("search")
//
// The [github.com/opengoui/icons/name] package has a constant for every icon,
// so a typo becomes a compile error and editors can complete names:
//
//	data, err := icons.SVG(name.Search)
//
// Use [Search] to find icons by keyword, [Lookup] for tags and categories, and
// [FS] to serve or walk the raw files.
//
// The data is vendored by "go generate" from a pinned Lucide release; see
// [Version]. Lucide is licensed under ISC (with some icons MIT, derived from
// Feather); the LICENSE file in this module must accompany redistributions.
package icons

//go:generate go run ./internal/sync
