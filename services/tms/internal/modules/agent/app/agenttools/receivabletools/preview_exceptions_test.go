package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentexceptionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingExceptions struct {
	agentports.AgentExceptionService

	saved *agentexception.AgentException
	guard agenttoolstest.WriteGuard
}

func (f *savingExceptions) Flag(
	_ context.Context,
	req *agentports.FlagAgentExceptionRequest,
	_ *permissionports.RequestActor,
) (*agentexception.AgentException, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	entity, err := agentexceptionservice.PlanException(req)
	if err != nil {
		return nil, err
	}
	entity.ID = pulid.MustNew("aexc_")
	f.saved = entity

	return entity, nil
}

func TestRaiseException_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := NewRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true}).(*RaiseExceptionTool)
	shipmentID := pulid.MustNew("shp_")
	params := raiseExceptionParams("Shipment", shipmentID)
	params.Params["blastRadius"] = float64(3)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&exceptions.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agentpreview.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceAgentException, change.Resource)
	about := agenttoolstest.FieldByPath(t, change, "subjectId")
	assert.Equal(t, "About", about.Label)
	require.NotNil(t, about.AfterRef)
	assert.Equal(t, permission.ResourceShipment, about.AfterRef.Resource)
	assert.Equal(t, shipmentID, about.AfterRef.ID)
	assert.Equal(t, "The report cannot compute an average dwell.",
		agenttoolstest.FieldByPath(t, change, "attemptSummary").After)
	assert.Contains(t, preview.Summary, "a shipment")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, exceptions.saved,
		raisedExceptionOptions(exceptions.saved.SubjectType)...)
}

func TestRaiseException_PreviewWarnsOutsideARun(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := NewRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true}).(*RaiseExceptionTool)
	params := raiseExceptionParams("Shipment", pulid.MustNew("shp_"))
	params.RunID = pulid.Nil

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&exceptions.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
}

func TestRaiseException_FlagsABillingQueueItemWithItsEvidence(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	subjects := &fakeSubjectRepository{exists: true}
	tool := NewRaiseExceptionTool(exceptions, subjects).(*RaiseExceptionTool)
	itemID := pulid.MustNew("bqi_")
	documentID := pulid.MustNew("doc_")
	params := raiseExceptionParams("BillingQueueItem", itemID)
	params.Params["category"] = string(agentexception.CategoryIncorrectRates)
	params.Params["severity"] = string(agentexception.SeverityHigh)
	params.Params["attemptSummary"] = "The rate confirmation and the invoice disagree by $120."
	params.Params["evidence"] = []any{map[string]any{
		"type": "document",
		"id":   " " + documentID.String() + " ",
		"note": "rate confirmation",
	}}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&exceptions.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "High", agenttoolstest.FieldByPath(t, change, "severity").After)
	about := agenttoolstest.FieldByPath(t, change, "subjectId")
	require.NotNil(t, about.AfterRef)
	assert.Equal(t, permission.ResourceBillingQueue, about.AfterRef.Resource)
	assert.Equal(t, itemID, about.AfterRef.ID)
	assert.Contains(t, preview.Summary, "billing queue item")

	require.NoError(t, tool.Validate(t.Context(), params))
	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, exceptions.saved,
		raisedExceptionOptions(exceptions.saved.SubjectType)...)

	saved := exceptions.saved
	assert.Equal(t, agent.SubjectBillingQueueItem, saved.SubjectType)
	assert.Equal(t, itemID, saved.SubjectID)
	assert.Equal(t, agentexception.CategoryIncorrectRates, saved.Category)
	assert.Equal(t, params.RunID, saved.RunID, "the run is the runtime's, never the model's")
	assert.Equal(t, []agentevidence.EvidenceRef{
		{Type: "agent_run", ID: params.RunID.String(), Note: "raised by the agent"},
		{Type: "document", ID: documentID.String(), Note: "rate confirmation"},
	}, saved.Evidence, "the run comes first, then the records the agent named")
	require.Len(t, subjects.asked, 3)
	assert.Equal(t, agent.SubjectBillingQueueItem, subjects.asked[0].SubjectType)
}

func TestRaiseException_KeepsTheRunAsEvidenceWhenNoneIsGiven(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := NewRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true})
	params := raiseExceptionParams("BillingQueueItem", pulid.MustNew("bqi_"))

	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, exceptions.saved)
	assert.Equal(t, []agentevidence.EvidenceRef{
		{Type: "agent_run", ID: params.RunID.String(), Note: "raised by the agent"},
	}, exceptions.saved.Evidence)
}

func TestRaiseException_RefusesEvidenceWithoutATypeOrAnID(t *testing.T) {
	t.Parallel()

	for name, evidence := range map[string]any{
		"no id":        []any{map[string]any{"type": "document"}},
		"a blank type": []any{map[string]any{"type": " ", "id": pulid.MustNew("doc_").String()}},
		"not a list":   "the rate confirmation",
	} {
		exceptions := &savingExceptions{}
		subjects := &fakeSubjectRepository{exists: true}
		tool := NewRaiseExceptionTool(exceptions, subjects)
		params := raiseExceptionParams("BillingQueueItem", pulid.MustNew("bqi_"))
		params.Params["evidence"] = evidence

		validator, ok := tool.(agenttoolports.ToolValidator)
		require.True(t, ok)
		require.Error(t, validator.Validate(t.Context(), params), name)
		require.Error(t, tool.Execute(t.Context(), params), name)
		assert.Nil(t, exceptions.saved, name)
	}
}
