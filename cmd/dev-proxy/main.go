package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gwuah/dev-proxy/internal"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	doneCh := make(chan os.Signal, 1)
	signal.Notify(doneCh, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-doneCh
		cancel()
	}()

	dir, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get cwd", "err", err)
		os.Exit(1)
	}

	_, err = internal.ConnectToDB(filepath.Join(dir, "proxy.db"), internal.Migrations)
	if err != nil {
		logger.Error("failed to connect to db", "err", err)
		os.Exit(1)
	}

	proxy := internal.NewProxy(logger)

	httpServer := &http.Server{
		Addr:    "7777",
		Handler: proxy,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
	}
	httpServer.RegisterOnShutdown(cancel)

	<-ctx.Done()
	logger.Info("shutting down proxy")

	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(gracefulCtx); err != nil {
		logger.Error("shutdown error", "err", err, "http_addr", httpServer.Addr)
		os.Exit(1)
	}

}
