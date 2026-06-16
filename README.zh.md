# nght

> nginx-gin-http-test —— 一个 Go 单二进制的 HTTP 测试目标，专门给 nginx（及任意 HTTP 网关）做可控行为陪练：任意状态码、可控延迟、可手动切换的健康状态。

[English version](./README.md)

## 为什么用

调 nginx upstream 行为时 —— 重试触发、健康探测窗口、`proxy_next_upstream`、超时层级 —— 你需要一个"听话"的下游，让它干啥就干啥。`nght` 就是这个下游：

- `/status/418` 返回任意状态码
- `/response_time/3` 睡 N 秒再返
- `/health/false` 和 `/health/true` 不重启切换"健康"
- `/random/200502503` 在多个状态码间随机；`/random_crash/30/500502` 按概率乱码

Go 单二进制，零运行时依赖。内含 gin 和 fiber 两套引擎，`--type` 切换。

## nght client —— 同 binary 自带的压测客户端

v0.0.5 起，`nght` 同一个二进制多了 `client` 子命令，可对任意 HTTP target 做压测并报 p50/p95/p99 延迟 + 状态码直方图。不用额外装 `wrk` / `vegeta` / `ab` 就能压 nght。

```bash
# 5 秒压本地 nght，JSON 报告
nght client -H 127.0.0.1 -p 8080 -c 10 -d 5s --output json

# 1000 RPS 上限压 nginx 上游
nght client -H nginx -p 80 -c 50 -d 30s --rps 1000

# 累计 10000 个请求后停
nght client -H nght-svc -p 8080 -c 20 -n 10000

# 30 秒 或 5000 个请求，先到先停
nght client -H 127.0.0.1 -p 8080 -c 10 -d 30s -n 5000
```

参数表：

| 参数 | 短选项 | 默认 | 含义 |
|------|--------|------|------|
| `--host` | `-H` | `127.0.0.1` | 目标 host（短选项是大写 `-H`，cobra 占用 `-h` 给 `--help`） |
| `--port` | `-p` | `8080` | 目标端口 |
| `--concurrency` | `-c` | `10` | 并发 worker 数 |
| `--duration` | `-d` | `10s` | 测试时长；`0` = 无限（需要 `--total`） |
| `--total` | `-n` | `0` | 跑满 N 个请求后停；`0` = 无限（需要 `--duration`） |
| `--rps` | `-q` | `0` | 上限 RPS；`0` = 满速 |
| `--output` | `-o` | `text` | 报告格式：`text` 或 `json` |
| `--timeout` | — | `5s` | 单请求超时 |
| `--path` | — | `/` | URL path（必须以 `/` 开头） |

停止语义：**两个停止触发**（duration + total），OR 关系 —— 哪个先到就先停。`--rps` 是限速，不是停止条件。如果同时给 `--duration 0` 和 `--total 0`，客户端会直接 fatal 拒绝启动（否则会跑死循环）。

文本报告示例：

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

JSON 报告示例（`--output json`）：

```json
{
  "config": { "host":"127.0.0.1","port":8080,"concurrency":10,"duration":"5s","total":0,"rps":0,"timeout":"5s","path":"/echo/hello" },
  "summary": { "total_requests":87421,"actual_rps":17484.20,"duration_ms":5000,"errors":0 },
  "latency_ms": { "p50":0.21,"p95":0.45,"p99":0.78,"min":0.05,"max":4.92,"samples":10000 },
  "status_histogram": { "200":87421 }
}
```

状态码桶：HTTP 响应按数字分桶（`200`、`404`、`500` …）；单请求超时进 `"timeout"`；连接被拒 / DNS 失败 / EOF 进 `"network_error"`。所有 `>= 400` 的 HTTP 码 + 两个 error 桶都计入 `errors`。

百分位用 reservoir sampling（Algorithm R），最多保留 10000 个延迟样本。所以 100 万请求的测试报 p99 标准误约 ±1%，同时聚合器内存有上界。

### SRE 工作流：client + 动态路由（v0.0.4）配对

```bash
# Terminal 1: 跑 client
nght client -H nght-svc -p 8080 -c 20 -d 30s --path /api/inject --output json

# Terminal 2: 测试期间注入 5xx
curl -X POST http://nght-svc:8080/admin/routes \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/api/inject","status_code":502,"latency_ms":3000}'

# 30s 后 client 报告会显示 502 增加 + p99 飙升
# 然后清理：
curl -X DELETE http://nght-svc:8080/admin/routes/api/inject \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

## 安装

```bash
go install github.com/xunull/nght@latest
```

或从 [Releases](https://github.com/xunull/nght/releases) 下载预编译 binary（darwin/linux/windows × amd64/arm64）。

## 快速开始

```bash
# 默认 gin 引擎，:8080
nght server

# fiber 引擎，JSON 响应
nght server -t fiber -p 8080 --response-json

# 版本
nght --version

# Docker
docker build -t nght . && docker run --rm -p 8080:8080 nght
```

## 端点速查

| 路径 | 行为 | 示例 |
|------|------|------|
| `/echo/:text` | 回显 text，附带 hostname。 | `curl :8080/echo/hello` |
| `/echo_url` | 回显请求 URL 和 hostname。 | `curl :8080/echo_url` |
| `/echo_header` | 回显请求所有 header（仅 fiber）。 | `curl :8080/echo_header -H 'X-Foo: bar'` |
| `/log_req_data` | 服务端打 body 到日志，返 200（仅 fiber）。 | `curl -d 'payload' :8080/log_req_data` |
| `/status/:status` | 返回路径里写的状态码。 | `curl -i :8080/status/502` |
| `/response_time/:time` | 睡 N 秒再返 200。 | `time curl :8080/response_time/3` |
| `/random/:statusRandom` | 在 3 字符一组的状态码列表里随机返。 | `curl :8080/random/200502503` |
| `/random_crash/:percentage/:statusRandom` | N% 概率返 200，否则随机失败。 | `curl :8080/random_crash/30/500502` |
| `/health`, `/healthz` | "up" 状态 200，"down" 状态 502。 | `curl :8080/health` |
| `/health/true` | 切"up"，后续 `/health` 返 200。 | `curl :8080/health/true` |
| `/health/false` | 切"down"，后续 `/health` 返 502。 | `curl :8080/health/false` |
| `/health/random/:percentage` | N% 概率返 200，否则 502。 | `curl :8080/health/random/30` |
| `/livez` | k8s liveness probe —— 永真 200，独立于 `/health` 状态。 | `curl :8080/livez` |

## 动态路由 API（admin，仅 fiber）

`/admin/routes` 系列端点让你运行时注册任意 path，并让 nght 按指定 status code + latency 响应 —— 用于临时压测 nginx 配置或按需故障注入，**不用重构建**。

| 方法 | 路径 | Body / 参数 | 行为 |
|------|------|--------------|------|
| `POST` | `/admin/routes` | `{"path":"/api/timeout","status_code":504,"latency_ms":3000}` | 注册动态路由。成功 201，校验失败或命中保留路径 400。 |
| `GET` | `/admin/routes` | — | 列出当前注册的所有路由。 |
| `DELETE` | `/admin/routes/<path>` | path 是完整 URL 后缀，如 `/admin/routes/api/timeout` | 撤销。永远 204（幂等）。 |

动态路由**仅 fiber 引擎**支持（Gin 按设计没有 admin）。状态**只在内存**，进程重启清空。注册的 path 不能跟上面 11 个硬编码端点冲突 —— `Register` 命中硬编码 path 或其 prefix 保留时返 400 Conflict。保留路径包括：`/echo`、`/echo_header`、`/echo_url`、`/status`、`/log_req_data`、`/response_time`、`/random`、`/random_crash`、`/healthz`、`/livez`，外加 prefix 保留 `/echo/`、`/status/`、`/response_time/`、`/random/`、`/random_crash/`、`/health/`。

**鉴权（`NGHT_ADMIN_TOKEN`）：** opt-in。env var 未设时 `/admin/*` 完全开放 —— 只在你用 NetworkPolicy 或监听地址限制访问时安全。设了之后所有 `/admin/*` 请求必须带 `X-Admin-Token: <value>` header。比对是 constant-time，按字节比较（不 trim）—— 如果 env 里的 secret 带了意外空白，每个 admin 请求都会静默 401（启动时若检测到 secret 含空白会 warn）。

```bash
# 注册一个返 504 + 3s 延迟的路由
curl -X POST http://nght:8080/admin/routes \
  -H 'Content-Type: application/json' \
  -H 'X-Admin-Token: mysecret' \
  -d '{"path":"/api/timeout","status_code":504,"latency_ms":3000}'

# 打它（SRE 场景：验证 nginx retry / proxy_next_upstream）
time curl -i http://nght:8080/api/timeout

# 清理
curl -X DELETE http://nght:8080/admin/routes/api/timeout \
  -H 'X-Admin-Token: mysecret'
```

Kubernetes 里通过 chart 的 `adminToken` 值传入（见下方 [Kubernetes / Helm](#kubernetes--helm)）。

### gin vs fiber 引擎对比

`nght` 同时打包两套 HTTP 引擎，`--type gin`（默认）或 `--type fiber` 切换。

| 端点 | gin | fiber |
|------|:---:|:---:|
| `/echo/:text` | ✓ | ✓ |
| `/echo_url` |   | ✓ |
| `/echo_header` |   | ✓ |
| `/log_req_data` |   | ✓ |
| `/status/:status` | ✓ | ✓ |
| `/response_time/:time` | ✓ | ✓ |
| `/random/:statusRandom` | ✓ | ✓ |
| `/random_crash/:percentage/:statusRandom` | ✓ | ✓ |
| `/health` (含 `/healthz`) | ✓ | ✓ |
| `/health/true`, `/health/false` | ✓ | ✓ |
| `/health/random/:percentage` | ✓ | ✓ |
| `--response-json` flag 接通 |   | ✓ |
| 通配 `*` 回显 url |   | ✓ |
| `NGHT-Hostname` 响应头 |   | ✓ |
| `/livez` (k8s liveness probe, 永真 200) | ✓ | ✓ |

fiber 引擎更完整（底层用 `valyala/fasthttp`）。gin 保留作"双引擎对照测试"用 —— 同一负载同时打到 `:8080` (gin) 和 `:8081` (fiber)，对比框架级行为差异。

## nginx 场景配方

### 1. 验证 `proxy_next_upstream` 在 502 时回落

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
nght server -p 8080 -t fiber &  # 主节点
nght server -p 8081 -t fiber &  # 备节点
curl -i http://nginx/api/status/502   # 应该一直 200（nginx 自动落到 backup）
```

### 2. 探 nginx upstream 健康探测窗口

```bash
nght server -p 8080 -t fiber &
# 假设 nginx 配置：5 秒内 3 次 502 就摘除
curl http://nght:8080/health/false
# nginx 应在窗口内把 nght 摘除
curl http://nght:8080/health/true
# 验证 nginx 重新接纳
```

### 3. 压超时配置

```nginx
proxy_connect_timeout 1s;
proxy_read_timeout    2s;
```

```bash
nght server -t fiber &
curl -i http://nginx/api/response_time/5   # nginx 应该 504，而不是返 nght 的 200
```

### 4. 在 k8s 里做故障注入

临时故障注入不用改代码 —— 注册一个动态路由，打它，删掉。不用 PR、不用重构建、不用重启 pod：

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4 \
  --set adminToken=$(openssl rand -hex 32)
kubectl port-forward svc/nght 8080:8080 &

# 注入一个 502（nginx 会 retry past）
curl -X POST http://localhost:8080/admin/routes \
  -H 'Content-Type: application/json' \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -d '{"path":"/api/inject","status_code":502,"latency_ms":0}'
curl -i http://localhost:8080/api/inject

# 清理
curl -X DELETE http://localhost:8080/admin/routes/api/inject \
  -H "X-Admin-Token: $ADMIN_TOKEN"
```

持续 health-down 行为仍可用经典 `/health/false`：

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4
kubectl port-forward svc/nght 8080:8080 &
curl http://localhost:8080/echo/hello
curl -X POST http://localhost:8080/health/false
kubectl get pods -w
# livenessProbe 用 /livez，pod 不重启
# 但 readinessProbe 返 502，Service endpoint 把 pod 摘掉
```

## 构建 & 测试

```bash
make build         # 带 ldflags 注入 version
make test          # go test ./...
make vet           # go vet ./...
make fmt-check     # gofmt -l . （有差异就 fail）
make release       # goreleaser release --clean （tag 触发）
```

## 分发

- `go install` 或 GitHub Releases 下载 binary。
- 跨平台编译 darwin/linux/windows × amd64/arm64（[goreleaser](https://goreleaser.com/)）。
- 多阶段 Dockerfile（`golang:1.22-bookworm` → `ubuntu:22.04`），ENTRYPOINT 默认 fiber。

## 容器镜像

```bash
docker pull ghcr.io/xunull/nght:0.0.3    # 多架构 manifest (linux/amd64 + linux/arm64)
docker run --rm -p 8080:8080 ghcr.io/xunull/nght:0.0.3
```

基础镜像 `gcr.io/distroless/static-debian12:nonroot`（~25MB，nonroot UID 65532，无 shell —— `kubectl exec nght -- /bin/sh` 会失败）。`latest` tag 跟最新推送；**生产请 pin 具体 tag**。

## Kubernetes / Helm

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4
```

启用 admin-token 鉴权：

```bash
helm install nght oci://ghcr.io/xunull/charts/nght --version 0.0.4 \
  --set adminToken=$(openssl rand -hex 32)
```

`adminToken` 不设时 `/admin/*` 完全开放。**生产必须设 + 用 NetworkPolicy 限制 `/admin/*` 访问** —— chart 本身不做网络级隔离。示例策略：

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
    - podSelector: {}                # 允许同 namespace pod
    - namespaceSelector: {}           # 改成你的 SRE namespace
    ports:
    - port: 8080
      protocol: TCP
```

Chart 只带 `Deployment` + `Service`（无 probes、无 securityContext、无 resources）。生产请覆盖 probes：

```yaml
livenessProbe:
  httpGet: { path: /livez, port: 8080 }
readinessProbe:
  httpGet: { path: /health, port: 8080 }
```

**livenessProbe 用 `/livez`**——`/health/false` 切后端下线时**不应该**重启 pod。**readinessProbe 用 `/health`**——让手动降级的 pod 摘出流量但不杀掉。

## Roadmap

完整路线见项目 office-hours design doc。短列表：

- **近期**：实现 `nght client` 压测子命令（目前是占位）、fiber package-level state 重构成 struct 字段、扩 nginx 场景配方
- **路线 B** *(v0.0.3 已 ship multi-arch GHCR + 极简 Helm chart；v0.0.4 已 ship 动态路由 API)*：ghcr.io Docker image、Helm chart、prometheus `/metrics`
- **路线 C**：Web UI 控制面板、HTTP/3 + QUIC、录回放

## License

见 [LICENSE](./LICENSE)。

## Python 镜像

为兼顾 Python 场景，`nght.py` 是核心端点的 FastAPI 镜像：

```bash
uvicorn nght:app --host 0.0.0.0 --port 8000 --reload
```
