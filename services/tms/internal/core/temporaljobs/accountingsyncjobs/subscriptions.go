package accountingsyncjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	KeepAccountingWebhookSubscriptionsWorkflowName = "KeepAccountingWebhookSubscriptionsWorkflow"
	subscriptionPageSize                           = 25
	subscriptionMaxPages                           = 40
)

type SubscriptionSweepResult struct {
	Listed  int `json:"listed"`
	Created int `json:"created"`
	Renewed int `json:"renewed"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
	Pages   int `json:"pages"`
}

func subscriptionWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        KeepAccountingWebhookSubscriptionsWorkflowName,
			Fn:          KeepAccountingWebhookSubscriptionsWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Create, renew and remove the webhook subscriptions accounting systems require",
		},
	}
}

func KeepAccountingWebhookSubscriptionsWorkflow(
	ctx workflow.Context,
) (*SubscriptionSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, healthActivityOptions)

	var a *Activities
	result := new(SubscriptionSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.KeepAccountingWebhookSubscriptionsActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Accounting webhook subscription sweep failed", "error", err)
		return nil, err
	}

	return result, nil
}

func (a *Activities) KeepAccountingWebhookSubscriptionsActivity(
	ctx context.Context,
) (*SubscriptionSweepResult, error) {
	result := &SubscriptionSweepResult{}
	var after pulid.ID
	for page := range subscriptionMaxPages {
		sweep, err := a.connections.SyncWebhookSubscriptions(
			ctx,
			&services.SyncAccountingWebhookSubscriptionsRequest{
				AfterID: after,
				Limit:   subscriptionPageSize,
			},
		)
		if err != nil {
			return result, err
		}
		result.Pages = page + 1
		result.Listed += sweep.Listed
		result.Created += sweep.Created
		result.Renewed += sweep.Renewed
		result.Removed += sweep.Removed
		result.Failed += sweep.Failed
		activity.RecordHeartbeat(ctx, result.Pages)

		if sweep.Listed < subscriptionPageSize || sweep.NextAfterID.IsNil() {
			break
		}
		after = sweep.NextAfterID
	}

	if result.Failed > 0 {
		a.l.Warn("accounting webhook subscriptions could not all be kept",
			zap.Int("failed", result.Failed), zap.Int("listed", result.Listed))
	}
	return result, nil
}
