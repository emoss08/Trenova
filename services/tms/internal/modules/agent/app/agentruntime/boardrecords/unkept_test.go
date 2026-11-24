package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/stretchr/testify/assert"
)

func TestUnkeptFigures_NamesAFigureTheMetricReadAsNothing(t *testing.T) {
	t.Parallel()

	read := toolresult.Fence("metric_read", `{"metric":"fuelCostPerMile","figures":[`+
		`{"name":"currentCostPerMile","value":"$0.543"},{"name":"previousCostPerMile","value":null}]}`)
	words := "{currentCostPerMile:fuelCostPerMile} a mile, against " +
		"{previousCostPerMile:fuelCostPerMile} for October."

	assert.Equal(t, []string{"{previousCostPerMile:fuelCostPerMile}"},
		boardrecords.UnkeptFigures([]string{read}, words),
		"z2-final fuel #3 drew \"against — for October\": October is withheld (two currencies)")
	assert.Empty(t, boardrecords.UnkeptFigures([]string{read},
		"{currentCostPerMile:fuelCostPerMile} a mile."))
}
