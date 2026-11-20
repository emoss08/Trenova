package retrievaljobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type organizationPager struct {
	tenantports.TenantSyncRepository
	tenants []pagination.TenantInfo
	after   pulid.ID
	limit   int
}

func (p *organizationPager) ListTenantsPage(
	_ context.Context,
	_ entitlement.Need,
	after pulid.ID,
	limit int,
) ([]pagination.TenantInfo, error) {
	p.after = after
	p.limit = limit
	return p.tenants[:min(limit, len(p.tenants))], nil
}

func organizations(n int) []pagination.TenantInfo {
	tenants := make([]pagination.TenantInfo, 0, n)
	for range n {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		})
	}
	return tenants
}

func newListActivities(pager *organizationPager) *Activities {
	return NewActivities(ActivitiesParams{Tenants: pager, Logger: zap.NewNop()})
}

func TestListRetrievalOrganizationsActivity_PagesOverEveryOrganization(t *testing.T) {
	t.Parallel()

	pager := &organizationPager{tenants: organizations(3)}

	page, err := newListActivities(pager).ListRetrievalOrganizationsActivity(
		t.Context(),
		&ListOrganizationsInput{Need: entitlement.For(platformcatalog.FeatureAIDesk), Limit: 2},
	)
	require.NoError(t, err)

	assert.Equal(t, pulid.Nil, pager.after)
	assert.Equal(t, 3, pager.limit, "one extra row tells whether another page follows")
	assert.True(t, page.HasMore)
	require.Len(t, page.Tenants, 2)
	assert.Equal(t, pager.tenants[0].OrgID, page.Tenants[0].OrganizationID)
	assert.Equal(t, pager.tenants[1].BuID, page.Tenants[1].BusinessUnitID)
}

func TestListRetrievalOrganizationsActivity_ContinuesAfterTheCursor(t *testing.T) {
	t.Parallel()

	pager := &organizationPager{tenants: organizations(1)}
	cursor := temporaljobs.TenantWorkItem{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	page, err := newListActivities(pager).ListRetrievalOrganizationsActivity(
		t.Context(),
		&ListOrganizationsInput{
			Need:  entitlement.For(platformcatalog.FeatureAIDesk),
			After: &cursor,
			Limit: 2,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, cursor.OrganizationID, pager.after)
	assert.False(t, page.HasMore)
	assert.Len(t, page.Tenants, 1)
}

func TestListRetrievalOrganizationsActivity_DefaultsThePageSize(t *testing.T) {
	t.Parallel()

	pager := &organizationPager{}

	page, err := newListActivities(pager).ListRetrievalOrganizationsActivity(
		t.Context(),
		&ListOrganizationsInput{Need: entitlement.For(platformcatalog.FeatureAIDesk)},
	)
	require.NoError(t, err)

	assert.Equal(t, temporaljobs.DefaultOrganizationPageSize+1, pager.limit)
	assert.False(t, page.HasMore)
	assert.Empty(t, page.Tenants)
}

type sweepPipeline struct {
	retrievalports.RetrievalIndexPipeline
	result retrievalports.RetrievalSweepResult
	woken  []pagination.TenantInfo
}

func (p *sweepPipeline) Sweep(
	context.Context,
	pagination.TenantInfo,
) (retrievalports.RetrievalSweepResult, error) {
	return p.result, nil
}

func (p *sweepPipeline) Wake(_ context.Context, tenant pagination.TenantInfo) {
	p.woken = append(p.woken, tenant)
}

func TestSweepRetrievalSourcesActivity_WakesTheIndexerWhenTheRouteChanged(t *testing.T) {
	t.Parallel()

	pipeline := &sweepPipeline{result: retrievalports.RetrievalSweepResult{RoutingChanged: true}}
	activities := NewActivities(ActivitiesParams{Pipeline: pipeline, Logger: zap.NewNop()})
	tenant := temporaljobs.TenantWorkItem{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, err := activities.SweepRetrievalSourcesActivity(
		t.Context(),
		&OrganizationSweepInput{TenantWorkItem: tenant, Wake: true},
	)
	require.NoError(t, err)
	assert.Equal(t, []pagination.TenantInfo{tenant.TenantInfo()}, pipeline.woken,
		"an organization that never indexed starts once a model is routed to it")
}

func TestSweepRetrievalSourcesActivity_LeavesAnIdleOrganizationAlone(t *testing.T) {
	t.Parallel()

	pipeline := &sweepPipeline{}
	activities := NewActivities(ActivitiesParams{Pipeline: pipeline, Logger: zap.NewNop()})

	_, err := activities.SweepRetrievalSourcesActivity(
		t.Context(),
		&OrganizationSweepInput{
			TenantWorkItem: temporaljobs.TenantWorkItem{
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			},
			Wake: true,
		},
	)
	require.NoError(t, err)
	assert.Empty(
		t,
		pipeline.woken,
		"the sweep starts no workflow for an organization with nothing to do",
	)
}
