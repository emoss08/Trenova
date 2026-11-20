package entitlementports

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/shared/pulid"
)

type Filter interface {
	Organizations(
		ctx context.Context,
		organization keel.Col,
		need entitlement.Need,
	) (keel.Pred, error)
	ForWork(ctx context.Context, organization keel.Col) (keel.Pred, error)
	Keep(ctx context.Context, need entitlement.Need, organizations []pulid.ID) ([]pulid.ID, error)
}
