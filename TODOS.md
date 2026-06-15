# nght TODOS

Deferred from v0.0.3 (container distribution milestone). Each item was considered in the /plan-eng-review on 2026-06-15 and explicitly parked for a future milestone.

---

## 1. chart `pullPolicy: Always` opt-in for production users

**What:** Add a documented override in `charts/nght/values.yaml` so production users can enable `pullPolicy: Always` without hand-editing the deployment.

**Why:** Default `IfNotPresent` is fine for dev, but production users expect pods to always pull the latest image at the same tag — particularly when `image.tag` is overridden to a non-AppVersion value (e.g., a SHA-based tag). Currently this requires the user to pass `--set 'image.pullPolicy=Always'` and know to do so.

**Pros:**
- One-line `values.yaml` comment with example `helm install` command
- Prevents the "we deployed the fix but pods are still on the old image" failure mode
- Standard pattern from Bitnami / ingress-nginx / other popular charts

**Cons:**
- None at the code level — it's a doc/comment change
- Doesn't break v0.0.3 minimal-chart scope (it's just a comment)

**Context:** Helm install of nght v0.0.3 with default `IfNotPresent` works fine for the first deploy. But if a user does `helm upgrade nght ... --set image.tag=0.0.4` while the previous image is still on the node, kubelet may serve the cached 0.0.3 image. Production users won't tolerate this. The fix is a single comment in `values.yaml` showing the override command.

**Depends on:** None. v0.0.3 ships.

---

## 2. `/metrics` Prometheus endpoint

**What:** Expose a `/metrics` endpoint with Prometheus-format text exposing `nght_requests_total{path, status_code}` and `nght_response_time_seconds_bucket{path, le}` counters.

**Why:** SREs running nght behind nginx need observability — currently they have to grep nginx logs to see traffic patterns. A `/metrics` endpoint + PodMonitor (Prometheus Operator) is the standard k8s observability hook. The README's `path B` roadmap calls this out explicitly.

**Pros:**
- Adopts the k8s-standard observability primitive (vs. log scraping)
- Enables alerts on nght failure-rate or latency percentiles
- Promotes nght from "test rig" to "ops-grade test rig" — fits the maintainer position
- Existing `fiber` middleware stack makes this low-cost (`github.com/ansrivas/fiber-prometheus` or hand-rolled)

**Cons:**
- Adds a Go dependency (Prometheus client library) — but it's a tiny one
- ~50-100 LOC of handler + middleware
- Slightly more surface area for the binary

**Context:** Fiber's middleware ecosystem has a Prometheus adapter; gin has its own. Both follow the same `prometheus.NewCounterVec` + middleware patterns. Implementation: middleware that observes request status + path, registers counters/histograms, exposes them at `/metrics`. Skeleton is 30 lines; counter/histogram registration is another 20. Test: 1 unit test confirming the endpoint returns Prometheus text format. Estimated effort: 1-2 days, 1 PR.

**Depends on:** Prometheus Operator (or any Prometheus scrape config) at the deployment site. Documentation must explain the PodMonitor CR.

---

## 3. Dynamic path API (`POST /admin/route`) for v0.0.4

**What:** Add `POST /admin/route` and `DELETE /admin/route/{path}` endpoints with admin token auth. Each registered route carries its own status code, latency, and health behavior — all hot-reloadable without nght restart.

**Why:** Currently nght's behavior is fixed at compile time (the 11 endpoints in `routes.go`). A user wanting "an endpoint that returns 503 for the next 60 seconds" has to do a code change + redeploy. Dynamic routing lets SREs configure the test rig from the control plane, no rebuild needed. README's `path C` roadmap calls this out as item #1.

**Pros:**
- Eliminates the "I need one more endpoint, let me PR" friction
- Enables ad-hoc test scenarios without code deploys (timing-sensitive ones, in particular)
- Sets up nght to be a true "nginx test rig" rather than a fixed set of behaviors
- Concurrent-safe in-memory routing table with RWMutex (Go-idiomatic)

**Cons:**
- ~200-400 LOC of new code (admin auth, route table, registration, deletion, listing)
- Admin token must be configurable (env var, ConfigMap, or k8s Secret)
- New attack surface — any endpoint reachable through dynamic routes inherits the same auth model
- Behavior must be tested under restart (in-memory table = no persistence)

**Context:** Design the route table as `map[string]RouteConfig` with `RouteConfig = {StatusCode int, LatencyMs int, HealthFlip bool}`. Endpoints can be queried via `GET /admin/routes`. Auth: `X-Admin-Token` header checked against `NGHT_ADMIN_TOKEN` env var. Routes are NOT persisted — restart = fresh state. Worth a separate /plan-eng-review session before implementation.

**Depends on:** v0.0.3 ships. This is a standalone PR for v0.0.4.
