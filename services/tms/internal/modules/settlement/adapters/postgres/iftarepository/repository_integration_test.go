//go:build integration

package iftarepository_test

import (
	"context"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/iftarepository"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/internal/testutil/cachetest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testPeriodZone = "America/New_York"

type fixture struct {
	ctx        context.Context
	ctxA       context.Context
	ctxB       context.Context
	db         *keel.DB
	dsn        string
	repo       iftaports.IFTARepository
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

	ctx, db, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	orgs, err := keel.QueryAll(
		ctx,
		db,
		pgx.RowToStructByPos[orgRow],
		"SELECT id, business_unit_id FROM organizations ORDER BY created_at ASC LIMIT 1",
	)
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	tenantA := pagination.TenantInfo{OrgID: orgs[0].ID, BuID: orgs[0].BusinessUnitID}

	other := seedtest.SeedAdditionalTenant(t, ctx, db, "IFB")
	tenantB := pagination.TenantInfo{OrgID: other.Organization.ID, BuID: other.BusinessUnit.ID}

	f := &fixture{
		ctx:  ctx,
		ctxA: dbscope.WithTenant(ctx, tenantA.DBTenant()),
		ctxB: dbscope.WithTenant(ctx, tenantB.DBTenant()),
		db:   db,
		dsn:  dsn,
		repo: iftarepository.New(iftarepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenantA: tenantA,
		tenantB: tenantB,
		period:  ifta.NewPeriod(2020, 1),
	}
	f.userID = f.scalar(t, "users", "id", "current_organization_id", tenantA.OrgID)
	f.shipmentID = f.tenantScalar(t, "shipments", "id", tenantA)
	f.locationID = f.tenantScalar(t, "locations", "id", tenantA)
	f.tractorID = f.tenantScalar(t, "tractors", "id", tenantA)

	f.tx, err = f.repo.GetJurisdictionByCode(f.ctxA, "US", "TX")
	require.NoError(t, err)
	f.ok, err = f.repo.GetJurisdictionByCode(f.ctxA, "US", "OK")
	require.NoError(t, err)

	loc, err := time.LoadLocation(testPeriodZone)
	require.NoError(t, err)
	f.start, f.end = f.period.Bounds(loc)

	return f
}

type orgRow struct {
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

func (f *fixture) as(tenant pagination.TenantInfo) context.Context {
	return dbscope.WithTenant(f.ctx, tenant.DBTenant())
}

func (f *fixture) scalar(t *testing.T, table, column, where string, value any) pulid.ID {
	t.Helper()
	id, err := keel.QueryValue[pulid.ID](
		f.ctx,
		f.db,
		"SELECT "+column+" FROM "+table+" WHERE "+where+" = $1 LIMIT 1",
		value,
	)
	require.NoError(t, err, "no %s row for the fixture", table)
	return id
}

func (f *fixture) tenantScalar(
	t *testing.T,
	table, column string,
	tenant pagination.TenantInfo,
) pulid.ID {
	t.Helper()
	id, err := keel.QueryValue[pulid.ID](
		f.ctx,
		f.db,
		"SELECT "+column+" FROM "+table+
			" WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY created_at ASC LIMIT 1",
		tenant.OrgID.String(),
		tenant.BuID.String(),
	)
	require.NoError(t, err, "no %s row for the fixture", table)
	return id
}

type moveSpec struct {
	status      shipmentvocab.MoveStatus
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
	require.NoError(t, shipmentdb.ShipmentMoves.Create(f.ctxA, f.db, move))

	stop := &shipment.Stop{
		OrganizationID:       f.tenantA.OrgID,
		BusinessUnitID:       f.tenantA.BuID,
		ShipmentMoveID:       move.ID,
		LocationID:           f.locationID,
		Status:               shipmentvocab.StopStatusCompleted,
		Sequence:             1,
		ScheduledWindowStart: f.start,
		ActualDeparture:      spec.completedAt,
	}
	require.NoError(t, shipmentdb.Stops.Create(f.ctxA, f.db, stop))

	if spec.updatedAt != nil {
		_, err := keel.Exec(
			f.ctxA,
			f.db,
			"UPDATE shipment_moves SET updated_at = $1 WHERE id = $2 AND organization_id = $3 AND business_unit_id = $4",
			*spec.updatedAt,
			move.ID.String(),
			f.tenantA.OrgID.String(),
			f.tenantA.BuID.String(),
		)
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
		Source:           shipmentvocab.JurisdictionMileSourceRouteCalculation,
		CalculatedAt:     f.start,
	}
	require.NoError(t, shipmentdb.ShipmentMoveJurisdictionMiles.Create(f.ctxA, f.db, row))
}

func (f *fixture) assign(t *testing.T, move *shipment.ShipmentMove, archived bool) {
	t.Helper()
	tractorID := f.tractorID
	assignment := &shipment.Assignment{
		OrganizationID: f.tenantA.OrgID,
		BusinessUnitID: f.tenantA.BuID,
		ShipmentMoveID: move.ID,
		TractorID:      &tractorID,
		Status:         shipmentvocab.AssignmentStatusCompleted,
	}
	if archived {
		archivedAt := f.start + 10
		assignment.ArchivedAt = &archivedAt
	}
	require.NoError(t, shipmentdb.Assignments.Create(f.ctxA, f.db, assignment))
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
	created, err := f.repo.CreateMileageEntry(f.as(tenant), entry)
	require.NoError(t, err)
	return created
}

func (f *fixture) draft(
	t *testing.T,
	tenant pagination.TenantInfo,
	amendment int,
	amends *pulid.ID,
) *ifta.Return {
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
	created, err := f.repo.CreateReturn(f.as(tenant), ret)
	require.NoError(t, err)
	return created
}

func ptr(v int64) *int64 { return &v }

func rowFor(rows []*iftaports.MileRow, tractor pulid.ID, code string) *iftaports.MileRow {
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
	miles := shipmentvocab.JurisdictionDistanceUnitsMiles
	km := shipmentvocab.JurisdictionDistanceUnitsKilometers

	m1 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    500,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m1, "US", "TX", 300, miles)
	f.insertRow(t, m1, "US", "OK", 200, miles)
	f.assign(t, m1, false)

	m2 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      false,
			distance:    200,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m2, "US", "OK", 120, miles)
	f.insertRow(t, m2, "US", "TX", 80, miles)
	f.assign(t, m2, false)

	m3 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    150,
			units:       miles,
			completedAt: inside,
		},
	)
	f.assign(t, m3, false)

	m4 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    100,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m4, "US", "TX", 100, miles)
	f.assign(t, m4, true)

	m5 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    250,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m5, "US", "TX", 250, miles)
	f.assign(t, m5, false)
	f.manualEntry(t, f.tenantA, "250", &m5.ID)

	m6 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    100,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m6, "US", "TX", 160.9344, km)
	f.assign(t, m6, false)

	m7 := f.insertMove(
		t,
		moveSpec{
			status:    shipmentvocab.MoveStatusCompleted,
			loaded:    true,
			distance:  100,
			units:     miles,
			updatedAt: ptr(f.start + 7200),
		},
	)
	f.insertRow(t, m7, "US", "TX", 100, miles)
	f.assign(t, m7, false)

	boundary := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    100,
			units:       miles,
			completedAt: ptr(f.end),
		},
	)
	f.insertRow(t, boundary, "US", "TX", 100, miles)
	f.assign(t, boundary, false)

	m9 := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusCompleted,
			loaded:      true,
			distance:    300,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, m9, "US", "TX", 100, miles)
	f.assign(t, m9, false)

	inTransit := f.insertMove(
		t,
		moveSpec{
			status:      shipmentvocab.MoveStatusInTransit,
			loaded:      true,
			distance:    100,
			units:       miles,
			completedAt: inside,
		},
	)
	f.insertRow(t, inTransit, "US", "TX", 100, miles)
	f.assign(t, inTransit, false)

	result, err := f.repo.AccumulateMiles(f.ctxA, &iftaports.AccumulateMilesRequest{
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

	otherTenant, err := f.repo.AccumulateMiles(f.ctxB, &iftaports.AccumulateMilesRequest{
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

	_, err := f.repo.CreateReturn(f.ctxA, &ifta.Return{
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

	open, err := f.repo.GetOpenReturnForPeriod(f.ctxA, &iftaports.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	require.NotNil(t, open)
	assert.Equal(t, original.ID, open.ID)

	openB, err := f.repo.GetOpenReturnForPeriod(f.ctxB, &iftaports.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantB, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Nil(t, openB, "the other tenant sees no open return")

	_, err = f.repo.GetReturnByID(f.ctxB, &iftaports.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantB,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)

	err = f.repo.DeleteReturn(f.ctxB, &iftaports.DeleteReturnRequest{
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
	computed, err := f.repo.ReplaceReturnLines(f.ctxA, original, lines)
	require.NoError(t, err)
	assert.Equal(t, int64(1), computed.Version)
	require.NotNil(t, computed.ComputedAt)

	loaded, err := f.repo.GetReturnByID(f.ctxA, &iftaports.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantA, IncludeLines: true, IncludeJurisdictions: true,
	})
	require.NoError(t, err)
	require.Len(t, loaded.Lines, 1)
	assert.Equal(t, "1000", loaded.Lines[0].TotalMiles.String())
	require.NotNil(t, loaded.Lines[0].Jurisdiction)
	assert.Equal(t, "TX", loaded.Lines[0].Jurisdiction.Code)

	recomputed, err := f.repo.ReplaceReturnLines(f.ctxA, loaded, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), recomputed.Version)
	reloaded, err := f.repo.GetReturnByID(f.ctxA, &iftaports.GetReturnByIDRequest{
		ID: original.ID, TenantInfo: f.tenantA, IncludeLines: true,
	})
	require.NoError(t, err)
	assert.Empty(t, reloaded.Lines, "lines are replaced wholesale")

	stale := *reloaded
	stale.Version = 0
	_, err = f.repo.UpdateReturn(f.ctxA, &stale)
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
	filed, err := f.repo.UpdateReturn(f.ctxA, original)
	require.NoError(t, err)
	assert.Equal(t, ifta.ReturnStatusFiled, filed.Status)

	open, err := f.repo.GetOpenReturnForPeriod(f.ctxA, &iftaports.GetOpenReturnForPeriodRequest{
		TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Nil(t, open, "a filed return is not open")

	amendsID := filed.ID
	amendment := f.draft(t, f.tenantA, 1, &amendsID)
	assert.Equal(t, 1, amendment.AmendmentNumber)

	latest, err := f.repo.GetLatestReturnForPeriod(
		f.ctxA,
		&iftaports.GetLatestReturnForPeriodRequest{
			TenantInfo: f.tenantA, Year: f.period.Year, Quarter: f.period.Quarter,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, amendment.ID, latest.ID)

	list, err := f.repo.ListReturns(f.ctxA, &iftaports.ListReturnsRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.tenantA},
		Cursor: pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
		Year:   f.period.Year,
	})
	require.NoError(t, err)
	require.NotNil(t, list.TotalCount)
	assert.Equal(t, 2, *list.TotalCount)
	require.Len(t, list.Items, 2)
	assert.Equal(t, amendment.ID, list.Items[0].ID, "highest amendment first")

	listB, err := f.repo.ListReturns(f.ctxB, &iftaports.ListReturnsRequest{
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

	first, err := f.repo.UpsertTaxRates(f.ctxA, []*ifta.TaxRate{{
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

	second, err := f.repo.UpsertTaxRates(f.ctxA, []*ifta.TaxRate{
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

	resolved, err := f.repo.ResolveRates(f.ctxA, &iftaports.ResolveRatesRequest{
		Year: f.period.Year, Quarter: f.period.Quarter,
	})
	require.NoError(t, err)
	assert.Len(t, resolved, 2)
	assert.Equal(t, "0.1900", resolved[ifta.RateKey{
		JurisdictionID: f.ok.ID, FuelType: domaintypes.IFTAFuelTypeDiesel,
	}].RatePerGallon.StringFixed(4))

	list, err := f.repo.ListTaxRates(f.ctxA, &iftaports.ListTaxRatesRequest{
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

	require.NoError(t, f.repo.DeleteTaxRate(f.ctxA, second[1].ID, second[1].Version))
	require.Error(t, f.repo.DeleteTaxRate(f.ctxA, second[0].ID, second[0].Version+7),
		"a stale version does not delete")
}

func TestMileageEntries_TenantIsolation(t *testing.T) {
	f := setup(t)

	entry := f.manualEntry(t, f.tenantA, "123.45", nil)
	assert.False(t, entry.ID.IsNil())

	_, err := f.repo.GetMileageEntryByID(f.ctxB, &iftaports.GetMileageEntryByIDRequest{
		ID: entry.ID, TenantInfo: f.tenantB,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)

	listB, err := f.repo.ListMileageEntries(f.ctxB, &iftaports.ListMileageEntriesRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.tenantB},
		Cursor: pagination.CursorInfo{Limit: 10},
	})
	require.NoError(t, err)
	assert.Empty(t, listB.Items)

	listA, err := f.repo.ListMileageEntries(f.ctxA, &iftaports.ListMileageEntriesRequest{
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

	err = f.repo.DeleteMileageEntry(f.ctxB, &iftaports.DeleteMileageEntryRequest{
		ID: entry.ID, TenantInfo: f.tenantB, Version: entry.Version,
	})
	require.Error(t, err, "the other tenant cannot delete it")

	entry.Miles = decimal.RequireFromString("200.00")
	updated, err := f.repo.UpdateMileageEntry(f.ctxA, entry)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.Version)

	require.NoError(t, f.repo.DeleteMileageEntry(f.ctxA, &iftaports.DeleteMileageEntryRequest{
		ID: entry.ID, TenantInfo: f.tenantA, Version: updated.Version,
	}))
}

func TestJurisdictions_LookupByCode(t *testing.T) {
	f := setup(t)

	found, err := f.repo.FindJurisdictionsByCodes(f.ctxA, []string{"US_TX", "TX", "ca-on", "MX_CH"})
	require.NoError(t, err)
	assert.Equal(t, f.tx.ID, found["US_TX"].ID)
	assert.Equal(t, f.tx.ID, found["TX"].ID, "a bare code is a US state")
	require.NotNil(t, found["ca-on"])
	assert.Equal(t, "ON", found["ca-on"].Code)
	_, unknown := found["MX_CH"]
	assert.False(t, unknown)

	members, err := f.repo.ListJurisdictions(
		f.ctxA,
		&iftaports.ListJurisdictionsRequest{MembersOnly: true},
	)
	require.NoError(t, err)
	all, err := f.repo.ListJurisdictions(f.ctxA, &iftaports.ListJurisdictionsRequest{})
	require.NoError(t, err)
	assert.Less(t, len(members), len(all))
	for _, j := range members {
		assert.True(t, j.IsIftaMember)
	}
}

func TestJurisdictionLookupsShareOneCachedCatalogRead(t *testing.T) {
	f := setup(t)

	cached := cachetest.Open(t, f.ctx, f.dsn)
	repo := iftarepository.New(iftarepository.Params{
		DB:     postgres.NewTestConnection(cached),
		Logger: zap.NewNop(),
	})

	var all []*ifta.Jurisdiction
	cachetest.RequireMiss(t, cached, func() error {
		var err error
		all, err = repo.ListJurisdictions(f.ctxA, &iftaports.ListJurisdictionsRequest{})
		return err
	})
	require.NotEmpty(t, all)

	var texas *ifta.Jurisdiction
	cachetest.RequireHit(t, cached, func() error {
		var err error
		texas, err = repo.GetJurisdictionByCode(f.ctxA, "us", " tx ")
		return err
	})
	assert.Equal(t, "TX", texas.Code)

	cachetest.RequireHit(t, cached, func() error {
		found, err := repo.GetJurisdictionsByIDs(f.ctxA, []pulid.ID{texas.ID})
		if err == nil {
			assert.Len(t, found, 1)
		}
		return err
	})
}
