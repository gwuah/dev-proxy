package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
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
	caCert, err := tls.LoadX509KeyPair("./keys/ca.crt", "./keys/ca.key")
	if err != nil {
		log.Fatal(err)
	}

	ca, err := x509.ParseCertificate(caCert.Certificate[0])
	if err != nil {
		log.Fatal(err)
	}

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

	db, err := internal.ConnectToDB(filepath.Join(dir, "proxy.db"), internal.Migrations)
	if err != nil {
		logger.Error("failed to connect to db", "err", err)
		os.Exit(1)
	}

	proxy := internal.NewDevProxy(logger, db)

	httpServer := &http.Server{
		Addr: "127.0.0.1:7777",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info(fmt.Sprintf("Forwarding request: %s -> %s", r.Method, r.URL.String()))
			if r.Method == http.MethodConnect {
				proxy.HandleHTTPS(w, r, ca, &caCert)
			} else {
				proxy.HandleHTTP(w, r)
			}
		}),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
	}
	httpServer.RegisterOnShutdown(cancel)

	go func() {
		logger.Info(fmt.Sprintf("listening on %s", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			logger.With("err", err).Error(fmt.Sprintf("failed to listen on %s", httpServer.Addr))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down proxy")

	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(gracefulCtx); err != nil {
		logger.Error("shutdown error", "err", err, "http_addr", httpServer.Addr)
		os.Exit(1)
	}

}
