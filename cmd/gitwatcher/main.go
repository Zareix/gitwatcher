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

	"gitwatcher/internal/config"
	"gitwatcher/internal/watcher"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

const Version = "1.8.0"

const jobRunTimeout = 10 * time.Minute

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		os.Exit(1)
	}

	if cfg.WatcherJobCron != "" {
		if err := setupWatcherJob(ctx, cfg, scheduler); err != nil {
			slog.Error("Failed to schedule Watcher Job", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("No CRON configured, watcher job disabled")
	}

	scheduler.Start()
	slog.Info("Scheduler started", "cron", cfg.WatcherJobCron)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitwatcher Scheduler")).Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go runWatcherJob(ctx, cfg)

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := scheduler.StopJobsWithContext(shutdownCtx); err != nil {
			slog.Error("Failed to stop scheduler", "error", err)
		}
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to stop http server", "error", err)
		}
	}()

	slog.Info("Starting server", "port", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupWatcherJob(ctx context.Context, cfg config.Config, scheduler gocron.Scheduler) error {
	_, err := scheduler.NewJob(
		gocron.CronJob(cfg.WatcherJobCron, true),
		gocron.NewTask(runWatcherJob, ctx, cfg),
		gocron.WithName("Watcher Job"),
		gocron.WithIdentifier(cfg.WatcherJobUUID),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return fmt.Errorf("schedule watcher job: %w", err)
	}

	return nil
}

func runWatcherJob(ctx context.Context, cfg config.Config) {
	slog.Info("Running Gitwatcher job...")

	jobCtx, cancel := context.WithTimeout(ctx, jobRunTimeout)
	defer cancel()

	if err := watcher.RunWatcher(jobCtx, cfg); err != nil {
		slog.Error("Gitwatcher job failed", "error", err)
	}
}
