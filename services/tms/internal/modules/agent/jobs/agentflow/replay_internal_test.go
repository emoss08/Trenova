package agentflow

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordedCases struct {
	recording *agentquality.Recording
	err       error
	asked     []pulid.ID
	tenants   []pagination.TenantInfo
}

func (r *recordedCases) RecordedReads(
	_ context.Context,
	tenant pagination.TenantInfo,
	caseID pulid.ID,
) (agentruntimeports.RecordedReads, error) {
	r.asked = append(r.asked, caseID)
	r.tenants = append(r.tenants, tenant)
	if r.err != nil {
		return nil, r.err
	}
	if r.recording == nil {
		return nil, nil
	}

	return r.recording, nil
}

func replayRequest(replay *agentruntimeports.ReplayOf) *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agd_")},
		Actor: &permissionports.RequestActor{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		Context: agentturn.RuntimeContext{Timezone: "America/Chicago", Now: 1_772_530_200},
		Input:   "Where is S-100?",
		Replay:  replay,
	}
}

// A run's tool calls are dispatched in activities of their own, from the run
// context alone. What makes a replay a replay has to survive that trip, the
// way it travels through Temporal: as JSON.
func TestRunContext_AReplayKeepsItsCaseAndItsClockAcrossActivities(t *testing.T) {
	t.Parallel()

	caseID := pulid.MustNew("aec_")
	sent := NewRunContext(replayRequest(&agentruntimeports.ReplayOf{
		CaseID: caseID,
		AsOf:   1_772_530_200,
	}), 0)

	encoded, err := sonic.Marshal(sent)
	require.NoError(t, err)
	var received RunContext
	require.NoError(t, sonic.Unmarshal(encoded, &received))

	req := received.request()
	require.NotNil(t, req.Replay)
	assert.Equal(t, caseID, req.Replay.CaseID)
	assert.Equal(t, int64(1_772_530_200), req.Context.Clock(),
		"a tool result's dates are worded against the day the case was asked")
	assert.Equal(t, "America/Chicago", req.Context.Timezone)
	assert.Nil(t, req.Recorded, "the recording is loaded by the activity, not carried")
}

func TestRunContext_ARunThatReplaysNothingKeepsTheClockOfTheActivityItRunsIn(t *testing.T) {
	t.Parallel()

	rc := NewRunContext(replayRequest(nil), 0)
	req := rc.request()

	assert.Nil(t, req.Replay)
	assert.Zero(t, req.Context.Now,
		"an ordinary run words its dates against now, however long ago its turn began")
}

func TestRunContext_AReplayOfACaseWithNoAskedTimeKeepsTheClockToo(t *testing.T) {
	t.Parallel()

	rc := NewRunContext(replayRequest(&agentruntimeports.ReplayOf{
		CaseID: pulid.MustNew("aec_"),
	}), 0)
	req := rc.request()

	require.NotNil(t, req.Replay)
	assert.Zero(t, req.Context.Now)
}

func TestRecordedReads_AReplaysCallLoadsWhatItsCaseWasGiven(t *testing.T) {
	t.Parallel()

	recording := (&agentquality.EvalCase{ToolFixtures: []agentquality.ToolFixture{{
		Tool:   "shipment_get",
		Args:   map[string]any{"id": "shp_1"},
		Result: map[string]any{"status": "InTransit"},
	}}}).Recording()
	cases := &recordedCases{recording: recording}
	activities := &Activities{l: zap.NewNop(), recorded: cases}
	caseID := pulid.MustNew("aec_")
	req := replayRequest(&agentruntimeports.ReplayOf{CaseID: caseID})

	recorded, err := activities.recordedReads(t.Context(), req)
	require.NoError(t, err)

	require.NotNil(t, recorded)
	result, _, ok := recorded.Recorded("shipment_get", map[string]any{"id": "shp_1"})
	require.True(t, ok)
	assert.Equal(t, map[string]any{"status": "InTransit"}, result)
	assert.Equal(t, []pulid.ID{caseID}, cases.asked)
	require.Len(t, cases.tenants, 1)
	assert.Equal(t, req.Actor.OrganizationID, cases.tenants[0].OrgID,
		"the case is read in the organization the run belongs to")
}

func TestRecordedReads_ARunThatReplaysNothingLoadsNothing(t *testing.T) {
	t.Parallel()

	cases := &recordedCases{}
	activities := &Activities{l: zap.NewNop(), recorded: cases}

	recorded, err := activities.recordedReads(t.Context(), replayRequest(nil))
	require.NoError(t, err)

	assert.Nil(t, recorded)
	assert.Empty(t, cases.asked, "an ordinary run costs no read of a case")
}

func TestRecordedReads_ACaseThatCannotBeReadFailsTheCallRatherThanReadingTheRecords(t *testing.T) {
	t.Parallel()

	activities := &Activities{l: zap.NewNop(), recorded: &recordedCases{err: errors.New("database is down")}}

	_, err := activities.recordedReads(
		t.Context(),
		replayRequest(&agentruntimeports.ReplayOf{CaseID: pulid.MustNew("aec_")}),
	)

	require.Error(t, err,
		"a replay that silently read live records would be scored as if it had not")
}
