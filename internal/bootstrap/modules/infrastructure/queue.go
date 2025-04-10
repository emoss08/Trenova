package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/queue"
	"go.uber.org/fx"
)

var QueueModule = fx.Module("queue",
	fx.Provide(queue.NewClient),
	fx.Invoke(func(client *queue.Client) {
		// No need to use the lifecycle hook directly here, we'll handle it in the client
		client.SetupWithFx()
	}),
)
