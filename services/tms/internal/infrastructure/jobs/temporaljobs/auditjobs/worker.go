package auditjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
)

type WorkerParams struct {
	fx.In

	Client     client.Client
	Activities *Activities
	Logger     *logger.Logger
	LC         fx.Lifecycle
}

func NewWorker(p WorkerParams) {
	log := p.Logger.With().
		Str("worker", "audit").
		Str("taskQueue", AuditTaskQueue).
		Logger()

	w := worker.New(p.Client, AuditTaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
		MaxConcurrentWorkflowTaskPollers:       2,
		MaxConcurrentActivityTaskPollers:       2,
		EnableSessionWorker:                    true,
	})

	w.RegisterActivity(p.Activities.ProcessAuditBatchActivity)
	w.RegisterActivity(p.Activities.FlushAuditBufferActivity)
	w.RegisterActivity(p.Activities.GetAuditBufferStatusActivity)

	for _, wf := range RegisterWorkflows() {
		w.RegisterWorkflow(wf.Fn)
		log.Info().
			Str("workflow", wf.Name).
			Str("description", wf.Description).
			Msg("registered workflow")
	}

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				log.Info().Msg("starting audit worker")
				if err := w.Run(worker.InterruptCh()); err != nil {
					log.Fatal().Err(err).Msg("failed to start audit worker")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("stopping audit worker")
			w.Stop()
			return nil
		},
	})
}