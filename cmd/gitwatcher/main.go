package main

import (
	"context"
	"fmt"
	"gitwatcher/internal/config"
	"gitwatcher/internal/watcher"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

const Version = "1.3.0"

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		os.Exit(1)
	}

	setupWatcherJob(ctx, cfg, scheduler)

	if len(scheduler.Jobs()) == 0 {
		slog.Info("No backup jobs scheduled. Exiting.")
		return
	}

	scheduler.Start()

	srv := server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitwatcher Scheduler"))
	slog.Info("Starting server", "port", cfg.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Router)
	if err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupWatcherJob(ctx context.Context, cfg config.Config, scheduler gocron.Scheduler) {
	if cfg.WatcherJobCron != "" {
		_, err := scheduler.NewJob(
			gocron.CronJob(cfg.WatcherJobCron, true),
			gocron.NewTask(runWatcherJob, ctx, cfg),
			gocron.WithName("Watcher Job"),
			gocron.WithIdentifier(cfg.WatcherJobUUID),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		)
		if err != nil {
			slog.Error("Failed to schedule Watcher Job", "error", err)
			os.Exit(1)
		}
		slog.Info("Scheduled Watcher Job", "cron", cfg.WatcherJobCron)
	}

	runWatcherJob(ctx, cfg)
}

func runWatcherJob(ctx context.Context, cfg config.Config) {
	slog.Info("Running Gitwatcher job...")

	err := watcher.RunWatcher(ctx, cfg)
	if err != nil {
		slog.Error("Gitwatcher job failed", "error", err)
	}
}
