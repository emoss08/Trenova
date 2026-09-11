//go:build integration

package iftarepository_test

import (
	"context"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/iftarepository"
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

const testPeriodZone = "America/New_York"

type fixture struct {
	ctx        context.Context
	db         *bun.DB
	repo       repositories.IFTARepository
	tenantA    pagination.TenantInfo
	tenantB    pagination.TenantInfo
	userID     pulid.ID
	shipmentID pulid.ID
	locationID pulid.ID
	tractorID  pulid.ID
	tx         *ifta.Jurisdiction
	ok         *ifta.Jurisdiction
	period     ifta.Period
	start      int64
	end        int64
}

func setup(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
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

	other := seedtest.SeedAdditionalTenant(t, ctx, db, "IFB")
	tenantB := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}

	f := &fixture{
		ctx: ctx,
		db:  db,
		repo: iftarepository.New(iftarepository.Params{
			DB:                postgres.NewTestConnection(db),
			Logger:            zap.NewNop(),
			JurisdictionCache: &FakeJurisdictionCache{},
		}),
		tenantA: tenantA,
		tenantB: tenantB,
		period:  ifta.NewPeriod(2020, 1),
	}
	f.userID = f.scalar(t, "users", "id", "current_organization_id", tenantA.OrgID)
	f.shipmentID = f.tenantScalar(t, "shipments", "id", tenantA)
	f.locationID = f.tenantScalar(t, "locations", "id", tenantA)
	f.tractorID = f.tenantScalar(t, "tractors", "id", tenantA)

	f.tx, err = f.repo.GetJurisdictionByCode(ctx, "US", "TX")
	require.NoError(t, err)
	f.ok, err = f.repo.GetJurisdictionByCode(ctx, "US", "OK")
	require.NoError(t, err)

	loc, err := time.LoadLocation(testPeriodZone)
	require.NoError(t, err)
	f.start, f.end = f.period.Bounds(loc)

	return f
}

func (f *fixture) scalar(t *testing.T, table, column, where string, value any) pulid.ID {
	t.Helper()
	var id pulid.ID
	require.NoError(t, f.db.NewSelect().
		TableExpr(table).
		Column(column).
		Where(where+" = ?", value).
		Limit(1).
		Scan(f.ctx, &id), "no %s row for the fixture", table)
	return id
}

func (f *fixture) tenantScalar(t *testing.T, table, column string, tenant pagination.TenantInfo) pulid.ID {
	t.Helper()
	var id pulid.ID
	require.NoError(t, f.db.NewSelect().
		TableExpr(table).
		Column(column).
		Where("organization_id = ?", tenant.OrgID).
		Where("business_unit_id = ?", tenant.BuID).
		Order("created_at ASC").
		Limit(1).
		Scan(f.ctx, &id), "no %s row for the fixture", table)
	return id
}

type moveSpec struct {
	status      shipment.MoveStatus
	loaded      bool
	distance    float64
	units       string
	completedAt *int64
	updatedAt   *int64
}

func (f *fixture) insertMove(t *testing.T, spec moveSpec) *shipment.ShipmentMove {
	t.Helper()
	distance := spec.distance
	move := &shipment.ShipmentMove{
		OrganizationID: f.tenantA.OrgID,
		BusinessUnitID: f.tenantA.BuID,
		ShipmentID:     f.shipmentID,
		Status:         spec.status,
		Loaded:         spec.loaded,
		Sequence:       1,
		Distance:       &distance,
		DistanceUnits:  spec.units,
	}
	_, err := f.db.NewInsert().Model(move).Exec(f.ctx)
	require.NoError(t, err)

	stop := &shipment.Stop{
		OrganizationID:       f.tenantA.OrgID,
		BusinessUnitID:       f.tenantA.BuID,
		ShipmentMoveID:       move.ID,
		LocationID:           f.locationID,
		Status:               shipment.StopStatusCompleted,
		Sequence:             1,
		ScheduledWindowStart: f.start,
		ActualDeparture:      spec.completedAt,
	}
	_, err = f.db.NewInsert().Model(stop).Exec(f.ctx)
	require.NoError(t, err)

	if spec.updatedAt != nil {
		_, err = f.db.NewUpdate().
			Model((*shipment.ShipmentMove)(nil)).
			Set("updated_at = ?", *spec.updatedAt).
			Where("id = ?", move.ID).
			Where("organization_id = ?", f.tenantA.OrgID).
			Where("business_unit_id = ?", f.tenantA.BuID).
			Exec(f.ctx)
		require.NoError(t, err)
	}

	return move
}

func (f *fixture) insertRow(
	t *testing.T,
	move *shipment.ShipmentMove,
	country, code string,
	distance float64,
	units string,
) {
	t.Helper()
	row := &shipment.ShipmentMoveJurisdictionMile{
		OrganizationID:   f.tenantA.OrgID,
		BusinessUnitID:   f.tenantA.BuID,
		ShipmentMoveID:   move.ID,
		ShipmentID:       move.ShipmentID,
		CountryCode:      country,
		JurisdictionCode: code,
		Sequence:         1,
		Distance:         distance,
		DistanceUnits:    units,
		Loaded:           move.Loaded,
		Source:           shipment.JurisdictionMileSourceRouteCalculation,
		CalculatedAt:     f.start,
	}
	_, err := f.db.NewInsert().Model(row).Exec(f.ctx)
	require.NoError(t, err)
}

func (f *fixture) assign(t *testing.T, move *shipment.ShipmentMove, archived bool) {
	t.Helper()
	tractorID := f.tractorID
	assignment := &shipment.Assignment{
		OrganizationID: f.tenantA.OrgID,
		BusinessUnitID: f.tenantA.BuID,
		ShipmentMoveID: move.ID,
		TractorID:      &tractorID,
		Status:         shipment.AssignmentStatusCompleted,
	}
	if archived {
		archivedAt := f.start + 10
		assignment.ArchivedAt = &archivedAt
	}
	_, err := f.db.NewInsert().Model(assignment).Exec(f.ctx)
	require.NoError(t, err)
}

func (f *fixture) manualEntry(
	t *testing.T,
	tenant pagination.TenantInfo,
	miles string,
	moveID *pulid.ID,
) *ifta.JurisdictionMileageEntry {
	t.Helper()
	entry := &ifta.JurisdictionMileageEntry{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		TractorID:      f.tractorID,
		JurisdictionID: f.tx.ID,
		TraveledAt:     f.start + 3600,
		Year:           f.period.Year,
		Quarter:        f.period.Quarter,
		Miles:          decimal.RequireFromString(miles),
		Loaded:         true,
		Source:         ifta.MileageSourceManual,
		ShipmentMoveID: moveID,
		CreatedByID:    f.userID,
	}
	created, err := f.repo.CreateMileageEntry(f.ctx, entry)
	require.NoError(t, err)
	return created
}

func (f *fixture) draft(t *testing.T, tenant pagination.TenantInfo, amendment int, amends *pulid.ID) *ifta.Return {
	t.Helper()
	ret := &ifta.Return{
		OrganizationID:  tenant.OrgID,
		BusinessUnitID:  tenant.BuID,
		Year:            f.period.Year,
		Quarter:         f.period.Quarter,
		AmendmentNumber: amendment,
		AmendsReturnID:  amends,
		Status:          ifta.ReturnStatusDraft,
		Timezone:        testPeriodZone,
		PeriodStart:     f.start,
		PeriodEnd:       f.end,
		CurrencyCode:    "USD",
	}
	created, err := f.repo.CreateReturn(f.ctx, ret)
	require.NoError(t, err)
	return created
}

func ptr(v int64) *int64 { return &v }

func rowFor(rows []*repositories.MileRow, tractor pulid.ID, code string) *repositories.MileRow {
	for _, row := range rows {
		if row.TractorID == tractor && row.JurisdictionCode == code {
			return row
		}
	}
	return nil
}

func TestAccumulateMiles(t *testing.T) {
	f := setup(t)
	inside := ptr(f.start + 86_400)
	miles := shipment.JurisdictionDistanceUnitsMiles
	km := shipment.JurisdictionDistanceUnitsKilometers

	m1 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 500, units: miles, completedAt: inside})
	f.insertRow(t, m1, "US", "TX", 300, miles)
	f.insertRow(t, m1, "US", "OK", 200, miles)
	f.assign(t, m1, false)

	m2 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: false, distance: 200, units: miles, completedAt: inside})
	f.insertRow(t, m2, "US", "OK", 120, miles)
	f.insertRow(t, m2, "US", "TX", 80, miles)
	f.assign(t, m2, false)

	m3 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 150, units: miles, completedAt: inside})
	f.assign(t, m3, false)

	m4 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 100, units: miles, completedAt: inside})
	f.insertRow(t, m4, "US", "TX", 100, miles)
	f.assign(t, m4, true)

	m5 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 250, units: miles, completedAt: inside})
	f.insertRow(t, m5, "US", "TX", 250, miles)
	f.assign(t, m5, false)
	f.manualEntry(t, f.tenantA, "250", &m5.ID)

	m6 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 100, units: miles, completedAt: inside})
	f.insertRow(t, m6, "US", "TX", 160.9344, km)
	f.assign(t, m6, false)

	m7 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 100, units: miles, updatedAt: ptr(f.start + 7200)})
	f.insertRow(t, m7, "US", "TX", 100, miles)
	f.assign(t, m7, false)

	boundary := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 100, units: miles, completedAt: ptr(f.end)})
	f.insertRow(t, boundary, "US", "TX", 100, miles)
	f.assign(t, boundary, false)

	m9 := f.insertMove(t, moveSpec{status: shipment.MoveStatusCompleted, loaded: true, distance: 300, units: miles, completedAt: inside})
	f.insertRow(t, m9, "US", "TX", 100, miles)
	f.assign(t, m9, false)

	inTransit := f.insertMove(t, moveSpec{status: shipment.MoveStatusInTransit, loaded: true, distance: 100, units: miles, completedAt: inside})
	f.insertRow(t, inTransit, "US", "TX", 100, miles)
	f.assign(t, inTransit, false)

	result, err := f.repo.AccumulateMiles(f.ctx, &repositories.AccumulateMilesRequest{
		TenantInfo: f.tenantA,
		Start:      f.start,
		End:        f.end,
		Year:       f.period.Year,
		Quarter:    f.period.Quarter,
	})
	require.NoError(t, err)

	tx := rowFor(result.RouteRows, f.tractorID, "TX")
	require.NotNil(t, tx, "tractor TX row")
	assert.Equal(t, f.tx.ID, tx.JurisdictionID, "joined to the jurisdiction by code")
	assert.Equal(t, "680.000", tx.Miles.StringFixed(3), "m1 300 + m2 80 + m6 100 + m7 100 + m9 100")
	assert.Equal(t, "600.000", tx.LoadedMiles.StringFixed(3))
	assert.Equal(t, "80.000", tx.EmptyMiles.StringFixed(3))
	assert.Equal(t, 5, tx.MoveCount)

	ok := rowFor(result.RouteRows, f.tractorID, "OK")
	require.NotNil(t, ok, "tractor OK row")
	assert.Equal(t, "320.000", ok.Miles.StringFixed(3))
	assert.Equal(t, "200.000", ok.LoadedMiles.StringFixed(3))
	assert.Equal(t, "120.000", ok.EmptyMiles.StringFixed(3))
	assert.Equal(t, 2, ok.MoveCount)

	noTractor := rowFor(result.RouteRows, pulid.Nil, "TX")
	require.NotNil(t, noTractor, "archived assignment leaves the move without a tractor")
	assert.Equal(t, "100.000", noTractor.Miles.StringFixed(3))
	assert.Equal(t, 1, result.NoTractor.MoveCount)
	assert.Equal(t, "100.000", result.NoTractor.Miles.StringFixed(3))
	assert.Equal(t, []pulid.ID{m4.ID}, result.NoTractor.MoveIDs)

	assert.Equal(t, 1, result.Unattributed.MoveCount)
	assert.Equal(t, "150.000", result.Unattributed.Miles.StringFixed(3))
	assert.Equal(t, []pulid.ID{m3.ID}, result.Unattributed.MoveIDs)

	require.Len(t, result.ManualRows, 1)
	assert.Equal(t, f.tractorID, result.ManualRows[0].TractorID)
	assert.Equal(t, "TX", result.ManualRows[0].JurisdictionCode)
	assert.Equal(t, f.tx.ID, result.ManualRows[0].JurisdictionID)
	assert.Equal(t, "250.00", result.ManualRows[0].Miles.StringFixed(2))
	assert.Equal(t, "250.00", result.ManualRows[0].LoadedMiles.StringFixed(2))

	require.Len(t, result.Mismatches, 1)
	assert.Equal(t, m9.ID, result.Mismatches[0].MoveID)
	assert.Equal(t, "300.000", result.Mismatches[0].MoveDistance.StringFixed(3))
	assert.Equal(t, "100.000", result.Mismatches[0].AttributedMiles.StringFixed(3))

	otherTenant, err := f.repo.AccumulateMiles(f.ctx, &repositories.AccumulateMilesRequest{
		TenantInfo: f.tenantB,
		Start:      f.start,
		End:        f.end,
		Year:       f.period.Year,
		Quarter:    f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Empty(t, otherTenant.RouteRows)
	assert.Empty(t, otherTenant.ManualRows)
	assert.Empty(t, otherTenant.Mismatches)
	assert.Equal(t, 0, otherTenant.Unattributed.MoveCount)
	assert.Equal(t, 0, otherTenant.NoTractor.MoveCount)

	for _, excluded := range []pulid.ID{boundary.ID, inTransit.ID} {
		assert.NotContains(t, result.Unattributed.MoveIDs, excluded)
		assert.NotContains(t, result.NoTractor.MoveIDs, excluded)
	}
}

func TestReturns_TenantIsolationAndOpenReturnRule(t *testing.T) {
	f := setup(t)

	original := f.draft(t, f.tenantA, 0, nil)

	_, err := f.repo.CreateReturn(f.ctx, &ifta.Return{
		OrganizationID: f.tenantA.OrgID,
		BusinessUnitID: f.tenantA.BuID,
		Year:           f.period.Year,
		Quarter:        f.period.Quarter,
		Status:         ifta.ReturnStatusDraft,
		Timezone:       testPeriodZone,
		PeriodStart:    f.start,
		PeriodEnd:      f.end,
	})
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict, "two open returns for one period are rejected")

	open, err := f.repo.GetOpenReturnForPeriod(f.ctx, &repositories.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	require.NotNil(t, open)
	assert.Equal(t, original.ID, open.ID)

	openB, err := f.repo.GetOpenReturnForPeriod(f.ctx, &repositories.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantB, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Nil(t, openB, "the other tenant sees no open return")

	_, err = f.repo.GetReturnByID(f.ctx, &repositories.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantB,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)

	err = f.repo.DeleteReturn(f.ctx, &repositories.DeleteReturnRequest{
		ID: original.ID, TenantInfo: f.tenantB, Version: original.Version,
	})
	require.Error(t, err, "the other tenant cannot delete it")

	otherDraft := f.draft(t, f.tenantB, 0, nil)
	assert.NotEqual(t, original.ID, otherDraft.ID, "each tenant may hold its own open return")

	lines := []*ifta.ReturnLine{{
		JurisdictionID: f.tx.ID,
		FuelType:       domaintypes.IFTAFuelTypeDiesel,
		IsIftaMember:   true,
		TotalMiles:     decimal.NewFromInt(1000),
		TaxableMiles:   decimal.NewFromInt(1000),
		RouteMiles:     decimal.NewFromInt(1000),
		TaxableGallons: decimal.NewFromInt(100),
		RatePerGallon:  decimal.NewNullDecimal(decimal.RequireFromString("0.2000")),
		TaxDueMinor:    2000,
		LineTotalMinor: 2000,
	}}
	original.TotalMiles = decimal.NewFromInt(1000)
	computed, err := f.repo.ReplaceReturnLines(f.ctx, original, lines)
	require.NoError(t, err)
	assert.Equal(t, int64(1), computed.Version)
	require.NotNil(t, computed.ComputedAt)

	loaded, err := f.repo.GetReturnByID(f.ctx, &repositories.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantA, IncludeLines: true, IncludeJurisdictions: true,
	})
	require.NoError(t, err)
	require.Len(t, loaded.Lines, 1)
	assert.Equal(t, "1000", loaded.Lines[0].TotalMiles.String())
	require.NotNil(t, loaded.Lines[0].Jurisdiction)
	assert.Equal(t, "TX", loaded.Lines[0].Jurisdiction.Code)

	recomputed, err := f.repo.ReplaceReturnLines(f.ctx, loaded, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recomputed.Version)
	reloaded, err := f.repo.GetReturnByID(f.ctx, &repositories.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantA, IncludeLines: true,
	})
	require.NoError(t, err)
	assert.Empty(t, reloaded.Lines, "lines are replaced wholesale")

	stale := *reloaded
	stale.Version = 0
	_, err = f.repo.UpdateReturn(f.ctx, &stale)
	require.Error(t, err, "stale version is rejected")
}

func TestReturns_AmendmentCoexistsWithTheFiledOriginal(t *testing.T) {
	f := setup(t)

	original := f.draft(t, f.tenantA, 0, nil)
	now := time.Now().Unix()
	original.Status = ifta.ReturnStatusFiled
	original.FinalizedAt = ptr(now - 120)
	original.FinalizedByID = f.userID
	original.FiledAt = ptr(now - 60)
	original.FiledByID = f.userID
	original.FilingReference = "TX-2020-Q1-0001"
	filed, err := f.repo.UpdateReturn(f.ctx, original)
	require.NoError(t, err)
	assert.Equal(t, ifta.ReturnStatusFiled, filed.Status)

	open, err := f.repo.GetOpenReturnForPeriod(f.ctx, &repositories.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Nil(t, open, "a filed return is not open")

	amendsID := filed.ID
	amendment := f.draft(t, f.tenantA, 1, &amendsID)
	assert.Equal(t, 1, amendment.AmendmentNumber)

	latest, err := f.repo.GetLatestReturnForPeriod(f.ctx, &repositories.GetLatestReturnForPeriodRequest{
		TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, amendment.ID, latest.ID)

	list, err := f.repo.ListReturns(f.ctx, &repositories.ListReturnsRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.tenantA},
		Cursor: pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		Year:   f.period.Year,
	})
	require.NoError(t, err)
	require.NotNil(t, list.TotalCount)
	assert.Equal(t, 2, *list.TotalCount)
	require.Len(t, list.Items, 2)
	assert.Equal(t, amendment.ID, list.Items[0].ID, "highest amendment first")

	listB, err := f.repo.ListReturns(f.ctx, &repositories.ListReturnsRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.tenantB},
		Cursor: pagination.CursorInfo{Limit: 10},
		Year:   f.period.Year,
	})
	require.NoError(t, err)
	assert.Empty(t, listB.Items)
	assert.Nil(t, listB.TotalCount, "count is skipped unless requested")
}

func TestTaxRates_UpsertUpdatesInPlace(t *testing.T) {
	f := setup(t)

	first, err := f.repo.UpsertTaxRates(f.ctx, []*ifta.TaxRate{{
		JurisdictionID: f.tx.ID,
		Year:           f.period.Year,
		Quarter:        f.period.Quarter,
		FuelType:       domaintypes.IFTAFuelTypeDiesel,
		RatePerGallon:  decimal.RequireFromString("0.2000"),
		SourceNote:     "IFTA Inc. matrix",
	}})
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.False(t, first[0].ID.IsNil())
	assert.Equal(t, int64(0), first[0].Version)

	second, err := f.repo.UpsertTaxRates(f.ctx, []*ifta.TaxRate{
		{
			JurisdictionID:         f.tx.ID,
			Year:                   f.period.Year,
			Quarter:                f.period.Quarter,
			FuelType:               domaintypes.IFTAFuelTypeDiesel,
			RatePerGallon:          decimal.RequireFromString("0.2100"),
			SurchargeRatePerGallon: decimal.NewNullDecimal(decimal.RequireFromString("0.0100")),
			SourceNote:             "corrected",
		},
		{
			JurisdictionID: f.ok.ID,
			Year:           f.period.Year,
			Quarter:        f.period.Quarter,
			FuelType:       domaintypes.IFTAFuelTypeDiesel,
			RatePerGallon:  decimal.RequireFromString("0.1900"),
		},
	})
	require.NoError(t, err)
	require.Len(t, second, 2)
	assert.Equal(t, first[0].ID, second[0].ID, "same row updated in place")
	assert.Equal(t, int64(1), second[0].Version)
	assert.Equal(t, "0.2100", second[0].RatePerGallon.StringFixed(4))
	assert.True(t, second[0].SurchargeRatePerGallon.Valid)
	assert.Equal(t, "corrected", second[0].SourceNote)

	resolved, err := f.repo.ResolveRates(f.ctx, &repositories.ResolveRatesRequest{
		Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "0.1900", resolved[ifta.RateKey{
		JurisdictionID: f.ok.ID, FuelType: domaintypes.IFTAFuelTypeDiesel,
	}].RatePerGallon.StringFixed(4))

	list, err := f.repo.ListTaxRates(f.ctx, &repositories.ListTaxRatesRequest{
		Cursor:              pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		Year:                f.period.Year,
		Quarter:             f.period.Quarter,
		IncludeJurisdiction: true,
	})
	require.NoError(t, err)
	require.NotNil(t, list.TotalCount)
	assert.Equal(t, 2, *list.TotalCount)
	require.Len(t, list.Items, 2)
	require.NotNil(t, list.Items[0].Jurisdiction)

	require.NoError(t, f.repo.DeleteTaxRate(f.ctx, second[1].ID, second[1].Version))
	require.Error(t, f.repo.DeleteTaxRate(f.ctx, second[0].ID, second[0].Version+7),
		"a stale version does not delete")
}

func TestMileageEntries_TenantIsolation(t *testing.T) {
	f := setup(t)

	entry := f.manualEntry(t, f.tenantA, "123.45", nil)
	assert.False(t, entry.ID.IsNil())

	_, err := f.repo.GetMileageEntryByID(f.ctx, &repositories.GetMileageEntryByIDRequest{
		ID: entry.ID, TenantInfo: f.tenantB,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)

	listB, err := f.repo.ListMileageEntries(f.ctx, &repositories.ListMileageEntriesRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.tenantB},
		Cursor: pagination.CursorInfo{Limit: 10},
	})
	require.NoError(t, err)
	assert.Empty(t, listB.Items)

	listA, err := f.repo.ListMileageEntries(f.ctx, &repositories.ListMileageEntriesRequest{
		Filter:         &pagination.QueryOptions{TenantInfo: f.tenantA},
		Cursor:         pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		TractorID:      f.tractorID,
		Year:           f.period.Year,
		Quarter:        f.period.Quarter,
		IncludeTractor: true,
	})
	require.NoError(t, err)
	require.Len(t, listA.Items, 1)
	require.NotNil(t, listA.Items[0].Tractor)

	err = f.repo.DeleteMileageEntry(f.ctx, &repositories.DeleteMileageEntryRequest{
		ID: entry.ID, TenantInfo: f.tenantB, Version: entry.Version,
	})
	require.Error(t, err, "the other tenant cannot delete it")

	entry.Miles = decimal.RequireFromString("200.00")
	updated, err := f.repo.UpdateMileageEntry(f.ctx, entry)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.Version)

	require.NoError(t, f.repo.DeleteMileageEntry(f.ctx, &repositories.DeleteMileageEntryRequest{
		ID: entry.ID, TenantInfo: f.tenantA, Version: updated.Version,
	}))
}

func TestJurisdictions_LookupByCode(t *testing.T) {
	f := setup(t)

	found, err := f.repo.FindJurisdictionsByCodes(f.ctx, []string{"US_TX", "TX", "ca-on", "MX_CH"})
	require.NoError(t, err)
	assert.Equal(t, f.tx.ID, found["US_TX"].ID)
	assert.Equal(t, f.tx.ID, found["TX"].ID, "a bare code is a US state")
	require.NotNil(t, found["ca-on"])
	assert.Equal(t, "ON", found["ca-on"].Code)
	_, unknown := found["MX_CH"]
	assert.False(t, unknown)

	members, err := f.repo.ListJurisdictions(f.ctx, &repositories.ListJurisdictionsRequest{MembersOnly: true})
	require.NoError(t, err)
	all, err := f.repo.ListJurisdictions(f.ctx, &repositories.ListJurisdictionsRequest{})
	require.NoError(t, err)
	assert.Less(t, len(members), len(all))
	for _, j := range members {
		assert.True(t, j.IsIftaMember)
	}
}
