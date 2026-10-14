//go:build integration

package formulatemplaterepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func createTemplate(
	t *testing.T,
	ctx context.Context,
	repo formulatemplateports.FormulaTemplateRepository,
	tenantInfo pagination.TenantInfo,
	name string,
	status formulatemplate.Status,
) *formulatemplate.FormulaTemplate {
	t.Helper()

	created, err := repo.Create(ctx, formulatemplate.Seed{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           name,
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "totalDistance * 2",
		SchemaID:       "shipment",
		Status:         status,
	}.Build())
	require.NoError(t, err)
	return created
}

func TestCountUsagesCountsTheBillingFallback(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	template := createTemplate(t, ctx, repo, tenantInfo, "Fallback", formulatemplate.StatusActive)
	result, err := db.NewUpdate().
		TableExpr("billing_controls").
		Set("fallback_formula_template_id = ?", template.ID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	require.NoError(t, err)
	if updated, _ := result.RowsAffected(); updated == 0 {
		_, err = db.NewInsert().
			Model(&tenant.BillingControl{
				OrganizationID:            tenantInfo.OrgID,
				BusinessUnitID:            tenantInfo.BuID,
				FallbackFormulaTemplateID: &template.ID,
			}).
			Exec(ctx)
		require.NoError(t, err)
	}

	usage, err := repo.CountUsages(ctx, &formulatemplateports.GetTemplateUsageRequest{
		TemplateID: template.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)
	assert.True(t, usage.InUse, "the organization's fallback rates every uncovered shipment")
	assert.Contains(t, usage.Usages, formulatemplateports.TemplateUsageCount{
		Type:  "billing_control_fallback",
		Count: 1,
	})
}

func TestBulkUpdateStatusRefusesATemplateThatLeftActiveAndBumpsTheVersion(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	active := createTemplate(t, ctx, repo, tenantInfo, "Active", formulatemplate.StatusActive)
	inReview := createTemplate(t, ctx, repo, tenantInfo, "In review", formulatemplate.StatusInReview)

	_, err := repo.BulkUpdateStatus(ctx, &formulatemplateports.BulkUpdateFormulaTemplateStatusRequest{
		TenantInfo:  tenantInfo,
		TemplateIDs: []pulid.ID{active.ID, inReview.ID},
		Status:      formulatemplate.StatusInactive,
	})
	require.Error(t, err, "a template that went into review after the check is not archived")

	updated, err := repo.BulkUpdateStatus(ctx, &formulatemplateports.BulkUpdateFormulaTemplateStatusRequest{
		TenantInfo:  tenantInfo,
		TemplateIDs: []pulid.ID{active.ID},
		Status:      formulatemplate.StatusInactive,
	})
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, formulatemplate.StatusInactive, updated[0].Status)
	assert.Equal(t, active.Version+1, updated[0].Version)

	var stored formulatemplate.FormulaTemplate
	require.NoError(t, db.NewSelect().Model(&stored).Where("id = ?", inReview.ID).Scan(ctx))
	assert.Equal(t, formulatemplate.StatusInReview, stored.Status)
}
