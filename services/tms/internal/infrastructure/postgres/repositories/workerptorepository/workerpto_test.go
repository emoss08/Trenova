package workerptorepository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAggregatePTOData(t *testing.T) {
	repo := &repository{l: zap.NewNop()}

	rows := []ptoAggregateRow{
		{Date: "2026-01-01", Type: "Vacation", Count: 2, Workers: "[]"},
		{Date: "2026-01-01", Type: "Sick", Count: 1, Workers: "[]"},
		{Date: "2026-01-02", Type: "Vacation", Count: 3, Workers: "[]"},
	}

	got := repo.aggregatePTOData(rows)

	assert.Len(t, got, 2)
	assert.Equal(t, 2, got["2026-01-01"]["Vacation"].Count)
	assert.Equal(t, 1, got["2026-01-01"]["Sick"].Count)
	assert.Equal(t, 3, got["2026-01-02"]["Vacation"].Count)
}

func TestBuildPTOChartDataQuery(t *testing.T) {
	repo := &repository{l: zap.NewNop()}

	dateSeries := []string{"2026-01-01", "2026-01-02"}
	ptoMap := map[string]map[string]ptoAggregateRow{
		"2026-01-01": {
			"Vacation": {
				Date:    "2026-01-01",
				Type:    "Vacation",
				Count:   2,
				Workers: `[{"id":"wrk_1","firstName":"Ada","lastName":"Lovelace","ptoType":"Vacation"}]`,
			},
			"Sick": {
				Date:    "2026-01-01",
				Type:    "Sick",
				Count:   1,
				Workers: `invalid_json`,
			},
			"UnknownType": {
				Date:    "2026-01-01",
				Type:    "UnknownType",
				Count:   99,
				Workers: `[]`,
			},
		},
	}

	result := repo.buildPTOChartDataQuery(dateSeries, ptoMap, zap.NewNop())
	assert.Len(t, result, 2)

	dayOne := result[0]
	assert.Equal(t, "2026-01-01", dayOne.Date)
	assert.Equal(t, 2, dayOne.Vacation)
	assert.Equal(t, 1, dayOne.Sick)
	assert.Len(t, dayOne.Workers["Vacation"], 1)
	assert.Len(t, dayOne.Workers["Sick"], 0)
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
