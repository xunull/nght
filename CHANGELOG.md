# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
