package deskscheduleresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/modules/agent/app/conversationscheduleservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (d *Deps) schedulesReady() error {
	if d.ConversationScheduleService != nil {
		return nil
	}

	return errortypes.NewBusinessError("Scheduled requests are not available")
}

func (r *MutationResolver) scheduleRequest(
	ctx context.Context,
	operation permission.Operation,
	id string,
) (conversationscheduleservice.ScheduleRequest, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, operation)
	if err != nil {
		return conversationscheduleservice.ScheduleRequest{}, err
	}
	if err = r.schedulesReady(); err != nil {
		return conversationscheduleservice.ScheduleRequest{}, err
	}
	scheduleID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return conversationscheduleservice.ScheduleRequest{}, err
	}

	return conversationscheduleservice.ScheduleRequest{
		ScheduleID: scheduleID,
		Actor:      *actorutil.FromAuthContext(authCtx),
	}, nil
}

func scheduleEdit(
	input *gqlmodel.UpdateAssistantScheduleInput,
) (*conversationschedule.Edit, error) {
	if input.Prompt == nil && input.Repeat == nil && input.Time == nil {
		if input.Enabled == nil {
			return nil, errortypes.NewValidationError(
				"enabled", errortypes.ErrRequired, "Say whether the schedule should run",
			)
		}

		return nil, nil
	}

	multiErr := errortypes.NewMultiError()
	if input.Prompt == nil {
		multiErr.Add("prompt", errortypes.ErrRequired, "Say what to ask on each run")
	}
	if input.Repeat == nil {
		multiErr.Add("repeat", errortypes.ErrRequired, "Choose how often it runs")
	}
	if input.Time == nil {
		multiErr.Add("time", errortypes.ErrRequired, "Enter a time of day, such as 08:00")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &conversationschedule.Edit{
		Prompt: *input.Prompt,
		Repeat: *input.Repeat,
		Time:   *input.Time,
	}, nil
}
