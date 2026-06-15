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

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.3
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
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.3
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
- **path B** *(in progress — v0.0.3 ships multi-arch GHCR + minimal Helm chart)*: GitHub Container Registry Docker images, Helm chart, prometheus `/metrics`
- **path C**: dynamic path API (`POST /admin/route`), Web UI control panel, HTTP/3 + QUIC, record/replay

## License

See [LICENSE](./LICENSE).

## Python parity

A small FastAPI mirror of the core endpoints lives in `nght.py` for cases where you need a Python equivalent:

```bash
uvicorn nght:app --host 0.0.0.0 --port 8000 --reload
```
