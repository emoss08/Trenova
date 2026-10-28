package driversettlementresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type disputesBySettlement struct {
	driversettlementports.SettlementDisputeRepository

	disputes map[pulid.ID][]*driversettlement.Dispute
	calls    int
}

func (d *disputesBySettlement) ListBySettlementIDs(
	_ context.Context,
	_ *driversettlementports.ListSettlementDisputesBySettlementIDsRequest,
) (map[pulid.ID][]*driversettlement.Dispute, error) {
	d.calls++
	return d.disputes, nil
}

type advancesByID struct {
	driverpayports.PayAdvanceRepository

	advances []*driverpay.PayAdvance
}

func (a *advancesByID) GetByIDs(
	_ context.Context,
	_ driverpayports.GetPayAdvancesByIDsRequest,
) ([]*driverpay.PayAdvance, error) {
	return a.advances, nil
}

var cardTenant = pagination.TenantInfo{
	OrgID: pulid.MustNew("org_"),
	BuID:  pulid.MustNew("bu_"),
}

func signedIn(ctx context.Context) context.Context {
	userID := pulid.MustNew("usr_")
	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: cardTenant.OrgID,
		BusinessUnitID: cardTenant.BuID,
	})
}

func coreGranting(resources ...permission.Resource) *base.Core {
	granted := make(map[string]bool, len(resources))
	for _, resource := range resources {
		granted[resource.String()+"|"+string(permission.OpRead)] = true
	}
	return &base.Core{PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted}}
}

func TestLineSummaryIsNullWhenTheLinesWereNotReadAndSumsThemWhenTheyWere(t *testing.T) {
	t.Parallel()

	resolver := &DriverSettlementResolver{&Deps{}}
	unread, err := resolver.LineSummary(t.Context(), &driversettlement.Settlement{})
	require.NoError(t, err)
	assert.Nil(t, unread)

	shipment := pulid.MustNew("shp_")
	groups, err := resolver.LineSummary(t.Context(), &driversettlement.Settlement{
		Lines: []*driversettlement.SettlementLine{
			{
				Category:      driversettlement.LineCategoryEarning,
				ComponentKind: driverpay.ComponentKindStopPay,
				Quantity:      decimal.NewFromInt(9),
				Rate:          decimal.RequireFromString("25"),
				AmountMinor:   22500,
				ShipmentID:    &shipment,
			},
			{
				Category:      driversettlement.LineCategoryEarning,
				ComponentKind: driverpay.ComponentKindStopPay,
				Quantity:      decimal.NewFromInt(5),
				Rate:          decimal.RequireFromString("25"),
				AmountMinor:   12500,
				ShipmentID:    &shipment,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, int64(35000), groups[0].AmountMinor)

	group := &DriverSettlementLineGroupResolver{&Deps{}}
	rate, err := group.Rate(t.Context(), groups[0])
	require.NoError(t, err)
	require.NotNil(t, rate)
	assert.Equal(t, "25", *rate)
	quantity, err := group.Quantity(t.Context(), groups[0])
	require.NoError(t, err)
	assert.Equal(t, "14", quantity)
}

func TestLoadSharesTotalsEachLoadTheSettlementPays(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("shp_")
	shares, err := (&DriverSettlementResolver{&Deps{}}).LoadShares(
		t.Context(),
		&driversettlement.Settlement{Lines: []*driversettlement.SettlementLine{
			{ShipmentID: &first, ProNumber: "S-240193", AmountMinor: 61220},
			{AmountMinor: -9600},
			{ShipmentID: &first, AmountMinor: 4200},
		}},
	)
	require.NoError(t, err)
	require.Len(t, shares, 1)
	assert.Equal(t, first, shares[0].ShipmentID)
	assert.Equal(t, int64(65420), shares[0].AmountMinor)
	assert.Equal(t, "S-240193", shares[0].ProNumber)
}

func TestDisputesAreReadOnlyForSomeoneWhoMayReadSettlementDisputes(t *testing.T) {
	t.Parallel()

	settlementID := pulid.MustNew("dstl_")
	dispute := &driversettlement.Dispute{ID: pulid.MustNew("dsd_"), SettlementID: settlementID}
	repo := &disputesBySettlement{disputes: map[pulid.ID][]*driversettlement.Dispute{
		settlementID: {dispute},
	}}
	factory := loaders.NewSettlementDisputesBySettlementIDLoaderFactory(
		loaders.SettlementDisputesBySettlementIDLoaderFactoryParams{SettlementDisputeRepo: repo},
	)
	ctx := factory.Attach(signedIn(t.Context()), cardTenant)
	obj := &driversettlement.Settlement{ID: settlementID}

	refused, err := (&DriverSettlementResolver{&Deps{Core: coreGranting(
		permission.ResourceDriverSettlement,
	)}}).Disputes(ctx, obj)
	require.NoError(t, err)
	assert.Empty(t, refused)
	assert.NotNil(t, refused)
	assert.Zero(t, repo.calls)

	allowed, err := (&DriverSettlementResolver{&Deps{Core: coreGranting(
		permission.ResourceDriverSettlement,
		permission.ResourceSettlementDispute,
	)}}).Disputes(ctx, obj)
	require.NoError(t, err)
	assert.Equal(t, []*driversettlement.Dispute{dispute}, allowed)
}

func TestAnAdvanceIsReadOnlyForSomeoneWhoMayReadPayAdvances(t *testing.T) {
	t.Parallel()

	advanceID := pulid.MustNew("padv_")
	advance := &driverpay.PayAdvance{ID: advanceID, AmountMinor: 30000, RecoveredMinor: 20000}
	factory := loaders.NewPayAdvanceByIDLoaderFactory(loaders.PayAdvanceByIDLoaderFactoryParams{
		PayAdvanceRepo: &advancesByID{advances: []*driverpay.PayAdvance{advance}},
	})
	ctx := factory.Attach(signedIn(t.Context()), cardTenant)
	group := &driversettlement.LineGroup{AdvanceID: &advanceID}

	refused, err := (&DriverSettlementLineGroupResolver{&Deps{Core: coreGranting()}}).Advance(
		ctx,
		group,
	)
	require.NoError(t, err)
	assert.Nil(t, refused)

	read, err := (&DriverSettlementLineGroupResolver{&Deps{Core: coreGranting(
		permission.ResourcePayAdvance,
	)}}).Advance(ctx, group)
	require.NoError(t, err)
	assert.Same(t, advance, read)

	none, err := (&DriverSettlementLineGroupResolver{&Deps{Core: coreGranting(
		permission.ResourcePayAdvance,
	)}}).Advance(ctx, &driversettlement.LineGroup{})
	require.NoError(t, err)
	assert.Nil(t, none)
}

// Posting a settlement writes its journal and paying it releases money, so a
// bulk action needs what the single action needs: update alone is not enough.
func TestBulkSettlementOperationsMatchTheSingleActions(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []permission.Operation{permission.OpSubmit},
		bulkSettlementOperations(driversettlementservice.BulkActionSubmit))
	assert.Equal(t, []permission.Operation{permission.OpSubmit, permission.OpApprove},
		bulkSettlementOperations(driversettlementservice.BulkActionApprove),
		"bulk approve submits a draft first")
	assert.Equal(t, []permission.Operation{permission.OpApprove},
		bulkSettlementOperations(driversettlementservice.BulkActionPost))
	assert.Equal(t, []permission.Operation{permission.OpUpdate},
		bulkSettlementOperations(driversettlementservice.BulkActionMarkPaid))
}
