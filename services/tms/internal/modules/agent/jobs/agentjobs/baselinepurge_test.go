package agentjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sweptProposals struct {
	agentproposalports.AgentProposalRepository

	expired int
}

func (p *sweptProposals) ExpirePending(
	context.Context,
	agentproposalports.ExpireAgentProposalsRequest,
) (int, error) {
	return p.expired, nil
}

type orphanBaselines struct {
	agentproposalports.AgentProposalBaselineRepository

	asked []agentproposalports.PurgeOrphanProposalBaselinesRequest
	err   error
}

func (b *orphanBaselines) PurgeOrphans(
	_ context.Context,
	req agentproposalports.PurgeOrphanProposalBaselinesRequest,
) (int, error) {
	b.asked = append(b.asked, req)

	return 3, b.err
}

type sweptRuns struct {
	agentruntimeports.AgentRunRepository
}

func (sweptRuns) FailAbandoned(context.Context, agentruntimeports.FailAbandonedRunsRequest) (int, error) {
	return 0, nil
}

type closingRuns struct {
	agentruntimeports.RunCloser

	closed int
}

func (r *closingRuns) CloseSettled(context.Context, int64) (int, error) {
	r.closed++

	return 0, nil
}

type countingWaker struct {
	woken int
}

func (w *countingWaker) WakeDue(context.Context, int) (int, error) {
	w.woken++

	return 0, nil
}

type noPlansToExpire struct {
	agentplanports.AgentPlanRepository
}

func (noPlansToExpire) ExpirePending(
	context.Context,
	agentplanports.ExpireAgentPlansRequest,
) (int, error) {
	return 0, nil
}

// A retried dispatch mints a new proposal id and leaves the first attempt's
// baseline behind. The expiry sweep removes baselines past the age at which
// their proposal would have been filed.
func TestExpireStaleProposalsActivity_PurgesBaselinesNoProposalWasFiledFor(t *testing.T) {
	t.Parallel()

	baselines := &orphanBaselines{}
	closer := &closingRuns{}
	waker := &countingWaker{}
	a := &Activities{
		logger:       zap.NewNop(),
		proposalRepo: &sweptProposals{expired: 2},
		baselines:    baselines,
		plans:        noPlansToExpire{},
		runRepo:      sweptRuns{},
		runCloser:    closer,
		queuedWork:   []queuedWaker{waker},
	}

	now := int64(1_800_000_000)
	result, err := a.ExpireStaleProposalsActivity(t.Context(), &ExpireStaleProposalsInput{Now: now})
	require.NoError(t, err)

	assert.Equal(t, 2, result.Expired)
	require.Len(t, baselines.asked, 1)
	assert.Equal(t, now-int64(orphanBaselineAge.Seconds()), baselines.asked[0].Before,
		"only a baseline older than its proposal could still arrive is an orphan")
	assert.Equal(t, 1, closer.closed, "the sweep closes the runs its expiry settled")
	assert.Equal(t, 1, waker.woken, "and wakes queued agent work whose wake was lost")
}

func TestExpireStaleProposalsActivity_APurgeThatFailsDoesNotFailTheSweep(t *testing.T) {
	t.Parallel()

	a := &Activities{
		logger:       zap.NewNop(),
		proposalRepo: &sweptProposals{expired: 1},
		baselines:    &orphanBaselines{err: errors.New("database unavailable")},
		plans:        noPlansToExpire{},
		runRepo:      sweptRuns{},
		runCloser:    &closingRuns{},
	}

	result, err := a.ExpireStaleProposalsActivity(t.Context(), &ExpireStaleProposalsInput{
		Now: 1_800_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Expired)
}
