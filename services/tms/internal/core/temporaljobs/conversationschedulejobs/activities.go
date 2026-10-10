package conversationschedulejobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/services/conversationscheduleservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// reconciler is what the reconcile activity asks of the schedules.
type reconciler interface {
	Reconcile(ctx context.Context) (*ReconcileResult, error)
}

type ActivitiesParams struct {
	fx.In

	Runner    conversationscheduleports.ConversationScheduleRunner
	Syncer    conversationscheduleports.ConversationScheduleSyncer
	Schedules *Schedules
	Logger    *zap.Logger
}

type Activities struct {
	runner     conversationscheduleports.ConversationScheduleRunner
	syncer     conversationscheduleports.ConversationScheduleSyncer
	reconciler reconciler
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runner:     p.Runner,
		syncer:     p.Syncer,
		reconciler: p.Schedules,
		l:          p.Logger.Named("conversation-schedule-activities"),
	}
}

// FireConversationScheduleActivity starts the turn one slot asks for. A
// schedule that no longer exists has its Temporal schedule removed here, so
// a schedule deleted with its conversation stops firing at its next slot
// rather than at the next reconcile.
func (a *Activities) FireConversationScheduleActivity(
	ctx context.Context,
	input *FireInput,
) (*conversationscheduleports.FireConversationScheduleResult, error) {
	if input == nil || input.Payload == nil || input.Payload.ScheduleID.IsNil() {
		return nil, temporal.NewNonRetryableApplicationError(
			"a scheduled run must name its schedule", "InvalidInput", nil,
		)
	}

	payload := input.Payload
	result, err := a.runner.Fire(ctx, conversationscheduleports.FireConversationScheduleRequest{
		ScheduleID: payload.ScheduleID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  payload.OrganizationID,
			BuID:   payload.BusinessUnitID,
			UserID: payload.UserID,
		},
		FiredAt: input.FiredAt,
	})
	if errors.Is(err, conversationscheduleservice.ErrConversationBusy) {
		return nil, temporal.NewApplicationError(
			"the conversation is busy with another reply", "ConversationBusy",
		)
	}
	if err != nil {
		return nil, err
	}

	if result.Gone {
		a.syncer.Remove(ctx, payload.ScheduleID)
	}
	if result.Skipped != "" {
		a.l.Info("scheduled run skipped",
			zap.String("schedule", payload.ScheduleID.String()),
			zap.String("reason", result.Skipped),
		)
	}

	return result, nil
}

// ReconcileConversationSchedulesActivity makes every conversation schedule's
// Temporal schedule match its row.
func (a *Activities) ReconcileConversationSchedulesActivity(
	ctx context.Context,
) (*ReconcileResult, error) {
	return a.reconciler.Reconcile(ctx)
}
