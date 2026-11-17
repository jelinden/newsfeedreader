#!/bin/bash
set -euo pipefail
CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -ldflags='-s' -o newsfeedreader .
docker build -t default:newsfeedreader .
docker run -it -p 1300:1300 -d default:newsfeedreader
