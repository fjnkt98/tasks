package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var serviceName = semconv.ServiceNameKey.String("tasks")

type traceHandler struct {
	slog.Handler
	operations []operation
}

type operation struct {
	attrs []slog.Attr
	group string
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	handler := h.Handler

	path := fmt.Sprintf("projects/%s/traces/", os.Getenv("GOOGLE_CLOUD_PROJECT_NAME"))
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		handler = handler.WithAttrs([]slog.Attr{
			slog.String("logging.googleapis.com/trace", path+s.TraceID().String()),
			slog.String("logging.googleapis.com/spanId", s.SpanID().String()),
			slog.Bool("logging.googleapis.com/trace_sampled", s.TraceFlags().IsSampled()),
		})
	}

	for _, op := range h.operations {
		if op.group != "" {
			handler = handler.WithGroup(op.group)
		} else {
			handler = handler.WithAttrs(slices.Clone(op.attrs))
		}
	}
	return handler.Handle(ctx, record)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{
		Handler: h.Handler,
		operations: append(slices.Clone(h.operations), operation{
			attrs: slices.Clone(attrs),
		}),
	}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &traceHandler{
		Handler: h.Handler,
		operations: append(slices.Clone(h.operations), operation{
			group: name,
		}),
	}
}

func setup(ctx context.Context) (*http.Server, func() error, error) {
	// logger
	logger := slog.New(&traceHandler{
		Handler: slog.NewJSONHandler(os.Stdout, nil),
	})
	slog.SetDefault(logger)

	// opentelemetry
	var shutdowns []func(context.Context) error

	shutdown := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var err error
		for _, fn := range slices.Backward(shutdowns) {
			err = errors.Join(err, fn(ctx))
		}
		shutdowns = nil
		return err
	}

	// grpc
	conn, err := grpc.NewClient(
		os.Getenv("OTEL_COLLECTOR_URL"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, shutdown, fmt.Errorf("create grpc connection to otel collector: %w", err)
	}
	shutdowns = append(shutdowns, func(context.Context) error { return conn.Close() })

	// resource
	res, err := resource.New(
		ctx,
		resource.WithAttributes(serviceName),
	)
	if err != nil {
		return nil, shutdown, fmt.Errorf("create resource: %w", err)
	}

	// propagator
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	// tracer provider
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, shutdown, fmt.Errorf("create trace exporter: %w", err)
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
	)
	shutdowns = append(shutdowns, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// db
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		return nil, shutdown, fmt.Errorf("parse PORT: %w", err)
	}

	db, err := NewDB(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, shutdown, fmt.Errorf("open database: %w", err)
	}
	shutdowns = append(shutdowns, func(context.Context) error { return db.Close() })

	if _, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at <= ?", time.Now().Unix()); err != nil {
		return nil, shutdown, fmt.Errorf("delete expired session: %w", err)
	}

	// server
	server, err := NewServer(port, db)
	if err != nil {
		return nil, shutdown, fmt.Errorf("create server: %w", err)
	}

	return server, shutdown, nil
}

func serve(ctx context.Context, server *http.Server, timeout time.Duration) error {
	errs := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "start server", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errs <- err
		}
	}()

	select {
	case err := <-errs:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return errors.Join(fmt.Errorf("shutdown server: %w", err), server.Close())
		}
	}
	slog.InfoContext(ctx, "shutting down server", slog.String("addr", server.Addr))

	return nil
}

func run(ctx context.Context) (err error) {
	server, shutdown, err := setup(ctx)
	defer func() {
		err = errors.Join(err, shutdown())
	}()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(ctx, server, 5*time.Second)
}

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "comnand failed", slog.Any("error", err))
		os.Exit(1)
	}
}
