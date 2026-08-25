# Images that belong to their row

An `avatar` column that points at `picsum.photos` is a placeholder twice over:
it needs the network, and it depicts nobody. Synth draws the picture instead,
from the row's own text.

```go
type User struct {
    ID     uuid.UUID `synth:"pk"`
    Name   string
    Avatar string `synth:"avatar,from=Name"`      // initials, colour, shape from the name
    Icon   string `synth:"identicon,from=ID"`     // the GitHub-style pixel mark
}
```

Four kinds: `avatar` (a person), `productimage` (a catalogue thumbnail), `logo`
(a company monogram), `identicon` (a symmetric mark for any key).

The image is a **pure function of the subject**, so the same person keeps the
same face across runs, across formats and across datasets — regenerate the
fixture and the diff is empty. `from=` is what ties it to the row; without it
the picture depicts an unrelated name.

## Parameters

| Param | Meaning |
| --- | --- |
| `from=<field>` | take the subject from a sibling column |
| `format=` | `dataurl` (default, base64 SVG), `svg`, `png` |
| `size=` | edge length in pixels (default 128, max 1024) |
| `dir=` | write files there, put the path in the column |
| `seed=` | a different but equally stable image |
| `vary=true` | let repeated subjects differ (off by default) |

## No network, no fonts

Nothing is fetched and no font is required: text is rasterized from a built-in
5×7 bitmap into plain rectangles, so the SVG renders identically everywhere and
matches the PNG exactly. The renderer is usable on its own as
[`imagegen`](https://github.com/bakhod1r/synth/tree/main/imagegen), and the
`dataurl` default drops straight into a browser with no file to ship alongside
it.

Worked example:
[examples/images](https://github.com/bakhod1r/synth/tree/main/examples/images),
[examples/catalog.yaml](https://github.com/bakhod1r/synth/blob/main/examples/catalog.yaml).

The `imageurl` kind is unchanged — it still returns a placeholder-service URL
for cases that want one.
