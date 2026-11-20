package managedusageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ReceiverParams struct {
	fx.In

	Usage     *Service
	Observers []managedusageports.StateObserver `group:"managed_state_observers"`
	Logger    *zap.Logger
}

type Receiver struct {
	usage     *Service
	observers []managedusageports.StateObserver
	l         *zap.Logger
}

var (
	_ managedusageports.Writer               = (*Receiver)(nil)
	_ managedusageports.OrganizationReadings = (*Receiver)(nil)
)

func NewReceiver(p ReceiverParams) *Receiver {
	return &Receiver{
		usage:     p.Usage,
		observers: p.Observers,
		l:         p.Logger.Named("managed-usage-receiver"),
	}
}

func (r *Receiver) Readings(
	ctx context.Context,
	scope managedscope.Scope,
	since int64,
) ([]Reading, error) {
	return r.usage.Readings(ctx, scope, since)
}

func (r *Receiver) ReadingsForOrganizations(
	ctx context.Context,
	organizationIDs []pulid.ID,
	since int64,
) (map[pulid.ID][]Reading, error) {
	return r.usage.ReadingsForOrganizations(ctx, organizationIDs, since)
}

func (r *Receiver) SetState(
	ctx context.Context,
	scope managedscope.Scope,
	state *managedusage.State,
) error {
	change, err := r.usage.SetState(ctx, scope, state)
	if err != nil {
		return err
	}
	for _, o := range r.observers {
		if err = o.StateChanged(ctx, change); err != nil {
			r.l.Warn("a usage state observer failed", zap.Error(err))
		}
	}
	return nil
}
