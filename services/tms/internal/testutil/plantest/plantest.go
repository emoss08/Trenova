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
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
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
