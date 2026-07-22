#!/usr/bin/env bash
#
# Builds the npm package: the generator compiled to WebAssembly, plus the Go
# runtime shim.
#
#   scripts/build-npm.sh          # build and test
#   scripts/build-npm.sh --pack   # also produce the tarball npm publish sends
#
# The package version is taken from the git tag, so a published package and a
# tagged release cannot disagree about what is inside them.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)"
version="${version#v}"

echo "building the module ($version)"
GOOS=js GOARCH=wasm go build \
  -trimpath \
  -ldflags "-s -w -X main.version=v$version" \
  -o npm/synth.wasm ./wasm

# The shim ships with the toolchain and must match the compiler that produced
# the module, so it is copied at build time rather than committed.
goroot="$(go env GOROOT)"
for candidate in "$goroot/lib/wasm/wasm_exec.js" "$goroot/misc/wasm/wasm_exec.js"; do
  if [ -f "$candidate" ]; then
    cp "$candidate" npm/wasm_exec.js
    break
  fi
done
if [ ! -f npm/wasm_exec.js ]; then
  echo "cannot find wasm_exec.js in $goroot" >&2
  exit 1
fi

# Keep package.json in step with the tag.
python3 - "$version" <<'PY'
import json, pathlib, sys
p = pathlib.Path('npm/package.json')
pkg = json.loads(p.read_text())
if pkg['version'] != sys.argv[1]:
    pkg['version'] = sys.argv[1]
    p.write_text(json.dumps(pkg, indent=2) + '\n')
    print(f"package.json set to {sys.argv[1]}")
PY

echo
echo "testing"
(cd npm && node test.mjs)

if [ "${1:-}" = "--pack" ]; then
  echo
  (cd npm && npm pack)
fi

echo
du -h npm/synth.wasm | awk '{print "wasm     " $1}'
gzip -9 -c npm/synth.wasm | wc -c | awk '{printf "gzipped  %.1f MB (what a consumer downloads)\n", $1/1048576}'
