package assistantproposals

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type subsetTool struct{}

func (subsetTool) Name() string        { return "billing_transfer" }
func (subsetTool) Description() string { return "Transfer shipments." }
func (subsetTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipmentIds": toolschema.RecordSubset(permission.ResourceShipment.String(),
				map[string]any{
					toolschema.KeyType:  toolschema.TypeArray,
					toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
				}),
			"billType": map[string]any{toolschema.KeyType: toolschema.TypeString},
		},
		toolschema.KeyRequired: []string{"shipmentIds"},
	}
}
func (subsetTool) Policy() agenttoolports.ToolPolicy { return agenttoolports.ToolPolicy{} }
func (subsetTool) Execute(context.Context, *agenttoolports.ToolExecuteParams) error {
	return nil
}

type subsetTools struct {
	agenttoolports.AgentToolRegistry
}

func (subsetTools) Get(string) (agenttoolports.AgentTool, bool) { return subsetTool{}, true }

type recordingLabeler struct {
	labels agentruntimeports.RecordLabels
	asked  []map[permission.Resource][]pulid.ID
	tenant pagination.TenantInfo
}

func (l *recordingLabeler) Labels(
	_ context.Context,
	tenant pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (agentruntimeports.RecordLabels, error) {
	l.asked = append(l.asked, refs)
	l.tenant = tenant

	return l.labels, nil
}

type choicesFixture struct {
	svc      *Service
	labeler  *recordingLabeler
	request  *conversationports.GetThreadRequest
	first    pulid.ID
	second   pulid.ID
	gone     pulid.ID
	other    pulid.ID
	notAnID  string
	proposal []*agent.AgentProposal
}

func newChoicesFixture(t *testing.T, mayReadShipments bool) *choicesFixture {
	t.Helper()

	f := &choicesFixture{
		first:   pulid.MustNew("shp_"),
		second:  pulid.MustNew("shp_"),
		gone:    pulid.MustNew("shp_"),
		other:   pulid.MustNew("shp_"),
		notAnID: "not-an-id",
	}
	f.labeler = &recordingLabeler{labels: agentruntimeports.RecordLabels{
		permission.ResourceShipment: {
			f.first:  "PRO-1001",
			f.second: "PRO-1002",
			f.other:  "PRO-2001",
		},
	}}
	def := &agentdefinition.Definition{ID: pulid.MustNew("agd_"), Name: "Billing desk"}
	run := &agent.AgentRun{ID: pulid.MustNew("ar_"), AgentDefinitionID: def.ID}
	f.proposal = []*agent.AgentProposal{
		{
			ID:       pulid.MustNew("ap_"),
			RunID:    run.ID,
			ToolName: "billing_transfer",
			Status:   agent.ProposalStatusPending,
			ToolParams: map[string]any{
				"shipmentIds": []any{
					f.first.String(),
					f.gone.String(),
					f.notAnID,
					f.second.String(),
				},
			},
		},
		{
			ID:         pulid.MustNew("ap_"),
			RunID:      run.ID,
			ToolName:   "billing_transfer",
			Status:     agent.ProposalStatusPending,
			ToolParams: map[string]any{"shipmentIds": []any{f.other.String()}},
		},
		{
			ID:         pulid.MustNew("ap_"),
			ToolName:   "billing_transfer",
			Status:     agent.ProposalStatusExecuted,
			ToolParams: map[string]any{"shipmentIds": []any{f.other.String()}},
		},
	}
	threadID := pulid.MustNew("thr_")
	f.request = &conversationports.GetThreadRequest{
		ID:     threadID,
		UserID: pulid.MustNew("usr_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}
	f.request.TenantInfo.UserID = f.request.UserID

	f.svc = newProposalReadsBehind(
		&assistanttest.StubRunRepo{ByID: map[pulid.ID]*agent.AgentRun{run.ID: run}},
		&assistanttest.StubProposalRepo{ByThread: f.proposal},
		&assistanttest.StubConversationRepo{Thread: &conversation.Thread{ID: threadID}},
		assistanttest.ShadowSwitches{Definition: def},
	)
	f.svc.tools = subsetTools{}
	f.svc.labeler = f.labeler
	f.svc.permissions = &assistanttest.SubjectPermissions{Allowed: map[string]bool{
		permission.ResourceShipment.String() + ":" + string(permission.OpRead): mayReadShipments,
	}}

	return f
}

func subsetFieldOf(
	t *testing.T,
	proposal assistantproposalports.AssistantProposal,
) toolschema.Field {
	t.Helper()

	for _, field := range proposal.Fields {
		if field.Name == "shipmentIds" {
			return field
		}
	}
	require.FailNow(t, "the proposal has no shipmentIds field")

	return toolschema.Field{}
}

// The chat card lists every shipment a pending transfer proposed, by its PRO
// number, so a person can untick the ones they do not want. Every proposal in
// the thread is labelled in one read, inside the tenant, and a record that is
// gone, or an id that names no record, is shown by its id.
func TestListThreadProposals_LabelsTheRecordsASubsetOffers(t *testing.T) {
	t.Parallel()

	f := newChoicesFixture(t, true)

	result, err := f.svc.ListThreadProposals(t.Context(), f.request)
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, []toolschema.Choice{
		{ID: f.first.String(), Label: "PRO-1001"},
		{ID: f.gone.String(), Label: f.gone.String()},
		{ID: f.notAnID, Label: f.notAnID},
		{ID: f.second.String(), Label: "PRO-1002"},
	}, subsetFieldOf(t, result[0]).Choices)
	assert.Equal(t, []toolschema.Choice{{ID: f.other.String(), Label: "PRO-2001"}},
		subsetFieldOf(t, result[1]).Choices)
	assert.Empty(t, result[2].Fields, "a decided proposal has nothing to untick")

	require.Len(t, f.labeler.asked, 1, "one read labels every proposal in the thread")
	assert.ElementsMatch(t,
		[]pulid.ID{f.first, f.gone, f.second, f.other},
		f.labeler.asked[0][permission.ResourceShipment],
	)
	assert.Equal(t, f.request.TenantInfo.OrgID, f.labeler.tenant.OrgID)
	assert.Equal(t, f.request.TenantInfo.BuID, f.labeler.tenant.BuID)
}

// A person who may not read shipments is not told their PRO numbers through a
// proposal: the records are listed by the ids the proposal already carries,
// and no label is read.
func TestListThreadProposals_LeavesUnreadableRecordsNamedByTheirIDs(t *testing.T) {
	t.Parallel()

	f := newChoicesFixture(t, false)

	result, err := f.svc.ListThreadProposals(t.Context(), f.request)
	require.NoError(t, err)

	assert.Equal(t, []toolschema.Choice{{ID: f.other.String(), Label: f.other.String()}},
		subsetFieldOf(t, result[1]).Choices)
	assert.Empty(t, f.labeler.asked, "nothing the reader may not see is read")
}
