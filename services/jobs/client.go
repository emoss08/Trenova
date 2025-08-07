package jobsservice

import "github.com/hibiken/asynq"

func NewClient() *asynq.Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: "localhost:6379",
	})

	return client
}
