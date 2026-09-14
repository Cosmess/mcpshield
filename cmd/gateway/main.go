package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Cosmess/mcpshield/internal/approval"
	"github.com/Cosmess/mcpshield/internal/audit"
	"github.com/Cosmess/mcpshield/internal/auth"
	"github.com/Cosmess/mcpshield/internal/config"
	"github.com/Cosmess/mcpshield/internal/httpapi"
	"github.com/Cosmess/mcpshield/internal/mcpproxy"
	"github.com/Cosmess/mcpshield/internal/policy"
	"github.com/Cosmess/mcpshield/internal/upstream"
	"github.com/jackc/pgx/v5/pgxpool"
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
	if config.AuthIssuer != "" {
		validator, err := auth.NewValidator(auth.Config{Issuer: config.AuthIssuer, Audience: config.AuthAudience, JWKSURL: config.AuthJWKSURL, CacheTTL: 5 * time.Minute, Timeout: config.RequestTimeout})
		if err != nil {
			return fmt.Errorf("create authenticator: %w", err)
		}
		api.SetAuthenticator(validator)
	}
	registry, err := upstream.NewRegistry(config.Upstreams)
	if err != nil {
		return fmt.Errorf("create upstream registry: %w", err)
	}
	var auditSink audit.Sink = audit.NewMemorySink(1024)
	proxy, err := mcpproxy.New(parent, registry, logger, auditSink)
	if err != nil {
		return fmt.Errorf("create MCP proxy: %w", err)
	}
	defer proxy.Close()
	approvalRepository := approval.Repository(approval.NewMemoryRepository())
	var databasePool *pgxpool.Pool
	if config.DatabaseURL != "" {
		databasePool, err = pgxpool.New(parent, config.DatabaseURL)
		if err != nil {
			return fmt.Errorf("create database pool: %w", err)
		}
		defer databasePool.Close()
		if err := databasePool.Ping(parent); err != nil {
			return fmt.Errorf("ping database: %w", err)
		}
		if err := approval.ApplyMigration(parent, databasePool); err != nil {
			return err
		}
		if err := audit.ApplyMigration(parent, databasePool); err != nil {
			return err
		}
		auditSink = audit.NewPostgresSink(databasePool)
		approvalRepository = approval.NewPostgresRepository(databasePool)
	}
	approvalService := approval.NewService(approvalRepository)
	approvalService.SetTransitionHook(func(status approval.Status, record approval.Record) {
		auditSink.Record(audit.Event{UpstreamID: record.UpstreamID, MCPMethod: record.Method, Outcome: "approval_" + strings.ToLower(string(status)), OccurredAt: time.Now()})
	})
	proxy.SetApprovalService(approvalService)
	api.SetApprovalService(approvalService)
	if config.PolicyFile != "" {
		policyFile, err := os.Open(config.PolicyFile)
		if err != nil {
			return fmt.Errorf("open policy file: %w", err)
		}
		engine, loadErr := policy.LoadJSON(policyFile)
		closeErr := policyFile.Close()
		if loadErr != nil {
			return fmt.Errorf("load policy file: %w", loadErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close policy file: %w", closeErr)
		}
		proxy.SetPolicy(engine)
	}
	api.SetMCPHandler(proxy.Handler())
	api.SetExtraMetrics(proxy.RiskMetrics)
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
