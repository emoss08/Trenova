package accountingsyncjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Connections services.AccountingConnectionService
	Logger      *zap.Logger
}

type Activities struct {
	connections services.AccountingConnectionService
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		connections: p.Connections,
		l:           p.Logger.Named("job.accounting-sync"),
	}
}

func (a *Activities) CheckAccountingConnectionsActivity(
	ctx context.Context,
) (*HealthSweepResult, error) {
	result := &HealthSweepResult{}
	for page := range healthMaxPages {
		sweep, err := a.connections.CheckDue(ctx, healthPageSize)
		if err != nil {
			return result, err
		}
		result.Pages = page + 1
		result.Listed += sweep.Listed
		result.Checked += sweep.Checked
		result.Failed += sweep.Failed
		activity.RecordHeartbeat(ctx, result.Pages)

		if sweep.Listed < healthPageSize || sweep.Checked == 0 {
			break
		}
	}

	if result.Failed > 0 {
		a.l.Warn("accounting connection checks failed",
			zap.Int("failed", result.Failed), zap.Int("checked", result.Checked))
	}
	return result, nil
}
