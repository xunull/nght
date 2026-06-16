# nght

> nginx-gin-http-test — a tiny, single-binary HTTP server with controllable behavior, built for stress-testing nginx (and any HTTP gateway) against arbitrary status codes, latency, and health states.

[中文版 / Chinese](./README.zh.md)

## Why

When you debug nginx upstream behavior — retry triggers, health probe windows, `proxy_next_upstream`, timeout layers — you need a "compliant" backend that does exactly what you ask. `nght` is that backend:

- Return any status code you want via `/status/418`
- Sleep N seconds before replying via `/response_time/3`
- Flip "healthy" → "unhealthy" without restart via `/health/false` and `/health/true`
- Roll dice on the status code (`/random/200502503`) or on the failure rate (`/random_crash/30/500502`)

Single Go binary. Zero runtime deps. Two engines (gin & fiber) bundled, switchable via `--type`.

## nght client — load-test from the same binary

As of v0.0.5 the same `nght` binary also ships a `client` subcommand that load-tests any HTTP target and reports p50/p95/p99 latency plus a status code histogram. No need to install `wrk` / `vegeta` / `ab` to drive nght — the client is in the box.

```bash
# Quick smoke against local nght server: 10 workers, 5 seconds, JSON output
nght client -H 127.0.0.1 -p 8080 -c 10 -d 5s --output json

# Drive nght to 1000 RPS against an upstream behind nginx (validate retry)
nght client -H nginx -p 80 -c 50 -d 30s --rps 1000

# Stop after exactly 10000 requests, plain text report
nght client -H nght-svc -p 8080 -c 20 -n 10000

# Combine: 30s OR 5000 requests, whichever first
nght client -H 127.0.0.1 -p 8080 -c 10 -d 30s -n 5000
```

Flags:

| Flag | Short | Default | Meaning |
|------|-------|---------|---------|
| `--host` | `-H` | `127.0.0.1` | target host (capital `-H` because cobra reserves `-h` for `--help`) |
| `--port` | `-p` | `8080` | target port |
| `--concurrency` | `-c` | `10` | number of concurrent workers |
| `--duration` | `-d` | `10s` | test wall-clock duration; `0` = unlimited (requires `--total`) |
| `--total` | `-n` | `0` | stop after N total requests; `0` = unlimited (requires `--duration`) |
| `--rps` | `-q` | `0` | cap requests per second; `0` = full speed |
| `--output` | `-o` | `text` | report format: `text` or `json` |
| `--timeout` | — | `5s` | per-request timeout |
| `--path` | — | `/` | URL path to hit (must start with `/`) |

Stop semantics: **2 stop triggers** (duration + total), OR — the test ends as soon as either is reached. `--rps` is a pacing cap, not a stop condition. If you pass `--duration 0` and `--total 0` together, the client refuses to start (otherwise it would run forever).

Sample text report:

```
Target:        127.0.0.1:8080/echo/hello
Concurrency:   10
Duration:      5s (actual 5000 ms)

Total requests: 87421
Actual RPS:     17484.20
Errors:         0

Latency (ms):
  p50 = 0.21  p95 = 0.45  p99 = 0.78  min = 0.05  max = 4.92  samples = 10000

Status histogram:
  200             87421
```

Sample JSON report (`--output json`):

```json
{
  "config": { "host":"127.0.0.1","port":8080,"concurrency":10,"duration":"5s","total":0,"rps":0,"timeout":"5s","path":"/echo/hello" },
  "summary": { "total_requests":87421,"actual_rps":17484.20,"duration_ms":5000,"errors":0 },
  "latency_ms": { "p50":0.21,"p95":0.45,"p99":0.78,"min":0.05,"max":4.92,"samples":10000 },
  "status_histogram": { "200":87421 }
}
```

Status taxonomy: HTTP responses land in their numeric bucket (`200`, `404`, `500`, …); per-request timeouts land in `"timeout"`; connection refused / DNS failure / EOF land in `"network_error"`. Any `>= 400` HTTP code, plus both error buckets, count toward `errors`.

Percentiles use reservoir sampling (Algorithm R) over at most 10,000 latency samples, so a 1M-request test reports p99 with roughly ±1% standard error while keeping the aggregator's memory bounded.

### SRE workflow: pair client + dynamic route (v0.0.4)

```bash
# Terminal 1: fire client
nght client -H nght-svc -p 8080 -c 20 -d 30s --path /api/inject --output json

# Terminal 2: inject a 5xx mid-test
curl -X POST http://nght-svc:8080/admin/routes \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/api/inject","status_code":502,"latency_ms":3000}'

# ... 30s later, client report shows 502s + 3s p99 spike in the histogram
# Then clean up:
curl -X DELETE http://nght-svc:8080/admin/routes/api/inject \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

## Install

```bash
go install github.com/xunull/nght@latest
```

Or download a pre-built binary from [Releases](https://github.com/xunull/nght/releases) (darwin/linux/windows × amd64/arm64).

## Quick start

```bash
# default: gin engine on :8080
nght server

# fiber engine, JSON responses
nght server -t fiber -p 8080 --response-json

# version
nght --version

# Docker
docker build -t nght . && docker run --rm -p 8080:8080 nght
```

## Endpoint reference

| Path | Behavior | Example |
|------|----------|---------|
| `/echo/:text` | Echo the text. Response also carries hostname. | `curl :8080/echo/hello` |
| `/echo_url` | Echo the request URL + hostname. | `curl :8080/echo_url` |
| `/echo_header` | Echo all request headers (fiber only). | `curl :8080/echo_header -H 'X-Foo: bar'` |
| `/log_req_data` | Log the body server-side, return 200 (fiber only). | `curl -d 'payload' :8080/log_req_data` |
| `/status/:status` | Return the status code in the path. | `curl -i :8080/status/502` |
| `/response_time/:time` | Sleep N seconds, then return 200. | `time curl :8080/response_time/3` |
| `/random/:statusRandom` | Return a random status from the 3-char-grouped list. | `curl :8080/random/200502503` |
| `/random_crash/:percentage/:statusRandom` | With N% chance, return 200; otherwise a random failure. | `curl :8080/random_crash/30/500502` |
| `/health`, `/healthz` | Return 200 if "up", 502 if "down". | `curl :8080/health` |
| `/health/true` | Flip "up" — subsequent `/health` returns 200. | `curl :8080/health/true` |
| `/health/false` | Flip "down" — subsequent `/health` returns 502. | `curl :8080/health/false` |
| `/health/random/:percentage` | With N% chance return 200, else 502. | `curl :8080/health/random/30` |
| `/livez` | k8s liveness probe — always 200, independent of `/health` state. | `curl :8080/livez` |

## Dynamic route API (admin, fiber only)

The `/admin/routes` endpoints let you register an arbitrary path at runtime and have nght respond with a specific status code and latency — useful for pressure-testing nginx config or fault-injecting on demand, without rebuilding the binary.

| Method | Path | Body / Param | Behavior |
|--------|------|--------------|----------|
| `POST` | `/admin/routes` | `{"path":"/api/timeout","status_code":504,"latency_ms":3000}` | Register a dynamic route. Returns 201 on success, 400 on validation failure or reserved path. |
| `GET` | `/admin/routes` | — | List all currently-registered routes. |
| `DELETE` | `/admin/routes/<path>` | path is the full URL suffix, e.g. `/admin/routes/api/timeout` | Unregister. Always returns 204 (idempotent). |

Dynamic routes are **fiber-engine only** (Gin has no admin support, by design). They are stored in memory only; restart clears them. Registered paths cannot collide with the 11 hardcoded endpoints above — a `Register` of a hardcoded path or any path under a reserved prefix returns 400 Conflict. (The reserved paths are: `/echo`, `/echo_header`, `/echo_url`, `/status`, `/log_req_data`, `/response_time`, `/random`, `/random_crash`, `/healthz`, `/livez`, plus prefix reservations `/echo/`, `/status/`, `/response_time/`, `/random/`, `/random_crash/`, `/health/`.)

**Auth (`NGHT_ADMIN_TOKEN`):** opt-in. If the env var is unset, `/admin/*` is fully open — safe only if you restrict access via NetworkPolicy or `listen` address. If set, every request to `/admin/*` must include `X-Admin-Token: <value>`. Mismatch returns 401. The comparison is constant-time; the value is matched byte-for-byte (no trim) — if the secret in your env has stray whitespace, every admin request will silently 401 (the server warns at startup if it detects whitespace in `NGHT_ADMIN_TOKEN`).

Example session:

```bash
# Register a route that returns 504 after 3s
curl -X POST http://nght:8080/admin/routes \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Token: mysecret' \
  -d '{"path":"/api/timeout","status_code":504,"latency_ms":3000}'

# Hit it (SRE scenario: validate nginx retry / proxy_next_upstream)
time curl -i http://nght:8080/api/timeout

# Clean up
curl -X DELETE http://nght:8080/admin/routes/api/timeout \
  -H 'X-Admin-Token: mysecret'
```

In Kubernetes, set `adminToken` in your values (see [Kubernetes / Helm](#kubernetes--helm) below).

### gin vs fiber engine compatibility

`nght` ships two HTTP engines side by side. Use `--type gin` (default) or `--type fiber`.

| Endpoint | gin | fiber |
|----------|:---:|:---:|
| `/echo/:text` | ✓ | ✓ |
| `/echo_url` |   | ✓ |
| `/echo_header` |   | ✓ |
| `/log_req_data` |   | ✓ |
| `/status/:status` | ✓ | ✓ |
| `/response_time/:time` | ✓ | ✓ |
| `/random/:statusRandom` | ✓ | ✓ |
| `/random_crash/:percentage/:statusRandom` | ✓ | ✓ |
| `/health` (and `/healthz`) | ✓ | ✓ |
| `/health/true`, `/health/false` | ✓ | ✓ |
| `/health/random/:percentage` | ✓ | ✓ |
| `--response-json` flag wiring |   | ✓ |
| Wildcard `*` fallback (echo url) |   | ✓ |
| `NGHT-Hostname` response header |   | ✓ |
| `/livez` (k8s liveness probe, always 200) | ✓ | ✓ |

The fiber engine is the more feature-complete option and uses `valyala/fasthttp` under the hood. gin is kept around for dual-engine A/B comparison (run the same workload against `:8080` gin and `:8081` fiber to see framework-level behavior differences).

## nginx use-case recipes

### 1. Verify `proxy_next_upstream` falls through on 502

```nginx
upstream nght_pool {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081 backup;
}
location /api/ {
    proxy_next_upstream error timeout http_502;
    proxy_pass http://nght_pool;
}
```

```bash
# primary returns 502 100% of the time, backup returns 200
nght server -p 8080 -t fiber &
nght server -p 8081 -t fiber &
# now hit nginx; you should always see 200, not 502
curl -i http://nginx/api/status/502
```

### 2. Probe health window of nginx upstream healthcheck

```bash
nght server -p 8080 -t fiber &
# nginx is configured to mark "down" after 3 consecutive 502s in 5 seconds
curl http://nght:8080/health/false   # flip
# nginx should drop nght from rotation within the window
curl http://nght:8080/health/true    # flip back
# verify nginx re-admits nght
```

### 3. Stress-test connection / read timeouts

```nginx
proxy_connect_timeout 1s;
proxy_read_timeout    2s;
```

```bash
nght server -t fiber &
curl -i http://nginx/api/response_time/5   # should hit 504 from nginx, not 200 from nght
```

### 4. Fault-inject via k8s Deployment

For ad-hoc fault injection that doesn't require a code change: register a
dynamic route, hit it, delete it. No PR, no rebuild, no pod restart.

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4 \
  --set adminToken=$(openssl rand -hex 32)
kubectl port-forward svc/nght 8080:8080 &

# Inject a 502 that nginx will retry past
curl -X POST http://localhost:8080/admin/routes \
  -H 'Content-Type: application/json' \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -d '{"path":"/api/inject","status_code":502,"latency_ms":0}'
curl -i http://localhost:8080/api/inject

# Clean up
curl -X DELETE http://localhost:8080/admin/routes/api/inject \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

For persistent health-down behavior, the classic `/health/false` still works:

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4
kubectl port-forward svc/nght 8080:8080 &
curl http://localhost:8080/echo/hello
# flip health down — readinessProbe should mark NotReady, NO pod restart
curl -X POST http://localhost:8080/health/false
kubectl get pods -w
# livenessProbe uses /livez, so pod stays Ready for liveness
# but Service endpoints remove the pod (readinessProbe=NotReady)
```

## Build & test

```bash
make build         # produces ./nght with ldflags-injected version
make test          # go test ./...
make vet           # go vet ./...
make fmt-check     # gofmt -l . (fail on diff)
make release       # goreleaser release --clean (tag-driven)
```

## Distribution

- Single-binary install via `go install` or downloadable binary from GitHub Releases.
- Cross-compiled for darwin/linux/windows × amd64/arm64 via [goreleaser](https://goreleaser.com/).
- Multi-stage Dockerfile (`golang:1.22-bookworm` → `ubuntu:22.04`) included; entrypoint defaults to fiber.

## Container images

```bash
docker pull ghcr.io/xunull/nght:0.0.3    # multi-arch manifest (linux/amd64 + linux/arm64)
docker run --rm -p 8080:8080 ghcr.io/xunull/nght:0.0.3
```

Image is `gcr.io/distroless/static-debian12:nonroot` (~25MB, nonroot UID 65532, no shell — `kubectl exec nght -- /bin/sh` will fail). `latest` tag tracks the most recent push. Pin to a specific tag in production.

## Kubernetes / Helm

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4
```

To opt in to admin-token auth, pass `adminToken` (32+ random bytes recommended):

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4 \
  --set adminToken=$(openssl rand -hex 32)
```

With `adminToken` unset, `/admin/*` is fully open. **In production, set it AND restrict `/admin/*` access via NetworkPolicy** — the chart does not enforce network-level isolation on its own. Example policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nght-admin-lockdown
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: nght
  ingress:
  - from:
    - podSelector: {}                # allow from any pod in the namespace
    - namespaceSelector: {}           # adjust to your SRE namespace
    ports:
    - port: 8080
      protocol: TCP
```

The chart ships a `Deployment` + `Service` only (no probes, no securityContext, no resources). For production, override with `livenessProbe` and `readinessProbe`:

```yaml
livenessProbe:
  httpGet: { path: /livez, port: 8080 }
readinessProbe:
  httpGet: { path: /health, port: 8080 }
```

**Use `/livez` for liveness** — flipping `/health/false` to take a backend out of rotation must NOT restart the pod. **Use `/health` for readiness** so traffic stops flowing to a manually-degraded pod without killing it.

## Roadmap

See the [project office-hours design doc](https://github.com/xunull/nght) for full roadmap. Short list:

- **near-term**: real `nght client` load test subcommand (currently a stub), refactor fiber package-level state into struct fields, more nginx recipes
- **path B** *(v0.0.3 ships multi-arch GHCR + minimal Helm chart; v0.0.4 ships the dynamic-route API)*: GitHub Container Registry Docker images, Helm chart, prometheus `/metrics`
- **path C**: Web UI control panel, HTTP/3 + QUIC, record/replay

## License

See [LICENSE](./LICENSE).

## Python parity

A small FastAPI mirror of the core endpoints lives in `nght.py` for cases where you need a Python equivalent:

```bash
uvicorn nght:app --host 0.0.0.0 --port 8000 --reload
```
