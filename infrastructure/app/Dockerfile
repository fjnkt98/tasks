# App
FROM golang:1.27.3-trixie@sha256:fbb7ae40b62ff8afe31662df570d39b4f5052ac32b90dae8e60d7657a20a74f4 AS stage

ARG TARGETARCH

WORKDIR /app

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
RUN go mod download

COPY main.go /app/main.go
COPY internal/ /app/internal/

COPY internal/schema/schema.sql /app/schema.sql

ENV CGO_ENABLED=1
RUN go build -o /app/app

# Litestream
FROM litestream/litestream:0.5.17-scratch@sha256:f757c70d070ac278d45b8847d31a54ab2de24de5e77b09018c642eca263e3967 AS litestream

# sqldef
FROM sqldef/sqlite3def:3.11.26@sha256:91db0e5a8d1a3fc64ca0e33eab9ba29b768e7f0a870f4a8f93565640d0ed2977 AS sqldef

# Main
FROM debian:trixie-20260918-slim@sha256:a99cfc517144bc59b1978475ec53b46ecabec7e43635402ee5b77cc54cd1b20a

WORKDIR /app

RUN ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime
ENV TZ="Asia/Tokyo"

COPY --from=stage /app/app /app/app
COPY --from=litestream /usr/local/bin/litestream /usr/local/bin/litestream
COPY --from=sqldef /usr/local/bin/sqldef /usr/local/bin/sqlite3def
COPY --from=stage /app/schema.sql /app/schema.sql

COPY infrastructure/app/litestream.yaml /etc/litestream.yaml
COPY --chmod=755 infrastructure/app/run.sh /app/run.sh

CMD ["/app/run.sh"]
