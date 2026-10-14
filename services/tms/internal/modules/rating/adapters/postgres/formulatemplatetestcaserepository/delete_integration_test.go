//go:build integration

package formulatemplatetestcaserepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDeleteRemovesOnlyTheScenarioOfThatTemplateAndTenant(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	conn := postgres.NewTestConnection(db)
	templates := formulatemplaterepository.New(formulatemplaterepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	template, err := templates.Create(ctx, formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Per Mile",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "totalDistance * 2",
		SchemaID:       "shipment",
		Status:         formulatemplate.StatusDraft,
	}.Build())
	require.NoError(t, err)

	scenario, err := repo.Create(ctx, &formulatemplate.TestCase{
		ID:             pulid.MustNew("ftc_"),
		TemplateID:     template.ID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Short haul",
		Variables:      map[string]any{"totalDistance": 100},
		ExpectedAmount: decimal.NewFromInt(200),
		Tolerance:      decimal.RequireFromString("0.01"),
		CreatedByID:    tenantInfo.UserID,
	})
	require.NoError(t, err)

	wrongTemplate := formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: tenantInfo,
		TestCaseID: scenario.ID,
		TemplateID: pulid.MustNew("ft_"),
	}
	err = repo.Delete(ctx, wrongTemplate)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	otherTenant := formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		TestCaseID: scenario.ID,
		TemplateID: template.ID,
	}
	err = repo.Delete(ctx, otherTenant)
	require.Error(t, err)

	own := formulatemplateports.GetTestCaseByIDRequest{
		TenantInfo: tenantInfo,
		TestCaseID: scenario.ID,
		TemplateID: template.ID,
	}
	_, err = repo.GetByID(ctx, own)
	require.NoError(t, err, "a refused delete leaves the scenario in place")

	require.NoError(t, repo.Delete(ctx, own))
	_, err = repo.GetByID(ctx, own)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
