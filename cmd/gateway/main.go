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
	"time"

	"github.com/Cosmess/mcpshield/internal/config"
	"github.com/Cosmess/mcpshield/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(context.Background(), logger); err != nil {
		logger.Error("gateway_exit", "error", err)
		os.Exit(1)
	}
}

func run(parent context.Context, logger *slog.Logger) error {
	config, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	api := httpapi.New(logger, config.RequestTimeout, config.MaxBodyBytes)
	server := &http.Server{
		Addr:              config.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: config.RequestTimeout,
		WriteTimeout:      config.RequestTimeout,
		IdleTimeout:       config.RequestTimeout,
	}

	ctx, stop := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("gateway_started", "address", config.HTTPAddr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve gateway: %w", err)
	case <-ctx.Done():
		logger.Info("gateway_shutdown")
		if err := shutdownServer(server, config.ShutdownTimeout); err != nil {
			return fmt.Errorf("shutdown gateway: %w", err)
		}
		return nil
	}
}

func shutdownServer(server *http.Server, timeout time.Duration) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return server.Shutdown(shutdownContext)
}
