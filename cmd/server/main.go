package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/compute/metadata"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	gcppropagator "github.com/GoogleCloudPlatform/opentelemetry-operations-go/propagator"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/vvakame/atproto-vvvot/httpapi"
	"github.com/vvakame/atproto-vvvot/internal/cliutils"
	"github.com/vvakame/sdlog/gcpslog"
	octrace "go.opencensus.io/trace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/bridge/opencensus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
)

func main() {
	{
		h := gcpslog.HandlerOptions{
			Level: slog.LevelDebug,
			TraceInfo: func(ctx context.Context) (traceID string, spanID string) {
				span := trace.SpanFromContext(ctx)
				if span == nil {
					return "", ""
				}
				if span.SpanContext().HasTraceID() {
					traceID = span.SpanContext().TraceID().String()
				}
				if span.SpanContext().HasSpanID() {
					spanID = span.SpanContext().SpanID().String()
				}

				return traceID, spanID
			},
		}.NewHandler(os.Stdout)
		slog.SetDefault(slog.New(h))
	}

	if metadata.OnGCE() {
		ctx := context.Background()

		slog.Info("setup otel exporter and misc")

		exporter, err := texporter.New()
		if err != nil {
			slog.Error("error received from texporter.New", "error", err)
			panic(err)
		}

		res, err := resource.New(ctx,
			resource.WithDetectors(gcp.NewDetector()),
			resource.WithTelemetrySDK(),
			resource.WithAttributes(
				semconv.ServiceNameKey.String("vvvot"),
			),
		)
		if err != nil {
			slog.Error("error received from resource.New", "error", err)
			panic(err)
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		defer func() {
			err := tp.ForceFlush(ctx)
			if err != nil {
				slog.Error("error on tp.ForceFlush", "error", err)
			}
		}()
		otel.SetTracerProvider(tp)

		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				gcppropagator.CloudTraceOneWayPropagator{},
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)

		{ // gcloud clients still uses OpenCensus.
			tracer := otel.GetTracerProvider().Tracer("ocbridge")
			octrace.DefaultTracer = opencensus.NewTracer(tracer)
		}
	}

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	xrpcc := &xrpc.Client{
		Client: otelhttp.DefaultClient,
		Host:   "https://bsky.social",
	}

	err := cliutils.CheckTokenExpired(ctx, xrpcc)
	if err != nil {
		slog.Error("error on cliutils.CheckTokenExpired", "error", err)
		panic(err)
	}

	mux := http.NewServeMux()

	h, err := httpapi.New(xrpcc)
	if err != nil {
		slog.Error("error on httpapi.New", "error", err)
		panic(err)
	}

	h.Serve(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: otelhttp.NewHandler(mux, "http.request"),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error received from srv.ListenAndServe", "error", err)
			panic(err)
		}
	}()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		slog.Error("error received from srv.Shutdown", "error", err)
		panic(err)
	}

	slog.Info("server shutdown properly")
}
