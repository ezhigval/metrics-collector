package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ezhigval/go-toolkit/httputil"
	"github.com/ezhigval/go-toolkit/logger"
	tkmw "github.com/ezhigval/go-toolkit/middleware"
	tkpgx "github.com/ezhigval/go-toolkit/pgx"
	"github.com/ezhigval/metrics-collector/internal/config"
	"github.com/ezhigval/metrics-collector/internal/handler"
	promreg "github.com/ezhigval/metrics-collector/internal/prom"
	"github.com/ezhigval/metrics-collector/internal/repository"
	"github.com/ezhigval/metrics-collector/internal/service"
	"github.com/ezhigval/metrics-collector/internal/telemetry"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})
	ctx := context.Background()

	var shutdownTracer func(context.Context) error
	if cfg.EnableOTel {
		var err error
		shutdownTracer, err = telemetry.InitTracer(ctx, cfg.ServiceName)
		if err != nil {
			log.Error("otel init failed", "error", err)
			os.Exit(1)
		}
		defer func() { _ = shutdownTracer(context.Background()) }()
	}

	pool, err := tkpgx.NewPool(ctx, tkpgx.Config{URL: cfg.DatabaseURL, MaxConns: 10})
	if err != nil {
		log.Error("postgres failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := tkpgx.Ping(ctx, pool); err != nil {
		log.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}

	prom := promreg.NewRegistry()
	repo := repository.NewMetricRepository(pool)
	svc := service.NewMetricService(repo, prom)
	h := handler.New(svc, prom)

	r := chi.NewRouter()
	r.Use(tkmw.RequestID, tkmw.RealIP, tkmw.Recoverer(log), tkmw.AccessLog(log))
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", httputil.HealthHandler(map[string]func() error{
		"postgres": func() error { return tkpgx.Ping(ctx, pool) },
	}))

	r.Get("/metrics", h.Prometheus)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return otelhttp.NewHandler(next, "api")
		})
		r.Post("/ingest", h.Ingest)
		r.Get("/metrics", h.ListNames)
		r.Get("/metrics/{name}", h.Query)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("metrics-collector started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("server stopped")
}
