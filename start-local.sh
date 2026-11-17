#!/bin/bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=mongo-url.sh
. ./mongo-url.sh

bash minify.sh
go build -mod=vendor -o newsfeedreader .
exec ./newsfeedreader
