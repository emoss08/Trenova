package infrastructure

import (
	"github.com/emoss08/trenova/internal/infrastructure/messaging/rabbitmq"
	"github.com/emoss08/trenova/internal/pkg/workflow"
	"go.uber.org/fx"
)

var MessagingModule = fx.Module("messaging",
	fx.Provide(rabbitmq.NewWorkflowPublisher),
	fx.Invoke(func(lc fx.Lifecycle, p *rabbitmq.WorkflowPublisher) {
		lc.Append(p.RegisterHooks())
	}),
	fx.Provide(workflow.NewService),
)
