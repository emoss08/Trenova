package metricfind_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/metricfind"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingboard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homeboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func matchedNames(need string) []string {
	metrics := metricfind.Matching(need, 3)
	names := make([]string, 0, len(metrics))
	for idx := range metrics {
		names = append(names, metrics[idx].Name)
	}

	return names
}

func TestMatchingFindsAMetricInAPersonsWords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		need  string
		first string
		also  []string
	}{
		{
			need:  "how has on-time delivery trended this quarter, and where did it slip?",
			first: operationsboard.MetricOnTimeByWeek,
			also:  []string{operationsboard.MetricLateByCustomer},
		},
		{
			need:  "on-time delivery metric by week for shipments this quarter",
			first: operationsboard.MetricOnTimeByWeek,
		},
		{
			need:  "late deliveries by customer last week",
			first: operationsboard.MetricLateByCustomer,
		},
		{
			need:  "what's our on-time percent today?",
			first: homeboard.MetricDispatchToday,
		},
		{
			need:  "open and overdue receivables",
			first: billingboard.MetricReceivables,
		},
		{
			need:  "read onTimeTrend",
			first: operationsboard.MetricOnTimeByWeek,
		},
	}
	for _, tc := range cases {
		t.Run(tc.need, func(t *testing.T) {
			t.Parallel()

			names := matchedNames(tc.need)
			require.NotEmpty(t, names)
			assert.Equal(t, tc.first, names[0])
			for _, also := range tc.also {
				assert.Contains(t, names, also)
			}
		})
	}
}

func TestMatchingLeavesOutQuestionsNoMetricMeasures(t *testing.T) {
	t.Parallel()

	for _, need := range []string{
		"customer list to look up FreshHaul's id",
		"list customers by name",
		"assign a driver to this load",
		"quantum chromodynamics lattice",
		"",
	} {
		assert.Empty(t, matchedNames(need), need)
		assert.False(t, metricfind.Mentions(need), need)
	}
}

func TestASpanOfTimeInAMetricsWordsIsNotWhatItMeasures(t *testing.T) {
	t.Parallel()

	names := matchedNames(
		"on-time delivery performance by week this quarter, late deliveries by customer",
	)
	assert.NotContains(t, names, billingboard.MetricCollecting,
		"collecting is compared against a week or quarter ago; that is not on-time delivery")
	assert.Contains(t, names, operationsboard.MetricOnTimeByWeek)
}

func TestMatchingIsBounded(t *testing.T) {
	t.Parallel()

	assert.Len(t, metricfind.Matching("on-time late customer week today receivables", 1), 1)
	assert.Empty(t, metricfind.Matching("on-time delivery by week", 0))
}

type gate struct {
	denied map[string]bool
	asked  []string
	access permission.FieldSensitivity
}

func (g *gate) Name() string                      { return agenttoolports.MetricReadTool }
func (g *gate) Description() string               { return "" }
func (g *gate) ParamSchema() map[string]any       { return nil }
func (g *gate) Policy() agenttoolports.ToolPolicy { return agenttoolports.ToolPolicy{} }
func (g *gate) Query(context.Context, *agenttoolports.QueryToolParams) (any, error) {
	return nil, nil
}

func (g *gate) Readable(
	_ context.Context,
	_ *permissionports.RequestActor,
	access permission.FieldSensitivity,
	metric string,
) bool {
	g.access = access
	g.asked = append(g.asked, metric)

	return !g.denied[metric]
}

type registry struct {
	tools map[string]agenttoolports.AgentQueryTool
}

func (r *registry) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := r.tools[name]

	return tool, ok
}

func (r *registry) All() []agenttoolports.AgentQueryTool              { return nil }
func (r *registry) Descriptors() []agenttoolports.AgentToolDescriptor { return nil }

func request(tools agenttoolports.AgentQueryToolRegistry, reads, draws bool) *metricfind.Request {
	return &metricfind.Request{
		Tools:  tools,
		Actor:  &permissionports.RequestActor{},
		Access: permission.SensitivityRestricted,
		Need:   "on-time delivery performance by week this quarter, late deliveries by customer",
		Reads:  reads,
		Draws:  draws,
	}
}

func TestFindNamesTheMetricAndHowToUseIt(t *testing.T) {
	t.Parallel()

	tools := &registry{tools: map[string]agenttoolports.AgentQueryTool{
		agenttoolports.MetricReadTool: &gate{},
	}}
	section := metricfind.Find(t.Context(), request(tools, true, true))

	assert.ElementsMatch(t, []string{
		operationsboard.MetricOnTimeByWeek,
		operationsboard.MetricLateByCustomer,
	}, section.Metrics)
	assert.Contains(t, section.Text, "- onTimeByWeek: ")
	assert.Contains(t, section.Text, "periods thisQuarter (the default), lastQuarter")
	assert.Contains(t, section.Text, `metric_read {"metric": "`+section.Metrics[0]+`"}`)
	assert.Contains(t, section.Text, `show_answer {"title": "…", "answer": "…`)
	assert.Contains(t, section.Text, `{"title": "…", "metric": "onTimeByWeek"}`)
	assert.Contains(t, section.Text, `{"title": "…", "metric": "lateByCustomer"}`)
}

func TestFindOffersNoMetricTheReaderMayNotSee(t *testing.T) {
	t.Parallel()

	denying := &gate{denied: map[string]bool{operationsboard.MetricLateByCustomer: true}}
	tools := &registry{tools: map[string]agenttoolports.AgentQueryTool{
		agenttoolports.MetricReadTool: denying,
	}}
	section := metricfind.Find(t.Context(), request(tools, true, true))

	assert.Equal(t, []string{operationsboard.MetricOnTimeByWeek}, section.Metrics)
	assert.NotContains(t, section.Text, operationsboard.MetricLateByCustomer)
	assert.Contains(t, denying.asked, operationsboard.MetricLateByCustomer)
	assert.Equal(t, permission.SensitivityRestricted, denying.access,
		"the search asks with the agent's own data access, as the read does")

	everything := &gate{denied: map[string]bool{
		operationsboard.MetricOnTimeByWeek:   true,
		operationsboard.MetricLateByCustomer: true,
	}}
	none := metricfind.Find(t.Context(), request(&registry{
		tools: map[string]agenttoolports.AgentQueryTool{agenttoolports.MetricReadTool: everything},
	}, true, true))
	assert.Empty(t, none.Text)
}

func TestFindSaysOnlyWhatTheTurnCanCall(t *testing.T) {
	t.Parallel()

	tools := &registry{tools: map[string]agenttoolports.AgentQueryTool{
		agenttoolports.MetricReadTool: &gate{},
	}}

	drawOnly := metricfind.Find(t.Context(), request(tools, false, true))
	assert.NotContains(t, drawOnly.Text, "metric_read")
	assert.Contains(t, drawOnly.Text, "show_answer")

	readOnly := metricfind.Find(t.Context(), request(tools, true, false))
	assert.Contains(t, readOnly.Text, "metric_read")
	assert.NotContains(t, readOnly.Text, "show_answer")

	assert.Empty(t, metricfind.Find(t.Context(), request(tools, false, false)).Text)
}

func TestFindFailsClosedWithoutTheGate(t *testing.T) {
	t.Parallel()

	empty := &registry{tools: map[string]agenttoolports.AgentQueryTool{}}
	assert.Empty(t, metricfind.Find(t.Context(), request(empty, true, true)).Text)
}

func TestFoundAddsTheReadToolOnlyWhenLoaded(t *testing.T) {
	t.Parallel()

	found := metricfind.Found{Section: metricfind.Section{Text: "metrics"}, Loaded: true}
	assert.Equal(t, []string{"report_list", agenttoolports.MetricReadTool},
		found.Names([]string{"report_list"}))
	assert.Equal(t, "metrics\n\ntools", found.WithTools("tools"))

	unloaded := metricfind.Found{}
	assert.Equal(t, []string{"report_list"}, unloaded.Names([]string{"report_list"}))
	assert.Equal(t, "tools", unloaded.WithTools("tools"))
}
