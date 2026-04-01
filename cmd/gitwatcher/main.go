package main

import (
	"context"
	"fmt"
	"gitwatcher/internal/config"
	"gitwatcher/internal/puller"
	"gitwatcher/internal/pusher"
	"log"
	"net/http"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

const Version = "1.1.0"

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	setupWatcherJob(ctx, cfg, scheduler)

	if len(scheduler.Jobs()) == 0 {
		log.Println("No backup jobs scheduled. Exiting.")
		return
	}

	scheduler.Start()

	srv := server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitwatcher Scheduler"))
	log.Printf("Gitwatcher available at http://localhost:%d", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Router))
}

func setupWatcherJob(ctx context.Context, cfg config.Config, scheduler gocron.Scheduler) {
	if cfg.WatcherJobCron != "" {
		_, err := scheduler.NewJob(
			gocron.CronJob(cfg.WatcherJobCron, true),
			gocron.NewTask(runWatcherJob, ctx, cfg),
			gocron.WithName("Watcher Job"),
			gocron.WithIdentifier(cfg.WatcherJobUUID),
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Scheduled Watcher Job with cron:", cfg.WatcherJobCron)
	}

	runWatcherJob(ctx, cfg)
}

func runWatcherJob(ctx context.Context, cfg config.Config) {
	log.Println("Running Gitwatcher job...")

	err := puller.PullRepo(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = pusher.PushRepo(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
}
