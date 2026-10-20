package fuelpurchaseresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fueldeskcardservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type deskPurchases struct {
	fuelpurchaseports.FuelPurchaseRepository
	purchase *fuelpurchase.FuelPurchase
	card     *fuelpurchase.FuelCard
}

func (d *deskPurchases) GetPurchaseByID(
	context.Context,
	*fuelpurchaseports.GetFuelPurchaseByIDRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return d.purchase, nil
}

func (d *deskPurchases) GetCardByID(
	context.Context,
	*fuelpurchaseports.GetFuelCardByIDRequest,
) (*fuelpurchase.FuelCard, error) {
	return d.card, nil
}

func (d *deskPurchases) GetFeedState(
	context.Context,
	*fuelpurchaseports.GetFuelFeedStateRequest,
) (*fuelpurchase.CardFeedState, error) {
	return &fuelpurchase.CardFeedState{}, nil
}

type deskReads struct {
	sums      int
	held      int
	provenant int
}

func (d *deskReads) SumPurchases(
	context.Context,
	*fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelSpendRow, error) {
	d.sums++
	return nil, nil
}

func (d *deskReads) SumGallonsByJurisdiction(
	context.Context,
	*fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelJurisdictionRow, error) {
	return nil, nil
}

func (d *deskReads) GetPreviousFill(
	context.Context,
	*fuelpurchaseports.GetPreviousFillRequest,
) (*fuelpurchase.FuelPurchase, error) {
	return nil, nil
}

func (d *deskReads) GetPurchaseImportRow(
	context.Context,
	*fuelpurchaseports.GetPurchaseImportRowRequest,
) (*fuelpurchase.ImportRow, error) {
	d.provenant++
	return &fuelpurchase.ImportRow{RowNumber: 37, Batch: &fuelpurchase.ImportBatch{}}, nil
}

func (d *deskReads) ListHeldCardRows(
	context.Context,
	*fuelpurchaseports.ListHeldCardRowsRequest,
) ([]*fuelpurchase.ImportRow, error) {
	d.held++
	return nil, nil
}

func (d *deskReads) ListRecentCardPurchases(
	context.Context,
	*fuelpurchaseports.ListRecentCardPurchasesRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	return nil, nil
}

type deskRates struct{ iftaports.IFTARepository }

func (deskRates) ResolveRates(
	context.Context,
	*iftaports.ResolveRatesRequest,
) (map[ifta.RateKey]*ifta.TaxRate, error) {
	return map[ifta.RateKey]*ifta.TaxRate{}, nil
}

type deskOrgs struct{ tenantports.OrganizationRepository }

func (deskOrgs) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type deskUsers struct{ tenantports.UserRepository }

func (deskUsers) GetByIDs(
	context.Context,
	tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	return nil, nil
}

func deskResolver(
	t *testing.T,
	engine permissionports.PermissionEngine,
) (*QueryResolver, *deskPurchases, *deskReads, context.Context) {
	t.Helper()

	purchases := &deskPurchases{
		purchase: &fuelpurchase.FuelPurchase{
			ID:          pulid.MustNew("fpur_"),
			TractorID:   pulid.MustNew("tr_"),
			FuelType:    domaintypes.IFTAFuelTypeDiesel,
			Gallons:     decimal.RequireFromString("118.42"),
			TaxPaid:     true,
			Source:      fuelpurchase.PurchaseSourceCardImport,
			PurchasedAt: 1_792_495_800,
		},
		card: &fuelpurchase.FuelCard{
			ID:       pulid.MustNew("fcard_"),
			Provider: fuelpurchase.CardProviderComdata,
			Status:   fuelpurchase.CardStatusActive,
		},
	}
	reads := &deskReads{}
	cards := fueldeskcardservice.New(fueldeskcardservice.Params{
		Logger:        zap.NewNop(),
		Purchases:     purchases,
		Desk:          reads,
		IFTA:          deskRates{},
		Organizations: deskOrgs{},
		Users:         deskUsers{},
	})
	deps := &Deps{
		Core:          &base.Core{L: zap.NewNop(), PermissionEngine: engine},
		FuelDeskCards: cards,
	}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  string(permissionports.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	return &QueryResolver{deps}, purchases, reads, ctx
}

func TestFuelPurchaseDeskCardShowsWhereItWasReadOnlyToAnImportReader(t *testing.T) {
	t.Parallel()

	resolver, purchases, reads, ctx := deskResolver(t, &mocks.AllowAllPermissionEngine{})
	card, err := resolver.FuelPurchaseDeskCard(ctx, purchases.purchase.ID.String())
	require.NoError(t, err)
	require.NotNil(t, card.Provenance)
	assert.Equal(t, 37, card.Provenance.RowNumber)

	resolver, purchases, reads, ctx = deskResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceFuelPurchaseImport.String()},
	})
	card, err = resolver.FuelPurchaseDeskCard(ctx, purchases.purchase.ID.String())
	require.NoError(t, err)
	assert.Nil(t, card.Provenance)
	assert.Zero(t, reads.provenant)
}

func TestFuelPurchaseDeskCardRefusesSomeoneWhoMayNotReadFuelPurchases(t *testing.T) {
	t.Parallel()

	resolver, purchases, _, ctx := deskResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceFuelPurchase.String()},
	})

	_, err := resolver.FuelPurchaseDeskCard(ctx, purchases.purchase.ID.String())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestFuelCardDeskCardLeavesOutPurchasesAndImportsTheReaderMayNotRead(t *testing.T) {
	t.Parallel()

	resolver, purchases, reads, ctx := deskResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{
			permission.ResourceFuelPurchase.String(),
			permission.ResourceFuelPurchaseImport.String(),
		},
	})

	card, err := resolver.FuelCardDeskCard(ctx, purchases.card.ID.String())
	require.NoError(t, err)
	assert.Same(t, purchases.card, card.Card)
	assert.Zero(t, reads.sums)
	assert.Zero(t, reads.held)
	assert.Empty(t, card.Month)
	assert.Empty(t, card.Held)

	resolver, purchases, reads, ctx = deskResolver(t, &mocks.AllowAllPermissionEngine{})
	_, err = resolver.FuelCardDeskCard(ctx, purchases.card.ID.String())
	require.NoError(t, err)
	assert.Equal(t, 2, reads.sums)
	assert.Equal(t, 1, reads.held)
}

func TestFuelCardDeskCardRefusesSomeoneWhoMayNotReadFuelCards(t *testing.T) {
	t.Parallel()

	resolver, purchases, _, ctx := deskResolver(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceFuelCard.String()},
	})

	_, err := resolver.FuelCardDeskCard(ctx, purchases.card.ID.String())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}
