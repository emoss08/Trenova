package edipartnerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPartnersByIDsIsOneBoundedStatementInsideTheTenant(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, partnersByIDs(
		keeltest.TenantContext(t),
		edipartnerports.GetEDIPartnersByIDsRequest{PartnerIDs: []pulid.ID{"edip_1", "edip_2"}},
	))

	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, sql, `"organization_id" = `)
	assert.Contains(t, args, string(keeltest.TestOrganizationID))
}

func TestPartnersByIDsUnderASystemScopeStaysInsideTheRequestTenant(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithSystem(t.Context(), "test")
	sql, args := keeltest.RenderIn(t, ctx, partnersByIDs(
		ctx,
		edipartnerports.GetEDIPartnersByIDsRequest{
			TenantInfo: pagination.TenantInfo{OrgID: "org_9", BuID: "bu_9"},
			PartnerIDs: []pulid.ID{"edip_1", "edip_2"},
		},
	))

	assert.Contains(t, sql, "= ANY(")
	assert.Contains(t, sql, "LIMIT 2")
	assert.Contains(t, args, "org_9")
	assert.Contains(t, args, "bu_9")
}

func TestGetByIDsSkipsTheQueryForAnEmptyIDSet(t *testing.T) {
	t.Parallel()

	repo := &repository{db: postgres.NewTestConnection(keeltest.RenderDB(t)), l: zap.NewNop()}

	entities, err := repo.GetByIDs(
		keeltest.TenantContext(t),
		edipartnerports.GetEDIPartnersByIDsRequest{},
	)
	require.NoError(t, err)
	assert.Empty(t, entities)
}
