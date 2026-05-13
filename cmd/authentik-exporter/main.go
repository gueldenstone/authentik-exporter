package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/lukas/authentik-exporter/internal/client"
	"github.com/lukas/authentik-exporter/internal/collector"
	"github.com/lukas/authentik-exporter/internal/config"
)

func main() {
	log := newLogger(os.Getenv("EXPORTER_LOG_LEVEL"))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}
	log = newLogger(cfg.LogLevel)

	log.Info("starting authentik-exporter",
		"url", cfg.AuthentikURL,
		"listen", cfg.ListenAddr,
		"poll_interval", cfg.PollInterval,
		"windows", cfg.Windows,
		"actions", cfg.Actions,
	)

	cli, err := client.New(cfg.AuthentikURL, cfg.AuthentikToken, cfg.InsecureSkipVerify)
	if err != nil {
		log.Error("client init failed", "err", err)
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	metrics := collector.New(reg)

	poller := collector.NewPoller(cfg, cli, metrics, log)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go poller.Run(ctx)

	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h1>authentik-exporter</h1><p><a href="` + cfg.MetricsPath + `">metrics</a></p></body></html>`))
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", cfg.ListenAddr, "path", cfg.MetricsPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-srvErr:
		log.Error("http server failed", "err", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", "err", err)
	}
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
}
