//go:build integration

package fuelpurchaserepository_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/fuelpurchaserepository"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (f *fixture) deskReader() fuelpurchaseports.FuelDeskCardReader {
	return fuelpurchaserepository.NewDeskCardReader(fuelpurchaserepository.Params{
		DB:     postgres.NewTestConnection(f.db),
		Logger: zap.NewNop(),
	})
}

type cardPurchase struct {
	spec     purchaseSpec
	cardID   pulid.ID
	currency string
	odometer int64
}

func (f *fixture) createPurchase(t *testing.T, p cardPurchase) *fuelpurchase.FuelPurchase {
	t.Helper()

	purchase := f.newPurchase(p.spec)
	if !p.cardID.IsNil() {
		card := p.cardID
		purchase.FuelCardID = &card
	}
	if p.currency != "" {
		purchase.CurrencyCode = p.currency
	}
	if p.odometer > 0 {
		odometer := p.odometer
		purchase.Odometer = &odometer
	}
	created, err := f.repo.CreatePurchase(f.as(p.spec.tenant), purchase)
	require.NoError(t, err)

	return created
}

func TestDeskSumsTotalOneCardPerCurrencyInsideTheWindow(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	card, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "4412"))
	require.NoError(t, err)
	other, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "0931"))
	require.NoError(t, err)

	base := purchaseSpec{tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas}
	spec := func(ref, gallons string, taxPaid bool, at int64) purchaseSpec {
		s := base
		s.reference, s.gallons, s.taxPaid, s.purchasedAt = ref, gallons, taxPaid, at
		return s
	}

	f.createPurchase(t, cardPurchase{spec: spec("SUM-1", "100", true, windowStart+10), cardID: card.ID})
	f.createPurchase(t, cardPurchase{spec: spec("SUM-2", "15", false, windowStart+20), cardID: card.ID})
	f.createPurchase(t, cardPurchase{
		spec: spec("SUM-3", "50", true, windowStart+30), cardID: card.ID, currency: "CAD",
	})
	f.createPurchase(t, cardPurchase{spec: spec("SUM-4", "70", true, windowEnd+10), cardID: card.ID})
	f.createPurchase(t, cardPurchase{spec: spec("SUM-5", "40", true, windowStart+40), cardID: other.ID})

	rows, err := desk.SumPurchases(f.ctxA, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: f.tenantA,
		FuelCardID: card.ID,
		From:       windowStart,
		To:         windowEnd,
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	usd, cad := rows[0], rows[1]
	assert.Equal(t, "USD", usd.CurrencyCode)
	assert.Equal(t, 2, usd.PurchaseCount)
	assert.Equal(t, "115", usd.Gallons.String())
	assert.Equal(t, "15", usd.UntaxedGallons.String())
	assert.Equal(t, int64(38990+5848), usd.TotalAmountMinor)
	assert.Equal(t, windowStart+20, usd.LastPurchasedAt)
	assert.Equal(t, "CAD", cad.CurrencyCode)
	assert.Equal(t, 1, cad.PurchaseCount)
	assert.Equal(t, "0", cad.UntaxedGallons.String())

	lifetime, err := desk.SumPurchases(f.ctxA, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: f.tenantA,
		FuelCardID: card.ID,
	})
	require.NoError(t, err)
	count := 0
	for _, row := range lifetime {
		count += row.PurchaseCount
	}
	assert.Equal(t, 4, count)

	elsewhere, err := desk.SumPurchases(f.ctxB, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: f.tenantB,
		FuelCardID: card.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, elsewhere)
}

func TestDeskSumsGallonsByJurisdictionMostFirstAndIFTAFuelOnly(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	card, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "4412"))
	require.NoError(t, err)

	add := func(ref, gallons string, jurisdiction pulid.ID, fuelType domaintypes.IFTAFuelType) {
		f.createPurchase(t, cardPurchase{spec: purchaseSpec{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: jurisdiction,
			fuelType: fuelType, gallons: gallons, taxPaid: true, reference: ref,
		}, cardID: card.ID})
	}
	add("J-1", "120", f.texas, domaintypes.IFTAFuelTypeDiesel)
	add("J-2", "130", f.oklahoma, domaintypes.IFTAFuelTypeDiesel)
	add("J-3", "110", f.oklahoma, domaintypes.IFTAFuelTypeDiesel)
	add("J-4", "500", f.texas, domaintypes.IFTAFuelTypeDEF)

	rows, err := desk.SumGallonsByJurisdiction(f.ctxA, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: f.tenantA,
		FuelCardID: card.ID,
		FuelTypes:  domaintypes.IFTAFuelTypes(),
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, f.oklahoma, rows[0].JurisdictionID)
	assert.Equal(t, "240", rows[0].Gallons.String())
	assert.Equal(t, 2, rows[0].PurchaseCount)
	assert.Equal(t, f.texas, rows[1].JurisdictionID)
	assert.Equal(t, "120", rows[1].Gallons.String())
}

func TestDeskPreviousFillIsTheLastEarlierRoadFuelFillWithAnOdometer(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	fill := func(ref string, at, odometer int64, fuelType domaintypes.IFTAFuelType) *fuelpurchase.FuelPurchase {
		return f.createPurchase(t, cardPurchase{spec: purchaseSpec{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
			fuelType: fuelType, gallons: "100", taxPaid: true, reference: ref, purchasedAt: at,
		}, odometer: odometer})
	}

	first := fill("ODO-1", windowStart+100, 180000, domaintypes.IFTAFuelTypeDiesel)
	expected := fill("ODO-2", windowStart+200, 180600, domaintypes.IFTAFuelTypeDiesel)
	fill("ODO-3", windowStart+250, 0, domaintypes.IFTAFuelTypeDiesel)
	fill("ODO-4", windowStart+260, 180650, domaintypes.IFTAFuelTypeDEF)
	current := fill("ODO-5", windowStart+300, 181336, domaintypes.IFTAFuelTypeDiesel)
	fill("ODO-6", windowStart+400, 181900, domaintypes.IFTAFuelTypeDiesel)
	f.createPurchase(t, cardPurchase{spec: purchaseSpec{
		tenant: f.tenantB, tractorID: f.tractorB, jurisdiction: f.texas,
		gallons: "100", taxPaid: true, reference: "ODO-B", purchasedAt: windowStart + 290,
	}, odometer: 99000})

	previous, err := desk.GetPreviousFill(f.ctxA, &fuelpurchaseports.GetPreviousFillRequest{
		TenantInfo: f.tenantA,
		Purchase:   current,
	})
	require.NoError(t, err)
	require.NotNil(t, previous)
	assert.Equal(t, expected.ID, previous.ID)

	none, err := desk.GetPreviousFill(f.ctxA, &fuelpurchaseports.GetPreviousFillRequest{
		TenantInfo: f.tenantA,
		Purchase:   first,
	})
	require.NoError(t, err)
	assert.Nil(t, none)
}

func (f *fixture) feedBatch(
	t *testing.T,
	tenant pagination.TenantInfo,
	origin fuelpurchase.ImportOrigin,
	status fuelpurchase.ImportStatus,
) *fuelpurchase.ImportBatch {
	t.Helper()

	batch := f.newBatch(tenant)
	batch.Origin = origin
	if origin == fuelpurchase.ImportOriginFeed {
		batch.UploadedByID = pulid.Nil
	}
	ctx := f.as(tenant)
	created, err := f.repo.CreateImportBatch(ctx, batch)
	require.NoError(t, err)
	if status != fuelpurchase.ImportStatusPending {
		b := fuelpurchasedb.ImportBatches
		sets := []keel.Assignment{b.Status.To(status)}
		if status == fuelpurchase.ImportStatusCommitted {
			sets = append(sets, b.CommittedAt.To(windowStart), b.CommittedByID.To(f.userA))
		}
		_, err = b.Where(b.ID.Eq(created.ID)).Update(ctx, f.db, sets...)
		require.NoError(t, err)
		created.Status = status
	}

	return created
}

func heldRow(number int, cardID pulid.ID, status fuelpurchase.ImportRowStatus) *fuelpurchase.ImportRow {
	card := cardID
	row := &fuelpurchase.ImportRow{
		RowNumber:          number,
		Cells:              []string{"row"},
		Status:             status,
		ResolvedFuelCardID: &card,
	}
	if status == fuelpurchase.ImportRowStatusError {
		row.Error = "No tractor matched unit T130"
	}

	return row
}

func TestDeskHeldCardRowsAreTheCardsRowsStillWaitingInAnOpenImport(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	card, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "7720"))
	require.NoError(t, err)
	other, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "0931"))
	require.NoError(t, err)

	feed := f.feedBatch(t, f.tenantA, fuelpurchase.ImportOriginFeed, fuelpurchase.ImportStatusCommitted)
	require.NoError(t, f.repo.ReplaceImportRows(f.ctxA, feed, []*fuelpurchase.ImportRow{
		heldRow(2, card.ID, fuelpurchase.ImportRowStatusError),
		heldRow(3, card.ID, fuelpurchase.ImportRowStatusError),
		heldRow(4, other.ID, fuelpurchase.ImportRowStatusError),
		heldRow(5, card.ID, fuelpurchase.ImportRowStatusSkipped),
	}))

	discarded := f.feedBatch(
		t, f.tenantA, fuelpurchase.ImportOriginFeed, fuelpurchase.ImportStatusDiscarded,
	)
	require.NoError(t, f.repo.ReplaceImportRows(f.ctxA, discarded, []*fuelpurchase.ImportRow{
		heldRow(2, card.ID, fuelpurchase.ImportRowStatusError),
	}))

	upload := f.feedBatch(t, f.tenantA, fuelpurchase.ImportOriginUpload, fuelpurchase.ImportStatusPending)
	require.NoError(t, f.repo.ReplaceImportRows(f.ctxA, upload, []*fuelpurchase.ImportRow{
		heldRow(2, card.ID, fuelpurchase.ImportRowStatusNew),
	}))

	committedUpload := f.feedBatch(
		t, f.tenantA, fuelpurchase.ImportOriginUpload, fuelpurchase.ImportStatusCommitted,
	)
	require.NoError(t, f.repo.ReplaceImportRows(f.ctxA, committedUpload, []*fuelpurchase.ImportRow{
		heldRow(2, card.ID, fuelpurchase.ImportRowStatusError),
	}))

	rows, err := desk.ListHeldCardRows(f.ctxA, &fuelpurchaseports.ListHeldCardRowsRequest{
		TenantInfo: f.tenantA,
		CardID:     card.ID,
	})
	require.NoError(t, err)
	require.Len(t, rows, 3)

	byBatch := map[pulid.ID][]int{}
	for _, row := range rows {
		require.NotNil(t, row.Batch)
		byBatch[row.Batch.ID] = append(byBatch[row.Batch.ID], row.RowNumber)
	}
	assert.Equal(t, map[pulid.ID][]int{feed.ID: {2, 3}, upload.ID: {2}}, byBatch)

	elsewhere, err := desk.ListHeldCardRows(f.ctxB, &fuelpurchaseports.ListHeldCardRowsRequest{
		TenantInfo: f.tenantB,
		CardID:     card.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, elsewhere)
}

func TestDeskPurchaseImportRowIsTheRowThePurchaseWasCommittedFrom(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	purchase := f.createPurchase(t, cardPurchase{spec: purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "118.42", taxPaid: true, reference: "PROV-1",
	}})
	manual := f.createPurchase(t, cardPurchase{spec: purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "50", taxPaid: true, reference: "PROV-2",
	}})

	batch := f.feedBatch(t, f.tenantA, fuelpurchase.ImportOriginFeed, fuelpurchase.ImportStatusCommitted)
	purchaseID := purchase.ID
	require.NoError(t, f.repo.ReplaceImportRows(f.ctxA, batch, []*fuelpurchase.ImportRow{{
		RowNumber:       37,
		Cells:           []string{"PROV-1"},
		Status:          fuelpurchase.ImportRowStatusCommitted,
		FuelPurchaseID:  &purchaseID,
		ResolutionNotes: []string{"Tractor matched by license plate FA-1001"},
	}}))

	row, err := desk.GetPurchaseImportRow(f.ctxA, &fuelpurchaseports.GetPurchaseImportRowRequest{
		TenantInfo: f.tenantA,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, 37, row.RowNumber)
	assert.Equal(t, []string{"Tractor matched by license plate FA-1001"}, row.ResolutionNotes)
	require.NotNil(t, row.Batch)
	assert.Equal(t, batch.ID, row.Batch.ID)

	none, err := desk.GetPurchaseImportRow(f.ctxA, &fuelpurchaseports.GetPurchaseImportRowRequest{
		TenantInfo: f.tenantA,
		PurchaseID: manual.ID,
	})
	require.NoError(t, err)
	assert.Nil(t, none)

	elsewhere, err := desk.GetPurchaseImportRow(f.ctxB, &fuelpurchaseports.GetPurchaseImportRowRequest{
		TenantInfo: f.tenantB,
		PurchaseID: purchase.ID,
	})
	require.NoError(t, err)
	assert.Nil(t, elsewhere)
}

func TestDeskRecentCardPurchasesAreNewestFirstWithTheirJurisdiction(t *testing.T) {
	f := setup(t)
	desk := f.deskReader()

	card, err := f.repo.CreateCard(f.ctxA, f.newCard(f.tenantA, "4412"))
	require.NoError(t, err)

	refs := []string{"REC-1", "REC-2", "REC-3"}
	for index, ref := range refs {
		f.createPurchase(t, cardPurchase{spec: purchaseSpec{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.oklahoma,
			gallons: "100", taxPaid: true, reference: ref,
			purchasedAt: windowStart + int64(index)*100,
		}, cardID: card.ID})
	}

	purchases, err := desk.ListRecentCardPurchases(
		f.ctxA,
		&fuelpurchaseports.ListRecentCardPurchasesRequest{
			TenantInfo: f.tenantA,
			CardID:     card.ID,
			Limit:      2,
		},
	)
	require.NoError(t, err)
	require.Len(t, purchases, 2)
	assert.Equal(t, "REC-3", purchases[0].TransactionReference)
	assert.Equal(t, "REC-2", purchases[1].TransactionReference)
	require.NotNil(t, purchases[0].Jurisdiction)
	assert.Equal(t, "OK", purchases[0].Jurisdiction.Code)
}
