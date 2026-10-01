FROM golang:1.27.1-trixie@sha256:3b77fc618ec235a1ab412de7737f120dd507c57e8d87de4cbb7994fb94275ed5 AS builder

WORKDIR /app

RUN go install go.opentelemetry.io/collector/cmd/builder@v0.162.0

COPY manifest.yaml /app/manifest.yaml
RUN CGO_ENABLE=0 builder --config=manifest.yaml

FROM gcr.io/distroless/static-debian13

COPY --from=builder --chown=nonroot:nonroot /app/dist/otelcol /otelcol
COPY config.prd.yaml /etc/otelcol/config.yaml

USER nonroot

ENTRYPOINT ["/otelcol"]
CMD ["--config", "/etc/otelcol/config.yaml"]
EXPOSE 4317
