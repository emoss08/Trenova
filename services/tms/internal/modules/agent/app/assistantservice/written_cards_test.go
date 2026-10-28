package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type shipmentReader struct {
	asked map[string]any
}

func (r *shipmentReader) Name() string        { return "shipment_get" }
func (r *shipmentReader) Description() string { return "Retrieve one shipment." }
func (r *shipmentReader) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:     toolschema.TypeObject,
		toolschema.KeyRequired: []string{"shipmentId"},
	}
}

func (r *shipmentReader) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name: "shipment_get", Kind: agent.ToolKindQuery,
		Resource: permission.ResourceShipment, Operation: permission.OpRead,
	}
}

func (r *shipmentReader) Query(
	_ context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	r.asked = params.Params

	return map[string]any{
		"id":        params.Params["shipmentId"],
		"proNumber": "S26100125117880",
		"status":    "New",
		"customer":  map[string]any{"name": "Harbor Line Foods"},
	}, nil
}

func writtenRecorder(
	t *testing.T,
	reads ...agenttoolports.AgentQueryTool,
) (*artifactRecorder, *stubArtifactRepo) {
	t.Helper()

	repo := &stubArtifactRepo{}
	svc := &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			QueryTools:  &stubQueryRegistry{Tools: reads},
			ActionTools: &stubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), nil), repo
}

func executedWrite(
	tool string,
	result *agent.ToolExecutionResult,
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

	reader := &shipmentReader{}
	recorder, repo := writtenRecorder(t, reader)
	shipmentID := pulid.MustNew("shp_").String()

	shown, err := recorder.observe(executedWrite("shipment_create", &agent.ToolExecutionResult{
		Action: "created", Kind: "shipment", Name: "S26100125117880",
		Record: &agent.RecordRef{EntityType: "shipment", ID: shipmentID},
	}))
	require.NoError(t, err)

	require.NotNil(t, shown)
	assert.Equal(t, "change created", shown.Written)
	assert.True(t, shown.MustBeLinked(), "the card stays whether or not the reply links it")
	assert.Equal(t, map[string]any{"shipmentId": shipmentID}, reader.asked)
	require.Len(t, repo.upserts, 1)
	card := repo.upserts[0]
	assert.Equal(t, assistantartifact.KindEntityCard, card.Kind)
	assert.Equal(t, "created", card.Payload[payloadWritten])
	assert.Equal(t, shipmentID, card.Payload[payloadRecordID])
	assert.Equal(t, "call_write", card.SourceToolCallID)
}

func TestObserve_ShowsARecordWithNoReaderByWhatTheWriteSaid(t *testing.T) {
	t.Parallel()

	recorder, repo := writtenRecorder(t)
	locationID := pulid.MustNew("loc_").String()

	shown, err := recorder.observe(executedWrite("location_create", &agent.ToolExecutionResult{
		Action: "created", Kind: "location", Name: "Harbor Line Cold Storage",
		Record: &agent.RecordRef{EntityType: "location", ID: locationID},
	}))
	require.NoError(t, err)

	require.NotNil(t, shown)
	assert.Equal(t, "change created", shown.Written)
	require.Len(t, repo.upserts, 1)
	assert.Equal(t, locationID, repo.upserts[0].Payload[payloadRecordID])
}

func TestObserve_AProposedWriteShowsNoRecord(t *testing.T) {
	t.Parallel()

	recorder, repo := writtenRecorder(t, &shipmentReader{})
	observation := executedWrite("shipment_create", nil)
	observation.Action.Executed = false

	shown, err := recorder.observe(observation)
	require.NoError(t, err)
	assert.Nil(t, shown)
	assert.Empty(t, repo.upserts)
}

func TestShowWritten_PointsTheFollowUpAtWhatTheApprovalMade(t *testing.T) {
	t.Parallel()

	repo := &stubArtifactRepo{}
	svc := &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		runtime: agentruntime.New(agentruntime.Params{
			Logger: zap.NewNop(),
			QueryTools: &stubQueryRegistry{
				Tools: []agenttoolports.AgentQueryTool{&shipmentReader{}},
			},
			ActionTools: &stubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	shipmentID := pulid.MustNew("shp_").String()
	executed := &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), ToolName: "shipment_create", Status: agent.ProposalStatusExecuted,
		ExecutionResult: &agent.ToolExecutionResult{
			Action: "created", Record: &agent.RecordRef{EntityType: "shipment", ID: shipmentID},
		},
	}
	declined := &agent.AgentProposal{ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusRejected}

	note := svc.showWritten(t.Context(), thread, testActor(), testActor().TenantInfo(),
		[]*agent.AgentProposal{executed, declined})

	require.Len(t, repo.upserts, 1)
	assert.Equal(t, decisionCallID+executed.ID.String(), repo.upserts[0].SourceToolCallID)
	assert.Contains(t, note, "The record this change created is shown beside the conversation")
	assert.Contains(t, note, "(artifact:"+repo.upserts[0].ID.String()+")")
}
