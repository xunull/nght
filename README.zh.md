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

## Roadmap

完整路线见项目 office-hours design doc。短列表：

- **近期**：实现 `nght client` 压测子命令（目前是占位）、fiber package-level state 重构成 struct 字段、扩 nginx 场景配方
- **路线 B**：ghcr.io Docker image、Helm chart、prometheus `/metrics`
- **路线 C**：动态 path API (`POST /admin/route`)、Web UI 控制面板、HTTP/3 + QUIC、录回放

## License

见 [LICENSE](./LICENSE)。

## Python 镜像

为兼顾 Python 场景，`nght.py` 是核心端点的 FastAPI 镜像：

```bash
uvicorn nght:app --host 0.0.0.0 --port 8000 --reload
```
