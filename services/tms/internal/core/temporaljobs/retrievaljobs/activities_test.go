package retrievaljobs

import (
	"context"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type indexedTenantsPipeline struct {
	serviceports.RetrievalIndexPipeline
	tenants []pagination.TenantInfo
	after   *pagination.TenantInfo
	limit   int
}

func (p *indexedTenantsPipeline) ListIndexedTenants(
	_ context.Context,
	after *pagination.TenantInfo,
	limit int,
) ([]pagination.TenantInfo, error) {
	p.after = after
	p.limit = limit
	return p.tenants[:min(limit, len(p.tenants))], nil
}

func indexedTenants(n int) []pagination.TenantInfo {
	tenants := make([]pagination.TenantInfo, 0, n)
	for range n {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		})
	}
	return tenants
}

func TestListRetrievalOrganizationsActivity_PagesOverIndexedOrganizations(t *testing.T) {
	t.Parallel()

	pipeline := &indexedTenantsPipeline{tenants: indexedTenants(3)}
	activities := NewActivities(ActivitiesParams{Pipeline: pipeline, Logger: zap.NewNop()})

	page, err := activities.ListRetrievalOrganizationsActivity(
		t.Context(),
		&ListOrganizationsInput{Limit: 2},
	)
	require.NoError(t, err)

	assert.Nil(t, pipeline.after)
	assert.Equal(t, 3, pipeline.limit, "one extra row tells whether another page follows")
	assert.True(t, page.HasMore)
	require.Len(t, page.Tenants, 2)
	assert.Equal(t, pipeline.tenants[0].OrgID, page.Tenants[0].OrganizationID)
	assert.Equal(t, pipeline.tenants[1].BuID, page.Tenants[1].BusinessUnitID)
}

func TestListRetrievalOrganizationsActivity_ContinuesAfterTheCursor(t *testing.T) {
	t.Parallel()

	pipeline := &indexedTenantsPipeline{tenants: indexedTenants(1)}
	activities := NewActivities(ActivitiesParams{Pipeline: pipeline, Logger: zap.NewNop()})
	cursor := temporaljobs.TenantWorkItem{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	page, err := activities.ListRetrievalOrganizationsActivity(
		t.Context(),
		&ListOrganizationsInput{After: &cursor, Limit: 2},
	)
	require.NoError(t, err)

	require.NotNil(t, pipeline.after)
	assert.Equal(t, cursor.TenantInfo(), *pipeline.after)
	assert.False(t, page.HasMore)
	assert.Len(t, page.Tenants, 1)
}

func TestListRetrievalOrganizationsActivity_DefaultsThePageSize(t *testing.T) {
	t.Parallel()

	pipeline := &indexedTenantsPipeline{}
	activities := NewActivities(ActivitiesParams{Pipeline: pipeline, Logger: zap.NewNop()})

	page, err := activities.ListRetrievalOrganizationsActivity(t.Context(), &ListOrganizationsInput{})
	require.NoError(t, err)

	assert.Equal(t, temporaljobs.DefaultOrganizationPageSize+1, pipeline.limit)
	assert.False(t, page.HasMore)
	assert.Empty(t, page.Tenants)
}
