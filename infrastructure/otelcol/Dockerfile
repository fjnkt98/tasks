FROM golang:1.27.3-trixie@sha256:fbb7ae40b62ff8afe31662df570d39b4f5052ac32b90dae8e60d7657a20a74f4 AS builder

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
