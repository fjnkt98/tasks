# App
FROM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS stage

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
FROM litestream/litestream:0.5.18-scratch@sha256:0ba1cdbb2d0be8876e27b1e646dc031309b7e3928e0ed91503ebaa73ff7e2c5d AS litestream

# sqldef
FROM sqldef/sqlite3def:3.11.30@sha256:87110de34d00e9bb7d607656f4bd0afe27890696755a721a5037c079ff86c6c7 AS sqldef

# Main
FROM debian:trixie-20261104-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2

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
