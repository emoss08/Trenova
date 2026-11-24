package operationsboardreader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const coverSlot = int64(4 * 3600)

var coverStart = time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC).Unix()

type fakeCoverBalances struct {
	balances *freightoutlookports.Balances
	err      error
	request  *freightoutlookports.Request
}

func (f *fakeCoverBalances) Balances(
	_ context.Context,
	req *freightoutlookports.Request,
) (*freightoutlookports.Balances, error) {
	f.request = req
	return f.balances, f.err
}

type coverSlotSpec struct {
	need, short, spare float64
	high               int
}

func coverStarts() []int64 {
	starts := make([]int64, demandforecast.OutlookSlots)
	for idx := range starts {
		starts[idx] = coverStart + int64(idx)*coverSlot
	}

	return starts
}

func coverBalance(cell int64, label string, specs map[int]coverSlotSpec) demandforecast.AreaBalance {
	starts := coverStarts()
	balance := demandforecast.AreaBalance{
		Cell:  cell,
		Label: label,
		Slots: make([]demandforecast.SlotBalance, len(starts)),
	}
	for idx, start := range starts {
		spec := specs[idx]
		open := demandforecast.Count{}.Shift(spec.high)
		balance.Slots[idx] = demandforecast.SlotBalance{
			Start: start,
			End:   start + coverSlot,
			Need:  spec.need,
			Short: spec.short,
			Spare: spec.spare,
			Open:  open,
		}
	}

	return balance
}

func readyCover(areas ...demandforecast.AreaBalance) *freightoutlookports.Balances {
	return &freightoutlookports.Balances{
		Quality: &demandforecast.Quality{
			Status:   demandforecast.StatusReady,
			FittedAt: coverStart,
			Timezone: "UTC",
		},
		GeneratedAt: coverStart,
		Starts:      coverStarts(),
		Areas:       areas,
	}
}

func TestDriverCover_DrawsShortDeepAndUncertainCellsFromTheForecastSlots(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	outlook := &fakeCoverBalances{balances: readyCover(
		coverBalance(1, "Memphis, TN", map[int]coverSlotSpec{
			0: {need: 5, spare: 1, high: 5},
			1: {need: 3, short: 1, high: 4},
		}),
		coverBalance(2, "Dallas, TX", map[int]coverSlotSpec{
			2:  {need: 6, short: 4, high: 8},
			3:  {need: 4, short: 2, high: 5},
			11: {need: 1, spare: 3, high: 1},
		}),
		coverBalance(3, "", map[int]coverSlotSpec{
			5: {need: 2, high: 4},
		}),
		coverBalance(4, "Idle, KS", nil),
	)}
	reader := NewDriverCover(DriverCoverParams{})
	reader.outlook = outlook

	read, err := reader.Read(t.Context(), &boardreaderports.MetricRequest{
		Tenant: tenant,
		Period: boardcatalog.PeriodNext48Hours,
		Now:    coverStart,
	})
	require.NoError(t, err)

	require.NotNil(t, outlook.request)
	assert.Equal(t, tenant, outlook.request.TenantInfo, "the forecast is read for the board's tenant")
	assert.Equal(t, operationsboard.MetricDriverCover, reader.Metric())
	assert.Contains(t, read.Title, "the next 48 hours")
	assert.Contains(t, read.Title, "forecast area")
	assert.Contains(t, read.Source, "4 forecast areas")
	assert.Contains(t, read.Source, "the next 48 hours")
	assert.NotContains(t, read.Title+read.Source, "week")
	assert.NotContains(t, read.Title+read.Source, "terminal")

	grid := read.Grid
	require.NotNil(t, grid)
	require.Len(t, grid.Columns, demandforecast.OutlookSlots, "one column a forecast slot")
	assert.Equal(t, "8 AM", grid.Columns[0].Label)
	assert.Equal(t, "Tue 24", grid.Columns[0].Sublabel)
	assert.Equal(t, "12 PM", grid.Columns[1].Label)
	assert.Equal(t, "12 AM", grid.Columns[4].Label)
	assert.Equal(t, "Wed 25", grid.Columns[4].Sublabel)

	require.Len(t, grid.Rows, 3, "an area with no need is not drawn")
	assert.Equal(t, "Dallas, TX", grid.Rows[0].Label, "the shortest area first")
	assert.Equal(t, "Memphis, TN", grid.Rows[1].Label)
	assert.Equal(t, "An area with no named origin", grid.Rows[2].Label)

	dallas := grid.Rows[0].Cells
	assert.Equal(t, boardcatalog.Cell{Have: 2, Need: 6, NeedHigh: 8, State: boardcatalog.CellDeep},
		dallas[2], "the deepest short cell is drawn deep")
	assert.Equal(t, boardcatalog.Cell{Have: 2, Need: 4, NeedHigh: 5, State: boardcatalog.CellShort},
		dallas[3])
	assert.Equal(t, boardcatalog.Cell{Have: 4, Need: 1, NeedHigh: 1, State: boardcatalog.CellCovered},
		dallas[11])

	memphis := grid.Rows[1].Cells
	assert.Equal(t, boardcatalog.Cell{Have: 6, Need: 5, NeedHigh: 5, State: boardcatalog.CellCovered},
		memphis[0])
	assert.Equal(t, boardcatalog.CellShort, memphis[1].State)

	unnamed := grid.Rows[2].Cells
	assert.Equal(t, boardcatalog.Cell{Have: 2, Need: 2, NeedHigh: 4, State: boardcatalog.CellMaybe},
		unnamed[5], "covered at the likely need, short at the high end: may be short")

	assert.Equal(t, boardcatalog.Count(2), *read.Figures["areasShort"])
	assert.Equal(t, boardcatalog.Count(7), *read.Figures["short"])
	assert.Equal(t, boardcatalog.Count(1), *read.Figures["mayBeShort"])
	assert.Equal(t, boardcatalog.Count(4), *read.Figures["worst"])
	assert.Equal(t, "Dallas, TX", read.Figures["worstArea"].Text)
	assert.Equal(t, "Tue 24, 4 PM to 8 PM", read.Figures["worstSlot"].Text)
	assert.Equal(t, boardcatalog.Count(3), *read.Figures["spare"])

	metric, ok := boardcatalog.New(boardcatalog.Entries{Metrics: operationsboard.Metrics}).
		MetricNamed(operationsboard.MetricDriverCover)
	require.True(t, ok)
	for _, figure := range metric.Figures {
		assert.Contains(t, read.Figures, figure.Name, "every declared figure is read")
	}
}

func TestDriverCover_DrawsAtMostSixAreas(t *testing.T) {
	t.Parallel()

	areas := make([]demandforecast.AreaBalance, 0, 9)
	for cell := range int64(9) {
		areas = append(areas, coverBalance(cell+1, "Area", map[int]coverSlotSpec{
			0: {need: float64(cell + 1), high: int(cell + 1)},
		}))
	}
	read := coverRead(t.Context(), readyCover(areas...), time.UTC)

	assert.Len(t, read.Grid.Rows, operationsboard.MaxCoverAreas)
	assert.Contains(t, read.Source, "9 forecast areas")
}

func TestDriverCover_SaysNothingIsShortWhenNoSlotIs(t *testing.T) {
	t.Parallel()

	read := coverRead(t.Context(), readyCover(
		coverBalance(1, "Tulsa, OK", map[int]coverSlotSpec{0: {need: 2, spare: 1, high: 2}}),
	), time.UTC)

	assert.Equal(t, boardcatalog.Count(0), *read.Figures["areasShort"])
	assert.Equal(t, boardcatalog.Count(0), *read.Figures["short"])
	assert.Equal(t, boardcatalog.Count(0), *read.Figures["worst"])
	assert.Equal(t, "No area is short", read.Figures["worstArea"].Text)
	assert.Equal(t, "No slot is short", read.Figures["worstSlot"].Text)
	for _, cell := range read.Grid.Rows[0].Cells {
		assert.Equal(t, boardcatalog.CellCovered, cell.State)
	}
}

func TestDriverCover_DrawsNothingUntilTheForecastIsReady(t *testing.T) {
	t.Parallel()

	for _, balances := range []*freightoutlookports.Balances{
		nil,
		{Quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill, FittedAt: coverStart}},
		{Quality: &demandforecast.Quality{Status: demandforecast.StatusInsufficientHistory}},
	} {
		read := coverRead(t.Context(), balances, time.UTC)

		assert.Empty(t, read.Grid.Rows)
		assert.Contains(t, read.Source, "not ready")
		for name, value := range read.Figures {
			assert.Nil(t, value, "%s is not read from a forecast that is not ready", name)
		}
	}
}

func TestDriverCover_ReturnsTheForecastsError(t *testing.T) {
	t.Parallel()

	failure := errors.New("forecast unavailable")
	reader := &DriverCover{outlook: &fakeCoverBalances{err: failure}}

	_, err := reader.Read(t.Context(), &boardreaderports.MetricRequest{})

	require.ErrorIs(t, err, failure)
}

func TestDriverCover_WritesSlotsInTheForecastsZone(t *testing.T) {
	t.Parallel()

	balances := readyCover(coverBalance(1, "Reno, NV", map[int]coverSlotSpec{0: {need: 1, high: 1}}))
	balances.Quality.Timezone = "America/Los_Angeles"
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	read := coverRead(t.Context(), balances, coverZone(balances, chicago))

	assert.Equal(t, "12 AM", read.Grid.Columns[0].Label)
	assert.Equal(t, "Tue 24", read.Grid.Columns[0].Sublabel)
}
