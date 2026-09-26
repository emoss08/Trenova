package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentexceptionservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingExceptions struct {
	serviceports.AgentExceptionService

	saved *agent.AgentException
	guard writeGuard
}

func (f *savingExceptions) Flag(
	_ context.Context,
	req *serviceports.FlagAgentExceptionRequest,
	_ *serviceports.RequestActor,
) (*agent.AgentException, error) {
	if err := f.guard.write(); err != nil {
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
	tool := newRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true}).(*raiseExceptionTool)
	shipmentID := pulid.MustNew("shp_")
	params := raiseExceptionParams("Shipment", shipmentID)
	params.Params["blastRadius"] = float64(3)

	preview := previewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceAgentException, change.Resource)
	about := fieldByPath(t, change, "subjectId")
	assert.Equal(t, "About", about.Label)
	require.NotNil(t, about.AfterRef)
	assert.Equal(t, permission.ResourceShipment, about.AfterRef.Resource)
	assert.Equal(t, shipmentID, about.AfterRef.ID)
	assert.Equal(t, "The report cannot compute an average dwell.",
		fieldByPath(t, change, "attemptSummary").After)
	assert.Contains(t, preview.Summary, "a shipment")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, exceptions.saved,
		raisedExceptionOptions(exceptions.saved.SubjectType)...)
}

func TestRaiseException_PreviewWarnsOutsideARun(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := newRaiseExceptionTool(exceptions, &fakeSubjectRepository{exists: true}).(*raiseExceptionTool)
	params := raiseExceptionParams("Shipment", pulid.MustNew("shp_"))
	params.RunID = pulid.Nil

	preview := previewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestFlagForManualReview_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	exceptions := &savingExceptions{}
	tool := newFlagManualReviewTool(exceptions).(*flagManualReviewTool)
	itemID := pulid.MustNew("bqi_")
	params := executeParams(map[string]any{
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

	preview := previewWithoutWrites(t, &exceptions.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "High", fieldByPath(t, change, "severity").After)
	about := fieldByPath(t, change, "subjectId")
	require.NotNil(t, about.AfterRef)
	assert.Equal(t, itemID, about.AfterRef.ID)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, exceptions.saved,
		raisedExceptionOptions(exceptions.saved.SubjectType)...)
}
