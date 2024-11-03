package main

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func configure(ctx context.Context) error {
	// Create resource.
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("my-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return fmt.Errorf("resources: %w", err)
	}

	// LOGS
	//
	// Create a logger provider.
	// You can pass this instance directly when creating bridges.
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithInsecure())
	if err != nil {
		return fmt.Errorf("otlploghttp: %w", err)
	}
	lp := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	// Handle shutdown properly so nothing leaks.
	go func() {
		<-ctx.Done()
		if err := lp.Shutdown(context.Background()); err != nil {
			slog.Warn("log provider shutdown", "error", err)
		}
	}()
	// Use it with SLOG.
	global.SetLoggerProvider(lp)
	logger := otelslog.NewLogger("pkgname", otelslog.WithLoggerProvider(lp))
	slog.SetDefault(logger)

	// METRICS
	//
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return fmt.Errorf("otlpmetrichttp: %w", err)
	}
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
	)
	go func() {
		<-ctx.Done()
		if err := mp.Shutdown(context.Background()); err != nil {
			slog.Warn("metric provider shutdown", "error", err)
		}
	}()
	// Baseline metrics of the Go runtime.
	err = runtime.Start()
	if err != nil {
		return fmt.Errorf("runtime metrics: %w", err)
	}
	otel.SetMeterProvider(mp)

	// TRACES
	//
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		return fmt.Errorf("otlptracehttp: %w", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exp),
	)
	go func() {
		<-ctx.Done()
		err := tp.Shutdown(context.Background())
		if err != nil {
			slog.Warn("trace provider shutdown", "error", err)
		}
	}()
	otel.SetTracerProvider(tp)

	return nil
}
