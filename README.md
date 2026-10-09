# icons

把 [Lucide](https://lucide.dev) 图标集以 SVG 形式嵌入的 Go 模块。**只依赖标准库**，不绑定任何 UI 框架（OpenGoUI 只是其中一个使用者）：它提供 SVG 文档与元数据，怎么渲染由调用方决定。

```go
import (
	"github.com/opengoui/icons"
	"github.com/opengoui/icons/name"
)

data, err := icons.SVG(name.Search)      // []byte，24x24，stroke="currentColor"
ids := icons.Search("arrow right")       // 按名称 / 别名 / 标签搜索
info, ok := icons.Lookup("alert-circle") // 旧名（别名）同样可用
```

| API | 说明 |
| --- | --- |
| `SVG(name)` | 图标 SVG，返回副本；未知名称返回包装了 `ErrNotFound` 的错误 |
| `Names()` / `Has` / `Resolve` | 全部图标名（升序）、是否存在、别名 → 规范名 |
| `Lookup(name)` | `Info{Name, Tags, Categories, Aliases}` |
| `Search(query)` | 空白分词，每个词需子串命中名称、别名或标签（忽略大小写） |
| `Draw(name)` / `Parse(svg)` | 化简成 `Drawing{ViewBox, StrokeWidth, Paths}`：方形视图框 + SVG path data（circle/rect/line/polyline 等已转成路径），渲染器无需 XML 解析器 |
| `FS()` | `fs.FS`，根目录下是 `<name>.svg`，可用于遍历或 `http.FileServer` |
| `Version()` | 当前嵌入的 Lucide 版本 |
| `name.*` | 每个图标一个字符串常量（`name.ArrowRight`），拼写错误在编译期暴露 |

## 在 OpenGoUI 中使用

`vigo/kit/lucide` 把 `Draw` 的结果交给 `kit.NewIconData`，并缓存；方向性图标（箭头、chevron）自动标记 `Directional`，RTL 下镜像：

```go
kit.NewIcon(lucide.Must(name.Search)).Size(20)
```

`vigo/go.mod` 目前用 `replace github.com/opengoui/icons => ../icons` 指向本地目录；`icons` 发布并打 tag 后，换成真实版本号并删除 replace。

## 同步图标

图标数据（`svg/`、`index.json`、`name/name_gen.go`、`LICENSE`）由脚本从 Lucide 的 GitHub release 源码包生成，**不要手改**：

```bash
go generate ./...                          # 重新同步 index.json 里记录的版本
go run ./internal/sync -version latest     # 升级到最新 release
go run ./internal/sync -version 1.53.0     # 固定到指定版本
go run ./internal/sync -archive l.tgz -version 1.53.0   # 离线：使用本地源码包
```

设置 `GITHUB_TOKEN` 可避开查询 `latest` 时的 API 限流。升级后运行 `go test ./...`：测试会校验 `svg/`、`index.json` 与 `name` 常量三者一致。

## 发布与体积

数据通过 `go:embed` 嵌入，`go get` 即用，无需网络或构建步骤；模块体积约 1.3 MB（1869 个 SVG）。链接器只会把被引用的包数据编入二进制。上游版本与本模块版本相互独立，升级图标后按语义化版本打 tag。

## 许可

Lucide 采用 ISC 许可（部分图标源自 Feather，MIT）。再分发时必须附带本目录的 `LICENSE`，同步脚本会随图标一并更新它。
