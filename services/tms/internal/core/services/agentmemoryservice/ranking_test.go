package agentmemoryservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const day = int64(24 * 60 * 60)

func memoryAt(createdAt int64, uses int) *agent.Memory {
	return &agent.Memory{
		ID:        pulid.MustNew("amem_"),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		UseCount:  uses,
	}
}

func TestRankByRecencyAndUse_PrefersRecentAndUsedMemories(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	stale := memoryAt(now-365*day, 0)
	fresh := memoryAt(now-day, 0)
	wellUsed := memoryAt(now-90*day, 400)

	ranked := RankByRecencyAndUse([]*agent.Memory{stale, wellUsed, fresh}, now)

	assert.Equal(t, []*agent.Memory{fresh, wellUsed, stale}, ranked)
}

func TestRankByRecencyAndUse_AnEditCountsAsRecent(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	edited := memoryAt(now-200*day, 0)
	edited.UpdatedAt = now - day
	untouched := memoryAt(now-100*day, 0)

	assert.Equal(t, []*agent.Memory{edited, untouched},
		RankByRecencyAndUse([]*agent.Memory{untouched, edited}, now))
}

func TestRankByRecencyAndUse_IsStableForEqualScores(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	first := &agent.Memory{ID: pulid.ID("amem_A"), CreatedAt: now, UpdatedAt: now}
	second := &agent.Memory{ID: pulid.ID("amem_B"), CreatedAt: now, UpdatedAt: now}

	assert.Equal(t, []*agent.Memory{first, second},
		RankByRecencyAndUse([]*agent.Memory{second, nil, first}, now),
		"ties break by id so the same memories always rank the same way")
}

func TestRecencyRanker_RanksThroughThePort(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	old, recent := memoryAt(now-60*day, 0), memoryAt(now, 0)

	ranked, err := NewRecencyRanker().RankMemories(t.Context(), &services.RankMemoriesRequest{
		TenantInfo: tenant(),
		Now:        now,
		Memories:   []*agent.Memory{old, recent},
	})
	require.NoError(t, err)
	assert.Equal(t, []*agent.Memory{recent, old}, ranked)
}
