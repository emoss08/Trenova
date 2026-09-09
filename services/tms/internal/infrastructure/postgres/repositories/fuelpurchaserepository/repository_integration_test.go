//go:build integration

package fuelpurchaserepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fuelpurchaserepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	purchasedInWindow  = int64(1_783_000_000)
	purchasedOutWindow = int64(1_770_000_000)
	windowStart        = int64(1_780_000_000)
	windowEnd          = int64(1_790_000_000)
)

type fixture struct {
	ctx      context.Context
	db       *bun.DB
	repo     repositories.FuelPurchaseRepository
	tenantA  pagination.TenantInfo
	tenantB  pagination.TenantInfo
	userA    pulid.ID
	tractorA pulid.ID
	tractorB pulid.ID
	texas    pulid.ID
	oklahoma pulid.ID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)

	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{
			SystemUserPassword: "integration-system-password",
		},
	})
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{
		Environment: common.EnvDevelopment,
		Force:       true,
	})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		TableExpr("organizations").
		Column("id", "business_unit_id").
		Order("created_at ASC").
		Limit(1).
		Scan(ctx, &org))
	tenantA := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	var userID pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("users").
		Column("id").
		Where("current_organization_id = ?", org.ID).
		Limit(1).
		Scan(ctx, &userID))

	var tractorA pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("tractors").
		Column("id").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("code ASC").
		Limit(1).
		Scan(ctx, &tractorA))

	other := seedtest.SeedAdditionalTenant(t, ctx, db, "FB")
	tenantB := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}
	tractorB := seedTractor(t, ctx, db, tenantB, other.State.ID)

	f := &fixture{
		ctx: ctx,
		db:  db,
		repo: fuelpurchaserepository.New(fuelpurchaserepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenantA:  tenantA,
		tenantB:  tenantB,
		userA:    userID,
		tractorA: tractorA,
		tractorB: tractorB,
		texas:    jurisdictionID(t, ctx, db, "US", "TX"),
		oklahoma: jurisdictionID(t, ctx, db, "US", "OK"),
	}

	return f
}

func jurisdictionID(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	country, code string,
) pulid.ID {
	t.Helper()

	var id pulid.ID
	require.NoError(t, db.NewSelect().
		TableExpr("ifta_jurisdictions").
		Column("id").
		Where("country_code = ?", country).
		Where("code = ?", code).
		Limit(1).
		Scan(ctx, &id))

	return id
}

func seedTractor(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenant pagination.TenantInfo,
	stateID pulid.ID,
) pulid.ID {
	t.Helper()

	equipType := &equipmenttype.EquipmentType{
		ID:             pulid.MustNew("et_"),
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		Code:           "TRACTOR",
		Description:    "Tractor",
		Class:          equipmenttype.ClassTractor,
		Status:         domaintypes.StatusActive,
	}
	_, err := db.NewInsert().Model(equipType).Exec(ctx)
	require.NoError(t, err)

	manufacturer := &equipmentmanufacturer.EquipmentManufacturer{
		ID:             pulid.MustNew("em_"),
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		Name:           "Freightliner",
		Description:    "Freightliner",
		Status:         domaintypes.StatusActive,
	}
	_, err = db.NewInsert().Model(manufacturer).Exec(ctx)
	require.NoError(t, err)

	driver := &worker.Worker{
		ID:                   pulid.MustNew("wrk_"),
		BusinessUnitID:       tenant.BuID,
		OrganizationID:       tenant.OrgID,
		StateID:              stateID,
		Status:               domaintypes.StatusActive,
		Type:                 worker.WorkerTypeEmployee,
		DriverType:           worker.DriverTypeOTR,
		FirstName:            "Fuel",
		LastName:             "Driver",
		AddressLine1:         "1 Depot Road",
		City:                 "Springfield",
		PostalCode:           "62701",
		Email:                "fuel-driver-fb@example.com",
		PhoneNumber:          "555-0100",
		Gender:               worker.GenderMale,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
	}
	_, err = db.NewInsert().Model(driver).Exec(ctx)
	require.NoError(t, err)

	entity := &tractor.Tractor{
		ID:                      pulid.MustNew("trac_"),
		BusinessUnitID:          tenant.BuID,
		OrganizationID:          tenant.OrgID,
		EquipmentTypeID:         equipType.ID,
		EquipmentManufacturerID: manufacturer.ID,
		PrimaryWorkerID:         driver.ID,
		StateID:                 stateID,
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "FB-TRC-001",
		Make:                    "Freightliner",
		Model:                   "Cascadia",
		LicensePlateNumber:      "FB-1001",
		FuelType:                domaintypes.IFTAFuelTypeDiesel,
		IFTAQualified:           true,
	}
	_, err = db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)

	return entity.ID
}

func (f *fixture) newCard(tenant pagination.TenantInfo, lastFour string) *fuelpurchase.FuelCard {
	return &fuelpurchase.FuelCard{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Provider:       fuelpurchase.CardProviderComdata,
		LastFour:       lastFour,
		Label:          "Card " + lastFour,
		Status:         fuelpurchase.CardStatusActive,
	}
}

type purchaseSpec struct {
	tenant       pagination.TenantInfo
	tractorID    pulid.ID
	jurisdiction pulid.ID
	fuelType     domaintypes.IFTAFuelType
	gallons      string
	taxPaid      bool
	reference    string
	purchasedAt  int64
}

func (f *fixture) newPurchase(spec purchaseSpec) *fuelpurchase.FuelPurchase {
	fuelType := spec.fuelType
	if fuelType == "" {
		fuelType = domaintypes.IFTAFuelTypeDiesel
	}
	purchasedAt := spec.purchasedAt
	if purchasedAt == 0 {
		purchasedAt = purchasedInWindow
	}
	quantity := decimal.RequireFromString(spec.gallons)
	purchase := &fuelpurchase.FuelPurchase{
		OrganizationID:       spec.tenant.OrgID,
		BusinessUnitID:       spec.tenant.BuID,
		TractorID:            spec.tractorID,
		JurisdictionID:       spec.jurisdiction,
		PurchasedAt:          purchasedAt,
		Vendor:               "Pilot",
		FuelType:             fuelType,
		Quantity:             quantity,
		QuantityUnit:         fuelpurchase.QuantityUnitGallon,
		Gallons:              quantity,
		TotalAmountMinor:     quantity.Mul(decimal.RequireFromString("3.899")).Shift(2).IntPart(),
		CurrencyCode:         "USD",
		TransactionReference: spec.reference,
		Source:               fuelpurchase.PurchaseSourceManual,
		TaxPaid:              spec.taxPaid,
		CreatedByID:          f.userA,
	}
	purchase.Normalize()

	return purchase
}

func requireDuplicate(t *testing.T, err error, field string) {
	t.Helper()

	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, field, fieldErr.Field)
	assert.Equal(t, errortypes.ErrDuplicate, fieldErr.Code)
}

func TestCardsAreTenantScopedAndLastFourReusableAfterCancel(t *testing.T) {
	f := setup(t)

	card, err := f.repo.CreateCard(f.ctx, f.newCard(f.tenantA, "1234"))
	require.NoError(t, err)
	require.False(t, card.ID.IsNil())

	_, err = f.repo.CreateCard(f.ctx, f.newCard(f.tenantA, "1234"))
	requireDuplicate(t, err, "lastFour")

	otherTenantCard, err := f.repo.CreateCard(f.ctx, f.newCard(f.tenantB, "1234"))
	require.NoError(t, err)

	_, err = f.repo.GetCardByID(f.ctx, &repositories.GetFuelCardByIDRequest{
		ID:         card.ID,
		TenantInfo: f.tenantB,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	found, err := f.repo.FindCardByLastFour(f.ctx, &repositories.FindFuelCardByLastFourRequest{
		TenantInfo: f.tenantB,
		Provider:   fuelpurchase.CardProviderComdata,
		LastFour:   "1234",
	})
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, otherTenantCard.ID, found.ID)

	active, err := f.repo.ListActiveCards(f.ctx, &repositories.ListActiveFuelCardsRequest{
		TenantInfo: f.tenantA,
	})
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, card.ID, active[0].ID)

	byIDs, err := f.repo.GetCardsByIDs(f.ctx, &repositories.GetFuelCardsByIDsRequest{
		TenantInfo: f.tenantB,
		IDs:        []pulid.ID{card.ID, otherTenantCard.ID},
	})
	require.NoError(t, err)
	require.Len(t, byIDs, 1)
	assert.Equal(t, otherTenantCard.ID, byIDs[0].ID)

	card.Cancel(purchasedInWindow, "Card reported lost by the driver")
	cancelled, err := f.repo.UpdateCard(f.ctx, card)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cancelled.Version)

	stale := *cancelled
	stale.Version = 0
	_, err = f.repo.UpdateCard(f.ctx, &stale)
	require.Error(t, err)

	none, err := f.repo.FindCardByLastFour(f.ctx, &repositories.FindFuelCardByLastFourRequest{
		TenantInfo: f.tenantA,
		Provider:   fuelpurchase.CardProviderComdata,
		LastFour:   "1234",
	})
	require.NoError(t, err)
	assert.Nil(t, none)

	replacement, err := f.repo.CreateCard(f.ctx, f.newCard(f.tenantA, "1234"))
	require.NoError(t, err)
	assert.NotEqual(t, card.ID, replacement.ID)

	listed, err := f.repo.ListCards(f.ctx, &repositories.ListFuelCardsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.tenantA,
			Pagination: pagination.Info{Limit: 10},
		},
		Cursor:   pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		Statuses: []fuelpurchase.CardStatus{fuelpurchase.CardStatusActive},
	})
	require.NoError(t, err)
	require.NotNil(t, listed.TotalCount)
	assert.Equal(t, 1, *listed.TotalCount)
	require.Len(t, listed.Items, 1)
	assert.Equal(t, replacement.ID, listed.Items[0].ID)
}

func TestPurchaseReferencesAreUniquePerTenant(t *testing.T) {
	f := setup(t)

	first, err := f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "100", taxPaid: true, reference: "ref-1",
	}))
	require.NoError(t, err)
	assert.Equal(t, "REF-1", first.TransactionReference)

	_, err = f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "50", taxPaid: true, reference: "REF-1",
	}))
	requireDuplicate(t, err, "transactionReference")

	_, err = f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantB, tractorID: f.tractorB, jurisdiction: f.texas,
		gallons: "50", taxPaid: true, reference: "REF-1",
	}))
	require.NoError(t, err)

	blankOne, err := f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "10", taxPaid: true,
	}))
	require.NoError(t, err)
	blankTwo, err := f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "20", taxPaid: true,
	}))
	require.NoError(t, err)
	assert.NotEqual(t, blankOne.ID, blankTwo.ID)

	_, err = f.repo.GetPurchaseByID(f.ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         first.ID,
		TenantInfo: f.tenantB,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	loaded, err := f.repo.GetPurchaseByID(f.ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:                  first.ID,
		TenantInfo:          f.tenantA,
		IncludeTractor:      true,
		IncludeJurisdiction: true,
	})
	require.NoError(t, err)
	require.NotNil(t, loaded.Tractor)
	require.NotNil(t, loaded.Jurisdiction)
	assert.Equal(t, "TX", loaded.Jurisdiction.Code)

	refs, err := f.repo.FindReferences(f.ctx, &repositories.FindFuelPurchaseReferencesRequest{
		TenantInfo: f.tenantA,
		References: []string{"ref-1", "REF-9", ""},
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]pulid.ID{"REF-1": first.ID}, refs)

	blankTwo.TransactionReference = "REF-1"
	_, err = f.repo.UpdatePurchase(f.ctx, blankTwo)
	requireDuplicate(t, err, "transactionReference")

	err = f.repo.DeletePurchase(f.ctx, &repositories.DeleteFuelPurchaseRequest{
		ID:         first.ID,
		TenantInfo: f.tenantB,
	})
	require.Error(t, err)

	err = f.repo.DeletePurchase(f.ctx, &repositories.DeleteFuelPurchaseRequest{
		ID:         first.ID,
		TenantInfo: f.tenantA,
	})
	require.NoError(t, err)

	page, err := f.repo.ListPurchases(f.ctx, &repositories.ListFuelPurchasesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.tenantA,
			Pagination: pagination.Info{Limit: 10},
		},
		Cursor:    pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		TractorID: f.tractorA,
	})
	require.NoError(t, err)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 2, *page.TotalCount)
}

func (f *fixture) newBatch(tenant pagination.TenantInfo) *fuelpurchase.ImportBatch {
	return &fuelpurchase.ImportBatch{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		Provider:        fuelpurchase.CardProviderEFS,
		Status:          fuelpurchase.ImportStatusPending,
		DefaultCurrency: "USD",
		UploadedByID:    f.userA,
	}
}

func TestCommitImportIsAtomicAndTenantScoped(t *testing.T) {
	f := setup(t)

	existing, err := f.repo.CreatePurchase(f.ctx, f.newPurchase(purchaseSpec{
		tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
		gallons: "100", taxPaid: true, reference: "DUP-1",
	}))
	require.NoError(t, err)

	batch, err := f.repo.CreateImportBatch(f.ctx, f.newBatch(f.tenantA))
	require.NoError(t, err)

	_, err = f.repo.GetImportBatchByID(f.ctx, &repositories.GetImportBatchByIDRequest{
		ID:         batch.ID,
		TenantInfo: f.tenantB,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	references := []string{"NEW-1", "DUP-1", "NEW-2"}
	rows := make([]*fuelpurchase.ImportRow, 0, len(references)+1)
	for i, ref := range references {
		parsed := f.newPurchase(purchaseSpec{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
			gallons: "10", taxPaid: true, reference: ref,
		})
		rows = append(rows, &fuelpurchase.ImportRow{
			RowNumber:            i + 2,
			Cells:                []string{ref},
			Parsed:               parsed,
			TransactionReference: ref,
			Status:               fuelpurchase.ImportRowStatusNew,
		})
	}
	rows = append(rows, &fuelpurchase.ImportRow{
		RowNumber: 5,
		Cells:     []string{"bad"},
		Status:    fuelpurchase.ImportRowStatusError,
		Error:     "quantity must be greater than zero",
	})
	require.NoError(t, f.repo.ReplaceImportRows(f.ctx, batch, rows))

	staged, err := f.repo.GetImportBatchByID(f.ctx, &repositories.GetImportBatchByIDRequest{
		ID:          batch.ID,
		TenantInfo:  f.tenantA,
		IncludeRows: true,
	})
	require.NoError(t, err)
	require.Len(t, staged.Rows, 4)
	assert.Equal(t, 2, staged.Rows[0].RowNumber)
	require.NotNil(t, staged.Rows[0].Parsed)
	assert.Equal(t, "NEW-1", staged.Rows[0].Parsed.TransactionReference)

	staged.RowCount = 4
	staged.ErrorCount = 1
	staged.Summary = &fuelpurchase.ImportSummary{RowCount: 4, NewCount: 3, ErrorCount: 1}
	staged.Rows = nil

	purchases := make([]*fuelpurchase.FuelPurchase, 0, len(rows))
	rowIDByReference := make(map[string]pulid.ID, len(references))
	for _, row := range rows {
		if !row.WillCommit() {
			continue
		}
		purchase := *row.Parsed
		purchase.ID = ""
		purchase.CreatedByID = f.userA
		purchases = append(purchases, &purchase)
		rowIDByReference[row.TransactionReference] = row.ID
	}
	require.Len(t, purchases, 3)

	result, err := f.repo.CommitImport(f.ctx, &repositories.CommitImportRequest{
		Batch:            staged,
		Purchases:        purchases,
		RowIDByReference: rowIDByReference,
		CommittedByID:    f.userA,
		CommittedAt:      purchasedInWindow,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Committed)
	assert.Equal(t, 1, result.AlreadyImported)

	committed, err := f.repo.GetImportBatchByID(f.ctx, &repositories.GetImportBatchByIDRequest{
		ID:          batch.ID,
		TenantInfo:  f.tenantA,
		IncludeRows: true,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.ImportStatusCommitted, committed.Status)
	assert.Equal(t, 2, committed.CommittedCount)
	assert.Equal(t, f.userA, committed.CommittedByID)
	require.NotNil(t, committed.CommittedAt)
	assert.Equal(t, int64(1), committed.Version)

	byRef := make(map[string]*fuelpurchase.ImportRow, len(committed.Rows))
	for _, row := range committed.Rows {
		byRef[row.TransactionReference] = row
	}
	assert.Equal(t, fuelpurchase.ImportRowStatusCommitted, byRef["NEW-1"].Status)
	require.NotNil(t, byRef["NEW-1"].FuelPurchaseID)
	assert.Equal(t, fuelpurchase.ImportRowStatusCommitted, byRef["NEW-2"].Status)
	assert.Equal(t, fuelpurchase.ImportRowStatusAlreadyImported, byRef["DUP-1"].Status)
	assert.Nil(t, byRef["DUP-1"].FuelPurchaseID)
	assert.Equal(t, fuelpurchase.ImportRowStatusError, byRef[""].Status)

	refs, err := f.repo.FindReferences(f.ctx, &repositories.FindFuelPurchaseReferencesRequest{
		TenantInfo: f.tenantA,
		References: references,
	})
	require.NoError(t, err)
	assert.Len(t, refs, 3)
	assert.Equal(t, existing.ID, refs["DUP-1"])

	imported, err := f.repo.GetPurchaseByID(f.ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         *byRef["NEW-1"].FuelPurchaseID,
		TenantInfo: f.tenantA,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.PurchaseSourceCardImport, imported.Source)
	require.NotNil(t, imported.ImportBatchID)
	assert.Equal(t, batch.ID, *imported.ImportBatchID)

	firstPage, err := f.repo.ListImportRows(f.ctx, &repositories.ListImportRowsRequest{
		BatchID:    batch.ID,
		TenantInfo: f.tenantA,
		Cursor:     pagination.CursorInfo{Limit: 2, IncludeTotalCount: true},
	})
	require.NoError(t, err)
	require.NotNil(t, firstPage.TotalCount)
	assert.Equal(t, 4, *firstPage.TotalCount)
	require.Len(t, firstPage.Items, 2)
	assert.True(t, firstPage.HasNextPage)
	assert.Equal(t, 2, firstPage.Items[0].RowNumber)
	assert.Equal(t, 3, firstPage.Items[1].RowNumber)

	after, err := pagination.EncodeCursorFromEntity(firstPage.Items[1])
	require.NoError(t, err)
	secondPage, err := f.repo.ListImportRows(f.ctx, &repositories.ListImportRowsRequest{
		BatchID:    batch.ID,
		TenantInfo: f.tenantA,
		Cursor:     pagination.CursorInfo{Limit: 2, After: after},
	})
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 2)
	assert.False(t, secondPage.HasNextPage)
	assert.Equal(t, 4, secondPage.Items[0].RowNumber)
	assert.Equal(t, 5, secondPage.Items[1].RowNumber)

	onlyErrors, err := f.repo.ListImportRows(f.ctx, &repositories.ListImportRowsRequest{
		BatchID:    batch.ID,
		TenantInfo: f.tenantA,
		Cursor:     pagination.CursorInfo{Limit: 10},
		Statuses:   []fuelpurchase.ImportRowStatus{fuelpurchase.ImportRowStatusError},
	})
	require.NoError(t, err)
	require.Len(t, onlyErrors.Items, 1)

	foreign, err := f.repo.ListImportRows(f.ctx, &repositories.ListImportRowsRequest{
		BatchID:    batch.ID,
		TenantInfo: f.tenantB,
		Cursor:     pagination.CursorInfo{Limit: 10},
	})
	require.NoError(t, err)
	assert.Empty(t, foreign.Items)
}

func TestAccumulateFuelGroupsByTractorJurisdictionAndFuelType(t *testing.T) {
	f := setup(t)

	specs := []purchaseSpec{
		{
			tenant:       f.tenantA,
			tractorID:    f.tractorA,
			jurisdiction: f.texas,
			gallons:      "100",
			taxPaid:      true,
		},
		{
			tenant:       f.tenantA,
			tractorID:    f.tractorA,
			jurisdiction: f.texas,
			gallons:      "50.5",
			taxPaid:      false,
		},
		{
			tenant:       f.tenantA,
			tractorID:    f.tractorA,
			jurisdiction: f.oklahoma,
			gallons:      "80",
			taxPaid:      true,
		},
		{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
			gallons: "10", taxPaid: true, fuelType: domaintypes.IFTAFuelTypeDEF,
		},
		{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
			gallons: "30", taxPaid: true, fuelType: domaintypes.IFTAFuelTypeGasoline,
		},
		{
			tenant: f.tenantA, tractorID: f.tractorA, jurisdiction: f.texas,
			gallons: "999", taxPaid: true, purchasedAt: purchasedOutWindow,
		},
		{
			tenant:       f.tenantB,
			tractorID:    f.tractorB,
			jurisdiction: f.texas,
			gallons:      "500",
			taxPaid:      true,
		},
	}
	for _, spec := range specs {
		_, err := f.repo.CreatePurchase(f.ctx, f.newPurchase(spec))
		require.NoError(t, err)
	}

	rows, err := f.repo.AccumulateFuel(f.ctx, &repositories.AccumulateFuelRequest{
		TenantInfo: f.tenantA,
		Start:      windowStart,
		End:        windowEnd,
	})
	require.NoError(t, err)
	require.Len(t, rows, 3)

	type key struct {
		jurisdiction pulid.ID
		fuelType     domaintypes.IFTAFuelType
	}
	byKey := make(map[key]*repositories.FuelAccumulationRow, len(rows))
	for _, row := range rows {
		assert.Equal(t, f.tractorA, row.TractorID)
		byKey[key{row.JurisdictionID, row.FuelType}] = row
	}

	texasDiesel := byKey[key{f.texas, domaintypes.IFTAFuelTypeDiesel}]
	require.NotNil(t, texasDiesel)
	assert.Equal(t, "150.500", texasDiesel.Gallons.StringFixed(3))
	assert.Equal(t, "100.000", texasDiesel.TaxPaidGallons.StringFixed(3))
	assert.Equal(t, 2, texasDiesel.PurchaseCount)

	oklahomaDiesel := byKey[key{f.oklahoma, domaintypes.IFTAFuelTypeDiesel}]
	require.NotNil(t, oklahomaDiesel)
	assert.Equal(t, "80.000", oklahomaDiesel.Gallons.StringFixed(3))
	assert.Equal(t, "80.000", oklahomaDiesel.TaxPaidGallons.StringFixed(3))
	assert.Equal(t, 1, oklahomaDiesel.PurchaseCount)

	texasGasoline := byKey[key{f.texas, domaintypes.IFTAFuelTypeGasoline}]
	require.NotNil(t, texasGasoline)
	assert.Equal(t, "30.000", texasGasoline.Gallons.StringFixed(3))

	otherTenant, err := f.repo.AccumulateFuel(f.ctx, &repositories.AccumulateFuelRequest{
		TenantInfo: f.tenantB,
		Start:      windowStart,
		End:        windowEnd,
	})
	require.NoError(t, err)
	require.Len(t, otherTenant, 1)
	assert.Equal(t, f.tractorB, otherTenant[0].TractorID)
	assert.Equal(t, "500.000", otherTenant[0].Gallons.StringFixed(3))
}
