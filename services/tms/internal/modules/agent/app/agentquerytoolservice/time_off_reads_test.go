package agentquerytoolservice_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeOffZone = "America/Chicago"

// timeOffReader records the input each read sends and answers with one
// request unless keep refuses that input.
type timeOffReader struct {
	keep func(input map[string]any) bool

	mu     sync.Mutex
	inputs []map[string]any
}

func (r *timeOffReader) Read(
	_ context.Context,
	req *graphqlreadports.GraphQLReadRequest,
) (*graphqlreadports.GraphQLReadResult, error) {
	input, _ := req.Variables["input"].(map[string]any)
	r.mu.Lock()
	r.inputs = append(r.inputs, input)
	r.mu.Unlock()

	edges := []any{}
	if r.keep == nil || r.keep(input) {
		edges = append(edges, map[string]any{"node": map[string]any{
			"id": "wrkpto_01", "status": "Approved", "type": "Vacation",
		}})
	}

	return &graphqlreadports.GraphQLReadResult{Data: map[string]any{
		"workerPTOEntries": map[string]any{
			"edges":    edges,
			"pageInfo": map[string]any{"hasNextPage": false},
		},
	}}, nil
}

func (r *timeOffReader) first() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.inputs[0]
}

func queryTimeOff(
	t *testing.T,
	reader graphqlreadports.GraphQLReader,
	args map[string]any,
) map[string]any {
	t.Helper()

	tool, err := agentquerytoolservice.ConvertedListTool(
		"worker_list_time_off", reader, &mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	person := agentreadtest.Person()
	result, err := tool.Query(t.Context(), &agenttoolports.QueryToolParams{
		OrganizationID: person.OrganizationID,
		BusinessUnitID: person.BusinessUnitID,
		Actor:          person,
		Timezone:       timeOffZone,
		Params:         args,
		RunID:          pulid.MustNew("ar_"),
	})
	require.NoError(t, err)

	return asMap(t, result)
}

// A call that names no dates reads the leave on at any point from today
// through two weeks out, including leave already under way, with each
// request's worker, as the repository tool's default window did.
func TestTimeOff_ReadsTheNextTwoWeeksOverlappingByDefault(t *testing.T) {
	t.Parallel()

	reader := &timeOffReader{}
	queryTimeOff(t, reader, map[string]any{})

	input := reader.first()
	clk := filtercatalog.NewClock(timeOffZone)
	from, _ := input["startDateFrom"].(int64)
	to, _ := input["startDateTo"].(int64)
	assert.Equal(t, clk.Today(), from, "the window opens on the organization's midnight")
	last := time.Unix(from, 0).In(clk.Location()).AddDate(0, 0, 14)
	assert.Equal(t, clk.DayEnd(last.Unix()), to,
		"through the end of the fourteenth day, whatever the clocks do between")
	assert.Equal(t, true, input["overlapping"])
	assert.Equal(t, true, input["includeWorker"], "each request carries its worker")
}

// Naming either date replaces the default window whole, and the window
// still overlaps unless the call says otherwise.
func TestTimeOff_AGivenDateReplacesTheDefaultWindow(t *testing.T) {
	t.Parallel()

	reader := &timeOffReader{}
	queryTimeOff(t, reader, map[string]any{"startDateFrom": "2026-11-02"})

	input := reader.first()
	assert.Contains(t, input, "startDateFrom")
	assert.NotContains(t, input, "startDateTo", "no default end is added to a given start")
	assert.Equal(t, true, input["overlapping"])

	starting := &timeOffReader{}
	queryTimeOff(t, starting, map[string]any{
		"startDateTo": "2026-11-30", "overlapping": false,
	})
	assert.NotContains(t, starting.first(), "startDateFrom")
	assert.Equal(t, false, starting.first()["overlapping"])
}

// An empty answer says which of the screen's own conditions kept every
// request out, so "nobody is off" is not read from a window that missed.
func TestTimeOff_AnEmptyAnswerNamesTheConditionThatKeptRequestsOut(t *testing.T) {
	t.Parallel()

	reader := &timeOffReader{keep: func(input map[string]any) bool {
		_, windowed := input["startDateFrom"]
		return !windowed
	}}
	outcome := queryTimeOff(t, reader, map[string]any{})

	assert.Equal(t, 0, int(outcome["count"].(float64)))
	assert.Contains(t, outcome["note"], "startDateFrom")
}

// At the default agent data access a time-off request shows who is off,
// when and how, and withholds why: a leave's reasons and the balance it left.
func TestTimeOff_WithholdsALeavesReasonsAtInternal(t *testing.T) {
	t.Parallel()

	registry := fieldsensitivity.DefaultRegistry()
	for _, field := range []string{
		"id", "workerId", "status", "type", "startDate", "endDate", "days", "approverId",
	} {
		assert.True(t, fieldsensitivity.Visible(registry, permission.ResourceWorkerPTO, field,
			permission.SensitivityInternal), field)
	}
	for _, field := range []string{
		"reason", "rejectionReason", "cancellationReason", "balanceAfterDays",
	} {
		assert.False(t, fieldsensitivity.Visible(registry, permission.ResourceWorkerPTO, field,
			permission.SensitivityInternal), field)
		assert.True(t, fieldsensitivity.Visible(registry, permission.ResourceWorkerPTO, field,
			permission.SensitivityRestricted), field)
	}
}
