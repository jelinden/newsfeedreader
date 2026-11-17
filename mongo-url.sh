# Sourced by start-local.sh and start-prod.sh. Resolves MONGO_URL from either
# MONGO_URL itself or MONGO_USER/MONGO_PASSWORD, and stops with a readable
# message instead of a bare "unbound variable" from set -u.
if [ -z "${MONGO_URL:-}" ]; then
  if [ -z "${MONGO_USER:-}" ] || [ -z "${MONGO_PASSWORD:-}" ]; then
    echo "newsfeedreader: set MONGO_URL, or MONGO_USER and MONGO_PASSWORD" >&2
    return 1
  fi
  MONGO_URL="mongodb://$MONGO_USER:$MONGO_PASSWORD@${MONGO_HOST:-192.168.0.1}:${MONGO_PORT:-27017}/news"
fi
export MONGO_URL
