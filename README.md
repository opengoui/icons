# icons

The [Lucide](https://lucide.dev) icon set as embedded SVG for Go. It depends only on the standard library and is not tied to any UI framework (OpenGoUI is just one consumer): it provides SVG documents, metadata and path data, and the caller decides how to render them.

```go
import (
	"github.com/opengoui/icons"
	"github.com/opengoui/icons/name"
)

svg, err := icons.SVG(name.Search)       // []byte, 24x24, stroke="currentColor"
d, err := icons.Draw(name.ArrowRight)    // view box, stroke width and path data
hits := icons.Search("arrow right")      // search by name, alias and tag
info, ok := icons.Lookup("alert-circle") // former names resolve too
```

| API | Description |
| --- | --- |
| `SVG(name)` | The icon's SVG document, as a copy; an unknown name returns an error wrapping `ErrNotFound` |
| `Names()` / `Has` / `Resolve` | All icon names (ascending), existence check, alias to canonical name |
| `Lookup(name)` | `Info{Name, Tags, Categories, Aliases}` |
| `Search(query)` | Whitespace-separated terms; each must be a case-insensitive substring of the name, an alias or a tag |
| `Draw(name)` / `Parse(svg)` | Reduces an icon to `Drawing{ViewBox, StrokeWidth, Paths}`: a square view box plus SVG path data (circle, rect, line, polyline, ... are converted to paths), so a renderer needs no XML parser |
| `FS()` | An `fs.FS` with `<name>.svg` at its root, for walking or `http.FileServer` |
| `Version()` | The embedded Lucide version |
| `name.*` | One string constant per icon (`name.ArrowRight`), so typos fail at compile time |

Run `go doc github.com/opengoui/icons` for the full reference.

## Using it with OpenGoUI

`vigo/kit/lucide` feeds `Draw` into `kit.NewIconData` and caches the result. Icons that point along the reading direction (arrows, chevrons) are marked `Directional`, so they mirror in right-to-left layouts:

```go
kit.NewIcon(lucide.Must(name.Search)).Size(20)
```

## Syncing the icons

The icon data (`svg/`, `index.json`, `name/name_gen.go`, `LICENSE.lucide`) is generated from a Lucide GitHub release; **do not edit it by hand**:

```bash
make sync                       # re-sync the version recorded in index.json
make upgrade                    # upgrade to the latest release
make sync VERSION=1.53.0        # pin an exact release
make sync VERSION=1.53.0 ARCHIVE=lucide.tgz   # offline, from a local source tarball
```

`go generate ./...` and `go run ./internal/sync [-version V] [-archive FILE]` do the same without make. Set `GITHUB_TOKEN` to avoid API rate limits when resolving `latest`. After upgrading, run `make test`: the tests check that `svg/`, `index.json` and the `name` constants agree, and that every icon still reduces to a `Drawing`.

## Distribution and size

The data is embedded with `go:embed`, so `go get` is all a consumer needs: no network or build step. The module is about 1.3 MB (1869 SVGs), and the linker only includes the data of packages that are actually referenced. Module tags follow the upstream Lucide version: this module's `vX.Y.Z` embeds Lucide `X.Y.Z`, so `go get github.com/opengoui/icons@v1.53.0` gives you Lucide 1.53.0. After `make upgrade`, tag the new release with the version printed by `icons.Version()`.

## License

The code of this module is under the license in `LICENSE`. The icons are Lucide's, licensed under ISC (some derive from Feather, MIT); redistributions of the icons must include `LICENSE.lucide`, which the sync updates along with them.
