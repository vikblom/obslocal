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

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func run(ctx context.Context) error {
	err := configure(ctx)
	if err != nil {
		return fmt.Errorf("configure Otel")
	}

	// TESTING
	//
	tracer := otel.GetTracerProvider().Tracer("main")
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
