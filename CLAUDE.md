# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A small Go HTTP server that fronts a single GCS bucket, mainly as the backend for Kaltura's nginx-vod-module (in "mapped" mode). This is a fork of the archived NYTimes/gcs-helper. The Go module path is still `github.com/NYTimes/gcs-helper/v3`, so internal imports use that path, not `gcs-proxy`.

## Commands

```sh
go build -o gcs-proxy .                                   # binary; version comes from -ldflags "-X main.version=..."
go test -race ./...                                       # all tests (no GCS credentials needed)
go test ./handlers -run TestProxyHandler                  # single test
go test ./vodmodule -run TestMap -v
make docker-build                                         # docker build tagged <branch>-<sha>
```


Running locally requires `GCS_HELPER_BUCKET_NAME` plus Google Application Default Credentials (the HTTP client asks for the `storage.ScopeReadOnly` scope).

## Architecture

Startup in `main.go`: start the gops agent → `handlers.LoadConfig()` → `ClientConfig.HTTPClient()`, an authenticated `*http.Client` used by both handlers → `getHandler` in `server.go`.

Routing (`server.go`) checks path prefixes, in this order:
1. `GCS_HELPER_PROXY_PREFIX`: strips the prefix (keeping a leading `/`) and sends the request to `handlers.Proxy`.
2. `GCS_HELPER_MAP_PREFIX`: strips the prefix and sends the request to `handlers.Map`.
3. `/`: returns 200 as a health check.
4. Anything else returns 404.

The prefixes default to empty, and `strings.HasPrefix(path, "")` is always true. So when no proxy prefix is set, every request goes to the proxy handler and the map handler can't be reached.

- **Proxy** (`handlers/proxy.go`): accepts only GET and HEAD. It rewrites the request to `https://<bucket>.storage.googleapis.com/<path>`, or to path-style `storage.googleapis.com/<path>` when `GCS_HELPER_PROXY_BUCKET_ON_PATH=true`. It copies request and response headers both ways and streams the body. `GCS_HELPER_PROXY_TIMEOUT` sets a context deadline over the whole request, and the client's own `GCS_CLIENT_TIMEOUT` limits each upstream call. `GCS_HELPER_ROOT_REDIRECT` rewrites `/` to a fixed object path.
- **Map** (`handlers/map.go` → `vodmodule.Mapper`): lists objects under the request path as a prefix, using the `/` delimiter so the listing isn't recursive. `GCS_HELPER_MAP_REGEX_FILTER` is matched against each object's basename. The handler returns the nginx-vod-module mapping JSON (`vodmodule/types.go`), with one sequence per object and clip paths of the form `/<bucket>/<object>`. A failed listing is retried up to `maxTries` (5) times.

### Configuration

`LoadConfig` calls `envconfig.Process("gcs_helper", ...)`. Untagged or short-tagged fields get the `GCS_HELPER_` prefix (`Listen` → `GCS_HELPER_LISTEN`). Fields in the nested structs carry full env var names as tags (`GCS_CLIENT_TIMEOUT`, `GCS_HELPER_MAP_PREFIX`), and they only work through envconfig's fallback to the alternate name. Follow the same pattern when adding a setting. Then add it to the README tables and to the expectations in `handlers/config_test.go`, which compares the whole struct with `reflect.DeepEqual`.

### Tests

Tests use `fake-gcs-server` (`fakestorage`) with `NoListener: true`. `server.HTTPClient()` intercepts requests to any host, so the proxy handler's hardcoded `storage.googleapis.com` URLs reach the fake server, and `server.Client()` backs the map handler. Shared fixtures (`FakeObjects` in bucket `my-bucket`; set `BucketName`/`Name` inside an explicit `ObjectAttrs:` literal, since `go.mod`'s language version doesn't allow promoted fields in composite literals) and the table-driven `testhelper.ServerTest` runner are in `internal/testhelper`. The config tests call `os.Clearenv()`.

## CI / release

- `.github/workflows/docker-publish-multiarch.yml` calls Docker's reusable `docker/github-builder` `build.yml` workflow. It builds amd64 and arm64 on native runners, merges them into one multi-arch image, signs the provenance and pushes to Google Artifact Registry, logging in through the builder's `gcp-wif` registry identity (Workload Identity Federation). Inputs to the reusable workflow can't use the `env` context, so the registry paths come straight from `vars.SHARED_WIF_*` (`SHARED_WIF_LOCATON` really is spelled that way). A separate `scan` job runs Trivy on the pushed digest for push events. The workflow runs on pushes and PRs to `master`, on `v*.*.*` tags and on manual dispatch. PR images are pushed as well, except from forks. The floating `main` image tag follows `master`.
- `codeql-analysis.yml` runs on `master`. No GitHub workflow runs `go test`.
- `.drone.jsonnet` / `.drone.yml` are left over from the upstream NYTimes project (Go 1.12, Docker Hub `nytimes/gcs-helper`) and aren't used.
- Dependabot checks Go modules, GitHub Actions and the Dockerfile base images daily. Actions are pinned to commit SHAs with a version comment.
