FROM golang:1.27.4-trixie@sha256:f18c3d73a184fb0582828510722e723b63a93f7188d11dd223d9be6f6880e65d AS builder

WORKDIR /app

RUN go install go.opentelemetry.io/collector/cmd/builder@v0.162.0

COPY infrastructure/otelcol/manifest.yaml /app/manifest.yaml
RUN CGO_ENABLED=0 builder --config=manifest.yaml

FROM gcr.io/distroless/static-debian13

COPY --from=builder --chown=nonroot:nonroot /app/dist/otelcol /otelcol
COPY infrastructure/otelcol/config.prd.yaml /etc/otelcol/config.yaml

USER nonroot

ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otelcol/config.yaml"]
EXPOSE 4317
