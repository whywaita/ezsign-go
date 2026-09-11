// Command ezsign-go-api serves image updates through an HTTP API.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ezsign "github.com/whywaita/ezsign-go"
	"github.com/whywaita/ezsign-go/httpapi"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP listen address; use :8080 for other hosts")
	timeout := flag.Duration("timeout", 90*time.Second, "maximum duration of one update")
	flag.Parse()
	client, err := ezsign.New(ezsign.Config{Timeout: *timeout})
	if err != nil {
		slog.Error("configure writer", "error", err)
		os.Exit(1)
	}
	server := &http.Server{Addr: *listen, Handler: httpapi.NewHandler(client, httpapi.Config{}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: *timeout + 20*time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { slog.Info("EZ Sign API listening", "address", *listen); done <- server.ListenAndServe() }()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), *timeout+5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			slog.Error("shutdown", "error", err)
			_ = server.Close()
		}
	}
}
