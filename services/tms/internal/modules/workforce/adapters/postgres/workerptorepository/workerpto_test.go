package workerptorepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAggregatePTOData(t *testing.T) {
	repo := &repository{l: zap.NewNop()}

	rows := []ptoDayRow{
		{Date: "2026-01-01", Type: worker.PTOTypeVacation, WorkerID: "wrk_1", FirstName: "Ada"},
		{Date: "2026-01-01", Type: worker.PTOTypeVacation, WorkerID: "wrk_2", FirstName: "Alan"},
		{Date: "2026-01-01", Type: worker.PTOTypeSick, WorkerID: "wrk_3"},
		{Date: "2026-01-02", Type: worker.PTOTypeVacation, WorkerID: "wrk_1"},
	}

	got := repo.aggregatePTOData(rows)

	assert.Len(t, got, 2)
	assert.Equal(t, 2, got["2026-01-01"][worker.PTOTypeVacation].Count)
	assert.Equal(t, 1, got["2026-01-01"][worker.PTOTypeSick].Count)
	assert.Equal(t, 1, got["2026-01-02"][worker.PTOTypeVacation].Count)
	require.Len(t, got["2026-01-01"][worker.PTOTypeVacation].Workers, 2)
	assert.Equal(t, "wrk_1", got["2026-01-01"][worker.PTOTypeVacation].Workers[0].ID)
	assert.Equal(t, "Ada", got["2026-01-01"][worker.PTOTypeVacation].Workers[0].FirstName)
	assert.Equal(t, "Vacation", got["2026-01-01"][worker.PTOTypeVacation].Workers[0].PTOType)
}

func TestBuildPTOChartDataQuery(t *testing.T) {
	repo := &repository{l: zap.NewNop()}

	dateSeries := []string{"2026-01-01", "2026-01-02"}
	ptoMap := repo.aggregatePTOData([]ptoDayRow{
		{
			Date:      "2026-01-01",
			Type:      worker.PTOTypeVacation,
			WorkerID:  "wrk_1",
			FirstName: "Ada",
			LastName:  "Lovelace",
		},
		{Date: "2026-01-01", Type: worker.PTOTypeSick, WorkerID: "wrk_2"},
	})

	result := repo.buildPTOChartDataQuery(dateSeries, ptoMap)
	assert.Len(t, result, 2)

	dayOne := result[0]
	assert.Equal(t, "2026-01-01", dayOne.Date)
	assert.Equal(t, 1, dayOne.Vacation)
	assert.Equal(t, 1, dayOne.Sick)
	assert.Len(t, dayOne.Workers["Vacation"], 1)
	assert.Len(t, dayOne.Workers["Sick"], 1)
	assert.Len(t, dayOne.Workers["Holiday"], 0)
	assert.Len(t, dayOne.Workers["Bereavement"], 0)
	assert.Len(t, dayOne.Workers["Maternity"], 0)
	assert.Len(t, dayOne.Workers["Paternity"], 0)
	assert.Len(t, dayOne.Workers["Personal"], 0)

	dayTwo := result[1]
	assert.Equal(t, "2026-01-02", dayTwo.Date)
	assert.Equal(t, 0, dayTwo.Vacation)
	assert.Len(t, dayTwo.Workers["Vacation"], 0)
	assert.Len(t, dayTwo.Workers["Sick"], 0)
}

func TestChartDatesCoverEveryLocalDay(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	from := time.Date(2026, time.March, 7, 23, 30, 0, 0, loc).Unix()
	to := time.Date(2026, time.March, 9, 0, 15, 0, 0, loc).Unix()

	assert.Equal(t, []string{"2026-03-07", "2026-03-08", "2026-03-09"}, chartDates(from, to, loc))
	assert.Empty(t, chartDates(to, from, loc))
}

func TestChartDaysQuery(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, chartDaysQuery(&chartDaysParams{
		timezone:     "America/Chicago",
		from:         100,
		to:           200,
		overlapStart: 90,
		overlapEnd:   210,
		ptoType:      worker.PTOTypeVacation,
		workerID:     "wrk_1",
	}))
}
