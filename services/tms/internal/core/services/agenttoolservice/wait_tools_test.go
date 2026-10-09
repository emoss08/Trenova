package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedWaits struct {
	registered []*serviceports.RegisterWaitRequest
	cancelled  []*serviceports.CancelWaitRequest
}

func (r *recordedWaits) Register(
	_ context.Context,
	req *serviceports.RegisterWaitRequest,
) (*agentwait.Wait, error) {
	r.registered = append(r.registered, req)

	return &agentwait.Wait{ID: pulid.MustNew(agentwait.IDPrefix), Description: req.Description}, nil
}

func (r *recordedWaits) Cancel(
	_ context.Context,
	req *serviceports.CancelWaitRequest,
) (*agentwait.Wait, error) {
	r.cancelled = append(r.cancelled, req)

	return &agentwait.Wait{ID: req.ID}, nil
}

func waitParams(args map[string]any) serviceports.ToolExecuteParams {
	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	return serviceports.ToolExecuteParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Params:         args,
		Timezone:       "America/Chicago",
		DefinitionID:   pulid.MustNew("agdef_"),
		ThreadID:       pulid.MustNew("athr_"),
	}
}

func TestWaitUntil_ReadsAWaitOnATimeInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	waits := &recordedWaits{}
	tool := newWaitUntilTool(waits).(*waitUntilTool)
	params := waitParams(map[string]any{
		"until":            "Time",
		"description":      "Call Acme back",
		"then":             "Ask whether the PO came through",
		"at":               "2026-10-09T15:30",
		"giveUpAfterHours": float64(500),
	})

	result, err := tool.ExecuteWithResult(context.Background(), params)
	require.NoError(t, err)

	require.Len(t, waits.registered, 1)
	req := waits.registered[0]
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 9, 15, 30, 0, 0, chicago).Unix(), req.Condition.At)
	assert.Equal(t, int64(maxWaitHours)*3600, req.LifetimeSeconds, "the give-up is held to a week")
	assert.Equal(t, params.ThreadID, req.ThreadID, "the conversation is the runtime's, not the model's")
	assert.Equal(t, params.DefinitionID, req.DefinitionID)
	assert.Equal(t, "Ask whether the PO came through", req.Then)
	assert.Equal(t, "wait", result.Kind)
	assert.NotEmpty(t, result.IDs["waitId"])
}

func TestWaitUntil_ReadsTheRecordsAndThresholds(t *testing.T) {
	t.Parallel()

	waits := &recordedWaits{}
	tool := newWaitUntilTool(waits).(*waitUntilTool)
	worker := pulid.MustNew("wrk_")

	_, err := tool.ExecuteWithResult(context.Background(), waitParams(map[string]any{
		"until":           "HOSDriveBelow",
		"description":     "Martinez to run low on hours",
		"workerId":        worker.String(),
		"driveHoursBelow": 1.5,
	}))
	require.NoError(t, err)
	assert.Equal(t, worker, waits.registered[0].Condition.WorkerID)
	assert.Equal(t, 90, waits.registered[0].Condition.DriveMinutesBelow)
}

func TestWaitUntil_RefusesAnUnknownKindAndABadID(t *testing.T) {
	t.Parallel()

	tool := newWaitUntilTool(&recordedWaits{}).(*waitUntilTool)

	err := tool.Validate(context.Background(), waitParams(map[string]any{
		"until": "Forever", "description": "x",
	}))
	assert.ErrorContains(t, err, "until must be one of")

	err = tool.Validate(context.Background(), waitParams(map[string]any{
		"until": "Reply", "description": "x", "carrierId": "car",
	}))
	assert.ErrorContains(t, err, "carrierId is not a valid id")
}

func TestCancelWait_NamesTheAgentForABackgroundRun(t *testing.T) {
	t.Parallel()

	waits := &recordedWaits{}
	params := waitParams(map[string]any{"waitId": pulid.MustNew(agentwait.IDPrefix).String()})
	params.ThreadID = pulid.Nil

	require.NoError(t, newCancelWaitTool(waits).Execute(context.Background(), params))
	require.Len(t, waits.cancelled, 1)
	assert.Equal(t, params.DefinitionID, waits.cancelled[0].DefinitionID)
	assert.True(t, waits.cancelled[0].ThreadID.IsNil())
}
