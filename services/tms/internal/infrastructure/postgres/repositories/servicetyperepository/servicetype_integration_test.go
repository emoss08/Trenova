//go:build integration

package servicetyperepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateReturnsVersionMismatchForStaleEntity(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	created, err := repo.Create(ctx, &servicetype.ServiceType{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         domaintypes.StatusActive,
		Code:           "LTL",
		Description:    "Less-than-truckload",
		Color:          "#111111",
	})
	require.NoError(t, err)

	firstCopy, err := repo.GetByID(ctx, repositories.GetServiceTypeByIDRequest{
		ID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
	})
	require.NoError(t, err)

	secondCopy, err := repo.GetByID(ctx, repositories.GetServiceTypeByIDRequest{
		ID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
	})
	require.NoError(t, err)

	firstCopy.Description = "Updated first copy"
	_, err = repo.Update(ctx, firstCopy)
	require.NoError(t, err)

	secondCopy.Description = "Stale update"
	updated, err := repo.Update(ctx, secondCopy)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrVersionMismatch, validationErr.Code)
}

func TestBulkUpdateStatusReturnsVersionMismatchWhenTenantDoesNotMatch(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	created, err := repo.Create(ctx, &servicetype.ServiceType{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		Status:         domaintypes.StatusActive,
		Code:           "FTL",
		Description:    "Full truckload",
		Color:          "#222222",
	})
	require.NoError(t, err)

	otherTenant := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	entities, err := repo.BulkUpdateStatus(ctx, &repositories.BulkUpdateServiceTypeStatusRequest{
		TenantInfo:     otherTenant,
		ServiceTypeIDs: []pulid.ID{created.ID},
		Status:         domaintypes.StatusInactive,
	})

	require.Nil(t, entities)
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrVersionMismatch, validationErr.Code)
}
