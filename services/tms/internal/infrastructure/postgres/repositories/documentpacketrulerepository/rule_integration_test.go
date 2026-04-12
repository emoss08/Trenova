//go:build integration

package documentpacketrulerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documenttyperepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDocumentPacketRuleRepositoryCRUD_Integration(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.NewTestConnection(db)
	repo := New(Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	docTypeRepo := documenttyperepository.New(documenttyperepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})

	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	docType, err := docTypeRepo.Create(ctx, &documenttype.DocumentType{
		OrganizationID:         data.Organization.ID,
		BusinessUnitID:         data.BusinessUnit.ID,
		Code:                   "POD",
		Name:                   "Proof of Delivery",
		DocumentClassification: documenttype.ClassificationPublic,
		DocumentCategory:       documenttype.CategoryShipment,
	})
	require.NoError(t, err)

	created, err := repo.Create(ctx, &documentpacketrule.DocumentPacketRule{
		OrganizationID:        data.Organization.ID,
		BusinessUnitID:        data.BusinessUnit.ID,
		ResourceType:          "Worker",
		DocumentTypeID:        docType.ID,
		Required:              true,
		AllowMultiple:         false,
		DisplayOrder:          15,
		ExpirationRequired:    true,
		ExpirationWarningDays: 45,
	})
	require.NoError(t, err)
	assert.Equal(t, docType.ID, created.DocumentTypeID)

	byID, err := repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.Equal(t, "Worker", byID.ResourceType)

	listed, err := repo.ListByResourceType(ctx, &repositories.ListDocumentPacketRulesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "Worker",
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)

	created.AllowMultiple = true
	created.DisplayOrder = 99
	created.ExpirationWarningDays = 10
	updated, err := repo.Update(ctx, created)
	require.NoError(t, err)
	assert.True(t, updated.AllowMultiple)
	assert.Equal(t, 99, updated.DisplayOrder)
	assert.Equal(t, 10, updated.ExpirationWarningDays)

	otherTenantListed, err := repo.ListByResourceType(ctx, &repositories.ListDocumentPacketRulesByResourceRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ResourceType: "Worker",
	})
	require.NoError(t, err)
	assert.Len(t, otherTenantListed, 0)

	err = repo.Delete(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.Error(t, err)
}
