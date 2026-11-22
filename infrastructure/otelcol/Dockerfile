FROM golang:1.27.4-trixie@sha256:2cd7e1ebf475c6208122d2b9cefc638d98fcc8dfd8554c8872c9dd0c21f4ac61 AS builder

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
