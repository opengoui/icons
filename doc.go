// Package icons embeds the Lucide icon set (https://lucide.dev) as SVG files
// and exposes them by name. It depends only on the standard library and knows
// nothing about any UI toolkit: it hands out SVG documents, metadata and
// ready-to-stroke path data, and callers decide how to render them.
//
// Every icon is a 24x24 SVG drawn with stroke="currentColor", so the stroke
// color is whatever the consumer substitutes for "currentColor". All functions
// are safe for concurrent use; the package has no mutable state.
//
// # Usage
//
// Pick the entry point by what you need:
//
//   - [SVG] returns the raw SVG document, for web views, files or an SVG
//     rasterizer.
//   - [Draw] returns a [Drawing] (view box, stroke width, path data) for a
//     renderer that strokes paths itself and has no XML parser.
//   - [Search], [Lookup] and [Names] find icons by keyword and list them.
//   - [FS] exposes the raw files as an [io/fs.FS], e.g. for net/http.
//
// A name is a Lucide name such as "arrow-right", or a former name of an icon
// that Lucide has since renamed. The [github.com/opengoui/icons/name] package
// has a constant for every icon, so a typo becomes a compile error:
//
//	import (
//		"github.com/opengoui/icons"
//		"github.com/opengoui/icons/name"
//	)
//
//	svg, err := icons.SVG(name.Search)       // <svg ...>...</svg>
//	d, err := icons.Draw("arrow-right")      // d.ViewBox == 24, d.StrokeWidth == 2
//	hits := icons.Search("magnifier")        // ["search", ...]
//
// An unknown name yields an error wrapping [ErrNotFound]; test for it with
// [errors.Is].
//
// # Updating the data
//
// The icons are vendored from a pinned Lucide release (see [Version]) by
// "go generate" in this module; never edit svg/, index.json or the name
// package by hand. Lucide is licensed under ISC, with some icons MIT-licensed
// and derived from Feather; LICENSE.lucide must accompany redistributions of
// the icons.
package icons

//go:generate go run ./internal/sync
