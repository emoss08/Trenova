package boardrecords_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricGroups_AddsTheGroupOfAMetricTheWordsPlace(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"answer": "{lapsingCarriers:coverAgainstLoads} carriers have cover ending within 30 " +
			"days, and {lapsedCarriers:coverAgainstLoads} has already lapsed.",
		"groups": []any{map[string]any{
			"title": "Cover already lapsed", "records": []any{ironhorse},
		}},
	}

	got, added := boardrecords.MetricGroups(args)

	assert.Equal(t, []string{"coverAgainstLoads"}, added)
	groups, _ := got["groups"].([]any)
	require.Len(t, groups, 2)
	assert.Equal(t, args["groups"].([]any)[0], groups[0], "the model's own group stays first")
	metric, _ := groups[1].(map[string]any)
	assert.Equal(t, "coverAgainstLoads", metric["metric"])
	assert.NotEmpty(t, metric["title"])
	assert.LessOrEqual(t, len([]rune(metric["title"].(string))), 60)
	assert.Len(t, args["groups"], 1, "the arguments the model sent are left as they were")
}

func TestMetricGroups_LeavesAShownMetricAndAFullBoard(t *testing.T) {
	t.Parallel()

	shown := map[string]any{
		"answer": "{average:onTimeByWeek} this quarter.",
		"groups": []any{map[string]any{"title": "Trend", "metric": "onTimeTrend"}},
	}
	_, added := boardrecords.MetricGroups(shown)
	assert.Empty(t, added, "an alias of the shown metric is the same metric")

	full := map[string]any{
		"answer": "{average:onTimeByWeek} this quarter.",
		"groups": []any{
			map[string]any{"title": "A", "records": []any{ironhorse}},
			map[string]any{"title": "B", "records": []any{ridgeline}},
			map[string]any{"title": "C", "records": []any{ironhorse}},
		},
	}
	got, added := boardrecords.MetricGroups(full)
	assert.Empty(t, added, "at the group limit the refusal stands")
	assert.Len(t, got["groups"], 3)

	_, added = boardrecords.MetricGroups(map[string]any{
		"answer": "{total:revenue} across {count} loads.",
	})
	assert.Empty(t, added, "a measure's figure is not a metric's")
}
