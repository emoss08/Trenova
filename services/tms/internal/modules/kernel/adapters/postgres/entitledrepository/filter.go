package entitledrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/managedfeaturesrepository"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures/managedfeaturesdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Scopes *managedscope.Resolver
	Gate   platformcatalogports.FeatureGate
}

type filter struct {
	db     *postgres.Connection
	scopes *managedscope.Resolver
	gate   platformcatalogports.FeatureGate
}

var _ entitlementports.Filter = (*filter)(nil)

func New(p Params) entitlementports.Filter {
	return &filter{db: p.DB, scopes: p.Scopes, gate: p.Gate}
}

func (f *filter) Organizations(
	ctx context.Context,
	organization keel.Col,
	need entitlement.Need,
) (keel.Pred, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	if !f.scopes.Shared() {
		return f.instanceAnswer(ctx, need)
	}
	return organization.InQuery(entitledScopes(need)), nil
}

func (f *filter) ForWork(ctx context.Context, organization keel.Col) (keel.Pred, error) {
	need, ok := entitlement.WorkFrom(ctx)
	if ok {
		return f.Organizations(ctx, organization, need)
	}
	if f.scopes.Shared() {
		return nil, entitlement.ErrNoWork
	}
	return keel.True(), nil
}

func (f *filter) Keep(
	ctx context.Context,
	need entitlement.Need,
	organizations []pulid.ID,
) ([]pulid.ID, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	if len(organizations) == 0 {
		return []pulid.ID{}, nil
	}
	if !f.scopes.Shared() {
		entitled, err := f.instanceEntitled(ctx, need)
		if err != nil || !entitled {
			return []pulid.ID{}, err
		}
		return organizations, nil
	}

	keys := make([]string, 0, len(organizations))
	for _, organization := range organizations {
		keys = append(keys, organization.String())
	}

	ctx = dbscope.WithSystem(
		ctx,
		"find which of a page of organizations a shared server's plans entitle to background work",
	)
	return dbtx.Read(ctx, f.db, func(ctx context.Context) ([]pulid.ID, error) {
		s := managedfeaturesdb.States
		held, err := keel.Into[string](
			ctx,
			f.db.Keel(ctx),
			entitledScopes(need).Where(s.ScopeKey.In(keys...)).Limit(len(keys)),
		)
		if err != nil {
			return nil, fmt.Errorf("read the entitled organizations: %w", err)
		}

		entitled := make(map[string]struct{}, len(held))
		for _, key := range held {
			entitled[key] = struct{}{}
		}
		kept := make([]pulid.ID, 0, len(organizations))
		for _, organization := range organizations {
			if _, ok := entitled[organization.String()]; ok {
				kept = append(kept, organization)
			}
		}
		return kept, nil
	})
}

func (f *filter) instanceAnswer(ctx context.Context, need entitlement.Need) (keel.Pred, error) {
	entitled, err := f.instanceEntitled(ctx, need)
	if err != nil {
		return nil, err
	}
	if entitled {
		return keel.True(), nil
	}
	return keel.Not(keel.True()), nil
}

func (f *filter) instanceEntitled(ctx context.Context, need entitlement.Need) (bool, error) {
	feature, named := need.Feature()
	if !named {
		return true, nil
	}
	return f.gate.EnabledAnywhere(ctx, feature)
}

func entitledScopes(need entitlement.Need) keel.Query[*managedfeatures.State] {
	s := managedfeaturesdb.States
	feature, _ := need.Feature()
	return s.
		Pick(s.ScopeKey).
		Where(
			managedfeaturesrepository.ActivePlan(),
			s.ScopeKey.Ne(managedscope.InstanceKey),
			keel.When(
				need.AddOn(),
				s.Features.Contains(keel.JSONB([]platformcatalog.FeatureKey{feature})),
			),
		)
}
