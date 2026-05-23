package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/observa/observad/internal/config"
	"github.com/observa/observad/internal/httpapi"
	"github.com/observa/observad/internal/ingest"
	"github.com/observa/observad/internal/realtime"
	"github.com/observa/observad/internal/store"
	"github.com/observa/observad/pkg/clog"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		return err
	}

	log := clog.New(cfg.LogLevel)
	log.Info("starting observad",
		"http_addr", cfg.HTTPAddr,
		"workers", cfg.Workers,
		"batch_size", cfg.BatchSize,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logStore, err := openStore(ctx, cfg, log)
	if err != nil {
		log.Error("storage init failed", "error", err)
		return err
	}
	defer logStore.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis unreachable", "addr", cfg.Redis.Addr, "error", err)
		return err
	}

	buffer, err := ingest.NewBuffer(ctx, rdb, cfg.Redis.Stream)
	if err != nil {
		log.Error("ingest buffer init failed", "error", err)
		return err
	}

	hub := realtime.NewHub()

	writer, err := ingest.NewWriter(ingest.WriterConfig{
		Buffer:        buffer,
		Store:         logStore,
		Hub:           hub,
		Logger:        log,
		BatchSize:     cfg.BatchSize,
		FlushInterval: cfg.FlushInterval,
		Workers:       cfg.Workers,
	})
	if err != nil {
		log.Error("writer init failed", "error", err)
		return err
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		if err := writer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("ingest writer stopped with error", "error", err)
		}
	}()

	api := httpapi.New(httpapi.Deps{
		Store:           logStore,
		Buffer:          buffer,
		Hub:             hub,
		Logger:          log,
		APIKey:          cfg.IngestAPIKey,
		CORSOrigins:     cfg.CORSOrigins,
		IngestRateLimit: cfg.IngestRateLimit,
		Redis:           rdb,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // WebSocket
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Error("http server failed", "error", err)
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, draining")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http graceful shutdown failed", "error", err)
	}

	select {
	case <-writerDone:
		log.Info("ingest writer drained cleanly")
	case <-shutdownCtx.Done():
		log.Warn("ingest writer did not drain before timeout")
	}

	log.Info("observad stopped")
	return nil
}

func openStore(ctx context.Context, cfg *config.Config, log interface {
	Warn(string, ...any)
}) (store.LogStore, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	ch, err := store.NewClickHouse(
		dialCtx,
		cfg.ClickHouse.Addr,
		cfg.ClickHouse.Database,
		cfg.ClickHouse.Username,
		cfg.ClickHouse.Password,
	)
	if err == nil {
		if err := ch.ApplyRetention(ctx, cfg.Retention); err != nil {
			log.Warn("clickhouse retention apply failed", "error", err)
		}
		return ch, nil
	}

	if os.Getenv("OBSERVA_ALLOW_MEMSTORE") == "1" {
		log.Warn("clickhouse unavailable, falling back to in-memory store (dev only)")
		return store.NewMemStore(), nil
	}
	return nil, err
}
