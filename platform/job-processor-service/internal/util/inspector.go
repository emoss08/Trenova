package util

import (
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

func InspectQueue(redisAddr string) error {
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisAddr})
	tasks, err := inspector.ListPendingTasks("default")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to list pending tasks")
		return err
	}
	for _, task := range tasks {
		log.Info().Msgf("Pending task: ID=%s, Type=%s, Payload=%s, Queue=%s", task.ID, task.Type, string(task.Payload), task.Queue)
	}

	return nil
}
