package tenderjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	PurgeTenderTokensWorkflowName = "PurgeTenderTokensWorkflow"

	// tokenRetentionDays is measured from the moment a token became dead —
	// used, revoked, or expired — not from when it was minted.
	tokenRetentionDays = 30

	tokenPurgeBatchSize = 500

	secondsPerDay = int64(24 * 60 * 60)
)

type PurgeTenderTokensResult struct {
	OfferTokensPurged int64 `json:"offerTokensPurged"`
	SignTokensPurged  int64 `json:"signTokensPurged"`
}

func PurgeTenderTokensWorkflow(ctx workflow.Context) (*PurgeTenderTokensResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	})

	var a *Activities
	result := new(PurgeTenderTokensResult)
	if err := workflow.ExecuteActivity(ctx, a.PurgeDeadTokensActivity).
		Get(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// PurgeDeadTokensActivity deletes hashed external-party credentials that can no
// longer authorize anything, once the retention window past the moment they
// died has elapsed. Live links are never eligible.
func (a *Activities) PurgeDeadTokensActivity(
	ctx context.Context,
) (*PurgeTenderTokensResult, error) {
	deadBefore := timeutils.NowUnix() - tokenRetentionDays*secondsPerDay
	result := new(PurgeTenderTokensResult)

	offerTokens, err := purgeTokensInBatches(ctx, deadBefore, a.repo.PurgeDeadOfferTokens)
	result.OfferTokensPurged = offerTokens
	if err != nil {
		a.l.Error("failed to purge dead tender offer tokens", zap.Error(err))
		return nil, err
	}

	signTokens, err := purgeTokensInBatches(ctx, deadBefore, a.rateConRepo.PurgeDeadSignTokens)
	result.SignTokensPurged = signTokens
	if err != nil {
		a.l.Error("failed to purge dead rate confirmation tokens", zap.Error(err))
		return nil, err
	}

	a.l.Info("dead token retention purge completed",
		zap.Int64("offerTokensPurged", result.OfferTokensPurged),
		zap.Int64("signTokensPurged", result.SignTokensPurged))

	return result, nil
}

func purgeTokensInBatches(
	ctx context.Context,
	deadBefore int64,
	purge func(context.Context, repositories.PurgeDeadTokensRequest) (int64, error),
) (int64, error) {
	req := repositories.PurgeDeadTokensRequest{
		DeadBefore: deadBefore,
		Limit:      tokenPurgeBatchSize,
	}

	var total int64
	for {
		purged, err := purge(ctx, req)
		if err != nil {
			return total, err
		}
		total += purged
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, total)
		}
		if purged < int64(req.Limit) {
			return total, nil
		}
	}
}
