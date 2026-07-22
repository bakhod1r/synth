package ui

// The schema-to-YAML conversion and the CSV/SQL writers moved to
// internal/webspec, so the WebAssembly build of this same page can use them
// without dragging net/http into a wasm binary — and, more importantly, so the
// two backends cannot drift apart while the page's JavaScript stays identical
// between them.
