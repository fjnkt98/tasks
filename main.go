package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/fjnkt98/tasks/repository"
	"github.com/fjnkt98/tasks/server"
	_ "github.com/mattn/go-sqlite3"

	"github.com/urfave/cli/v3"

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
	gcpProjectName string
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	path := fmt.Sprintf("projects/%s/traces/", h.gcpProjectName)
	if s := trace.SpanContextFromContext(ctx); s.IsValid() {
		record.AddAttrs(
			slog.String("logging.googleapis.com/trace", path+s.TraceID().String()),
			slog.String("logging.googleapis.com/spanId", s.SpanID().String()),
			slog.Bool("logging.googleapis.com/trace_sampled", s.TraceFlags().IsSampled()),
		)
	}
	return h.Handler.Handle(ctx, record)
}

func setup(ctx context.Context, otelCollectorURL string, gcpProjectName string) (func() error, error) {
	var shutdowns []func(context.Context) error

	shutdown := func() error {
		ctx := context.Background()
		var err error
		for _, fn := range shutdowns {
			err = errors.Join(err, fn(ctx))
		}
		shutdowns = nil
		return err
	}

	// grpc
	conn, err := grpc.NewClient(
		otelCollectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return shutdown, fmt.Errorf("create grpc connection to otel collector: %w", err)
	}

	// resource
	res, err := resource.New(
		ctx,
		resource.WithAttributes(serviceName),
	)
	if err != nil {
		return shutdown, fmt.Errorf("create resource: %w", err)
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
		return shutdown, fmt.Errorf("create trace exporter: %w", err)
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)),
	)
	shutdowns = append(shutdowns, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// logger
	logger := slog.New(&traceHandler{
		Handler:        slog.NewJSONHandler(os.Stdout, nil),
		gcpProjectName: gcpProjectName,
	})
	slog.SetDefault(logger)

	return shutdown, nil
}

func NewCmd() *cli.Command {
	return &cli.Command{
		Name: "app",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "port",
				Required: true,
				Sources:  cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:     "database-url",
				Required: true,
				Sources:  cli.EnvVars("DATABASE_URL"),
			},
			&cli.StringFlag{
				Name:     "gcp-project-name",
				Required: true,
				Sources:  cli.EnvVars("GCP_PROJECT_NAME"),
			},
			&cli.StringFlag{
				Name:     "otel-collector-url",
				Required: true,
				Sources:  cli.EnvVars("OTEL_COLLECTOR_URL"),
			},
			&cli.BoolFlag{
				Name: "migrate",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) (err error) {
			shutdown, err := setup(ctx, cmd.String("otel-collector-url"), cmd.String("gcp-project-name"))
			defer func() {
				err = errors.Join(err, shutdown())
			}()
			if err != nil {
				return err
			}

			port := cmd.Int("port")

			databaseURL := cmd.String("database-url")
			if cmd.Bool("migrate") {
				if err := repository.Migrate(databaseURL); err != nil {
					return fmt.Errorf("migrate: %w", err)
				}
			}
			db, err := repository.NewDB(databaseURL)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer func() {
				err = errors.Join(err, db.Close())
			}()

			s, err := server.NewServer(port, db)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}

			ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errs := make(chan error, 1)
			go func() {
				slog.InfoContext(ctx, "start server", slog.Int("port", port))
				if err := s.ListenAndServe(); err != http.ErrServerClosed {
					errs <- err
				}
			}()

			select {
			case err = <-errs:
				return fmt.Errorf("server error: %w", err)
			case <-ctx.Done():
				if err := s.Shutdown(ctx); err != nil {
					return fmt.Errorf("shutdown server: %w", err)
				}
			}
			slog.InfoContext(ctx, "shutting down server", slog.Int("port", port))

			return nil
		},
	}
}

func main() {
	ctx := context.Background()

	cmd := NewCmd()
	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, "comnand failed", slog.Any("error", err))
		os.Exit(1)
	}
}
