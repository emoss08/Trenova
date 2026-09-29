package aicorrectionjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const ExtractionAccuracyDriftWorkflowName = "ExtractionAccuracyDriftWorkflow"

type ExtractionAccuracyDriftInput struct {
	After *temporaljobs.TenantWorkItem `json:"after,omitempty"`
	Now   int64                        `json:"now,omitempty"`
}

type OrganizationDriftInput struct {
	temporaljobs.TenantWorkItem
	Now int64 `json:"now"`
}

type OrganizationDriftResult struct {
	Notified int `json:"notified"`
}

type ExtractionAccuracyDriftResult struct {
	OrganizationsProcessed int      `json:"organizationsProcessed"`
	Notified               int      `json:"notified"`
	FailedOrganizations    []string `json:"failedOrganizations"`
}

func driftWorkflowDefinition() temporaltype.WorkflowDefinition {
	return temporaltype.WorkflowDefinition{
		Name:        ExtractionAccuracyDriftWorkflowName,
		Fn:          ExtractionAccuracyDriftWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: "Tell each organization when an extraction provider read last week's documents worse than its own recent weeks",
	}
}

func ExtractionAccuracyDriftWorkflow(
	ctx workflow.Context,
	input *ExtractionAccuracyDriftInput,
) (*ExtractionAccuracyDriftResult, error) {
	if input == nil {
		input = &ExtractionAccuracyDriftInput{}
	}
	now := input.Now
	if now == 0 {
		now = workflow.Now(ctx).Unix()
	}

	var a *Activities
	result := &ExtractionAccuracyDriftResult{FailedOrganizations: make([]string, 0)}
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "extraction accuracy drift",
		Concurrency: organizationConcurrency,
		ListPage: func(
			wctx workflow.Context,
			after *temporaljobs.TenantWorkItem,
		) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			listCtx := workflow.WithActivityOptions(wctx, listOptions)
			var page temporaljobs.TenantPage
			err := workflow.ExecuteActivity(
				listCtx,
				a.ListAICorrectionOrganizationsActivity,
				&ListAICorrectionOrganizationsInput{
					After: after,
					Limit: temporaljobs.DefaultOrganizationPageSize,
				},
			).Get(listCtx, &page)

			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			options := organizationOptions
			options.Priority = temporal.Priority{FairnessKey: tenant.OrganizationID.String()}
			activityCtx := workflow.WithActivityOptions(wctx, options)
			var checked OrganizationDriftResult
			if err := workflow.ExecuteActivity(
				activityCtx,
				a.CheckOrganizationExtractionDriftActivity,
				&OrganizationDriftInput{TenantWorkItem: tenant, Now: now},
			).Get(activityCtx, &checked); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)

				return 0, err
			}
			result.OrganizationsProcessed++
			result.Notified += checked.Notified

			return checked.Notified, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, ExtractionAccuracyDriftWorkflow,
			&ExtractionAccuracyDriftInput{After: next, Now: now})
	}

	return result, nil
}

func (a *Activities) CheckOrganizationExtractionDriftActivity(
	ctx context.Context,
	input *OrganizationDriftInput,
) (*OrganizationDriftResult, error) {
	if a.drift == nil {
		return &OrganizationDriftResult{}, nil
	}

	tenant := input.TenantInfo()
	notified, err := a.drift.CheckDrift(ctx, &services.ExtractionDriftCheckRequest{
		TenantInfo: tenant,
		Now:        input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("check extraction accuracy drift: %w", err)
	}
	if notified > 0 {
		a.l.Info("extraction accuracy drift reported",
			zap.String("organization", tenant.OrgID.String()),
			zap.Int("providers", notified),
		)
	}

	return &OrganizationDriftResult{Notified: notified}, nil
}
