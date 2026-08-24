package tenderjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	RateConfirmationIssueSweepWorkflowName = "RateConfirmationIssueSweepWorkflow"

	// issueSweepGraceSeconds keeps the sweep clear of the owning workflow's own
	// IssueRateConfirmationActivity retries: an acceptance is only picked up
	// once its in-band issuance has had time to succeed or give up.
	issueSweepGraceSeconds = 900

	issueSweepLookbackSeconds = 7 * 24 * 60 * 60

	issueSweepBatchLimit = 100
)

type RateConIssueSweepResult struct {
	Checked int `json:"checked"`
	Issued  int `json:"issued"`
	Failed  int `json:"failed"`
}

func RateConfirmationIssueSweepWorkflow(
	ctx workflow.Context,
) (*RateConIssueSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	})

	var a *Activities
	result := new(RateConIssueSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.SweepMissingRateConfirmationsActivity).
		Get(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// SweepMissingRateConfirmationsActivity is the recovery arm under the
// acceptance path: an accepted tender whose carrier already covers the move
// but whose rate confirmation never landed gets the idempotent issuer run
// again. The repository query excludes every case the issuer would decline —
// a standing dispatcher-managed revision, or no matching assignment at all —
// so a row here is always genuinely missing its paperwork.
func (a *Activities) SweepMissingRateConfirmationsActivity(
	ctx context.Context,
) (*RateConIssueSweepResult, error) {
	result := new(RateConIssueSweepResult)

	now := timeutils.NowUnix()
	tenders, err := a.repo.ListAcceptedMissingRateConfirmation(
		ctx,
		repositories.ListAcceptedMissingRateConfirmationRequest{
			AcceptedAfter:  now - issueSweepLookbackSeconds,
			AcceptedBefore: now - issueSweepGraceSeconds,
			Limit:          issueSweepBatchLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, entity := range tenders {
		if entity == nil || entity.AcceptedOfferID == nil {
			continue
		}
		result.Checked++

		if issueErr := a.lifecycle.IssueRateConfirmation(
			ctx,
			pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			*entity.AcceptedOfferID,
		); issueErr != nil {
			result.Failed++
			a.l.Error("failed to issue a missing rate confirmation from the sweep",
				zap.Error(issueErr),
				zap.String("tenderId", entity.ID.String()))
			continue
		}
		result.Issued++
	}

	return result, nil
}
