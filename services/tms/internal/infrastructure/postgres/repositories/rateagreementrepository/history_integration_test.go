//go:build integration

package rateagreementrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A lane's history is why the amendment primitive never edits in place: after
// a GRI closes a rule out and inserts its successor, both must remain readable
// — the current read returning only the successor, the history read returning
// the whole lineage, newest first.
func TestListRules_HistoryKeepsTheClosedOutRule(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	billTo := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		StateID:        data.State.ID,
		Code:           "HISTCUST",
		Name:           "History Test Customer",
		AddressLine1:   "123 Main St",
		City:           "Atlanta",
		PostalCode:     "30303",
	}
	_, err := db.NewInsert().Model(billTo).Exec(ctx)
	require.NoError(t, err)

	agreement := &rateagreement.RateAgreement{
		ID:             pulid.MustNew("ragr_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		PartyType:      rateagreement.PartyTypeCustomer,
		CustomerID:     &billTo.ID,
		Code:           "HISTORY-TEST",
		Name:           "History Test Agreement",
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
		Name:           "Per Mile",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		Expression:     "baseRate * totalDistance",
		Status:         formulatemplate.StatusActive,
		SchemaID:       "shipment",
	}
	_, err = db.NewInsert().Model(template).Exec(ctx)
	require.NoError(t, err)

	original := &rateagreement.RateAgreementRule{
		ID:                    pulid.MustNew("rarl_"),
		OrganizationID:        tenantInfo.OrgID,
		BusinessUnitID:        tenantInfo.BuID,
		RateAgreementID:       agreement.ID,
		PartyType:             rateagreement.PartyTypeCustomer,
		PartyID:               billTo.ID,
		Label:                 "GA to FL",
		Status:                rateagreement.RuleStatusActive,
		OriginScopeType:       "State",
		OriginScopeValue:      "GA",
		DestinationScopeType:  "State",
		DestinationScopeValue: "FL",
		LaneKey:               "ST:GA>ST:FL",
		Direction:             rateagreement.DirectionDirectional,
		FormulaTemplateID:     &template.ID,
		Rate:                  decimal.NewNullDecimal(decimal.RequireFromString("2.00")),
		EffectiveFrom:         100,
	}
	_, err = db.NewInsert().Model(original).Exec(ctx)
	require.NoError(t, err)

	successorRate := decimal.RequireFromString("2.10")
	successor := *original
	successor.ID = pulid.Nil
	supersededID := original.ID
	successor.SupersedesRuleID = &supersededID
	successor.Rate = decimal.NewNullDecimal(successorRate)
	successor.EffectiveFrom = 0

	require.NoError(t, repo.AmendRules(ctx, &repositories.AmendRateAgreementRulesRequest{
		TenantInfo:      tenantInfo,
		RateAgreementID: agreement.ID,
		EffectiveFrom:   1_000,
		SupersededIDs:   []pulid.ID{original.ID},
		Rules:           []*rateagreement.RateAgreementRule{&successor},
	}))

	current, err := repo.ListRules(ctx, &repositories.ListRateAgreementRulesRequest{
		TenantInfo:      tenantInfo,
		RateAgreementID: agreement.ID,
		AsOf:            2_000,
	})
	require.NoError(t, err)
	require.Len(t, current, 1, "only the successor prices after the amendment")
	assert.True(t, successorRate.Equal(current[0].Rate.Decimal))

	history, err := repo.ListRules(ctx, &repositories.ListRateAgreementRulesRequest{
		TenantInfo:        tenantInfo,
		RateAgreementID:   agreement.ID,
		LaneKey:           "ST:GA>ST:FL",
		IncludeSuperseded: true,
	})
	require.NoError(t, err)
	require.Len(t, history, 2, "history keeps the closed-out rule")

	assert.True(t, successorRate.Equal(history[0].Rate.Decimal), "newest first")
	assert.True(t, decimal.RequireFromString("2.00").Equal(history[1].Rate.Decimal))
	require.NotNil(t, history[1].EffectiveTo, "the old rule's window closed at the amendment")
	assert.EqualValues(t, 1_000, *history[1].EffectiveTo)
	require.NotNil(t, history[0].SupersedesRuleID)
	assert.Equal(t, original.ID, *history[0].SupersedesRuleID)
}
