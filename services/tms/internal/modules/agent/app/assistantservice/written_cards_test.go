package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func writtenRecorder(
	t *testing.T,
	reads ...agenttoolports.AgentQueryTool,
) (*artifactrecorder.Recorder, *assistanttest.StubArtifactRepo) {
	t.Helper()

	repo := &assistanttest.StubArtifactRepo{}
	svc := &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			QueryTools:  &assistanttest.StubQueryRegistry{Tools: reads},
			ActionTools: &assistanttest.StubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, assistanttest.Actor(), nil), repo
}

func executedWrite(
	tool string,
	result *agentrecord.ToolExecutionResult,
) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_write", Name: tool},
		Action: &agentruntimeports.PendingAction{
			ToolName:        tool,
			Executed:        true,
			ExecutionResult: result,
		},
	}
}

/*
A record a change writes is shown as it now stands, without being asked for.

An approved shipment_create named its pro number and the person had to ask to
see the shipment. Any write that names the record it made or changed now
shows that record's card, read again through the tool that reads it, and the
model is told to point to it.
*/
func TestObserve_ShowsTheRecordAWriteMade(t *testing.T) {
	t.Parallel()

	reader := &assistanttest.ShipmentReader{}
	recorder, repo := writtenRecorder(t, reader)
	shipmentID := pulid.MustNew("shp_").String()

	shown, err := recorder.Observe(
		executedWrite("shipment_create", &agentrecord.ToolExecutionResult{
			Action: "created", Kind: "shipment", Name: "S26100125117880",
			Record: &agentrecord.RecordRef{EntityType: "shipment", ID: shipmentID},
		}),
	)
	require.NoError(t, err)

	require.NotNil(t, shown)
	assert.Equal(t, "change created", shown.Written)
	assert.True(t, shown.MustBeLinked(), "the card stays whether or not the reply links it")
	assert.Equal(t, map[string]any{"shipmentId": shipmentID}, reader.Asked)
	require.Len(t, repo.Upserts, 1)
	card := repo.Upserts[0]
	assert.Equal(t, assistantartifact.KindEntityCard, card.Kind)
	assert.Equal(t, "created", card.Payload[toolartifacts.PayloadWritten])
	assert.Equal(t, shipmentID, card.Payload[toolartifacts.PayloadRecordID])
	assert.Equal(t, "call_write", card.SourceToolCallID)
}

func TestObserve_ShowsARecordWithNoReaderByWhatTheWriteSaid(t *testing.T) {
	t.Parallel()

	recorder, repo := writtenRecorder(t)
	locationID := pulid.MustNew("loc_").String()

	shown, err := recorder.Observe(
		executedWrite("location_create", &agentrecord.ToolExecutionResult{
			Action: "created", Kind: "location", Name: "Harbor Line Cold Storage",
			Record: &agentrecord.RecordRef{EntityType: "location", ID: locationID},
		}),
	)
	require.NoError(t, err)

	require.NotNil(t, shown)
	assert.Equal(t, "change created", shown.Written)
	require.Len(t, repo.Upserts, 1)
	assert.Equal(t, locationID, repo.Upserts[0].Payload[toolartifacts.PayloadRecordID])
}

func TestObserve_AProposedWriteShowsNoRecord(t *testing.T) {
	t.Parallel()

	recorder, repo := writtenRecorder(t, &assistanttest.ShipmentReader{})
	observation := executedWrite("shipment_create", nil)
	observation.Action.Executed = false

	shown, err := recorder.Observe(observation)
	require.NoError(t, err)
	assert.Nil(t, shown)
	assert.Empty(t, repo.Upserts)
}

func TestShowWritten_PointsTheFollowUpAtWhatTheApprovalMade(t *testing.T) {
	t.Parallel()

	repo := &assistanttest.StubArtifactRepo{}
	svc := &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		runtime: agentruntime.New(agentruntime.Params{
			Logger: zap.NewNop(),
			QueryTools: &assistanttest.StubQueryRegistry{
				Tools: []agenttoolports.AgentQueryTool{&assistanttest.ShipmentReader{}},
			},
			ActionTools: &assistanttest.StubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	shipmentID := pulid.MustNew("shp_").String()
	executed := &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), ToolName: "shipment_create", Status: agent.ProposalStatusExecuted,
		ExecutionResult: &agentrecord.ToolExecutionResult{
			Action: "created", Record: &agentrecord.RecordRef{EntityType: "shipment", ID: shipmentID},
		},
	}
	declined := &agent.AgentProposal{ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusRejected}

	note := svc.showWritten(
		t.Context(),
		thread,
		assistanttest.Actor(),
		assistanttest.Actor().TenantInfo(),
		[]*agent.AgentProposal{executed, declined},
	)

	require.Len(t, repo.Upserts, 1)
	assert.Equal(
		t,
		toolartifacts.DecisionCallID+executed.ID.String(),
		repo.Upserts[0].SourceToolCallID,
	)
	assert.Contains(t, note, "The record this change created is shown beside the conversation")
	assert.Contains(t, note, "(artifact:"+repo.Upserts[0].ID.String()+")")
}
