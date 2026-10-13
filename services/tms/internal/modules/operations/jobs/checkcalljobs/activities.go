package checkcalljobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	PurgeTrackingLinksWorkflowName = "PurgeCarrierTrackingLinksWorkflow"

	linkRetentionDays  = 30
	linkPurgeBatchSize = 500
	secondsPerDay      = int64(24 * 60 * 60)
)

type ActivitiesParams struct {
	fx.In

	Links  checkcallports.TrackingLinkRepository
	Logger *zap.Logger
}

type Activities struct {
	links checkcallports.TrackingLinkRepository
	l     *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		links: p.Links,
		l:     p.Logger.Named("temporal.check-call-activities"),
	}
}

type PurgeTrackingLinksResult struct {
	LinksPurged int64 `json:"linksPurged"`
}

func PurgeTrackingLinksWorkflow(ctx workflow.Context) (*PurgeTrackingLinksResult, error) {
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
	result := new(PurgeTrackingLinksResult)
	if err := workflow.ExecuteActivity(ctx, a.PurgeDeadTrackingLinksActivity).
		Get(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Activities) PurgeDeadTrackingLinksActivity(
	ctx context.Context,
) (*PurgeTrackingLinksResult, error) {
	deadBefore := timeutils.NowUnix() - linkRetentionDays*secondsPerDay
	result := new(PurgeTrackingLinksResult)

	for {
		purged, err := a.links.PurgeDead(ctx, deadBefore, linkPurgeBatchSize)
		if err != nil {
			a.l.Error("failed to purge dead carrier tracking links",
				zap.Int64("linksPurged", result.LinksPurged),
				zap.Error(err))
			return nil, err
		}
		result.LinksPurged += int64(purged)
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, result.LinksPurged)
		}
		if purged < linkPurgeBatchSize {
			break
		}
	}

	a.l.Info("dead carrier tracking link purge completed",
		zap.Int64("linksPurged", result.LinksPurged))

	return result, nil
}
