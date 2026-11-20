package entitlementtest

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/shared/pulid"
)

func Instance() entitlementports.Filter {
	return instanceFilter{}
}

type instanceFilter struct{}

func (instanceFilter) Organizations(
	_ context.Context,
	_ keel.Col,
	need entitlement.Need,
) (keel.Pred, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	return keel.True(), nil
}

func (instanceFilter) ForWork(context.Context, keel.Col) (keel.Pred, error) {
	return keel.True(), nil
}

func (instanceFilter) Keep(
	_ context.Context,
	need entitlement.Need,
	organizations []pulid.ID,
) ([]pulid.ID, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	return organizations, nil
}
