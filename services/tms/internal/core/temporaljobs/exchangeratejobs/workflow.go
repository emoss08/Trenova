package exchangeratejobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const RefreshExchangeRatesWorkflowName = "RefreshExchangeRatesWorkflow"

var refreshExchangeRatesRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var refreshExchangeRatesActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         refreshExchangeRatesRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        RefreshExchangeRatesWorkflowName,
			Fn:          RefreshExchangeRatesWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Refresh cached exchange rates from ExchangeRate-API for all enabled tenants",
		},
	}
}

func RefreshExchangeRatesWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, refreshExchangeRatesActivityOptions)

	var a *Activities
	return workflow.ExecuteActivity(ctx, a.RefreshExchangeRatesActivity).Get(ctx, nil)
}
