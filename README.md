# Newsfeedreader

A small RSS news aggregator (fi/en) built with [echo](https://echo.labstack.com) and MongoDB.

## Prerequisites

- Go 1.26+
- MongoDB (with data populated by the rss fetcher)

No other toolchain is needed: css and js are minified by `cmd/assets`, which is
part of the module.

Set the database credentials in the environment (see `mongo-url.sh`), either
directly:

```sh
export MONGO_URL="mongodb://user:password@127.0.0.1:27017/news"
```

or as `MONGO_USER` and `MONGO_PASSWORD`, in which case `MONGO_HOST`
(default `192.168.0.1`) and `MONGO_PORT` (default `27017`) are also honoured.

The app exits at startup if the database is unreachable.

## Build and run

```sh
go build -mod=vendor && ./newsfeedreader
```

Then open http://localhost:1300/fi

Two scripts wrap that flow:

- `start-local.sh` rebuilds the assets, builds and runs in the foreground.
- `start-prod.sh` is the production entry point: same build steps, then it
  restarts the detached process and polls `http://127.0.0.1:1300/fi` until the
  app answers, exiting non-zero and tailing `news.log` when it never does.
  Assets and the binary are built *before* the running process is stopped, so a
  compile error leaves the old instance serving.

`docker.sh` builds and runs the container image.

## Assets

Templates reference `public/css/uutispuro.css` and `public/js/uutispuro.js`; the
manifest maps those names to fingerprinted files that are built from the sources
listed in `assets.json`:

```sh
./minify.sh              # purge unused css, minify, rewrite manifest.json
./minify.sh -check       # fail if the committed artifacts are out of date
./minify.sh -purge=false # keep every css selector
```

Unused css selectors are dropped by scanning `public/html` and `public/js` for
class and id names. The match is deliberately loose (a selector survives if any
of its names appears anywhere) and the build refuses to publish a stylesheet
that lost most of its bytes, so a bad purge shows up as a failed command instead
of an unstyled site.

## Test

```sh
go test -race -mod=vendor ./...
```
