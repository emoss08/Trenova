package boardanswer_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func metricAnswer(answer string, groups ...any) map[string]any {
	return map[string]any{
		"title":  "Where receivables stand",
		"answer": answer,
		"groups": groups,
	}
}

func refused(t *testing.T, args map[string]any) string {
	t.Helper()
	answer, problems := boardanswer.Translate(args)
	require.Nil(t, answer)
	require.NotEmpty(t, problems)

	return strings.Join(problems, "; ")
}

func TestTranslate_DrawsAMetricGroupAsAMetricBlock(t *testing.T) {
	t.Parallel()

	answer := translated(t, metricAnswer(
		"Receivables stand at {arTotalOpen:receivables}, {arOverdue:ar} of it overdue.",
		map[string]any{"title": "Receivables", "metric": "AR", "period": "thisWeek"},
	))

	require.Len(t, answer.Board.Blocks, 1)
	block := answer.Board.Blocks[0]
	assert.Equal(t, board.BlockMetric, block.Type)
	assert.Equal(t, billingboard.MetricReceivables, block.Metric)
	assert.Empty(t, block.Period, "a period the metric does not take becomes its default")
	require.Len(t, answer.Board.Readings, 2)
	for _, reading := range answer.Board.Readings {
		assert.Equal(t, billingboard.MetricReceivables, reading.Metric)
		assert.Contains(t, answer.Board.Verdict, "{"+reading.Key+"}")
	}
	labels := map[string]string{}
	for _, reading := range answer.Board.Readings {
		labels[reading.Figure] = reading.Label
	}
	assert.Equal(t, map[string]string{
		"arTotalOpen": "Open receivables",
		"arOverdue":   "Overdue receivables",
	}, labels)
}

func TestTranslate_ReadsABareFigureThatOneMetricGroupDeclares(t *testing.T) {
	t.Parallel()

	answer := translated(t, metricAnswer(
		"Days sales outstanding is {arDaysSalesOutstanding}.",
		map[string]any{"title": "Receivables", "metric": "receivables"},
	))

	require.Len(t, answer.Board.Readings, 1)
	assert.Equal(t, "arDaysSalesOutstanding", answer.Board.Readings[0].Figure)
}

func TestTranslate_KeepsRecordsAndAMetricGivenInOneGroupAsTwo(t *testing.T) {
	t.Parallel()

	answer := translated(t, metricAnswer(
		"{count} invoices are open, {arOverdue:receivables} overdue in all.",
		map[string]any{"title": "Open", "records": invoices, "metric": "receivables"},
	))

	require.Len(t, answer.Board.Blocks, 2)
	assert.Equal(t, board.BlockRecords, answer.Board.Blocks[0].Type)
	assert.Equal(t, board.BlockMetric, answer.Board.Blocks[1].Type)
}

func TestTranslate_RefusesWhatAMetricGroupCannotRead(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		args map[string]any
		want string
	}{
		"an unknown metric names the closest": {
			args: metricAnswer("It is {total}.",
				map[string]any{"title": "Aging", "metric": "receivablez"}),
			want: `metric "receivablez" is not a metric Trenova keeps; the closest are receivables`,
		},
		"a figure the metric does not declare": {
			args: metricAnswer("It is {lowest:receivables}.",
				map[string]any{"title": "AR", "metric": "receivables"}),
			want: "the receivables metric's figures are",
		},
		"a figure of a metric no group shows": {
			args: metricAnswer("It is {arOverdue:receivables}.",
				map[string]any{"title": "Open", "records": invoices}),
			want: "which no group shows; add a group with metric receivables",
		},
		"a count with no records": {
			args: metricAnswer("There are {count}.",
				map[string]any{"title": "AR", "metric": "receivables"}),
			want: "no group holds records",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, refused(t, test.args), test.want)
		})
	}
}

func TestTranslate_GivesRecordsTheMeasureTheirKindTakesByDefault(t *testing.T) {
	t.Parallel()

	answer := translated(t, map[string]any{
		"title":  "Credentials running out",
		"answer": "{count} run out soon, the soonest in {largest}.",
		"groups": []any{map[string]any{
			"title":   "Running out",
			"records": []any{"wcred_01JCRED00000000000000001", "wcred_01JCRED00000000000000002"},
		}},
	})

	assert.Equal(t, board.Measure("expiry"), answer.Board.Blocks[0].Measure)
	assert.Equal(t, board.OrderMeasure, answer.Board.Blocks[0].Order)
	largest := answer.Board.Reading("largest")
	require.NotNil(t, largest)
	assert.Equal(t, board.Measure("expiry"), largest.Measure)
	assert.Equal(t, "Soonest to run out", largest.Label)
}
