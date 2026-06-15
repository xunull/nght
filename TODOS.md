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

## 3. Dynamic path API (`POST /admin/routes`) for v0.0.4

**Status:** **IN PROGRESS** — user 2026-06-15 决定做(从最初的"roadmap 一句话 + AI 推测的 deferred 设计"升级为真实需求),design doc 写好 + plan-eng-review 完,等实现。

**Design doc:** `~/.gstack/projects/nght/quincy-master-design-20260615-212204.md` (15.8KB)

**Scope locked:**
- SRE 临时压测 nginx config 的真实场景
- `/admin/routes` (POST register / DELETE unregister / GET list) 端点
- `NGHT_ADMIN_TOKEN` opt-in auth (没设 = 全开放,设了 = 必须 X-Admin-Token header)
- 路由表 in-memory + sync.RWMutex,无持久化
- 只 fiber 引擎实现
- reserved path list 通过 `MarkReserved(path)` 集中维护,fiber routes.go 调它
- wildcard 改 middleware,先查 dynamic table,miss 才 fallback
- constant-time token 比对,不 trim,exact match,拒绝带 whitespace 的 token
- 动态路由 health 独立,不动 process-global /health
- helm chart values.yaml 加 `adminToken: ""` 默认空,k8s Secret 透传

**Implementation tasks:** 见 design doc 末尾的 "## GSTACK REVIEW REPORT" → Implementation Tasks

**Original 200-400 LOC estimate** holds, but only 3 core code files (table/middleware/fiber) + 3 tests + 2 wiring + 3 docs.

---

## Original item (now superseded by 3. v0.0.4)

This was originally deferred from v0.0.3, but the user has confirmed intent to implement on 2026-06-15. The original "AI-only drafted" version is below for historical reference; the v0.0.4 entry above is the source of truth.

**What (original):** Add `POST /admin/route` and `DELETE /admin/route/{path}` endpoints with admin token auth.
