#!/usr/bin/env bash
#
# Builds the static site: the workbench compiled to WebAssembly, plus the docs.
#
#   scripts/build-pages.sh [outdir]     # default: dist/pages
#
# The page's JavaScript is copied unchanged from ui/static. Only two files are
# added — the Go runtime shim and a fetch interceptor that routes /api/ calls
# into wasm — so the hosted demo cannot drift from the tool it demonstrates.
set -euo pipefail

out="${1:-dist/pages}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="$(git describe --tags --always 2>/dev/null || echo devel)"

rm -rf "$out"
mkdir -p "$out"

echo "building wasm ($version)"
GOOS=js GOARCH=wasm go build \
  -trimpath \
  -ldflags "-s -w -X main.version=$version" \
  -o "$out/synth.wasm" ./wasm

# The runtime shim ships with the Go toolchain and must match the compiler that
# produced the module, so it is copied rather than vendored.
goroot="$(go env GOROOT)"
for candidate in "$goroot/lib/wasm/wasm_exec.js" "$goroot/misc/wasm/wasm_exec.js"; do
  if [ -f "$candidate" ]; then
    cp "$candidate" "$out/wasm_exec.js"
    break
  fi
done
if [ ! -f "$out/wasm_exec.js" ]; then
  echo "cannot find wasm_exec.js in $goroot" >&2
  exit 1
fi

cp ui/static/*.js ui/static/*.css "$out/"
cp README.md CHANGELOG.md LICENSE "$out/"

# The static page differs from the served one by two script tags and a banner.
python3 - "$out" <<'PY'
import re, sys, pathlib

out = pathlib.Path(sys.argv[1])
html = pathlib.Path('ui/static/index.html').read_text()

# wasm_exec.js must load before the shim, and the shim before app.js — app.js
# calls fetch during boot, and an uninstalled interceptor would try the network.
html = html.replace(
    '<script src="i18n.js"></script>',
    '<script src="wasm_exec.js"></script>\n'
    '<script src="wasm-api.js"></script>\n'
    '<script src="i18n.js"></script>')

# Say plainly where the generator is running. On the hosted page this is the
# whole claim, and a claim nobody can see is not one.
banner = '''
<div class="wasm-note">
  Running entirely in this tab — the generator is compiled to WebAssembly and
  there is no server to send anything to.
  <a href="https://github.com/bakhod1r/synth">Source</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</div>
'''
html = html.replace('<body>', '<body>\n' + banner)
(out / 'index.html').write_text(html)

css = (out / 'app.css').read_text() + '''
/* Static build only. */
.wasm-note {
  padding: 7px 20px;
  background: #16233a;
  border-bottom: 1px solid #26334d;
  color: #9fb8ff;
  font-size: 12px;
}
.wasm-note a { color: #cfe0ff; }
'''
(out / 'app.css').write_text(css)
PY

# GitHub Pages serves this repository under /synth, and Jekyll would otherwise
# swallow files it does not recognise.
touch "$out/.nojekyll"

echo
echo "built $out"
du -h "$out/synth.wasm" | awk '{print "  wasm      " $1}'
gzip -9 -c "$out/synth.wasm" | wc -c | awk '{printf "  gzipped   %.1f MB (what a visitor downloads)\n", $1/1048576}'
ls "$out" | sed 's/^/  /'
