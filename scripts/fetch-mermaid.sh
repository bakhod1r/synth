#!/usr/bin/env bash
#
# Downloads the mermaid bundle the docs serve from their own origin.
#
#   scripts/fetch-mermaid.sh
#
# Material for MkDocs renders mermaid by lazy-loading it from unpkg.com unless a
# global `mermaid` already exists. This is that same build, served from the docs
# origin instead — it assigns the global, so the CDN request never happens and
# the docs match the workbench's no-external-origin rule.
#
# It is 4 MB, so it is fetched rather than committed. An existing copy is kept,
# which is what makes an offline `mkdocs build` work after the first run.
set -euo pipefail

version="${MERMAID_VERSION:-11}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dest="$root/docs/assets/js/mermaid.min.js"

if [ -s "$dest" ]; then
  echo "mermaid already present: $dest"
  exit 0
fi

mkdir -p "$(dirname "$dest")"
echo "fetching mermaid@$version"
curl -fsSL -o "$dest" "https://unpkg.com/mermaid@$version/dist/mermaid.min.js"

# The bundle ends by assigning globalThis.mermaid; without that line Material
# would decide mermaid is missing and fetch it from the CDN anyway.
if ! tail -c 200 "$dest" | grep -q 'globalThis\["mermaid"\]'; then
  echo "downloaded bundle does not set globalThis.mermaid — refusing it" >&2
  exit 1
fi

echo "wrote $dest"
