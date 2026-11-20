//go:build integration

package detentionrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedDetentionPolicies(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	codes ...string,
) []*detentionpolicy.DetentionPolicy {
	t.Helper()

	scoped := tenant.Ctx(ctx)
	charge := &accessorialcharge.AccessorialCharge{
		Code:        "DETENTION",
		Description: "Detention",
		Method:      accessorialcharge.MethodFlat,
		Amount:      decimal.NewFromInt(75),
	}
	require.NoError(t, accessorialchargedb.AccessorialCharges.Create(scoped, db, charge))

	policies := make([]*detentionpolicy.DetentionPolicy, 0, len(codes))
	for _, code := range codes {
		policy := &detentionpolicy.DetentionPolicy{
			Name:                code + " policy",
			Code:                code,
			AccessorialChargeID: charge.ID,
		}
		require.NoError(t, detentionpolicydb.DetentionPolicies.Create(scoped, db, policy))
		policies = append(policies, policy)
	}

	return policies
}

func TestGetByIDsReturnsOnlyTheTenantsPolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "DT")
	own := seedDetentionPolicies(t, ctx, db, tenantA, "SHIPPER", "RECEIVER")
	foreign := seedDetentionPolicies(t, ctx, db, tenantB, "FOREIGN")[0]

	repo := NewPolicyRepository(PolicyParams{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	policies, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&detentionports.GetDetentionPoliciesByIDsRequest{
			TenantInfo: tenantA.Info(),
			DetentionPolicyIDs: []pulid.ID{
				own[1].ID,
				foreign.ID,
				own[0].ID,
				pulid.MustNew("dtp_"),
			},
		},
	)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(policies))
	for _, policy := range policies {
		codes[policy.ID] = policy.Code
		assert.Empty(t, policy.Tiers, "a batch read loads no tiers")
	}
	assert.Equal(t, map[pulid.ID]string{own[0].ID: "SHIPPER", own[1].ID: "RECEIVER"}, codes)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		&detentionports.GetDetentionPoliciesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestGetResolutionCandidatesReadsTheShipmentsPoliciesOnce(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seedtest.SeedFullTestData(t, ctx, db)
	scoped := tenant.Ctx(ctx)

	customerID := pulid.MustNew("cus_")
	shipper := pulid.MustNew("loc_")
	consignee := pulid.MustNew("loc_")
	endsEarly := int64(50_000)
	startsLate := int64(500_000)

	policies := seedDetentionPolicies(
		t, ctx, db, tenant,
		"ANY", "CUSTOMER", "SHIPPER", "CUSTOMER-CONSIGNEE",
		"OTHER-CUSTOMER", "OTHER-FACILITY", "EXPIRED", "FUTURE", "DRAFT",
	)
	byCode := make(map[string]*detentionpolicy.DetentionPolicy, len(policies))
	for _, policy := range policies {
		byCode[policy.Code] = policy
		policy.Status = detentionpolicy.PolicyStatusActive
	}
	byCode["CUSTOMER"].CustomerID = &customerID
	byCode["SHIPPER"].LocationID = &shipper
	byCode["CUSTOMER-CONSIGNEE"].CustomerID = &customerID
	byCode["CUSTOMER-CONSIGNEE"].LocationID = &consignee
	otherCustomer := pulid.MustNew("cus_")
	byCode["OTHER-CUSTOMER"].CustomerID = &otherCustomer
	otherFacility := pulid.MustNew("loc_")
	byCode["OTHER-FACILITY"].LocationID = &otherFacility
	byCode["EXPIRED"].EffectiveEndDate = &endsEarly
	byCode["FUTURE"].EffectiveStartDate = &startsLate
	byCode["DRAFT"].Status = detentionpolicy.PolicyStatusDraft
	for _, policy := range policies {
		policy.SpecificityScore = policy.ComputeSpecificity()
		require.NoError(t, detentionpolicydb.DetentionPolicies.Save(scoped, db, policy))
	}

	repo := NewPolicyRepository(PolicyParams{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	candidates, err := repo.GetResolutionCandidates(
		scoped,
		&detentionports.GetResolutionCandidatesRequest{
			TenantInfo:  tenant.Info(),
			CustomerID:  customerID,
			LocationIDs: []pulid.ID{shipper, consignee},
			ArrivedFrom: 100_000,
			ArrivedTo:   300_000,
		},
	)
	require.NoError(t, err)

	codes := make([]string, 0, len(candidates))
	for _, policy := range candidates {
		codes = append(codes, policy.Code)
	}
	assert.ElementsMatch(t, []string{"ANY", "CUSTOMER", "SHIPPER", "CUSTOMER-CONSIGNEE"}, codes)
	assert.Equal(t, "CUSTOMER-CONSIGNEE", codes[0], "ordered by priority, then specificity")

	none, err := repo.GetResolutionCandidates(
		scoped,
		&detentionports.GetResolutionCandidatesRequest{
			TenantInfo: tenant.Info(),
			CustomerID: customerID,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestSelectOptionsNarrowsToDefaultEligiblePolicies(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seedtest.SeedFullTestData(t, ctx, db)
	scoped := tenant.Ctx(ctx)

	now := timeutils.NowUnix()
	customerID := pulid.MustNew("cus_")
	begun := now - 24*60*60
	later := now + 30*24*60*60

	policies := seedDetentionPolicies(
		t, ctx, db, tenant,
		"ALL", "BEGUN", "DRAFT", "CUSTOMER", "STOP-TYPE", "APPOINTMENT", "ENDING", "FUTURE",
	)
	byCode := make(map[string]*detentionpolicy.DetentionPolicy, len(policies))
	for _, policy := range policies {
		byCode[policy.Code] = policy
		policy.Status = detentionpolicy.PolicyStatusActive
	}
	byCode["BEGUN"].EffectiveStartDate = &begun
	byCode["DRAFT"].Status = detentionpolicy.PolicyStatusDraft
	byCode["CUSTOMER"].CustomerID = &customerID
	byCode["STOP-TYPE"].StopTypes = []shipmentvocab.StopType{shipmentvocab.StopTypePickup}
	byCode["APPOINTMENT"].AppointmentStopsOnly = true
	byCode["ENDING"].EffectiveEndDate = &later
	byCode["FUTURE"].EffectiveStartDate = &later
	for _, policy := range policies {
		require.NoError(t, detentionpolicydb.DetentionPolicies.Save(scoped, db, policy))
	}

	repo := NewPolicyRepository(PolicyParams{DB: postgres.WrapDB(db), Logger: zap.NewNop()})
	codes := func(defaultEligibleOnly bool) []string {
		t.Helper()

		result, err := repo.SelectOptions(
			scoped,
			&detentionports.DetentionPolicySelectOptionsRequest{
				SelectQueryRequest: &pagination.SelectQueryRequest{
					TenantInfo: tenant.Info(),
					Pagination: pagination.Info{Limit: 20},
				},
				DefaultEligibleOnly: defaultEligibleOnly,
			},
		)
		require.NoError(t, err)

		out := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			out = append(out, item.Code)
		}

		return out
	}

	assert.ElementsMatch(t, []string{"ALL", "BEGUN"}, codes(true))
	assert.ElementsMatch(
		t,
		[]string{"ALL", "BEGUN", "CUSTOMER", "STOP-TYPE", "APPOINTMENT", "ENDING", "FUTURE"},
		codes(false),
		"without the filter every active policy is offered",
	)
}
