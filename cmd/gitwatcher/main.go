package main

import (
	"context"
	"fmt"
	"gitwatcher/internal/config"
	"gitwatcher/puller"
	"log"
	"net/http"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

const Version = "1.0.0"

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.PullerJobCron != "" {
		_, err := scheduler.NewJob(
			gocron.CronJob(cfg.PullerJobCron, true),
			gocron.NewTask(runPullerJob, ctx, cfg),
			gocron.WithName("Watcher Job"),
			gocron.WithIdentifier(cfg.PullerJobUUID),
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Scheduled Puller Job with cron:", cfg.PullerJobCron)
	}

	runPullerJob(ctx, cfg)

	if len(scheduler.Jobs()) == 0 {
		log.Println("No backup jobs scheduled. Exiting.")
		return
	}

	scheduler.Start()

	srv := server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitwatcher Scheduler"))
	log.Printf("Gitwatcher available at http://localhost:%d", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Router))
}

func runPullerJob(ctx context.Context, cfg config.Config) {
	log.Println("Running Gitwatcher job...")

	err := puller.PullRepo(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
}
