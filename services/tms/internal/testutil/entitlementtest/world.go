package entitlementtest

import (
	"context"
	"sync"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

type World struct {
	Feature   platformcatalog.FeatureKey
	Entitled  dbscope.Tenant
	Lacking   dbscope.Tenant
	Suspended dbscope.Tenant
}

func NewWorld(feature platformcatalog.FeatureKey) *World {
	tenant := func() dbscope.Tenant {
		return dbscope.Tenant{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		}
	}
	return &World{
		Feature:   feature,
		Entitled:  tenant(),
		Lacking:   tenant(),
		Suspended: tenant(),
	}
}

func (w *World) Tenants() []dbscope.Tenant {
	return []dbscope.Tenant{w.Entitled, w.Lacking, w.Suspended}
}

func (w *World) baseFeature() bool {
	known, ok := platformcatalog.Lookup(w.Feature)
	return ok && known.IsBase()
}

func (w *World) has(organization pulid.ID, feature platformcatalog.FeatureKey) bool {
	switch organization {
	case w.Entitled.OrganizationID:
		return true
	case w.Lacking.OrganizationID:
		if w.baseFeature() {
			return false
		}
		return feature != w.Feature
	default:
		return false
	}
}

func (w *World) active(organization pulid.ID) bool {
	if organization == w.Entitled.OrganizationID {
		return true
	}
	return organization == w.Lacking.OrganizationID && !w.baseFeature()
}

func (w *World) entitled(organization pulid.ID, need entitlement.Need) bool {
	feature, named := need.Feature()
	if !named {
		return w.active(organization)
	}
	return w.has(organization, feature)
}

func (w *World) Outsider(organization pulid.ID) bool {
	return organization == w.Lacking.OrganizationID ||
		organization == w.Suspended.OrganizationID
}

func (w *World) Gate() platformcatalogports.FeatureGate {
	return worldGate{w: w}
}

type worldGate struct{ w *World }

func (g worldGate) Enabled(ctx context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || tenant.OrganizationID.IsNil() {
		return false, managedscope.ErrNoTenantScope
	}
	return g.w.has(tenant.OrganizationID, feature), nil
}

func (g worldGate) EnabledAnywhere(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return true, nil
}

func (w *World) Filter() entitlementports.Filter {
	return worldFilter{w: w}
}

type worldFilter struct{ w *World }

func (f worldFilter) Organizations(
	_ context.Context,
	_ keel.Col,
	need entitlement.Need,
) (keel.Pred, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	return keel.True(), nil
}

func (f worldFilter) ForWork(ctx context.Context, _ keel.Col) (keel.Pred, error) {
	if _, ok := entitlement.WorkFrom(ctx); !ok {
		return nil, entitlement.ErrNoWork
	}
	return keel.True(), nil
}

func (f worldFilter) Keep(
	_ context.Context,
	need entitlement.Need,
	organizations []pulid.ID,
) ([]pulid.ID, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	kept := make([]pulid.ID, 0, len(organizations))
	for _, organization := range organizations {
		if f.w.entitled(organization, need) {
			kept = append(kept, organization)
		}
	}
	return kept, nil
}

type refusals struct {
	mu    sync.Mutex
	count int
	names []string
}

func (r *refusals) RecordEntitlementRefusal(activityType string, _ platformcatalog.FeatureKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.count++
	r.names = append(r.names, activityType)
}

func (r *refusals) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.names...)
}
