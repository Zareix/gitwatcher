package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitwatcher/internal/config"
	"gitwatcher/internal/watcher"

	"github.com/robfig/cron/v3"
)

const Version = "1.8.1"

const jobRunTimeout = 10 * time.Minute

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.WatcherJobCron == "" {
		slog.Info("No CRON configured, exiting")
		return
	}

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	scheduler := cron.New(
		cron.WithParser(parser),
		cron.WithChain(cron.Recover(cron.DefaultLogger), cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	_, err = scheduler.AddFunc(cfg.WatcherJobCron, func() {
		runWatcherJob(ctx, cfg)
	})
	if err != nil {
		slog.Error("Failed to schedule Watcher Job", "error", err)
		os.Exit(1)
	}

	scheduler.Start()
	slog.Info("Scheduler started", "cron", cfg.WatcherJobCron)

	go runWatcherJob(ctx, cfg)

	<-ctx.Done()
	slog.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopCtx := scheduler.Stop()
	select {
	case <-stopCtx.Done():
	case <-shutdownCtx.Done():
		slog.Warn("Shutdown timeout reached, stopping scheduler")
	}
}

func runWatcherJob(ctx context.Context, cfg config.Config) {
	slog.Info("Running Gitwatcher job...")

	jobCtx, cancel := context.WithTimeout(ctx, jobRunTimeout)
	defer cancel()

	if err := watcher.RunWatcher(jobCtx, cfg); err != nil {
		slog.Error("Gitwatcher job failed", "error", err)
	}
}
