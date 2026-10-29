package insighttools_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/insighttools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInsightRestorer struct {
	guard    *agenttoolstest.WriteGuard
	found    *insight.Insight
	hidden   bool
	restored bool
}

func (f *fakeInsightRestorer) GetDetail(
	context.Context,
	insightports.GetInsightDetailRequest,
) (*insightports.InsightDetail, error) {
	if f.hidden {
		return nil, errortypes.NewAuthorizationError("You cannot see this insight")
	}

	return &insightports.InsightDetail{Insight: f.found}, nil
}

func (f *fakeInsightRestorer) PlanRestore(
	context.Context,
	insightports.RestoreInsightRequest,
) (*kernelports.RecordChange[insight.Insight], error) {
	restored := *f.found
	if err := restored.Restore(); err != nil {
		return nil, err
	}

	return &kernelports.RecordChange[insight.Insight]{Before: f.found, After: &restored}, nil
}

func (f *fakeInsightRestorer) Restore(
	context.Context,
	insightports.RestoreInsightRequest,
) (*insight.Insight, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.restored = true

	return f.found, nil
}

func TestRestoreInsight_OnlyADismissedFindingTheReaderMaySee(t *testing.T) {
	t.Parallel()

	dismissedAt := int64(1_789_000_000)
	insights := &fakeInsightRestorer{guard: &agenttoolstest.WriteGuard{}, found: &insight.Insight{
		ID:            pulid.MustNew("inst_"),
		Headline:      "Late deliveries for Acme",
		Status:        insight.StatusDismissed,
		DismissedAt:   &dismissedAt,
		DismissReason: "Seasonal",
		Version:       2,
	}}
	tool := insighttools.NewRestoreInsightTool(insights)
	params := agenttoolstest.ExecuteParams(
		map[string]any{insighttools.ParamInsightID: insights.found.ID.String()},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		insights.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Equal(
		t,
		string(insight.StatusActive),
		agenttoolstest.FieldByPath(
			t,
			agenttoolstest.PreviewChange(t, preview, 0),
			changepreview.FieldStatus,
		).After,
	)
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.True(t, insights.restored)

	insights.found.Status = insight.StatusActive
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	insights.hidden = true
	insights.restored = false
	require.Error(t, tool.Execute(t.Context(), params))
	assert.False(t, insights.restored)
}
