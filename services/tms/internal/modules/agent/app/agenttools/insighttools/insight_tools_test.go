package insighttools

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInsightDismisser struct {
	dismissed *insightports.DismissInsightRequest
	finding   *insight.Insight
	guard     agenttoolstest.WriteGuard
}

func (f *fakeInsightDismisser) GetDetail(
	_ context.Context,
	req insightports.GetInsightDetailRequest,
) (*insightports.InsightDetail, error) {
	if f.finding == nil || f.finding.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Insight not found")
	}
	copied := *f.finding

	return &insightports.InsightDetail{Insight: &copied}, nil
}

func (f *fakeInsightDismisser) Dismiss(
	_ context.Context,
	req insightports.DismissInsightRequest,
) (*insight.Insight, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.dismissed = &req
	if f.finding == nil {
		return &insight.Insight{ID: req.ID, Status: insight.StatusDismissed}, nil
	}
	if err := f.finding.Dismiss(req.UserID, req.Reason, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return f.finding, nil
}

func TestDismissInsight_RecordsTheReasonAsTheApprover(t *testing.T) {
	t.Parallel()

	insights := &fakeInsightDismisser{}
	tool := NewDismissInsightTool(insights)
	id := pulid.MustNew("inst_")
	params := agenttoolstest.MemoryParams(map[string]any{
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
	tool := NewDismissInsightTool(insights)
	id := pulid.MustNew("inst_")

	err := tool.Execute(
		t.Context(),
		agenttoolstest.MemoryParams(map[string]any{"insightId": id.String()}),
	)
	require.Error(t, err)

	err = tool.Execute(t.Context(), agenttoolstest.MemoryParams(map[string]any{
		"insightId": id.String(),
		"reason":    strings.Repeat("x", maxDismissReasonChars+1),
	}))
	require.Error(t, err)
	assert.Nil(t, insights.dismissed, "nothing is dismissed without a usable reason")
}

func TestDismissInsight_IsAnApprovedUpdateOnTheInsight(t *testing.T) {
	t.Parallel()

	tool := NewDismissInsightTool(&fakeInsightDismisser{})
	id := pulid.MustNew("inst_")

	assert.Equal(t, permission.ResourceInsight, tool.Policy().Resource)
	assert.Equal(t, permission.OpUpdate, tool.Policy().Operation)

	targeted, ok := tool.(agenttoolports.TargetedTool)
	require.True(t, ok)
	target, ok := targeted.Target(map[string]any{"insightId": id.String()})
	require.True(t, ok)
	assert.Equal(t, permission.ResourceInsight, target.Resource)
	assert.Equal(t, id, target.ID)
	_, ok = targeted.Target(map[string]any{})
	assert.False(t, ok)
}
