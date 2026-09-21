package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePTOLister struct {
	request *repositories.ListPTORequest
	items   []*worker.WorkerPTO
}

func (f *fakePTOLister) List(
	_ context.Context,
	req *repositories.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	f.request = req

	return &pagination.CursorListResult[*worker.WorkerPTO]{Items: f.items}, nil
}

func TestListTimeOff_ScopesToTheActorTenant(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	params := testParams(map[string]any{})

	_, err := newListTimeOffTool(pto).Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, pto.request)
	assert.Equal(t, params.OrganizationID, pto.request.Filter.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, pto.request.Filter.TenantInfo.BuID)
}

/*
The window is resolved here, not by the model.

"Who is off?" with no dates has to mean the near future. A model asked to
compute "today plus two weeks" will sometimes get it wrong and will always be
guessing at the server's clock, so the tool sets both ends from its own now.
*/
func TestListTimeOff_DefaultsToAWindowStartingNow(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	before := timeutils.NowUnix()

	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	// The window runs in whole days: from the start of today, so leave that
	// began this morning is still in it, through the end of the horizon's
	// last day.
	assert.LessOrEqual(t, pto.request.StartDateFrom, before)
	assert.Less(t, before-pto.request.StartDateFrom, int64(secondsPerDay))
	assert.Equal(t,
		int64(defaultTimeOffWindowDays+1)*secondsPerDay-1,
		pto.request.StartDateTo-pto.request.StartDateFrom,
	)
}

// An explicit pair wins over the horizon, because a person who named dates
// meant them.
func TestListTimeOff_HonoursAnExplicitWindow(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"startingFrom":   float64(1789560000),
		"startingBefore": float64(1790560000),
		"withinDays":     float64(7),
	}))
	require.NoError(t, err)

	assert.Equal(t, int64(1789560000), pto.request.StartDateFrom)
	assert.Equal(t, int64(1790560000), pto.request.StartDateTo)
}

// A horizon past the cap falls back to the default rather than returning a
// decade of leave the model then has to summarise.
func TestListTimeOff_ClampsAnOversizedHorizon(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"withinDays": float64(maxTimeOffWindowDays + 1),
	}))
	require.NoError(t, err)

	assert.Equal(t,
		int64(defaultTimeOffWindowDays+1)*secondsPerDay-1,
		pto.request.StartDateTo-pto.request.StartDateFrom,
	)
}

/*
A status outside the four is refused, not passed through.

"Pending" is the word people use, and it is not a status this system has. Sent
to the repository it would match nothing, and an empty list reads to a model as
"nobody has asked for time off" — a confident wrong answer about the
organization's data. The refusal names what would work instead.
*/
func TestListTimeOff_RefusesAStatusThatDoesNotExist(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"status": "Pending",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status")
	assert.Nil(t, pto.request, "nothing may be queried on a refused argument")
}

func TestListTimeOff_PassesAValidStatusAndType(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"status": "Requested",
		"type":   "Sick",
	}))
	require.NoError(t, err)

	assert.Equal(t, "Requested", pto.request.Status)
	assert.Equal(t, "Sick", pto.request.Type)
}

func TestListTimeOff_RefusesAWorkerNameWhereAnIdBelongs(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	_, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"workerId": "Maria Ortiz",
	}))

	require.Error(t, err)
	assert.Nil(t, pto.request)
}

// An empty page says which filters produced it, so the model reports what it
// searched for rather than that the organization has nobody off.
func TestListTimeOff_NamesItsFiltersWhenNothingMatched(t *testing.T) {
	t.Parallel()

	pto := &fakePTOLister{}
	result, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{
		"status": "Requested",
	}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
	assert.Contains(t, outcome.SearchedFor, "status Requested")
}

// The row is what a dispatcher covering a board needs, and the decision reason
// is carried so "why was this turned down" needs no second lookup.
func TestListTimeOff_ProjectsTheRowsADispatcherNeeds(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	pto := &fakePTOLister{items: []*worker.WorkerPTO{{
		ID:              pulid.MustNew("wpto_"),
		WorkerID:        workerID,
		Status:          worker.PTOStatusRejected,
		Type:            worker.PTOTypeVacation,
		StartDate:       1789560000,
		EndDate:         1789992000,
		Days:            decimal.NewFromInt(5),
		Reason:          "Family trip",
		RejectionReason: "Three drivers already off that week.",
		Worker:          &worker.Worker{FirstName: "Maria", LastName: "Ortiz"},
	}}}

	result, err := newListTimeOffTool(pto).Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)

	outcome, ok := result.(searchOutcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]timeOffRow)
	require.True(t, ok)
	require.Len(t, rows, 1)

	assert.Equal(t, "Maria Ortiz", rows[0].WorkerName)
	assert.Equal(t, workerID.String(), rows[0].WorkerID)
	assert.Equal(t, "Rejected", rows[0].Status)
	assert.Equal(t, "5", rows[0].Days)
	assert.Equal(t, "Three drivers already off that week.", rows[0].Decision)
}

// The tool answers about time off, and the resource it authorizes against has
// to be the one an organization grants for time off — not the worker record.
func TestListTimeOff_AuthorizesAgainstWorkerPTO(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		permission.ResourceWorkerPTO,
		newListTimeOffTool(&fakePTOLister{}).PermissionResource(),
	)
}

// Every status and type the schema offers has to be one the parser accepts.
func TestListTimeOffSchema_OffersOnlyValuesTheToolAccepts(t *testing.T) {
	t.Parallel()

	properties, ok := newListTimeOffTool(&fakePTOLister{}).
		ParamSchema()["properties"].(map[string]any)
	require.True(t, ok)

	statuses, ok := properties["status"].(map[string]any)["enum"].([]string)
	require.True(t, ok)
	for _, value := range statuses {
		_, err := worker.PTOStatusFromString(value)
		assert.NoError(t, err, "schema offers status %q", value)
	}

	types, ok := properties["type"].(map[string]any)["enum"].([]string)
	require.True(t, ok)
	for _, value := range types {
		_, err := worker.PTOTypeFromString(value)
		assert.NoError(t, err, "schema offers type %q", value)
	}
}
