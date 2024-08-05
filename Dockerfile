FROM golang:1.22.5-bookworm AS builder

ENV GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go build -o /app/

###########################################################################################

FROM ubuntu:22.04

RUN apt update && apt install -y lsof net-tools curl

WORKDIR /app

COPY --from=builder /app/nght /app/nght

ENTRYPOINT ["/app/nght","server"]

