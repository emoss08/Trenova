//go:build integration

package ratematrixversionrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/ratematrixversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen/ratematrixcol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type gridCell struct {
	origin, destination, value string
}

func publishGrid(
	t *testing.T,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	code string,
	status formulatemplate.Status,
	cells ...gridCell,
) *ratematrix.RateMatrix {
	t.Helper()

	matrix := &ratematrix.RateMatrix{
		ID:                     pulid.MustNew("rmx_"),
		OrganizationID:         tenantInfo.OrgID,
		BusinessUnitID:         tenantInfo.BuID,
		Code:                   code,
		Name:                   code,
		Status:                 status,
		ValueType:              ratematrix.ValueTypeMoney + "/mi",
		PublishedVersionNumber: 1,
		DraftCellCount:         int64(len(cells)),
	}
	for position := range int16(2) {
		matrix.Dimensions = append(matrix.Dimensions, &ratematrix.RateMatrixDimension{
			ID:             pulid.MustNew("rmd_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			Position:       position,
			Kind:           ratematrix.DimensionKindCustom,
			MatchMode:      ratematrix.MatchModeExact,
		})
	}
	draftCells := make([]*ratematrix.RateMatrixCell, 0, len(cells))
	for _, cell := range cells {
		draftCells = append(draftCells, &ratematrix.RateMatrixCell{
			ID:             pulid.MustNew("rmc_"),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			RateMatrixID:   matrix.ID,
			D0Key:          cell.origin,
			D1Key:          cell.destination,
			Value:          decimal.RequireFromString(cell.value),
		})
	}

	_, err := db.NewInsert().Model(matrix).Exec(t.Context())
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&matrix.Dimensions).Exec(t.Context())
	require.NoError(t, err)
	if len(draftCells) > 0 {
		_, err = db.NewInsert().Model(&draftCells).Exec(t.Context())
		require.NoError(t, err)
	}

	published := ratematrix.CopyPublished(matrix, draftCells, 1, pulid.Nil)
	_, err = db.NewInsert().Model(published.Version).Exec(t.Context())
	require.NoError(t, err)
	_, err = db.NewInsert().Model(&published.Dimensions).Exec(t.Context())
	require.NoError(t, err)
	if len(published.Cells) > 0 {
		_, err = db.NewInsert().Model(&published.Cells).Exec(t.Context())
		require.NoError(t, err)
	}

	return matrix
}

func TestTableReadsServeTheTariffHostByVersion(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	connection := postgres.NewTestConnection(db)

	grid := publishGrid(t, db, tenantInfo, "ZONES", formulatemplate.StatusActive,
		gridCell{"SE", "MW", "2.35"},
		gridCell{"SE", "NE", "2.60"},
		gridCell{"SW", "MW", "2.15"},
	)
	retired := publishGrid(t, db, tenantInfo, "RETIRED", formulatemplate.StatusInactive,
		gridCell{"SE", "MW", "9"})

	versions := ratematrixversionrepository.New(ratematrixversionrepository.Params{
		DB:     connection,
		Logger: zap.NewNop(),
	})
	matrices := ratematrixrepository.New(ratematrixrepository.Params{
		DB:     connection,
		Logger: zap.NewNop(),
	})

	served, err := versions.ListServed(ctx, tenantInfo)
	require.NoError(t, err)
	servedIDs := make([]pulid.ID, 0, len(served))
	for _, version := range served {
		servedIDs = append(servedIDs, version.RateMatrixID)
	}
	assert.Contains(t, servedIDs, grid.ID)
	assert.NotContains(t, servedIDs, retired.ID, "an archived matrix is not served")

	stamp, err := versions.ServedStamp(ctx, tenantInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, stamp)

	_, err = db.NewUpdate().
		Model((*ratematrix.RateMatrixCell)(nil)).
		Set(ratematrixcol.RateMatrixCellColumns.Value.Set(), decimal.RequireFromString("7.77")).
		Where(ratematrixcol.RateMatrixCellColumns.RateMatrixID.Eq(), grid.ID).
		Where(ratematrixcol.RateMatrixCellColumns.VersionNumber.Eq(), ratematrix.DraftVersion).
		Exec(ctx)
	require.NoError(t, err)

	v1 := ratematrix.VersionKey{RateMatrixID: grid.ID, VersionNumber: 1}
	cells, err := versions.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo: tenantInfo,
		Key:        v1,
		Keys:       map[int16][]string{0: {"SE"}, 1: {"MW", "NE"}},
	})
	require.NoError(t, err)
	require.Len(t, cells, 2)
	for _, cell := range cells {
		assert.Equal(t, "SE", cell.D0Key)
		assert.EqualValues(t, 1, cell.VersionNumber)
		assert.NotEqual(t, "7.77", cell.Value.String(), "editing the draft never changes v1")
	}

	origins, err := versions.ListAxisKeys(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo: tenantInfo,
		Key:        v1,
		Position:   0,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"SE", "SW"}, origins, "each key once")

	whole, err := versions.ListTableCells(ctx, &ratematrixports.ListTableCellsRequest{
		TenantInfo: tenantInfo,
		Keys:       []ratematrix.VersionKey{v1},
	})
	require.NoError(t, err)
	assert.Len(t, whole, 3)

	pinned, err := versions.GetByKeys(ctx, &ratematrixports.GetVersionsByKeysRequest{
		TenantInfo: tenantInfo,
		Keys:       []ratematrix.VersionKey{v1, {RateMatrixID: retired.ID, VersionNumber: 1}},
	})
	require.NoError(t, err)
	assert.Len(t, pinned, 2, "a pinned version reads even when its matrix is archived")
	for _, version := range pinned {
		assert.Len(t, version.Dimensions, 2)
	}

	copied, err := matrices.CopyVersion(ctx, &ratematrixports.CopyVersionRequest{
		TenantInfo:   tenantInfo,
		RateMatrixID: grid.ID,
		From:         ratematrix.DraftVersion,
		To:           2,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3, copied)
	misfits, err := matrices.CountMisfitCells(ctx, &ratematrixports.CountMisfitCellsRequest{
		TenantInfo:    tenantInfo,
		RateMatrixID:  grid.ID,
		VersionNumber: 2,
		Dimensions:    grid.Dimensions,
	})
	require.NoError(t, err)
	assert.Zero(t, misfits)

	foreign := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	other, err := versions.LoadTableCells(ctx, &ratematrixports.LoadTableCellsRequest{
		TenantInfo: foreign,
		Key:        v1,
		Keys:       map[int16][]string{0: {"SE"}},
	})
	require.NoError(t, err)
	assert.Empty(t, other, "another tenant reads none of the cells")
	otherServed, err := versions.ListServed(ctx, foreign)
	require.NoError(t, err)
	assert.Empty(t, otherServed)
}

func TestPostalRangesListOncePerAxis(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	grid := publishGrid(t, db, tenantInfo, "LANES", formulatemplate.StatusActive)

	cells := []*ratematrix.RateMatrixCell{
		{D0Key: "30000", D0KeyTo: "31999", D1Key: "A"},
		{D0Key: "30000", D0KeyTo: "31999", D1Key: "B"},
		{D0Key: "32000", D0KeyTo: "34999", D1Key: "A"},
	}
	for _, cell := range cells {
		cell.ID = pulid.MustNew("rmc_")
		cell.OrganizationID = tenantInfo.OrgID
		cell.BusinessUnitID = tenantInfo.BuID
		cell.RateMatrixID = grid.ID
		cell.VersionNumber = 1
		cell.Value = decimal.NewFromInt(1)
	}
	_, err := db.NewInsert().Model(&cells).Exec(ctx)
	require.NoError(t, err)

	versions := ratematrixversionrepository.New(ratematrixversionrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	ranges, err := versions.ListAxisRanges(ctx, &ratematrixports.ListAxisKeysRequest{
		TenantInfo: tenantInfo,
		Key:        ratematrix.VersionKey{RateMatrixID: grid.ID, VersionNumber: 1},
		Position:   0,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ratematrix.PostalRange{
		{From: "30000", To: "31999"},
		{From: "32000", To: "34999"},
	}, ranges)
}
