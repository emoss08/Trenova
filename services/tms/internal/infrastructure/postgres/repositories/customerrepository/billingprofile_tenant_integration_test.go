//go:build integration

package customerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func ensureBillingProfile(
	t *testing.T,
	db *bun.DB,
	cus *customer.Customer,
) {
	t.Helper()

	count, err := db.NewSelect().
		TableExpr("customer_billing_profiles").
		Where("customer_id = ?", cus.ID).
		Count(t.Context())
	require.NoError(t, err)
	if count > 0 {
		return
	}

	_, err = db.NewInsert().Model(&customer.CustomerBillingProfile{
		ID:             pulid.MustNew("cbp_"),
		BusinessUnitID: cus.BusinessUnitID,
		OrganizationID: cus.OrganizationID,
		CustomerID:     cus.ID,
	}).Exec(t.Context())
	require.NoError(t, err)
}

func TestGetBillingProfile_StaysInsideTheTenant(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	db.RegisterModel(domainregistry.RegisterManyToManyEntities()...)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "CB")
	infoA := pagination.TenantInfo{OrgID: tenantA.Organization.ID, BuID: tenantA.BusinessUnit.ID}
	infoB := pagination.TenantInfo{OrgID: tenantB.Organization.ID, BuID: tenantB.BusinessUnit.ID}

	fixtureB := testutil.SeedShipmentIntegrationFixture(t, ctx, db, tenantB, infoB)
	ensureBillingProfile(t, db, fixtureB.Customer)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	_, err := repo.GetBillingProfile(ctx, repositories.GetCustomerBillingProfileRequest{
		CustomerID: fixtureB.Customer.ID,
		TenantInfo: infoA,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	profile, err := repo.GetBillingProfile(ctx, repositories.GetCustomerBillingProfileRequest{
		CustomerID: fixtureB.Customer.ID,
		TenantInfo: infoB,
	})
	require.NoError(t, err)
	assert.Equal(t, fixtureB.Customer.ID, profile.CustomerID)
}
