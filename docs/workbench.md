# Browser workbench

```bash
synth ui            # then open http://127.0.0.1:8080
```

A local page with the type palette on the left, the schema in the middle, and a
live preview on the right that regenerates as you type. The seed is shown and
editable, because reproducibility is the thing a hosted generator cannot give
you — the UI teaches it rather than hiding it.

The palette is read from the provider registry itself, so it cannot drift from
what the engine actually supports, and each type is marked with whether its
values really follow the locale. Most do not, and saying so beats letting you
assume otherwise.

## It does not listen to the network

**The server binds `127.0.0.1` and refuses anything else** —
`Serve("0.0.0.0:8080")` returns an error, and a test enforces it. The page is
embedded in the binary with inline CSS and JavaScript: no CDN, no fonts, no
telemetry, no outbound request of any kind, and a test asserts the page contains
no external origin. The browser connects in; Synth never connects out.

## The hosted demo

**[bakhod1r.github.io/synth](https://bakhod1r.github.io/synth)** is the same page
and the same engine, with the generator compiled to WebAssembly; only the backend
differs, and the page's JavaScript is byte-identical between them.

Nothing is uploaded, because there is nowhere to upload it to — the generator
runs in your tab. That is a stronger claim than a promise not to send your schema
anywhere, and it is the whole reason the demo is built this way rather than
hosted.

About 1.6 MB to download, once.
