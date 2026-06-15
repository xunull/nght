# syntax=docker/dockerfile:1.7
FROM --platform=$BUILDPLATFORM golang:1.22.5-bookworm AS builder
ARG VERSION
ARG TARGETOS
ARG TARGETARCH

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X github.com/xunull/nght/cmd.Version=${VERSION}" \
    -o /out/nght .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/nght /nght
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/nght", "server", "--type", "fiber"]
