package fueldeskcardservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	orgID    = pulid.MustNew("org_")
	buID     = pulid.MustNew("bu_")
	tenantIn = pagination.TenantInfo{OrgID: orgID, BuID: buID}
	chicago  = mustLocation("America/Chicago")

	tractorID = pulid.MustNew("tr_")
	workerID  = pulid.MustNew("wrk_")
	texasID   = pulid.MustNew("ifj_")
	cardID    = pulid.MustNew("fcard_")
	userID    = pulid.MustNew("usr_")

	errUnexpected = errors.New("unexpected call")
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func at(year int, month time.Month, day, hour, minute int) int64 {
	return time.Date(year, month, day, hour, minute, 0, 0, chicago).Unix()
}

func dec(value string) decimal.Decimal { return decimal.RequireFromString(value) }

type fakePurchases struct {
	purchase    *fuelpurchase.FuelPurchase
	card        *fuelpurchase.FuelCard
	feed        *fuelpurchase.CardFeedState
	cards       map[pulid.ID][]*fuelpurchase.FuelCard
	listed      []*fuelpurchaseports.ListFuelCardsRequest
	purchaseReq *fuelpurchaseports.GetFuelPurchaseByIDRequest
	cardReq     *fuelpurchaseports.GetFuelCardByIDRequest
}

func (f *fakePurchases) GetPurchaseByID(
	_ context.Context,
	req *fuelpurchaseports.GetFuelPurchaseByIDRequest,
) (*fuelpurchase.FuelPurchase, error) {
	f.purchaseReq = req
	return f.purchase, nil
}

func (f *fakePurchases) GetCardByID(
	_ context.Context,
	req *fuelpurchaseports.GetFuelCardByIDRequest,
) (*fuelpurchase.FuelCard, error) {
	f.cardReq = req
	return f.card, nil
}

func (f *fakePurchases) ListCards(
	_ context.Context,
	req *fuelpurchaseports.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	f.listed = append(f.listed, req)
	holder := req.AssignedTractorID
	if holder.IsNil() {
		holder = req.AssignedWorkerID
	}
	return &pagination.CursorListResult[*fuelpurchase.FuelCard]{Items: f.cards[holder]}, nil
}

func (f *fakePurchases) GetFeedState(
	_ context.Context,
	req *fuelpurchaseports.GetFuelFeedStateRequest,
) (*fuelpurchase.CardFeedState, error) {
	if f.feed != nil {
		return f.feed, nil
	}
	return &fuelpurchase.CardFeedState{Provider: req.Provider, FeedType: req.FeedType}, nil
}

type fakeDesk struct {
	sums           [][]*fuelpurchaseports.FuelSpendRow
	sumReqs        []*fuelpurchaseports.SumFuelPurchasesRequest
	byJurisdiction []*fuelpurchaseports.FuelJurisdictionRow
	jurisdReq      *fuelpurchaseports.SumFuelPurchasesRequest
	previous       *fuelpurchase.FuelPurchase
	previousCalls  int
	importRow      *fuelpurchase.ImportRow
	importCalls    int
	held           []*fuelpurchase.ImportRow
	recent         []*fuelpurchase.FuelPurchase
	recentReq      *fuelpurchaseports.ListRecentCardPurchasesRequest
	forbidPurchase bool
	forbidImports  bool
}

func (f *fakeDesk) SumPurchases(
	_ context.Context,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelSpendRow, error) {
	if f.forbidPurchase && !req.FuelCardID.IsNil() {
		return nil, errUnexpected
	}
	f.sumReqs = append(f.sumReqs, req)
	if len(f.sums) == 0 {
		return []*fuelpurchaseports.FuelSpendRow{}, nil
	}
	next := f.sums[0]
	f.sums = f.sums[1:]
	return next, nil
}

func (f *fakeDesk) SumGallonsByJurisdiction(
	_ context.Context,
	req *fuelpurchaseports.SumFuelPurchasesRequest,
) ([]*fuelpurchaseports.FuelJurisdictionRow, error) {
	if f.forbidPurchase {
		return nil, errUnexpected
	}
	f.jurisdReq = req
	return f.byJurisdiction, nil
}

func (f *fakeDesk) GetPreviousFill(
	_ context.Context,
	_ *fuelpurchaseports.GetPreviousFillRequest,
) (*fuelpurchase.FuelPurchase, error) {
	f.previousCalls++
	return f.previous, nil
}

func (f *fakeDesk) GetPurchaseImportRow(
	_ context.Context,
	_ *fuelpurchaseports.GetPurchaseImportRowRequest,
) (*fuelpurchase.ImportRow, error) {
	f.importCalls++
	return f.importRow, nil
}

func (f *fakeDesk) ListHeldCardRows(
	_ context.Context,
	_ *fuelpurchaseports.ListHeldCardRowsRequest,
) ([]*fuelpurchase.ImportRow, error) {
	if f.forbidImports {
		return nil, errUnexpected
	}
	return f.held, nil
}

func (f *fakeDesk) ListRecentCardPurchases(
	_ context.Context,
	req *fuelpurchaseports.ListRecentCardPurchasesRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	if f.forbidPurchase {
		return nil, errUnexpected
	}
	f.recentReq = req
	return f.recent, nil
}

type fakeRates struct {
	rates  map[ifta.RateKey]*ifta.TaxRate
	asked  []*iftaports.ResolveRatesRequest
	forbid bool
}

func (f *fakeRates) ResolveRates(
	_ context.Context,
	req *iftaports.ResolveRatesRequest,
) (map[ifta.RateKey]*ifta.TaxRate, error) {
	if f.forbid {
		return nil, errUnexpected
	}
	f.asked = append(f.asked, req)
	return f.rates, nil
}

type fakeOrgs struct{}

func (fakeOrgs) GetByID(
	_ context.Context,
	_ tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type fakeUsers struct {
	users []*tenant.User
	calls int
}

func (f *fakeUsers) GetByIDs(
	_ context.Context,
	_ tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	f.calls++
	return f.users, nil
}

type fixture struct {
	purchases *fakePurchases
	desk      *fakeDesk
	rates     *fakeRates
	users     *fakeUsers
	service   *Service
}

func newFixture(now time.Time) *fixture {
	f := &fixture{
		purchases: &fakePurchases{cards: map[pulid.ID][]*fuelpurchase.FuelCard{}},
		desk:      &fakeDesk{},
		rates:     &fakeRates{rates: map[ifta.RateKey]*ifta.TaxRate{}},
		users:     &fakeUsers{},
	}
	f.service = &Service{
		l:             zap.NewNop(),
		purchases:     f.purchases,
		desk:          f.desk,
		ifta:          f.rates,
		organizations: fakeOrgs{},
		users:         f.users,
		now:           func() time.Time { return now },
	}
	return f
}

func dieselPurchase() *fuelpurchase.FuelPurchase {
	odometer := int64(181604)
	return &fuelpurchase.FuelPurchase{
		ID:               pulid.MustNew("fpur_"),
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		TractorID:        tractorID,
		JurisdictionID:   texasID,
		PurchasedAt:      at(2026, time.October, 19, 6, 42),
		FuelType:         domaintypes.IFTAFuelTypeDiesel,
		Gallons:          dec("118.420"),
		TotalAmountMinor: 46172,
		CurrencyCode:     "USD",
		Odometer:         &odometer,
		TaxPaid:          true,
		Source:           fuelpurchase.PurchaseSourceCardImport,
	}
}

func TestGetPurchaseCard_CreditsTaxPaidGallonsAtTheQuartersRate(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	purchase := dieselPurchase()
	f.purchases.purchase = purchase
	f.rates.rates[ifta.RateKey{JurisdictionID: texasID, FuelType: domaintypes.IFTAFuelTypeDiesel}] =
		&ifta.TaxRate{RatePerGallon: dec("0.2000")}
	f.desk.sums = [][]*fuelpurchaseports.FuelSpendRow{{
		{CurrencyCode: "USD", PurchaseCount: 4, Gallons: dec("500.120")},
		{CurrencyCode: "CAD", PurchaseCount: 1, Gallons: dec("112.720")},
	}}

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, 2026, card.IFTA.Year)
	assert.Equal(t, 4, card.IFTA.Quarter)
	assert.True(t, card.IFTA.CountsForIFTA)
	assert.Equal(t, "118.42", card.IFTA.TaxPaidGallons.String())
	require.True(t, card.IFTA.RatePerGallon.Valid)
	assert.Equal(t, "0.2", card.IFTA.RatePerGallon.Decimal.String())
	require.True(t, card.IFTA.TaxPaidCredit.Valid)
	assert.Equal(t, "23.68", card.IFTA.TaxPaidCredit.Decimal.StringFixed(2))
	assert.Equal(t, "612.84", card.IFTA.TractorGallons.String())
	assert.Equal(t, 5, card.IFTA.TractorPurchaseCount)

	require.Len(t, f.rates.asked, 1)
	assert.Equal(t, &iftaports.ResolveRatesRequest{Year: 2026, Quarter: 4}, f.rates.asked[0])

	require.Len(t, f.desk.sumReqs, 1)
	sum := f.desk.sumReqs[0]
	assert.Equal(t, tractorID, sum.TractorID)
	assert.Equal(t, texasID, sum.JurisdictionID)
	assert.Equal(t, domaintypes.IFTAFuelTypes(), sum.FuelTypes)
	assert.Equal(t, at(2026, time.October, 1, 0, 0), sum.From)
	assert.Equal(t, at(2027, time.January, 1, 0, 0), sum.To)

	assert.True(t, f.purchases.purchaseReq.IncludeTractor)
	assert.True(t, f.purchases.purchaseReq.IncludeWorker)
	assert.True(t, f.purchases.purchaseReq.IncludeJurisdiction)
	assert.True(t, f.purchases.purchaseReq.IncludeFuelCard)
}

func TestGetPurchaseCard_ReadsTheQuarterInTheOrganizationsTimeZone(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	purchase := dieselPurchase()
	purchase.PurchasedAt = at(2026, time.September, 30, 23, 30)
	f.purchases.purchase = purchase

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, card.IFTA.Quarter, "Sep 30, 11:30 PM in Chicago is Oct 1 in UTC")
	assert.Equal(t, at(2026, time.July, 1, 0, 0), f.desk.sumReqs[0].From)
	assert.Equal(t, at(2026, time.October, 1, 0, 0), f.desk.sumReqs[0].To)
}

func TestGetPurchaseCard_LeavesTheCreditOutWhenNoRateIsOnFile(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.purchase = dieselPurchase()

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: f.purchases.purchase.ID,
	})
	require.NoError(t, err)

	assert.False(t, card.IFTA.RatePerGallon.Valid)
	assert.False(t, card.IFTA.TaxPaidCredit.Valid)
	assert.Equal(t, "118.42", card.IFTA.TaxPaidGallons.String())
}

func TestGetPurchaseCard_CreditsNothingForUntaxedFuel(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	purchase := dieselPurchase()
	purchase.TaxPaid = false
	f.purchases.purchase = purchase
	f.rates.rates[ifta.RateKey{JurisdictionID: texasID, FuelType: domaintypes.IFTAFuelTypeDiesel}] =
		&ifta.TaxRate{RatePerGallon: dec("0.2000")}

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)

	assert.True(t, card.IFTA.TaxPaidGallons.IsZero())
	require.True(t, card.IFTA.TaxPaidCredit.Valid)
	assert.Equal(t, "0.00", card.IFTA.TaxPaidCredit.Decimal.StringFixed(2))
}

func TestGetPurchaseCard_TreatsDEFAsSpendNotIFTAFuel(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	purchase := dieselPurchase()
	purchase.FuelType = domaintypes.IFTAFuelTypeDEF
	purchase.TaxPaid = false
	f.purchases.purchase = purchase
	f.rates.forbid = true

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)

	assert.False(t, card.IFTA.CountsForIFTA)
	assert.True(t, card.IFTA.TaxPaidGallons.IsZero())
	assert.False(t, card.IFTA.TaxPaidCredit.Valid)
	assert.Nil(t, card.PreviousFill)
	assert.Zero(t, f.desk.previousCalls, "DEF does not move the truck, so it has no previous fill")
}

func TestGetPurchaseCard_ReadsThePreviousFillOnlyFromAFillWithAnOdometer(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	previous := dieselPurchase()
	previous.ID = pulid.MustNew("fpur_")
	f.desk.previous = previous

	f.purchases.purchase = dieselPurchase()
	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: f.purchases.purchase.ID,
	})
	require.NoError(t, err)
	assert.Same(t, previous, card.PreviousFill)
	assert.Equal(t, 1, f.desk.previousCalls)

	noOdometer := dieselPurchase()
	noOdometer.Odometer = nil
	f.purchases.purchase = noOdometer
	card, err = f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: noOdometer.ID,
	})
	require.NoError(t, err)
	assert.Nil(t, card.PreviousFill)
	assert.Equal(t, 1, f.desk.previousCalls)
}

func TestGetPurchaseCard_ShowsWhereAnImportedPurchaseWasRead(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.purchase = dieselPurchase()
	batch := &fuelpurchase.ImportBatch{ID: pulid.MustNew("fpib_"), RowCount: 112}
	f.desk.importRow = &fuelpurchase.ImportRow{RowNumber: 37, Batch: batch}

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo:        tenantIn,
		PurchaseID:        f.purchases.purchase.ID,
		IncludeProvenance: true,
	})
	require.NoError(t, err)

	require.NotNil(t, card.Provenance)
	assert.Same(t, batch, card.Provenance.Batch)
	assert.Equal(t, 37, card.Provenance.RowNumber)
	assert.Equal(t, []string{}, card.Provenance.ResolutionNotes)
}

func TestGetPurchaseCard_ReadsNoProvenanceForAHandEntryOrAReaderWithoutImports(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	manual := dieselPurchase()
	manual.Source = fuelpurchase.PurchaseSourceManual
	f.purchases.purchase = manual

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo:        tenantIn,
		PurchaseID:        manual.ID,
		IncludeProvenance: true,
	})
	require.NoError(t, err)
	assert.Nil(t, card.Provenance)

	f.purchases.purchase = dieselPurchase()
	card, err = f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo: tenantIn,
		PurchaseID: f.purchases.purchase.ID,
	})
	require.NoError(t, err)
	assert.Nil(t, card.Provenance)
	assert.Zero(t, f.desk.importCalls)
}

func TestGetPurchaseCard_NamesWhoEnteredIt(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	manual := dieselPurchase()
	manual.Source = fuelpurchase.PurchaseSourceManual
	manual.CreatedByID = userID
	f.purchases.purchase = manual
	dana := &tenant.User{ID: userID, Name: "Dana Park"}
	f.users.users = []*tenant.User{dana}

	card, err := f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo:       tenantIn,
		PurchaseID:       manual.ID,
		IncludeCreatedBy: true,
	})
	require.NoError(t, err)
	assert.Same(t, dana, card.CreatedBy)

	fed := dieselPurchase()
	f.purchases.purchase = fed
	card, err = f.service.GetPurchaseCard(t.Context(), &GetPurchaseCardRequest{
		TenantInfo:       tenantIn,
		PurchaseID:       fed.ID,
		IncludeCreatedBy: true,
	})
	require.NoError(t, err)
	assert.Nil(t, card.CreatedBy)
	assert.Equal(t, 1, f.users.calls, "a feed records no person to look up")
}

func activeCard() *fuelpurchase.FuelCard {
	tractor := tractorID
	worker := workerID
	return &fuelpurchase.FuelCard{
		ID:                cardID,
		Provider:          fuelpurchase.CardProviderComdata,
		LastFour:          "4412",
		Status:            fuelpurchase.CardStatusActive,
		AssignedTractorID: &tractor,
		AssignedWorkerID:  &worker,
	}
}

func TestGetFuelCardCard_TotalsTheMonthAndTheQuarterInTheOrganizationsTimeZone(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Date(2026, time.November, 1, 3, 0, 0, 0, time.UTC))
	f.purchases.card = activeCard()
	f.desk.sums = [][]*fuelpurchaseports.FuelSpendRow{
		{{CurrencyCode: "USD", PurchaseCount: 9, Gallons: dec("1046.3"), UntaxedGallons: dec("15.0"), TotalAmountMinor: 409218}},
		{
			{CurrencyCode: "USD", PurchaseCount: 200, LastPurchasedAt: at(2026, time.October, 19, 6, 51)},
			{CurrencyCode: "CAD", PurchaseCount: 3, LastPurchasedAt: at(2026, time.October, 31, 21, 0)},
		},
	}
	f.desk.byJurisdiction = []*fuelpurchaseports.FuelJurisdictionRow{
		{JurisdictionID: texasID, Gallons: dec("612.84"), PurchaseCount: 5},
	}

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo:       tenantIn,
		CardID:           cardID,
		IncludePurchases: true,
	})
	require.NoError(t, err)

	assert.True(t, f.purchases.cardReq.IncludeAssignments)
	assert.Equal(t, at(2026, time.October, 1, 0, 0), card.MonthStart,
		"3 AM UTC on Nov 1 is still October in Chicago")
	assert.Equal(t, 2026, card.Year)
	assert.Equal(t, 4, card.Quarter)

	require.Len(t, card.Month, 1)
	assert.Equal(t, &Spend{
		CurrencyCode:     "USD",
		PurchaseCount:    9,
		Gallons:          dec("1046.3"),
		UntaxedGallons:   dec("15.0"),
		TotalAmountMinor: 409218,
	}, card.Month[0])
	assert.Equal(t, 203, card.PurchaseCount)
	require.NotNil(t, card.LastPurchasedAt)
	assert.Equal(t, at(2026, time.October, 31, 21, 0), *card.LastPurchasedAt)

	require.Len(t, f.desk.sumReqs, 2)
	assert.Equal(t, cardID, f.desk.sumReqs[0].FuelCardID)
	assert.Equal(t, card.MonthStart, f.desk.sumReqs[0].From)
	assert.Zero(t, f.desk.sumReqs[0].To)
	assert.Zero(t, f.desk.sumReqs[1].From)

	assert.Equal(t, domaintypes.IFTAFuelTypes(), f.desk.jurisdReq.FuelTypes)
	assert.Equal(t, at(2026, time.October, 1, 0, 0), f.desk.jurisdReq.From)
	assert.Equal(t, at(2027, time.January, 1, 0, 0), f.desk.jurisdReq.To)
	assert.Equal(t, []*JurisdictionGallons{
		{JurisdictionID: texasID, Gallons: dec("612.84"), PurchaseCount: 5},
	}, card.QuarterByJurisdiction)
	assert.Equal(t, recentPurchaseLimit, f.desk.recentReq.Limit)
}

func TestGetFuelCardCard_ReadsNoPurchasesOrImportsWithoutPermission(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.card = activeCard()
	f.desk.forbidPurchase = true
	f.desk.forbidImports = true

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo: tenantIn,
		CardID:     cardID,
	})
	require.NoError(t, err)

	assert.Empty(t, card.Month)
	assert.Empty(t, card.QuarterByJurisdiction)
	assert.Empty(t, card.Recent)
	assert.Empty(t, card.Held)
	assert.Zero(t, card.PurchaseCount)
	assert.Nil(t, card.LastPurchasedAt)
}

func heldRow(batch *fuelpurchase.ImportBatch, currency, gallons string, minor int64) *fuelpurchase.ImportRow {
	return &fuelpurchase.ImportRow{
		ID:     pulid.MustNew("fpir_"),
		Status: fuelpurchase.ImportRowStatusError,
		Batch:  batch,
		Parsed: &fuelpurchase.FuelPurchase{
			CurrencyCode:     currency,
			Gallons:          dec(gallons),
			TotalAmountMinor: minor,
		},
	}
}

func TestGetFuelCardCard_TotalsTheRowsHeldAgainstTheCard(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.card = activeCard()
	first := &fuelpurchase.ImportBatch{ID: pulid.MustNew("fpib_")}
	second := &fuelpurchase.ImportBatch{ID: pulid.MustNew("fpib_")}

	rows := []*fuelpurchase.ImportRow{
		heldRow(first, "USD", "132.100", 49873),
		heldRow(first, "", "104.700", 40205),
		heldRow(second, "CAD", "98.400", 37733),
		{ID: pulid.MustNew("fpir_"), Status: fuelpurchase.ImportRowStatusError, Batch: second},
	}
	for range 9 {
		rows = append(rows, heldRow(second, "USD", "10.000", 1000))
	}
	f.desk.held = rows

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo:     tenantIn,
		CardID:         cardID,
		IncludeImports: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 13, card.HeldCount)
	assert.Len(t, card.Held, heldRowsShown)
	assert.Equal(t, rows[:heldRowsShown], card.Held)
	assert.Equal(t, []*fuelpurchase.ImportBatch{first, second}, card.HeldBatches)
	require.Len(t, card.HeldSpend, 2)
	assert.Equal(t, "USD", card.HeldSpend[0].CurrencyCode)
	assert.Equal(t, "326.8", card.HeldSpend[0].Gallons.String())
	assert.Equal(t, int64(99078), card.HeldSpend[0].TotalAmountMinor)
	assert.Equal(t, "CAD", card.HeldSpend[1].CurrencyCode)
	assert.Equal(t, "98.4", card.HeldSpend[1].Gallons.String())
	assert.Equal(t, int64(37733), card.HeldSpend[1].TotalAmountMinor)
}

func TestGetFuelCardCard_ReportsTheFeedOnlyWhenOneHasRun(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.card = activeCard()

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo: tenantIn,
		CardID:     cardID,
	})
	require.NoError(t, err)
	assert.Nil(t, card.Feed)

	state := &fuelpurchase.CardFeedState{
		Provider:      fuelpurchase.CardProviderComdata,
		LastPolledAt:  at(2026, time.October, 20, 8, 42),
		LastSuccessAt: at(2026, time.October, 20, 8, 42),
	}
	f.purchases.feed = state
	card, err = f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo: tenantIn,
		CardID:     cardID,
	})
	require.NoError(t, err)
	assert.Same(t, state, card.Feed)
}

func TestGetFuelCardCard_ListsTheActiveCardsThatReplacedACancelledOne(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	cancelled := activeCard()
	cancelled.Status = fuelpurchase.CardStatusCancelled
	f.purchases.card = cancelled

	replacement := &fuelpurchase.FuelCard{ID: pulid.MustNew("fcard_"), LastFour: "5306"}
	driverCard := &fuelpurchase.FuelCard{ID: pulid.MustNew("fcard_"), LastFour: "7781"}
	f.purchases.cards[tractorID] = []*fuelpurchase.FuelCard{replacement}
	f.purchases.cards[workerID] = []*fuelpurchase.FuelCard{replacement, driverCard}

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo: tenantIn,
		CardID:     cardID,
	})
	require.NoError(t, err)

	assert.Equal(t, []*fuelpurchase.FuelCard{replacement, driverCard}, card.Successors)
	require.Len(t, f.purchases.listed, 2)
	for _, req := range f.purchases.listed {
		assert.Equal(t, []fuelpurchase.CardStatus{fuelpurchase.CardStatusActive}, req.Statuses)
		assert.True(t, req.IncludeAssignments)
		assert.Equal(t, tenantIn, req.Filter.TenantInfo)
	}
}

func TestGetFuelCardCard_ListsNoSuccessorsForACardStillInUse(t *testing.T) {
	t.Parallel()

	f := newFixture(time.Unix(at(2026, time.October, 20, 9, 0), 0))
	f.purchases.card = activeCard()

	card, err := f.service.GetFuelCardCard(t.Context(), &GetFuelCardCardRequest{
		TenantInfo: tenantIn,
		CardID:     cardID,
	})
	require.NoError(t, err)

	assert.Empty(t, card.Successors)
	assert.Empty(t, f.purchases.listed)
}
