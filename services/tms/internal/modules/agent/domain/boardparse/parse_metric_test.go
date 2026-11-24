package boardparse_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metricBoard(block map[string]any) map[string]any {
	return map[string]any{
		"title":   "Receivables",
		"verdict": "Here is [where receivables stand][1].",
		"blocks":  []any{block},
	}
}

func TestParseReadsAMetricBlockByItsNameOrAlias(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"receivables", "AR", "arOverdue"} {
		parsed, problems := boardparse.Parse(metricBoard(map[string]any{
			"type": "metric", "metric": name, "period": "lastQuarter",
		}))
		require.Empty(t, problems, name)
		block := parsed.Blocks[0]
		assert.Equal(t, board.BlockMetric, block.Type)
		assert.Equal(t, billingboard.MetricReceivables, block.Metric, name)
		assert.Empty(t, block.Period, "a period the metric does not take is dropped")
		assert.Equal(
			t,
			[]board.MetricAsk{{Metric: billingboard.MetricReceivables}},
			parsed.Metrics(),
		)
	}
}

func TestParseRefusesAMetricBlockNamingNoMetric(t *testing.T) {
	t.Parallel()

	_, problems := boardparse.Parse(metricBoard(map[string]any{
		"type": "metric", "metric": "agingByMoonPhase",
	}))
	require.NotEmpty(t, problems)
	assert.Contains(t, strings.Join(problems, "\n"), `metric "agingByMoonPhase" is not one of`)

	_, problems = boardparse.Parse(metricBoard(map[string]any{"type": "metric"}))
	assert.Contains(t, strings.Join(problems, "\n"), "blocks[1].metric is required")
}

func TestParseReadsAMetricFigureByItsMetricAndName(t *testing.T) {
	t.Parallel()

	parsed, problems := boardparse.Parse(metricBoard(map[string]any{
		"type": "figures",
		"figures": []any{
			map[string]any{"label": "Overdue", "metric": "receivables", "figure": "arOverdue"},
			map[string]any{"label": "Late moves", "metric": "movesLate"},
		},
	}))
	require.Empty(t, problems)
	figures := parsed.Blocks[0].Figures
	assert.Equal(t, billingboard.MetricReceivables, figures[0].Metric)
	assert.Equal(t, "arOverdue", figures[0].Figure)
	assert.Equal(t, homeboard.MetricDispatchToday, figures[1].Metric)
	assert.Equal(t, "movesLate", figures[1].Figure)
	assert.Equal(t, "today", figures[1].Period)

	_, problems = boardparse.Parse(metricBoard(map[string]any{
		"type": "figures",
		"figures": []any{
			map[string]any{"label": "Overdue", "metric": "receivables", "figure": "lowest"},
		},
	}))
	assert.Contains(t, strings.Join(problems, "\n"), "the receivables metric's figures")
}
