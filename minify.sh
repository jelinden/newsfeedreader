#!/usr/bin/env bash
# Regenerates the fingerprinted css/js artifacts and manifest.json.
#
# The actual work lives in cmd/assets (Go, tdewolff/minify): it concatenates the
# sources listed in assets.json, drops unused css selectors, minifies, names the
# output by content hash and rewrites manifest.json. No Java or Node required.
#
#   ./minify.sh              build
#   ./minify.sh -check       verify committed artifacts without writing (CI)
#   ./minify.sh -purge=false keep every css selector
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

exec go run -mod=vendor ./cmd/assets "$@"
