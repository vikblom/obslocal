package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func run(ctx context.Context) error {
	// Create resource.
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("my-service"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		panic(err)
	}

	// LOGS
	//
	// Create a logger provider.
	// You can pass this instance directly when creating bridges.
	//
	// Note that Zerolog is not supported out-of-the-box by otel:
	// https://github.com/open-telemetry/opentelemetry-go-contrib/issues/5405#issuecomment-2276271236
	exporter, err := otlploghttp.New(ctx, otlploghttp.WithInsecure())
	if err != nil {
		panic(err)
	}
	provider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	// Handle shutdown properly so nothing leaks.
	defer func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			fmt.Println(err)
		}
	}()
	// Use it with SLOG.
	// TODO: There is also a otel.SetLogger.
	global.SetLoggerProvider(provider)
	logger := otelslog.NewLogger("pkgname", otelslog.WithLoggerProvider(provider))
	slog.SetDefault(logger)

	// TRACES
	//
	//
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		panic(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exp),
	)
	defer func() {
		err := tp.Shutdown(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	}()
	// TODO: Configure the propagataion format.
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("main")

	// TESTING
	//
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.DebugContext(r.Context(), "in the handler")

		ctx, span := tracer.Start(r.Context(), "my-function")
		defer span.End()
		time.Sleep(250 * time.Millisecond)

		_, span2 := tracer.Start(ctx, "my-function-2")
		span.AddEvent("bopity")
		span2.End()

		time.Sleep(250 * time.Millisecond)

	}))
	s := http.Server{Addr: ":1234", Handler: otelhttp.NewMiddleware("api")(mux)}
	lst, err := net.Listen("tcp", ":1234")
	if err != nil {
		panic(err)
	}
	go func() {
		_ = s.Serve(lst)
	}()
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	for ctx.Err() == nil {
		resp, err := http.Get("http://127.0.0.1:1234")
		if err != nil {
			panic(err)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		select {
		case <-ctx.Done():
			break
		case <-time.After(5 * time.Second):
		}
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	err := run(ctx)
	if err != nil {
		panic(err)
	}
	<-ctx.Done()
}
