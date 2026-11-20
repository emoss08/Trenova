package plantest

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/planfields"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/subscription"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func FreeDemo(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	status subscription.Status,
) *platformplan.ResolvedPlan {
	t.Helper()

	plan, err := platformplan.FreeDemo(nil)
	require.NoError(t, err)

	now := time.Now()
	sub := &subscription.Subscription{
		ID:             pulid.MustNew("osub_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         status,
		TrialEndsAt:    now.Add(24 * time.Hour).Unix(),
		ReadOnlyUntil:  now.Add(48 * time.Hour).Unix(),
		CreatedAt:      now.Add(-24 * time.Hour).Unix(),
	}

	return platformplan.NewManaged(plan, sub, now.Unix())
}

func Unmanaged(tenantInfo pagination.TenantInfo) *platformplan.ResolvedPlan {
	return platformplan.NewUnmanaged(
		platformplan.Unlimited(),
		platformplan.OriginInternal,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		time.Now().Unix(),
	)
}

type Gate struct {
	off map[platformcatalog.FeatureKey]struct{}
}

var _ platformcatalogports.FeatureGate = (*Gate)(nil)

func AllFeatures() *Gate {
	return &Gate{off: map[platformcatalog.FeatureKey]struct{}{}}
}

func Without(features ...platformcatalog.FeatureKey) *Gate {
	off := make(map[platformcatalog.FeatureKey]struct{}, len(features))
	for _, feature := range features {
		off[feature] = struct{}{}
	}

	return &Gate{off: off}
}

// BaseOnly is a plan of the base features alone: every add-on's feature is
// off.
func BaseOnly() *Gate {
	var addOns []platformcatalog.FeatureKey
	for _, feature := range platformcatalog.Features() {
		if !feature.IsBase() {
			addOns = append(addOns, feature.Key)
		}
	}

	return Without(addOns...)
}

func (g *Gate) Enabled(_ context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	_, off := g.off[feature]

	return !off, nil
}

func (g *Gate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return g.Enabled(ctx, feature)
}

// Organizations is a shared server's gate: an organization it was told about
// has every feature except those taken from it, one it was never told about
// has nothing (the control plane has sent it no plan), and a question asked
// with no tenant in the context is refused, as a shared server refuses it.
type Organizations struct {
	off map[pulid.ID]map[platformcatalog.FeatureKey]struct{}
}

var _ platformcatalogports.FeatureGate = (*Organizations)(nil)

func SharedServer() *Organizations {
	return &Organizations{off: map[pulid.ID]map[platformcatalog.FeatureKey]struct{}{}}
}

func (g *Organizations) Planned(organizationIDs ...pulid.ID) *Organizations {
	for _, organizationID := range organizationIDs {
		if g.off[organizationID] == nil {
			g.off[organizationID] = map[platformcatalog.FeatureKey]struct{}{}
		}
	}
	return g
}

func (g *Organizations) Without(
	organizationID pulid.ID,
	features ...platformcatalog.FeatureKey,
) *Organizations {
	off := g.off[organizationID]
	if off == nil {
		off = make(map[platformcatalog.FeatureKey]struct{}, len(features))
		g.off[organizationID] = off
	}
	for _, feature := range features {
		off[feature] = struct{}{}
	}
	return g
}

func (g *Organizations) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || tenant.OrganizationID.IsNil() {
		return false, managedscope.ErrNoTenantScope
	}
	off, planned := g.off[tenant.OrganizationID]
	if !planned {
		return false, nil
	}
	_, withheld := off[feature]
	return !withheld, nil
}

func (g *Organizations) EnabledAnywhere(
	context.Context,
	platformcatalog.FeatureKey,
) (bool, error) {
	return true, nil
}

// AllOn is the plan lookup over the process's registry with every feature on.
func AllOn() *onplan.Lookup {
	return onplan.New(permissioncatalog.Default(), AllFeatures())
}

// Views is the reference views over a registry, or the process's when nil,
// with every feature on.
func Views(registry *permissioncatalog.Registry) *refview.Views {
	if registry == nil {
		registry = permissioncatalog.NewRegistry()
	}
	return refview.New(registry, onplan.New(registry, AllFeatures()))
}

// Keeper is the plan field keeper over a registry, or the process's when nil,
// with every feature on: it keeps nothing back.
func Keeper(registry *permissioncatalog.Registry) *planfields.Keeper {
	if registry == nil {
		registry = permissioncatalog.Default()
	}
	return planfields.New(registry, onplan.New(registry, AllFeatures()))
}

func Restriction(feature platformcatalog.FeatureKey) error {
	return platformcatalogports.RequireFeature(context.Background(), Without(feature), feature)
}

func Cloud(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	resolved *platformplan.ResolvedPlan,
) *platformplanportsmock.MockPlanService {
	t.Helper()

	plans := platformplanportsmock.NewMockPlanService(t)
	plans.EXPECT().EnforcesPlans().Return(true).Maybe()
	plans.EXPECT().
		Resolve(mock.Anything, tenantInfo.OrgID, tenantInfo.BuID).
		Return(resolved, nil).
		Maybe()

	return plans
}

type PricingTemplates struct {
	Off map[pulid.ID]struct{}
}

var _ permissionports.PricingTemplatePlan = (*PricingTemplates)(nil)

func AllPricingTemplates() *PricingTemplates {
	return &PricingTemplates{Off: map[pulid.ID]struct{}{}}
}

func OwnTemplatesOffPlan(ids ...pulid.ID) *PricingTemplates {
	off := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		off[id] = struct{}{}
	}

	return &PricingTemplates{Off: off}
}

func (p *PricingTemplates) OffPlan(
	_ context.Context,
	_ pagination.TenantInfo,
	templateIDs ...pulid.ID,
) (map[pulid.ID]struct{}, error) {
	off := make(map[pulid.ID]struct{}, len(templateIDs))
	for _, id := range templateIDs {
		if _, withheld := p.Off[id]; withheld {
			off[id] = struct{}{}
		}
	}

	return off, nil
}

func (p *PricingTemplates) RefuseOffPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs ...pulid.ID,
) error {
	off, _ := p.OffPlan(ctx, tenantInfo, templateIDs...)
	if len(off) == 0 {
		return nil
	}

	return Restriction(platformcatalog.FeatureRating)
}
