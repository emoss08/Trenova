package metricquerytools

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBoards struct {
	refusal error
	read    *boardcatalog.MetricRead
	state   board.BlockState
	checked []*board.Board
	filled  []*board.Board
	agents  []*assistantports.BoardAgentRead
}

func (f *fakeBoards) Check(_ context.Context, req *assistantports.BoardFillRequest) error {
	f.checked = append(f.checked, req.Board)

	return f.refusal
}

func (f *fakeBoards) Fill(
	_ context.Context,
	req *assistantports.BoardFillRequest,
) (*board.Filled, error) {
	f.filled = append(f.filled, req.Board)
	f.agents = append(f.agents, req.Agent)
	block := board.FilledBlock{Type: board.BlockMetric, State: board.StateReady}
	if f.state != "" {
		block.State = f.state
	}
	if f.read != nil {
		spec, _ := boardregistry.Catalog().MetricNamed(req.Board.Blocks[0].Metric)
		block.Metric = &board.FilledMetric{Name: spec.Name, Form: spec.Form, Read: f.read}
	}

	return &board.Filled{Blocks: []board.FilledBlock{block}}, nil
}

func (f *fakeBoards) Card(
	context.Context,
	*assistantports.BoardCardRequest,
) (*assistantports.BoardCard, error) {
	return nil, nil
}

func (f *fakeBoards) Location(context.Context, pagination.TenantInfo) *time.Location {
	return time.UTC
}

const (
	orgID = pulid.ID("org_01JMETRICREADTEST000000000")
	buID  = pulid.ID("bu_01JMETRICREADTEST0000000000")
)

func params(args map[string]any) *agenttoolports.QueryToolParams {
	return &agenttoolports.QueryToolParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			UserID:         pulid.ID("usr_01JMETRICREADTEST000000000"),
		},
		Params: args,
	}
}

func percent(value string) *decimal.Decimal {
	number := decimal.RequireFromString(value)

	return &number
}

func monday(day int) int64 {
	return time.Date(2026, time.October, day, 0, 0, 0, 0, time.UTC).Unix()
}

func onTimeRead(weeks int) *boardcatalog.MetricRead {
	chart := &boardcatalog.Chart{
		Unit:   boardcatalog.ValuePercent,
		Series: []boardcatalog.Series{{Name: "On time", Role: boardcatalog.RoleNow}},
		Target: &boardcatalog.Line{Value: decimal.RequireFromString("0.92"), Label: "Target"},
		Ghost:  true,
	}
	for week := range weeks {
		chart.X = append(chart.X, boardcatalog.Tick{At: monday(1) + int64(week)*7*86400})
		chart.Series[0].Points = append(chart.Series[0].Points, percent("0.6"))
	}
	chart.Marked = []int{1}
	average := boardcatalog.Percent(decimal.RequireFromString("0.59"))
	lowest := boardcatalog.Percent(decimal.RequireFromString("0.549"))

	return &boardcatalog.MetricRead{
		Title:  "On time by week, this quarter",
		Source: "2,094 deliveries, 9 weeks",
		Figures: map[string]*boardcatalog.Value{
			"average": &average,
			"lowest":  &lowest,
			"target":  nil,
		},
		Chart: chart,
	}
}

func query(t *testing.T, boards *fakeBoards, args map[string]any) (*metricResult, error) {
	t.Helper()

	result, err := NewReadMetricTool(nil).(*ReadMetricTool).withBoards(boards).
		Query(t.Context(), params(args))
	if err != nil {
		return nil, err
	}
	out, ok := result.(*metricResult)
	require.True(t, ok)

	return out, nil
}

func (t *ReadMetricTool) withBoards(boards boardReader) *ReadMetricTool {
	t.boards = boards

	return t
}

func TestReadNamesEveryFigureAndPoint(t *testing.T) {
	t.Parallel()

	boards := &fakeBoards{read: onTimeRead(9)}
	result, err := query(t, boards, map[string]any{"metric": "onTimeTrend"})
	require.NoError(t, err)

	assert.Equal(t, operationsboard.MetricOnTimeByWeek, result.Metric)
	assert.Equal(t, "thisQuarter", result.Period)
	assert.Equal(t, "2,094 deliveries, 9 weeks", result.Source)
	require.NotEmpty(t, result.Figures)
	assert.Equal(t, "average", result.Figures[0].Name)
	require.NotNil(t, result.Figures[0].Value)
	assert.Equal(t, "59%", *result.Figures[0].Value)
	assert.Equal(t, "{average:onTimeByWeek}", result.Figures[0].WriteAs)

	require.Len(t, result.Points, 9)
	assert.Equal(t, []string{"On time"}, result.Series)
	assert.Equal(t, "Oct 1, 2026", result.Points[0].Point)
	assert.Equal(t, []string{"60%"}, result.Points[0].Values)
	assert.True(t, result.Points[1].Marked)
	assert.True(t, result.Points[8].Running)
	assert.Equal(t, "92%", result.Target)
	assert.Contains(t, result.HowToShow, `show_answer {"title": "…", "answer": "… `+
		`{average:onTimeByWeek} …", "groups": [{"title": "…", "metric": "onTimeByWeek"}]}`)
	assert.Contains(t, result.HowToShow, "how it moved over time")

	require.Len(t, boards.checked, 1)
	block := boards.checked[0].Blocks[0]
	assert.Equal(t, board.BlockMetric, block.Type)
	assert.Equal(t, operationsboard.MetricOnTimeByWeek, block.Metric)
	assert.Equal(t, "thisQuarter", block.Period)
}

func TestReadIsBounded(t *testing.T) {
	t.Parallel()

	result, err := query(t, &fakeBoards{read: onTimeRead(60)}, map[string]any{
		"metric": operationsboard.MetricOnTimeByWeek,
	})
	require.NoError(t, err)

	assert.Len(t, result.Points, maxEntries)
	assert.Equal(t, []string{fmt.Sprintf("points: the first %d of 60", maxEntries)}, result.Cut)
}

func TestReadWithholdsAFigureRatherThanZeroIt(t *testing.T) {
	t.Parallel()

	result, err := query(t, &fakeBoards{read: onTimeRead(3)}, map[string]any{
		"metric": operationsboard.MetricOnTimeByWeek,
	})
	require.NoError(t, err)

	var target *figureView
	for idx := range result.Figures {
		if result.Figures[idx].Name == "target" {
			target = &result.Figures[idx]
		}
	}
	require.NotNil(t, target)
	assert.Nil(t, target.Value)
	assert.Empty(t, target.WriteAs)
	assert.Contains(t, result.Notes, unreadNote)
}

func TestReadIsHeldToTheAgentsDataAccess(t *testing.T) {
	t.Parallel()

	boards := &fakeBoards{read: onTimeRead(3)}
	args := params(map[string]any{"metric": operationsboard.MetricOnTimeByWeek})
	args.DataAccessCeiling = permission.SensitivityInternal

	_, err := NewReadMetricTool(nil).(*ReadMetricTool).withBoards(boards).Query(t.Context(), args)

	require.NoError(t, err)
	require.Len(t, boards.agents, 1)
	require.NotNil(t, boards.agents[0], "the read is an agent's, so its data access applies")
	assert.Equal(t, permission.SensitivityInternal, boards.agents[0].DataAccess)
}

func TestReadRefusesAMetricTheReaderMayNotSee(t *testing.T) {
	t.Parallel()

	boards := &fakeBoards{
		read: onTimeRead(3),
		refusal: &agentruntimeports.PublishMistake{
			Reason: "the person may not read the lateByCustomer metric, so leave it out",
		},
	}
	_, err := query(t, boards, map[string]any{"metric": operationsboard.MetricLateByCustomer})

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, err.Error(), "may not read the lateByCustomer metric")
	assert.Empty(t, boards.filled)
}

func TestReadRefusesAMetricItDoesNotKnow(t *testing.T) {
	t.Parallel()

	boards := &fakeBoards{read: onTimeRead(3)}
	_, err := query(t, boards, map[string]any{"metric": "fuelCostPerParsec"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a metric Trenova keeps")
	assert.Contains(t, err.Error(), "search_tools")
	assert.Empty(t, boards.checked)
}

func TestReadTakesTheMetricsOwnPeriodWhenAskedForAnother(t *testing.T) {
	t.Parallel()

	boards := &fakeBoards{read: onTimeRead(3)}
	result, err := query(t, boards, map[string]any{
		"metric": operationsboard.MetricOnTimeByWeek,
		"period": "today",
		"for":    "cus_01JMETRICREADTEST000000000",
	})
	require.NoError(t, err)

	assert.Equal(t, "thisQuarter", result.Period)
	assert.Equal(t, "thisQuarter", boards.checked[0].Blocks[0].Period)
	assert.True(t, boards.checked[0].Blocks[0].For.IsNil())
	require.GreaterOrEqual(t, len(result.Notes), 2)
	assert.Contains(t, result.Notes[0], "does not take the period today")
	assert.Contains(t, result.Notes[1], "reads no single record")
}

func TestReadSaysWhenTheMetricCouldNotBeRead(t *testing.T) {
	t.Parallel()

	_, err := query(t, &fakeBoards{state: board.StateUnreadable}, map[string]any{
		"metric": operationsboard.MetricOnTimeByWeek,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be read")
}

func TestReadableAsksTheBoardsOwnCheck(t *testing.T) {
	t.Parallel()

	allowing := &fakeBoards{}
	tool := (&ReadMetricTool{}).withBoards(allowing)
	actor := &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID}
	onTime := operationsboard.MetricOnTimeByWeek
	assert.True(t, tool.Readable(t.Context(), actor, permission.SensitivityInternal, onTime))
	assert.Equal(t, operationsboard.MetricOnTimeByWeek, allowing.checked[0].Blocks[0].Metric)

	refusing := &fakeBoards{refusal: &agentruntimeports.PublishMistake{Reason: "no"}}
	assert.False(t, (&ReadMetricTool{}).withBoards(refusing).
		Readable(t.Context(), actor, permission.SensitivityInternal, onTime))
	assert.False(t, tool.Readable(t.Context(), nil, permission.SensitivityInternal, onTime))
}

func TestTheFiguresFormIsSaidInASentence(t *testing.T) {
	t.Parallel()

	onTime := boardcatalog.Percent(decimal.RequireFromString("0.873"))
	result, err := query(t, &fakeBoards{read: &boardcatalog.MetricRead{
		Title:   "Dispatch today",
		Figures: map[string]*boardcatalog.Value{"onTimePercent": &onTime},
	}}, map[string]any{"metric": "onTimePercent"})
	require.NoError(t, err)

	assert.Equal(t, homeboard.MetricDispatchToday, result.Metric)
	require.NotEmpty(t, result.Notes)
	assert.Contains(t, result.Notes[0], "onTimePercent is a figure of the dispatchToday metric")
	assert.Equal(t, "today", result.Period)
	assert.Contains(t, result.HowToShow, "Say the figure the person asked for in a sentence")
	for idx := range result.Figures {
		if result.Figures[idx].Name == "onTimePercent" {
			require.NotNil(t, result.Figures[idx].Value)
			assert.Equal(t, "87.3%", *result.Figures[idx].Value)
		}
	}
}

func TestAFiguresMetricThatTotalsRecordsAsksForTheRecordsBesideIt(t *testing.T) {
	t.Parallel()

	count := boardcatalog.Count(51)
	result, err := query(t, &fakeBoards{read: &boardcatalog.MetricRead{
		Title:   "Bank receipts not applied",
		Figures: map[string]*boardcatalog.Value{"receiptsToApply": &count},
	}}, map[string]any{"metric": "unappliedReceipts"})
	require.NoError(t, err)

	assert.NotContains(t, result.HowToShow, "Say the figure the person asked for in a sentence")
	assert.Contains(t, result.HowToShow, `"measure": "received"`)
	assert.Contains(t, result.HowToShow,
		"The second group holds the bank receipts not applied, read with "+
			"receivable_list_bank_receipt_exceptions")
	assert.Contains(t, result.HowToShow, "Only a question one figure answers is a sentence")
}

func TestEachValueOfAChartOfSeveralSeriesIsNamedBySeries(t *testing.T) {
	t.Parallel()

	was, now := decimal.RequireFromString("41650"), decimal.RequireFromString("74920")
	result, err := query(t, &fakeBoards{read: &boardcatalog.MetricRead{
		Title: "Aging, Nov 23, 2026 against Oct 23, 2026",
		Chart: &boardcatalog.Chart{
			Unit: boardcatalog.ValueMoney,
			X:    []boardcatalog.Tick{{Label: "31–60", Marked: true}},
			Series: []boardcatalog.Series{
				{Name: "Oct 23, 2026", Role: boardcatalog.RoleWas, Points: []*decimal.Decimal{&was}},
				{Name: "Nov 23, 2026", Role: boardcatalog.RoleNow, Points: []*decimal.Decimal{&now}},
			},
		},
	}}, map[string]any{"metric": "agingCompare"})
	require.NoError(t, err)

	require.Len(t, result.Points, 1)
	assert.Equal(t, []string{"Oct 23, 2026: $41,650.00", "Nov 23, 2026: $74,920.00"},
		result.Points[0].Values, "a value of two series reads as the series it belongs to")
}
