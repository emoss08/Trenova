package tariff

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeVersions struct {
	mu       sync.Mutex
	versions []*ratematrix.RateMatrixVersion
	served   map[pulid.ID]int64
	cells    []*ratematrix.RateMatrixCell
	loads    int
	lookups  []*ratematrixports.LoadTableCellsRequest
}

func (f *fakeVersions) Create(
	_ context.Context,
	version *ratematrix.RateMatrixVersion,
) (*ratematrix.RateMatrixVersion, error) {
	return version, nil
}

func (f *fakeVersions) Get(
	_ context.Context,
	req *ratematrixports.GetVersionRequest,
) (*ratematrix.RateMatrixVersion, error) {
	for _, version := range f.versions {
		if version.RateMatrixID == req.RateMatrixID && version.VersionNumber == req.VersionNumber {
			return version, nil
		}
	}
	return nil, nil
}

func (f *fakeVersions) List(
	context.Context,
	*ratematrixports.ListVersionsRequest,
) ([]*ratematrix.RateMatrixVersion, error) {
	return f.versions, nil
}

func (f *fakeVersions) GetByKeys(
	_ context.Context,
	req *ratematrixports.GetVersionsByKeysRequest,
) ([]*ratematrix.RateMatrixVersion, error) {
	f.mu.Lock()
	f.loads++
	f.mu.Unlock()
	out := make([]*ratematrix.RateMatrixVersion, 0, len(req.Keys))
	for _, version := range f.versions {
		if slices.Contains(req.Keys, version.Key()) {
			out = append(out, version)
		}
	}
	return out, nil
}

func (f *fakeVersions) ListServed(
	context.Context,
	pagination.TenantInfo,
) ([]*ratematrix.RateMatrixVersion, error) {
	out := make([]*ratematrix.RateMatrixVersion, 0, len(f.served))
	for _, version := range f.versions {
		if f.served[version.RateMatrixID] == version.VersionNumber {
			out = append(out, version)
		}
	}
	return out, nil
}

func (f *fakeVersions) ServedStamp(context.Context, pagination.TenantInfo) (string, error) {
	parts := make([]string, 0, len(f.served))
	for id, number := range f.served {
		parts = append(parts, id.String()+"@"+decimal.NewFromInt(number).String())
	}
	slices.Sort(parts)
	return strings.Join(parts, ","), nil
}

func (f *fakeVersions) LoadTableCells(
	_ context.Context,
	req *ratematrixports.LoadTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	f.mu.Lock()
	f.lookups = append(f.lookups, req)
	f.mu.Unlock()
	out := make([]*ratematrix.RateMatrixCell, 0, len(f.cells))
	for _, cell := range f.cells {
		if cell.RateMatrixID != req.Key.RateMatrixID || cell.VersionNumber != req.Key.VersionNumber {
			continue
		}
		matches := true
		for position, keys := range req.Keys {
			if !slices.Contains(keys, cell.KeyAt(position)) {
				matches = false
			}
		}
		if matches {
			out = append(out, cell)
		}
	}
	return out, nil
}

func (f *fakeVersions) ListTableCells(
	_ context.Context,
	req *ratematrixports.ListTableCellsRequest,
) ([]*ratematrix.RateMatrixCell, error) {
	out := make([]*ratematrix.RateMatrixCell, 0, len(f.cells))
	for _, cell := range f.cells {
		key := ratematrix.VersionKey{RateMatrixID: cell.RateMatrixID, VersionNumber: cell.VersionNumber}
		if slices.Contains(req.Keys, key) {
			out = append(out, cell)
		}
	}
	return out, nil
}

func (f *fakeVersions) ListAxisKeys(
	context.Context,
	*ratematrixports.ListAxisKeysRequest,
) ([]string, error) {
	return nil, nil
}

func (f *fakeVersions) ListAxisRanges(
	_ context.Context,
	req *ratematrixports.ListAxisKeysRequest,
) ([]ratematrix.PostalRange, error) {
	seen := make(map[ratematrix.PostalRange]struct{}, len(f.cells))
	out := make([]ratematrix.PostalRange, 0, len(f.cells))
	for _, cell := range f.cells {
		if cell.RateMatrixID != req.Key.RateMatrixID || cell.VersionNumber != req.Key.VersionNumber {
			continue
		}
		postal := cell.PostalRangeAt(req.Position)
		if _, dup := seen[postal]; dup {
			continue
		}
		seen[postal] = struct{}{}
		out = append(out, postal)
	}
	return out, nil
}

var _ ratematrixports.RateMatrixVersionRepository = (*fakeVersions)(nil)

func zoneVersion(matrixID pulid.ID, number int64, valueType string) *ratematrix.RateMatrixVersion {
	return &ratematrix.RateMatrixVersion{
		RateMatrixID:  matrixID,
		VersionNumber: number,
		Code:          "ZONES",
		Name:          "Zones",
		ValueType:     valueType,
		Dimensions: []*ratematrix.RateMatrixDimension{{
			RateMatrixID:  matrixID,
			VersionNumber: number,
			Position:      0,
			Kind:          ratematrix.DimensionKindCustom,
			MatchMode:     ratematrix.MatchModeExact,
		}},
	}
}

func zoneCell(matrixID pulid.ID, number int64, key, value string) *ratematrix.RateMatrixCell {
	return &ratematrix.RateMatrixCell{
		RateMatrixID:  matrixID,
		VersionNumber: number,
		D0Key:         key,
		Value:         decimal.RequireFromString(value),
	}
}

func freshTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestShapesServeTheLatestVersionAndPinsOverrideIt(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	repo := &fakeVersions{
		versions: []*ratematrix.RateMatrixVersion{
			zoneVersion(matrixID, 1, ratematrix.ValueTypeMoney),
			zoneVersion(matrixID, 2, ratematrix.ValueTypeNumber),
		},
		served: map[pulid.ID]int64{matrixID: 2},
	}
	tables := &Tables{versions: repo}
	tenant := freshTenant()

	latest, err := tables.Shapes(t.Context(), tenant, "mi", nil)
	require.NoError(t, err)
	require.Len(t, latest, 1)
	assert.Equal(t, "ZONES", latest[0].Code)
	assert.EqualValues(t, 2, latest[0].Version)
	assert.Equal(t, matrixID, latest[0].MatrixID)
	assert.Equal(t, ratematrix.ValueTypeNumber, latest[0].ValueType)
	assert.NotNil(t, latest[0].Loaded, "a keyed table loads its cells on demand")

	pinned, err := tables.Shapes(t.Context(), tenant, "mi", []tarifftypes.TablePin{{
		Code:          "ZONES",
		RateMatrixID:  matrixID,
		VersionNumber: 1,
	}})
	require.NoError(t, err)
	require.Len(t, pinned, 1)
	assert.EqualValues(t, 1, pinned[0].Version, "a pinned version wins over the served one")
	assert.Equal(t, ratematrix.ValueTypeMoney, pinned[0].ValueType)
}

func TestLoadCellsReadsThePinnedVersionOncePerCombination(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	repo := &fakeVersions{
		versions: []*ratematrix.RateMatrixVersion{
			zoneVersion(matrixID, 1, ratematrix.ValueTypeMoney),
			zoneVersion(matrixID, 2, ratematrix.ValueTypeMoney),
		},
		served: map[pulid.ID]int64{matrixID: 2},
		cells: []*ratematrix.RateMatrixCell{
			zoneCell(matrixID, 1, "SE", "10"),
			zoneCell(matrixID, 2, "SE", "99"),
		},
	}
	tables := &Tables{versions: repo}
	tenant := freshTenant()
	pins := []tarifftypes.TablePin{{Code: "ZONES", RateMatrixID: matrixID, VersionNumber: 1}}

	for range 2 {
		shapes, err := tables.Shapes(t.Context(), tenant, "mi", pins)
		require.NoError(t, err)
		tariffCtx := &rules.TariffContext{Tables: shapes}

		err = tables.LoadCells(t.Context(), tenant, tariffCtx, []rules.TariffNeed{
			{Table: "ZONES", Keys: []string{"SE"}},
			{Table: "ZONES", Keys: []string{"SE"}},
		})
		require.NoError(t, err)
		require.Len(t, tariffCtx.Tables[0].Cells, 1)
		assert.Equal(t, "10", tariffCtx.Tables[0].Cells[0].Value.String(),
			"a pinned template reads v1 even though v2 is published")
		assert.Equal(t, [][]string{{"SE"}}, tariffCtx.Tables[0].Loaded)
	}
	assert.Len(t, repo.lookups, 1, "a published version's cells are cached by version")
}

func TestPostalRangeAxesLoadByTheRangeHoldingTheKey(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	version := &ratematrix.RateMatrixVersion{
		RateMatrixID:  matrixID,
		VersionNumber: 1,
		Code:          "LANES",
		ValueType:     ratematrix.ValueTypeMoney + "/cwt",
		Dimensions: []*ratematrix.RateMatrixDimension{
			{
				RateMatrixID:  matrixID,
				VersionNumber: 1,
				Position:      0,
				Kind:          ratematrix.DimensionKindZipRange,
				MatchMode:     ratematrix.MatchModeRange,
			},
			{
				RateMatrixID:  matrixID,
				VersionNumber: 1,
				Position:      1,
				Kind:          ratematrix.DimensionKindWeightBreak,
				MatchMode:     ratematrix.MatchModeRange,
			},
		},
	}
	band := func(from, to, minimum, value string) *ratematrix.RateMatrixCell {
		cell := &ratematrix.RateMatrixCell{
			RateMatrixID:  matrixID,
			VersionNumber: 1,
			D0Key:         from,
			D0KeyTo:       to,
			D1Min:         decimal.NewNullDecimal(decimal.RequireFromString(minimum)),
			Value:         decimal.RequireFromString(value),
		}
		return cell
	}
	repo := &fakeVersions{
		versions: []*ratematrix.RateMatrixVersion{version},
		served:   map[pulid.ID]int64{matrixID: 1},
		cells: []*ratematrix.RateMatrixCell{
			band("30000", "31999", "0", "20"),
			band("30000", "31999", "1000", "15"),
			band("32000", "34999", "0", "25"),
		},
	}
	tables := &Tables{versions: repo}
	tenant := freshTenant()

	shapes, err := tables.Shapes(t.Context(), tenant, "mi", nil)
	require.NoError(t, err)
	require.Len(t, shapes, 1)
	assert.Equal(t, "ZipRange", shapes[0].Axes[0].Kind)
	assert.Empty(t, shapes[0].Axes[0].Overflow)
	tariffCtx := &rules.TariffContext{Tables: shapes}

	err = tables.LoadCells(t.Context(), tenant, tariffCtx, []rules.TariffNeed{
		{Table: "LANES", Keys: []string{"30303"}},
		{Table: "LANES", Keys: []string{"99999"}},
	})
	require.NoError(t, err)

	table := tariffCtx.Tables[0]
	require.Len(t, table.Cells, 2, "both bands of the range holding 30303, nothing for 99999")
	for _, cell := range table.Cells {
		assert.Equal(t, "30000", cell.At[0].From)
		assert.Equal(t, "31999", cell.At[0].To)
		assert.Empty(t, cell.At[0].Key)
	}
	assert.ElementsMatch(t, [][]string{{"30303"}, {"99999"}}, table.Loaded)
	require.Len(t, repo.lookups, 1)
	assert.Equal(t, map[int16][]string{0: {"30000"}}, repo.lookups[0].Keys,
		"the lookup reads the range's first code through the index")

	cached, ok := ratetablecache.Cells(tenant.OrgID, tenant.BuID,
		ratetablecache.VersionKey{MatrixID: matrixID, Version: 1}, []string{"99999"})
	require.True(t, ok)
	assert.Empty(t, cached, "a key no range holds is remembered as a miss")
}

func TestTablePinsComeFromTheTablesACheckRead(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	tariffCtx := &rules.TariffContext{Tables: []rules.TariffTable{
		{Code: "ZONES", Version: 3, MatrixID: matrixID},
		{Code: "UNREAD", Version: 1, MatrixID: pulid.MustNew("rmx_")},
		{Code: "STUDIO", Version: 0},
	}}
	three := int64(3)
	check := &rules.TariffCheck{Tables: []rules.TariffTableRead{
		{Code: "ZONES", Version: &three},
		{Code: "STUDIO"},
		{Code: "MISSING"},
	}}

	assert.Equal(t, []tarifftypes.TablePin{{
		Code:          "ZONES",
		RateMatrixID:  matrixID,
		VersionNumber: 3,
	}}, TablePins(check, tariffCtx))
	assert.Empty(t, TablePins(nil, tariffCtx))
}
