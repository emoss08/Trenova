package temporaljobs

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/startwait"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/pollwatch"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AwaitEngineParams struct {
	fx.In

	Client         client.Client
	TemporalConfig *temporalconfig.Config
	AppConfig      *appconfig.Config
	Logger         *zap.Logger
}

func AwaitEngine(p AwaitEngineParams) error {
	return startwait.Wait(context.Background(), startwait.Params{
		Logger:     p.Logger.Named("temporal-client"),
		Dependency: "Temporal",
		Deadline:   startwait.Deadline(p.AppConfig.StartupWait),
		Check:      engineAnswers(p.Client, p.TemporalConfig.GetNamespace()),
	})
}

func engineAnswers(c client.Client, namespace string) func(context.Context) error {
	return func(ctx context.Context) error {
		if _, err := c.CheckHealth(ctx, &client.CheckHealthRequest{}); err != nil {
			return err
		}
		_, err := c.WorkflowService().
			DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
				Namespace: namespace,
			})
		return err
	}
}

func bindPollProbe(polls *pollwatch.Watch, c client.Client, queues []string) {
	if len(queues) == 0 {
		return
	}
	queue := slices.Min(queues)
	polls.Bind(func(ctx context.Context) error {
		_, err := c.DescribeTaskQueue(ctx, queue, enumspb.TASK_QUEUE_TYPE_WORKFLOW)
		return err
	})
}
