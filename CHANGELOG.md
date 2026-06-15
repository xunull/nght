# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.4] — 2026-06-15

SRE 临时压测 nginx 配置:运行时动态路由 API,无需重启 nght。

### Added

- **Dynamic-route API** at `POST /admin/routes`, `GET /admin/routes`, `DELETE /admin/routes/<path>` (fiber engine only). SRE can register a route with a custom `status_code` and `latency_ms` at runtime, hit it immediately for nginx config validation, and delete it when done — no rebuild, no restart, no PR.
- In-memory `sync.RWMutex`-guarded `RouteTable` (no persistence; restart = fresh state, by design).
- `NGHT_ADMIN_TOKEN` opt-in auth middleware: unset = admin API fully open; set = requests must include matching `X-Admin-Token` header. Constant-time compare, no trim, startup warning if the secret contains whitespace.
- Single-source-of-truth `MarkReserved` on every hardcoded fiber path so dynamic `Register` of an existing endpoint returns 409 Conflict. Both exact and prefix reservations are supported.
- `helm template` and helm install now plumb `NGHT_ADMIN_TOKEN` through the chart via `values.adminToken` (default `""`).

### Changed

- `internal/fiber_server/routes.go`: the catch-all `app.All("*", ...)` is now preceded by a dynamic-dispatch `app.Use` middleware that consults the admin table and falls through to the wildcard only on miss.
- `cmd/server.go` reads `NGHT_ADMIN_TOKEN` from the env and warns on whitespace at startup. `fiber_server.Serve` signature gains an `adminToken` parameter; `gin_server.Serve` is unchanged (admin API is fiber-only).
- README and CHANGELOG add a "Dynamic route API" section (EN + ZH) plus NetworkPolicy advice for production deployments.

## [v0.0.3] — 2026-06-15

Container distribution milestone: nght 第一次有官方容器镜像和 Helm chart。

### Added

- `/livez` endpoint — 永真 liveness probe, 不受 `/health/false` 翻转
- Multi-arch container image on `ghcr.io/xunull/nght` (`linux/amd64` + `linux/arm64`)
- Minimal Helm chart at `charts/nght/` + OCI distribution at `oci://ghcr.io/xunull/charts/nght`
- Release workflow (`release.yml`) extended with buildx multi-arch build + chart-releaser

### Changed

- Dockerfile switched from `ubuntu:22.04` to `gcr.io/distroless/static-debian12:nonroot` — image size ~100MB → ~25MB
- CGO disabled in container build (fasthttp limitation)
- README adds "Container images" and "Kubernetes / Helm" sections (EN + ZH)

### Removed

- Legacy `.github/workflows/docker-publish.yml` (single-arch Docker Hub push, Oct 2024, superseded by multi-arch GHCR)

## [v0.0.2] — 2026-06-15

Cleanup pass. Wires up every CLI flag that was previously dead, makes every
documented endpoint behave as advertised, adds unit + handler tests from
zero, ships multi-platform binaries via goreleaser, and rewrites the README
in English + Chinese with endpoint reference and nginx use-case recipes.

### Added
- `nght --version` subcommand with `ldflags`-injected version string
- `--response-json` flag now actually controls the fiber engine's JSON vs text output (was dead code before)
- `README.md` (English) + `README.zh.md` (Chinese) with full endpoint reference, gin/fiber compatibility matrix, and 3 worked nginx use-case recipes
- Unit tests for `internal/utils.SplitStatus` covering empty / single / multi / zero-status / non-digit / partial / bad-length inputs
- `httptest`-based coverage for every fiber handler, including the JSON toggle, the `NGHT-Hostname` middleware, and wildcard fallback
- Regression tests for the four behavioral bugs fixed in this release (see _Fixed_ below)
- `.github/workflows/ci.yml` — vet + gofmt + race-enabled tests on linux/macOS/windows + cross-platform build matrix on every PR
- `.github/workflows/release.yml` + `.goreleaser.yaml` — push a `v*` tag to publish darwin/linux/windows × amd64/arm64 binaries to GitHub Releases
- `Makefile` with `build / test / vet / fmt / fmt-check / clean / release / release-snapshot` targets

### Fixed
- `internal/fiber_server.ResponseTimeResp` — sleep was nanoseconds, not seconds. `time.Duration(t) * time.Second`. Regression test included.
- `internal/fiber_server.StatusResp` — text branch was returning HTTP 200 regardless of the `:status` path param. Now `c.Status(status).SendString(...)`. Test included.
- `internal/fiber_server.HealthRandomResp` — was unconditionally returning UP, ignoring `:percentage`. Now respects the param. Distribution-based regression test included.
- `internal/fiber_server.LogReqData` — passed the user-supplied body to `log.Infof` as the format string (log injection). Now `log.Infof("%s", body)`. Crash-resistance test included.
- `internal/fiber_server.api` — two `fmt.Sprintf("%s", intArg)` calls flagged by `go vet`; corrected to `%d`.
- `internal/utils.SplitStatus("")` no longer silently returns `([]int{}, nil)` (which would crash callers via `status[0]`). Returns a proper error.
- `internal/gin_server` — removed deprecated per-request `rand.Seed(time.Now().UnixNano())` calls (Go 1.20+ auto-seeds the global RNG).

### Removed
- Empty placeholder `client.go` at repo root (the `cmd/client.go` cobra stub remains as the parent of the future `nght client` subcommand).
- Unused `--echo-hostname` CLI flag (was parsed but never read).

### Changed
- Empty `Makefile` is now populated.
- `.gitignore` now ignores the local `nght` binary and `dist/` (goreleaser output).
- Root command short/long description widened from "a gin web server" to "a gin/fiber web server" to match reality.

### Notes
- Both gin and fiber engines remain shipped. Eng-review session decided to keep both for dual-engine A/B comparison (run gin on `:8080` and fiber on `:8081` to see framework-level behavior diffs).
- `nght client` is still a placeholder; full load-test client (status distribution + p50/p99) is deferred.

[v0.0.2]: https://github.com/xunull/nght/releases/tag/v0.0.2
[v0.0.3]: https://github.com/xunull/nght/releases/tag/v0.0.3
