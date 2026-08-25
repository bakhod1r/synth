# Install

```bash
go get github.com/bakhod1r/synth              # library
go install github.com/bakhod1r/synth/cmd/synth@latest   # CLI
npm install @bakhod1r/synth                   # JavaScript, via WebAssembly
```

The MCP server is a separate module, so its 20 transitive dependencies stay out
of the core module's graph:

```bash
go install github.com/bakhod1r/synth/mcp/cmd/synth-mcp@latest
```

See [MCP](../mcp.md) for what it exposes.

## Try it without installing anything

**[bakhod1r.github.io/synth](https://bakhod1r.github.io/synth)** — the workbench
with the generator compiled to WebAssembly.

It is the same page and the same engine as `synth ui`; only the backend differs,
and the page's JavaScript is byte-identical between them. Nothing is uploaded,
because there is nowhere to upload it to — the generator runs in your tab. That
is a stronger claim than a promise not to send your schema anywhere, and it is
the whole reason the demo is built this way rather than hosted.

About 1.6 MB to download, once.

## Requirements

| | |
| --- | --- |
| Go | 1.26 or newer for the library and CLI (`go.mod` declares 1.26.2) |
| Core dependencies | exactly two, and CI fails if that slips |
| Node | only for the npm/WebAssembly package |
