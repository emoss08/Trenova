package worker

import (
	"backend/tasks"
	"log"

	"github.com/fatih/color"
	"github.com/hibiken/asynq"
)

const redisAddr = "127.0.0.1:6379"

// Init initializes and runs the asynq worker server
func Init() {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(tasks.TypeGenerateThumbnail, tasks.NewThumbnailProcessor())

	// Log a success message after starting the server
	successMsg := color.New(color.FgHiMagenta).SprintfFunc()
	log.Println(successMsg("👷 Successfully started the asynq worker server"))

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
