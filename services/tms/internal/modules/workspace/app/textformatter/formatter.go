package textformatter

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/dataentrycontrolports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Repo dataentrycontrolports.DataEntryControlRepository
}

type Formatter struct {
	repo dataentrycontrolports.DataEntryControlRepository
}

func New(p Params) dataentrycontrolports.TextFormatter {
	return &Formatter{repo: p.Repo}
}

func (f *Formatter) Format(ctx context.Context, entity dataentrycontrol.TextFormattable) error {
	ctrl, err := f.repo.GetOrCreate(ctx, entity.GetOrganizationID(), entity.GetBusinessUnitID())
	if err != nil {
		return err
	}

	entity.ApplyTextFormat(ctrl.TextFormat())

	return nil
}
