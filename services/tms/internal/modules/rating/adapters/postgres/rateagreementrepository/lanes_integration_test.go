//go:build integration

package rateagreementrepository

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type laneFixture struct {
	tenantInfo pagination.TenantInfo
	agreement  *rateagreement.RateAgreement
	templateID pulid.ID
}

func seedLaneFixture(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	data *seedtest.TestData,
	code string,
) *laneFixture {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	cust := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		StateID:        data.State.ID,
		Code:           code,
		Name:           "Lane Test Customer " + code,
		AddressLine1:   "1 Lane Way",
		City:           "Atlanta",
		PostalCode:     "30303",
	}
	_, err := db.NewInsert().Model(cust).Exec(ctx)
	require.NoError(t, err)

	agreement := &rateagreement.RateAgreement{
		ID:             pulid.MustNew("ragr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PartyType:      rateagreement.PartyTypeCustomer,
		CustomerID:     &cust.ID,
		Code:           code,
		Name:           "Lane Test Agreement " + code,
		AgreementType:  rateagreement.AgreementTypeContract,
		Status:         rateagreement.StatusActive,
		Currency:       "USD",
		EffectiveFrom:  100,
	}
	_, err = db.NewInsert().Model(agreement).Exec(ctx)
	require.NoError(t, err)

	template := &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Per Mile " + code,
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "baseRate * totalDistance",
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}
	_, err = db.NewInsert().Model(template).Exec(ctx)
	require.NoError(t, err)

	return &laneFixture{tenantInfo: tenantInfo, agreement: agreement, templateID: template.ID}
}

func (f *laneFixture) rule(
	mutate func(*rateagreement.RateAgreementRule),
) *rateagreement.RateAgreementRule {
	rule := &rateagreement.RateAgreementRule{
		ID:                    pulid.MustNew("rarl_"),
		OrganizationID:        f.tenantInfo.OrgID,
		BusinessUnitID:        f.tenantInfo.BuID,
		RateAgreementID:       f.agreement.ID,
		PartyType:             rateagreement.PartyTypeCustomer,
		PartyID:               *f.agreement.CustomerID,
		Status:                rateagreement.RuleStatusActive,
		OriginScopeType:       "State",
		OriginScopeValue:      "GA",
		DestinationScopeType:  "State",
		DestinationScopeValue: "FL",
		Direction:             rateagreement.DirectionDirectional,
		FormulaTemplateID:     &f.templateID,
		Rate:                  decimal.NewNullDecimal(decimal.RequireFromString("2.00")),
		EffectiveFrom:         100,
	}
	if mutate != nil {
		mutate(rule)
	}

	return rule
}

func insertRules(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	rules ...*rateagreement.RateAgreementRule,
) {
	t.Helper()

	for _, rule := range rules {
		_, err := db.NewInsert().Model(rule).Exec(ctx)
		require.NoError(t, err)
	}
}

func laneIDs(rules []*rateagreement.RateAgreementRule) []pulid.ID {
	ids := make([]pulid.ID, 0, len(rules))
	for _, rule := range rules {
		ids = append(ids, rule.ID)
	}

	return ids
}

// A card showing one agreement reads its lanes a page at a time. Only the
// lanes that price now or will — active and not closed out — are listed, most
// specific first with the id breaking ties, and the counts say how many live
// lanes there are and how many rules have been retired, without reading them.
func TestListLanes_PagesLiveLanesMostSpecificFirstAndCountsTheRest(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-PAGE")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	future := timeutils.NowUnix() + 1_000_000

	live := []*rateagreement.RateAgreementRule{
		fixture.rule(func(r *rateagreement.RateAgreementRule) {
			r.HazmatOnly = true
			r.TempControlOnly = true
		}),
		fixture.rule(func(r *rateagreement.RateAgreementRule) { r.HazmatOnly = true }),
		fixture.rule(func(r *rateagreement.RateAgreementRule) {
			r.TempControlOnly = true
			r.EffectiveFrom = future
		}),
		fixture.rule(nil),
		fixture.rule(func(r *rateagreement.RateAgreementRule) { r.EffectiveTo = &future }),
	}
	inactive := fixture.rule(func(r *rateagreement.RateAgreementRule) {
		r.Status = rateagreement.RuleStatusInactive
	})
	closedOutAt := int64(200)
	closedOut := fixture.rule(func(r *rateagreement.RateAgreementRule) {
		r.EffectiveTo = &closedOutAt
	})

	insertRules(t, ctx, db, live...)
	insertRules(t, ctx, db, inactive, closedOut)

	expected := slices.Clone(live)
	slices.SortStableFunc(expected, func(a, b *rateagreement.RateAgreementRule) int {
		return cmp.Or(
			cmp.Compare(b.SpecificityScore, a.SpecificityScore),
			cmp.Compare(a.ID.String(), b.ID.String()),
		)
	})
	expectedIDs := laneIDs(expected)

	breaks := []*rateagreement.RateAgreementRuleBreak{
		{
			OrganizationID:      fixture.tenantInfo.OrgID,
			BusinessUnitID:      fixture.tenantInfo.BuID,
			RateAgreementRuleID: expected[0].ID,
			FromWeight:          decimal.Zero,
			ToWeight:            decimal.NewNullDecimal(decimal.NewFromInt(5_000)),
			Rate:                decimal.RequireFromString("2.50"),
		},
		{
			OrganizationID:      fixture.tenantInfo.OrgID,
			BusinessUnitID:      fixture.tenantInfo.BuID,
			RateAgreementRuleID: expected[0].ID,
			FromWeight:          decimal.NewFromInt(5_000),
			Rate:                decimal.RequireFromString("2.25"),
			SortOrder:           1,
		},
	}
	_, err := db.NewInsert().Model(&breaks).Exec(ctx)
	require.NoError(t, err)

	request := func(limit, offset int) *rateagreementports.ListRateAgreementLanesRequest {
		return &rateagreementports.ListRateAgreementLanesRequest{
			TenantInfo:      fixture.tenantInfo,
			RateAgreementID: fixture.agreement.ID,
			Limit:           limit,
			Offset:          offset,
		}
	}

	first, err := repo.ListLanes(ctx, request(2, 0))
	require.NoError(t, err)
	assert.Equal(t, expectedIDs[:2], laneIDs(first.Lanes))
	assert.Equal(t, 5, first.Count, "the inactive and closed-out rules are not live")
	assert.Equal(t, 2, first.Retired, "one inactive rule and one closed-out rule")
	assert.Equal(t, 2, first.Limit)
	assert.Equal(t, 0, first.Offset)
	require.Len(t, first.Lanes[0].Breaks, 2, "each lane carries its breaks")
	assert.Empty(t, first.Lanes[1].Breaks)

	second, err := repo.ListLanes(ctx, request(2, 2))
	require.NoError(t, err)
	assert.Equal(t, expectedIDs[2:4], laneIDs(second.Lanes))

	last, err := repo.ListLanes(ctx, request(2, 4))
	require.NoError(t, err)
	assert.Equal(t, expectedIDs[4:], laneIDs(last.Lanes))
	assert.Equal(t, 5, last.Count)
	assert.Equal(t, 2, last.Retired)

	past, err := repo.ListLanes(ctx, request(2, 10))
	require.NoError(t, err)
	require.NotNil(t, past.Lanes, "an empty page still serializes as a list")
	assert.Empty(t, past.Lanes)
	assert.Equal(t, 5, past.Count)
	assert.Equal(t, 2, past.Retired)

	defaulted, err := repo.ListLanes(ctx, request(0, -3))
	require.NoError(t, err)
	assert.Equal(t, rateagreementports.DefaultLaneLimit, defaulted.Limit)
	assert.Equal(t, 0, defaulted.Offset)
	assert.Equal(t, expectedIDs, laneIDs(defaulted.Lanes))

	capped, err := repo.ListLanes(ctx, request(500, 0))
	require.NoError(t, err)
	assert.Equal(t, rateagreementports.MaxLaneLimit, capped.Limit)
	assert.Len(t, capped.Lanes, 5)
}

// An agreement with no rules at all is still found: zero live lanes and zero
// retired, not an error.
func TestListLanes_AgreementWithoutRulesCountsZero(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-EMPTY")
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	page, err := repo.ListLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      fixture.tenantInfo,
		RateAgreementID: fixture.agreement.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, page.Lanes)
	assert.Zero(t, page.Count)
	assert.Zero(t, page.Retired)
}

// Another tenant's agreement reads as not found, exactly like an id that does
// not exist, rather than as an agreement with no lanes.
func TestListLanes_AgreementOutsideTheTenantIsNotFound(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "LN")
	fixture := seedLaneFixture(t, ctx, db, data, "LN-TENANT")
	insertRules(t, ctx, db, fixture.rule(nil))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	_, err := repo.ListLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: other.Organization.ID,
			BuID:  other.BusinessUnit.ID,
		},
		RateAgreementID: fixture.agreement.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "got %v", err)

	_, err = repo.ListLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      fixture.tenantInfo,
		RateAgreementID: pulid.MustNew("ragr_"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "got %v", err)
}

// A reader that pages lanes separately loads the agreement without its rules
// but keeps everything else a child load brings: the accessorial schedule and
// the fuel binding.
func TestGetByID_OmitRulesKeepsAccessorialsAndFuelBinding(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	fixture := seedLaneFixture(t, ctx, db, data, "LN-OMIT")
	insertRules(t, ctx, db, fixture.rule(nil))

	charge := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		OrganizationID: fixture.tenantInfo.OrgID,
		BusinessUnitID: fixture.tenantInfo.BuID,
		Code:           "FUELOMIT",
		Description:    "Fuel surcharge",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(0),
	}
	detention := &accessorialcharge.AccessorialCharge{
		ID:             pulid.MustNew("acc_"),
		OrganizationID: fixture.tenantInfo.OrgID,
		BusinessUnitID: fixture.tenantInfo.BuID,
		Code:           "DETOMIT",
		Description:    "Detention",
		Method:         accessorialcharge.MethodFlat,
		Amount:         decimal.NewFromInt(60),
	}
	_, err := db.NewInsert().
		Model(&[]*accessorialcharge.AccessorialCharge{charge, detention}).
		Exec(ctx)
	require.NoError(t, err)

	index := &fuelsurcharge.FuelIndex{
		OrganizationID: fixture.tenantInfo.OrgID,
		BusinessUnitID: fixture.tenantInfo.BuID,
		Name:           "Omit Rules Index",
		Code:           "OMITIDX",
		Source:         fuelsurcharge.IndexSourceCustom,
		Currency:       "USD",
		IsActive:       true,
	}
	_, err = db.NewInsert().Model(index).Exec(ctx)
	require.NoError(t, err)

	program := &fuelsurcharge.FuelSurchargeProgram{
		OrganizationID:       fixture.tenantInfo.OrgID,
		BusinessUnitID:       fixture.tenantInfo.BuID,
		Name:                 "Omit Rules Program",
		Code:                 "OMITFSP",
		FuelIndexID:          index.ID,
		AccessorialChargeID:  charge.ID,
		Method:               fuelsurcharge.ProgramMethodPerMileStep,
		PegPrice:             decimal.NewNullDecimal(decimal.RequireFromString("1.20")),
		Increment:            decimal.NewNullDecimal(decimal.RequireFromString("0.05")),
		IncrementRate:        decimal.NewNullDecimal(decimal.RequireFromString("0.01")),
		RatePrecision:        4,
		DateBasis:            fuelsurcharge.DateBasisPickupDate,
		PriceEffectiveDay:    3,
		MissingPriceFallback: fuelsurcharge.FallbackUseLatestAvailable,
		Status:               fuelsurcharge.ProgramStatusActive,
	}
	_, err = db.NewInsert().Model(program).Exec(ctx)
	require.NoError(t, err)

	accessorial := &rateagreement.RateAgreementAccessorial{
		OrganizationID:      fixture.tenantInfo.OrgID,
		BusinessUnitID:      fixture.tenantInfo.BuID,
		RateAgreementID:     fixture.agreement.ID,
		AccessorialChargeID: detention.ID,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(75),
	}
	_, err = db.NewInsert().Model(accessorial).Exec(ctx)
	require.NoError(t, err)

	binding := &rateagreement.RateAgreementFuelBinding{
		OrganizationID:         fixture.tenantInfo.OrgID,
		BusinessUnitID:         fixture.tenantInfo.BuID,
		RateAgreementID:        fixture.agreement.ID,
		FuelSurchargeProgramID: program.ID,
	}
	_, err = db.NewInsert().Model(binding).Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	withoutRules, err := repo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: fixture.agreement.ID,
		TenantInfo:      fixture.tenantInfo,
		IncludeChildren: true,
		OmitRules:       true,
	})
	require.NoError(t, err)
	assert.Empty(t, withoutRules.Rules)
	require.Len(t, withoutRules.Accessorials, 1)
	assert.Equal(t, detention.ID, withoutRules.Accessorials[0].AccessorialChargeID)
	require.NotNil(t, withoutRules.FuelBinding)
	assert.Equal(t, program.ID, withoutRules.FuelBinding.FuelSurchargeProgramID)

	withRules, err := repo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: fixture.agreement.ID,
		TenantInfo:      fixture.tenantInfo,
		IncludeChildren: true,
	})
	require.NoError(t, err)
	assert.Len(t, withRules.Rules, 1, "the zero value still loads the rules")
	assert.Len(t, withRules.Accessorials, 1)
	assert.NotNil(t, withRules.FuelBinding)

	headerOnly, err := repo.GetByID(ctx, &rateagreementports.GetRateAgreementByIDRequest{
		RateAgreementID: fixture.agreement.ID,
		TenantInfo:      fixture.tenantInfo,
	})
	require.NoError(t, err)
	assert.Empty(t, headerOnly.Rules)
	assert.Empty(t, headerOnly.Accessorials)
	assert.Nil(t, headerOnly.FuelBinding)
}
