package extractionevaljobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/extractionevaljobs/extractionevalcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var freezeOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    4,
		MaximumInterval:    5 * time.Minute,
		NonRetryableErrorTypes: []string{
			temporaltype.ErrorTypeInvalidInput.String(),
		},
	},
}

type FreezeCaseInput struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	UserID         pulid.ID `json:"userId"`
	CaseID         pulid.ID `json:"caseId"`
}

type FreezeCaseOutcome struct {
	CaseID pulid.ID `json:"caseId"`
	Frozen bool     `json:"frozen"`
	Reason string   `json:"reason,omitempty"`
}

type RefreezeOutcome struct {
	Frozen   int                 `json:"frozen"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Outcomes []FreezeCaseOutcome `json:"outcomes"`
}

func RefreezeExtractionEvalCasesWorkflow(
	ctx workflow.Context,
	payload *extractionevalcontract.RefreezePayload,
) (*RefreezeOutcome, error) {
	freezeCtx := workflow.WithActivityOptions(
		ctx,
		withPriority(freezeOptions, payload.OrganizationID),
	)
	logger := workflow.GetLogger(ctx)

	var a *Activities
	outcome := &RefreezeOutcome{Outcomes: make([]FreezeCaseOutcome, 0, len(payload.CaseIDs))}
	if err := workflow.SetQueryHandler(
		ctx,
		extractionevalcontract.RefreezeProgressQuery,
		func() (extractionevalcontract.RefreezeProgress, error) {
			return extractionevalcontract.RefreezeProgress{
				Total:   len(payload.CaseIDs),
				Frozen:  outcome.Frozen,
				Skipped: outcome.Skipped,
				Failed:  outcome.Failed,
			}, nil
		},
	); err != nil {
		return nil, err
	}
	for _, caseID := range payload.CaseIDs {
		var frozen FreezeCaseOutcome
		if err := workflow.ExecuteActivity(freezeCtx, a.FreezeExtractionEvalCaseActivity, &FreezeCaseInput{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
			UserID:         payload.UserID,
			CaseID:         caseID,
		}).Get(freezeCtx, &frozen); err != nil {
			logger.Error("could not freeze an extraction evaluation case through Docling",
				"caseId", caseID.String(),
				"error", err,
			)
			outcome.Failed++
			outcome.Outcomes = append(outcome.Outcomes, FreezeCaseOutcome{
				CaseID: caseID,
				Reason: err.Error(),
			})
			continue
		}

		if frozen.Frozen {
			outcome.Frozen++
		} else {
			outcome.Skipped++
		}
		outcome.Outcomes = append(outcome.Outcomes, frozen)
	}

	return outcome, nil
}

func (a *Activities) FreezeExtractionEvalCaseActivity(
	ctx context.Context,
	input *FreezeCaseInput,
) (*FreezeCaseOutcome, error) {
	if a.freezer == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"extraction evaluation case freezer is not configured",
			temporaltype.ErrorTypeInvalidInput.String(),
			nil,
		)
	}

	result, err := a.freezer.FreezeDoclingPages(ctx, &extractionevalports.FreezeCasePagesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  input.OrganizationID,
			BuID:   input.BusinessUnitID,
			UserID: input.UserID,
		},
		CaseID: input.CaseID,
	})
	if err != nil {
		return nil, fmt.Errorf("freeze extraction evaluation case %s: %w", input.CaseID, err)
	}

	return &FreezeCaseOutcome{
		CaseID: input.CaseID,
		Frozen: result.Frozen,
		Reason: result.Reason,
	}, nil
}
