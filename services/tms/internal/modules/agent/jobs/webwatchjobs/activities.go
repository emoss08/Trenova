package webwatchjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/webwatchjobs/webwatchcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Runner agentextensionports.WebWatchRunner
	Waits  agentextensionports.WebWatchWaits
}

type Activities struct {
	runner agentextensionports.WebWatchRunner
	waits  agentextensionports.WebWatchWaits
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{runner: p.Runner, waits: p.Waits}
}

// CheckWebWatchActivity checks the watch once, hands what it found to the
// work waiting on it, and records it handed over only once that worked. A
// watch the vendor could no longer run tells the waiting work why it ended.
func (a *Activities) CheckWebWatchActivity(
	ctx context.Context,
	payload *webwatchcontract.Payload,
) (*webwatchcontract.CheckResult, error) {
	tenant := payload.Tenant()
	check, err := a.runner.CheckWebWatch(ctx, tenant, payload.WatchID)
	if err != nil {
		return nil, err
	}
	if check.Watch == nil {
		return &webwatchcontract.CheckResult{Ended: true}, nil
	}

	if len(check.Undelivered) > 0 {
		if err = a.waits.NotifyWebWatch(ctx, tenant, check.Watch, check.Undelivered); err != nil {
			return nil, err
		}
		if err = a.runner.MarkWebWatchDelivered(ctx, tenant, check.Undelivered); err != nil {
			return nil, err
		}
	}
	if check.Ended && check.Watch.Status != extensionwebwatch.WebWatchStatusEnded {
		a.waits.EndWebWatchWaits(ctx, tenant, check.Watch)
	}

	return &webwatchcontract.CheckResult{Ended: check.Ended}, nil
}
