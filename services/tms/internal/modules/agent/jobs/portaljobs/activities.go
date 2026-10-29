package portaljobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/portaljobs/portalcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Finisher agentextensionports.PortalRunFinisher
	Waits    agentextensionports.PortalRunWaits
}

type Activities struct {
	finisher agentextensionports.PortalRunFinisher
	waits    agentextensionports.PortalRunWaits
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{finisher: p.Finisher, waits: p.Waits}
}

func (a *Activities) PollPortalRunActivity(
	ctx context.Context,
	payload *portalcontract.Payload,
) (*portalcontract.PollResult, error) {
	poll, err := a.finisher.PollPortalRun(ctx, payload.Tenant(), payload.RunID)
	if err != nil {
		return nil, err
	}
	if poll.CodeRequested {
		a.waits.AnnouncePortalRun(ctx, payload.Tenant(), poll.Run)
	}

	return &portalcontract.PollResult{
		Finished:     poll.Run.Status.Finished(),
		AwaitingCode: poll.Run.Status == extensionportal.PortalRunStatusAwaitingCode,
	}, nil
}

func (a *Activities) StopPortalRunActivity(
	ctx context.Context,
	in *portalcontract.StopInput,
) error {
	_, err := a.finisher.StopPortalRun(
		ctx,
		in.Payload.Tenant(),
		in.Payload.RunID,
		in.Status,
		in.Reason,
	)

	return err
}

func (a *Activities) NotifyPortalRunActivity(
	ctx context.Context,
	payload *portalcontract.Payload,
) error {
	run, err := a.finisher.PortalRun(ctx, payload.Tenant(), payload.RunID)
	if err != nil {
		return err
	}
	a.waits.NotifyPortalRun(ctx, payload.Tenant(), run)

	return nil
}
