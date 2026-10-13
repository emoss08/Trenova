package watchtowerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweepAttention_EscalatesOverdueItemsUpToTheCap(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	due := int64(1_699_999_000)
	f.item.DueAt = &due
	tenant := f.actor.TenantInfo()

	now := int64(1_700_000_000)
	for level := 1; level <= 3; level++ {
		result, err := f.exceptions.SweepAttention(t.Context(), tenant, now)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Escalated)
		item := f.repo.items["ServiceFailure:sf_1"]
		assert.Equal(t, int16(level), item.EscalationLevel)
		require.NotNil(t, item.DueAt)
		assert.Equal(t, now+60*60, *item.DueAt)
		now = *item.DueAt + 1
	}

	item := f.repo.items["ServiceFailure:sf_1"]
	assert.Equal(t, watchtower.SeverityCritical, item.Severity)

	result, err := f.exceptions.SweepAttention(t.Context(), tenant, now)
	require.NoError(t, err)
	assert.Zero(t, result.Escalated)
	assert.Equal(t, int16(3), f.repo.items["ServiceFailure:sf_1"].EscalationLevel)
	assert.Equal(t, []watchtower.EventKind{
		watchtower.EventEscalated, watchtower.EventEscalated, watchtower.EventEscalated,
	}, eventKinds(f.repo.events))
}

func TestSweepAttention_WakesASnoozedItemAndLeavesAFutureOneAlone(t *testing.T) {
	t.Parallel()

	f := newExceptionFixture(t, &agenttest.StubPermissions{})
	require.NoError(t, f.item.Snooze(1_700_000_600, 1_700_000_000))
	tenant := f.actor.TenantInfo()

	result, err := f.exceptions.SweepAttention(t.Context(), tenant, 1_700_000_300)
	require.NoError(t, err)
	assert.Zero(t, result.Woke)
	assert.Equal(t, watchtower.StatusSnoozed, f.repo.items["ServiceFailure:sf_1"].Status)

	result, err = f.exceptions.SweepAttention(t.Context(), tenant, 1_700_000_600)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Woke)
	item := f.repo.items["ServiceFailure:sf_1"]
	assert.Equal(t, watchtower.StatusOpen, item.Status)
	assert.Nil(t, item.SnoozedUntil)
	assert.Equal(t, []watchtower.EventKind{watchtower.EventWoke}, eventKinds(f.repo.events))
}
