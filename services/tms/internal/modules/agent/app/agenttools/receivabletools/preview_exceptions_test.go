package receivabletools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentexceptionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingExceptions struct {
	agentports.AgentExceptionService

	saved *agent.AgentException
	guard agenttoolstest.WriteGuard
}

func (f *savingExceptions) Flag(
	_ context.Context,
	req *agentports.FlagAgentExceptionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentException, error) {
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

	preview := agenttoolstest.PreviewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
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

	preview := agenttoolstest.PreviewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestFlagForManualReview_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := NewFlagManualReviewTool(exceptions).(*FlagManualReviewTool)
	itemID := pulid.MustNew("bqi_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		"runId":          pulid.MustNew("arun_").String(),
		"subjectId":      itemID.String(),
		"category":       string(agent.CategoryOther),
		"severity":       string(agent.SeverityHigh),
		"attemptSummary": "The rate confirmation and the invoice disagree by $120.",
		"evidence": []any{map[string]any{
			"type": "document",
			"id":   pulid.MustNew("doc_").String(),
			"note": "rate confirmation",
		}},
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "High", agenttoolstest.FieldByPath(t, change, "severity").After)
	about := agenttoolstest.FieldByPath(t, change, "subjectId")
	require.NotNil(t, about.AfterRef)
	assert.Equal(t, itemID, about.AfterRef.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, exceptions.saved,
		raisedExceptionOptions(exceptions.saved.SubjectType)...)
}
