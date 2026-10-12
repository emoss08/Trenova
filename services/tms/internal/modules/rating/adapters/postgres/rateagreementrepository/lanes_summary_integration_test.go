//go:build integration

package rateagreementrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSummarizeLanes_GroupsLiveLanesByWhatPricesThem(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-SUM")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	flat := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: fixture.tenantInfo.OrgID,
		BusinessUnitID: fixture.tenantInfo.BuID,
		Name:           "Flat LN-SUM",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "baseRate",
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}
	_, err := db.NewInsert().Model(flat).Exec(ctx)
	require.NoError(t, err)

	rate := func(value string) decimal.NullDecimal {
		return decimal.NewNullDecimal(decimal.RequireFromString(value))
	}
	closedOutAt := int64(200)
	insertRules(t, ctx, db,
		fixture.rule(func(r *rateagreement.RateAgreementRule) { r.Rate = rate("1.50") }),
		fixture.rule(func(r *rateagreement.RateAgreementRule) { r.Rate = rate("2.75") }),
		fixture.rule(func(r *rateagreement.RateAgreementRule) { r.Rate = rate("2.10") }),
		fixture.rule(func(r *rateagreement.RateAgreementRule) {
			r.FormulaTemplateID = &flat.ID
			r.Rate = rate("950")
		}),
		fixture.rule(func(r *rateagreement.RateAgreementRule) {
			r.Status = rateagreement.RuleStatusInactive
			r.Rate = rate("9.99")
		}),
		fixture.rule(func(r *rateagreement.RateAgreementRule) {
			r.EffectiveTo = &closedOutAt
			r.Rate = rate("0.10")
		}),
	)

	summary, err := repo.SummarizeLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      fixture.tenantInfo,
		RateAgreementID: fixture.agreement.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, 4, summary.Count)
	require.Len(t, summary.Groups, 2)

	perMile := summary.Groups[0]
	require.NotNil(t, perMile.FormulaTemplateID)
	assert.Equal(t, fixture.templateID, *perMile.FormulaTemplateID)
	assert.Equal(t, "Per Mile LN-SUM", perMile.FormulaTemplateName)
	assert.Equal(t, 3, perMile.Lanes)
	assert.Equal(t, "1.5", perMile.MinRate.Decimal.String())
	assert.Equal(t, "2.75", perMile.MaxRate.Decimal.String())
	assert.Nil(t, perMile.RateMatrixID)

	flatGroup := summary.Groups[1]
	assert.Equal(t, "Flat LN-SUM", flatGroup.FormulaTemplateName)
	assert.Equal(t, 1, flatGroup.Lanes)
	assert.Equal(t, "950", flatGroup.MinRate.Decimal.String())
}

func TestSummarizeLanes_AgreementWithoutLanesCountsZero(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-SUM0")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	summary, err := repo.SummarizeLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      fixture.tenantInfo,
		RateAgreementID: fixture.agreement.ID,
	})
	require.NoError(t, err)
	assert.Zero(t, summary.Count)
	assert.Empty(t, summary.Groups)
	assert.NotNil(t, summary.Groups)
}

func TestSummarizeLanes_AgreementOutsideTheTenantIsNotFound(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-SUMX")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	outside := fixture.tenantInfo
	outside.OrgID = pulid.MustNew("org_")
	_, err := repo.SummarizeLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      outside,
		RateAgreementID: fixture.agreement.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
