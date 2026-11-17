#!/bin/bash
# Production entry point: rebuild assets and the binary, then restart the
# running app. Required environment variables are described in mongo-url.sh.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=mongo-url.sh
. ./mongo-url.sh

LOG=news.log
# Mirrors listenAddr in newsfeedreader.go.
HEALTH_URL="http://127.0.0.1:1300/fi"

bash minify.sh

# Both steps before this line may fail; with set -e the script then stops
# *before* the old process is killed, so a broken build cannot take the site
# down.
go build -mod=vendor -o newsfeedreader .

pkill -x newsfeedreader 2>/dev/null || true
sleep 1

nohup ./newsfeedreader > "$LOG" 2>&1 &
pid=$!

# A live pid proves nothing on its own: the app exits by itself when mongo is
# unreachable or manifest.json is broken, which can happen just after startup.
# Ask the process and the port instead of sleeping blindly.
is_up() {
  if command -v curl >/dev/null 2>&1; then
    curl -sf -o /dev/null "$HEALTH_URL"
  else
    kill -0 "$pid" 2>/dev/null
  fi
}

for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
  if is_up; then
    echo "newsfeedreader running (pid $pid, $HEALTH_URL)"
    exit 0
  fi
  kill -0 "$pid" 2>/dev/null || break # died during startup
  sleep 0.5
done

echo "start-prod: $HEALTH_URL did not answer; tail of $LOG:" >&2
tail -20 "$LOG" >&2 || true
exit 1
