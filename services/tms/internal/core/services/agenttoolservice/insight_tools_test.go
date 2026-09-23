package agenttoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInsightDismisser struct {
	dismissed *serviceports.DismissInsightRequest
}

func (f *fakeInsightDismisser) Dismiss(
	_ context.Context,
	req serviceports.DismissInsightRequest,
) (*insight.Insight, error) {
	f.dismissed = &req

	return &insight.Insight{ID: req.ID, Status: insight.StatusDismissed}, nil
}

func TestDismissInsight_RecordsTheReasonAsTheApprover(t *testing.T) {
	t.Parallel()

	insights := &fakeInsightDismisser{}
	tool := newDismissInsightTool(insights)
	id := pulid.MustNew("inst_")
	params := memoryParams(map[string]any{
		"insightId": id.String(),
		"reason":    "Known seasonal pattern for this customer.",
	})

	require.NoError(t, tool.Execute(t.Context(), params))

	require.NotNil(t, insights.dismissed)
	assert.Equal(t, id, insights.dismissed.ID)
	assert.Equal(t, params.Actor.UserID, insights.dismissed.UserID)
	assert.Equal(t, "Known seasonal pattern for this customer.", insights.dismissed.Reason)
	assert.Equal(t, params.OrganizationID, insights.dismissed.TenantInfo.OrgID)
}

func TestDismissInsight_RequiresAReasonOfBoundedLength(t *testing.T) {
	t.Parallel()

	insights := &fakeInsightDismisser{}
	tool := newDismissInsightTool(insights)
	id := pulid.MustNew("inst_")

	err := tool.Execute(t.Context(), memoryParams(map[string]any{"insightId": id.String()}))
	require.Error(t, err)

	err = tool.Execute(t.Context(), memoryParams(map[string]any{
		"insightId": id.String(),
		"reason":    strings.Repeat("x", maxDismissReasonChars+1),
	}))
	require.Error(t, err)
	assert.Nil(t, insights.dismissed, "nothing is dismissed without a usable reason")
}

func TestDismissInsight_IsAnApprovedUpdateOnTheInsight(t *testing.T) {
	t.Parallel()

	tool := newDismissInsightTool(&fakeInsightDismisser{})
	id := pulid.MustNew("inst_")

	assert.Equal(t, permission.ResourceInsight, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	targeted, ok := tool.(serviceports.TargetedTool)
	require.True(t, ok)
	target, ok := targeted.Target(map[string]any{"insightId": id.String()})
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInsight, target.Resource)
	assert.Equal(t, id, target.ID)
	_, ok = targeted.Target(map[string]any{})
	assert.False(t, ok)
}

func TestDismissInsight_SimulatesWithoutDismissing(t *testing.T) {
	t.Parallel()

	insights := &fakeInsightDismisser{}
	tool := newDismissInsightTool(insights)
	simulator, ok := tool.(serviceports.ToolSimulator)
	require.True(t, ok)

	preview, err := simulator.Simulate(t.Context(), memoryParams(map[string]any{
		"insightId": pulid.MustNew("inst_").String(),
		"reason":    "Already being handled by the account manager.",
	}))
	require.NoError(t, err)

	assert.Nil(t, insights.dismissed)
	assert.True(t, preview.Previewed)
	assert.Contains(t, preview.Describe(), "Dismissed")
	assert.Contains(t, preview.Describe(), "account manager")
}
