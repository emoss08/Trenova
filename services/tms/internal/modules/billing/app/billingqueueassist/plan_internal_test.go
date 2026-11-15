package billingqueueassist

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// novaOnPlan is Nova with a provider chosen, ready for a task only when the
// plan pays for it, as the harness answers.
type novaOnPlan struct {
	llmports.TaskRunner

	plan platformcatalogports.FeatureGate
	ran  int
}

func (n *novaOnPlan) Available(
	ctx context.Context,
	_ pagination.TenantInfo,
	task aitask.Task,
) (bool, error) {
	return llmports.RequireTask(ctx, n.plan, task) == nil, nil
}

func (n *novaOnPlan) Run(
	context.Context,
	*llmports.NovaTask,
	any,
) (*llmports.StructuredCompletionResult, error) {
	n.ran++

	return &llmports.StructuredCompletionResult{}, nil
}

// The review's verdict, summary, reads and hash are AI written on a base
// record: without Trenova AI's agents the review neither runs nor writes. The
// repository mock fails on any call, so a read of the evidence or a write of
// a review fails the test.
func TestReviewItem_WritesNothingOnABaseOnlyPlan(t *testing.T) {
	t.Parallel()

	nova := &novaOnPlan{plan: plantest.BaseOnly()}
	s := &Service{
		l:        zap.NewNop(),
		repo:     billingqueueportsmock.NewMockBillingQueueRepository(t),
		controls: reviewingControls{},
		nova:     nova,
	}

	outcome, err := s.ReviewItem(t.Context(), pagination.TenantInfo{}, pulid.MustNew("bqi_"))
	require.NoError(t, err)

	assert.False(t, outcome.Reviewed)
	assert.Zero(t, nova.ran)

	available, err := s.Availability(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.False(t, available.Review)
	assert.False(t, available.Brief)
	assert.False(t, available.Search)
}
