package agentjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type sweptProposals struct {
	repositories.AgentProposalRepository

	expired int
}

func (p *sweptProposals) ExpirePending(
	context.Context,
	repositories.ExpireAgentProposalsRequest,
) (int, error) {
	return p.expired, nil
}

type orphanBaselines struct {
	repositories.AgentProposalBaselineRepository

	asked []repositories.PurgeOrphanProposalBaselinesRequest
	err   error
}

func (b *orphanBaselines) PurgeOrphans(
	_ context.Context,
	req repositories.PurgeOrphanProposalBaselinesRequest,
) (int, error) {
	b.asked = append(b.asked, req)

	return 3, b.err
}

// A retried dispatch mints a new proposal id and leaves the first attempt's
// baseline behind. The expiry sweep removes baselines past the age at which
// their proposal would have been filed.
func TestExpireStaleProposalsActivity_PurgesBaselinesNoProposalWasFiledFor(t *testing.T) {
	t.Parallel()

	baselines := &orphanBaselines{}
	a := &Activities{
		logger:       zap.NewNop(),
		proposalRepo: &sweptProposals{expired: 2},
		baselines:    baselines,
	}

	now := int64(1_800_000_000)
	result, err := a.ExpireStaleProposalsActivity(t.Context(), &ExpireStaleProposalsInput{Now: now})
	require.NoError(t, err)

	assert.Equal(t, 2, result.Expired)
	require.Len(t, baselines.asked, 1)
	assert.Equal(t, now-int64(orphanBaselineAge.Seconds()), baselines.asked[0].Before,
		"only a baseline older than its proposal could still arrive is an orphan")
}

func TestExpireStaleProposalsActivity_APurgeThatFailsDoesNotFailTheSweep(t *testing.T) {
	t.Parallel()

	a := &Activities{
		logger:       zap.NewNop(),
		proposalRepo: &sweptProposals{expired: 1},
		baselines:    &orphanBaselines{err: errors.New("database unavailable")},
	}

	result, err := a.ExpireStaleProposalsActivity(t.Context(), &ExpireStaleProposalsInput{
		Now: 1_800_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Expired)
}
