package settlementreleaserepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAHoldIsWhatKeepsASettlementFirst(t *testing.T) {
	t.Parallel()

	repo := &repository{}
	for _, disputes := range []bool{false, true} {
		sql, _ := keeltest.Render(t, repo.settlementQuery(disputes))
		held := strings.Index(sql, `"hold_reason" IS NOT NULL`)
		critical := strings.Index(sql, `"severity":"Critical"`)
		require.GreaterOrEqual(t, held, 0, "the hold is read")
		require.GreaterOrEqual(t, critical, 0)
		assert.Less(t, held, critical, "the hold is named before a critical exception")
	}
}

func TestABatchCountsItsHeldSettlementsApart(t *testing.T) {
	t.Parallel()

	batch := batchOf([]batchRow{{Name: "Pay period ending Nov 21, 2026"}}, &splitRow{
		Settlements: 4, Held: 1, HeldNet: 98_000, Critical: 1, Disputed: 1, DisputedNet: 141_250,
	})

	assert.Equal(t,
		settlementreleaseports.Tally{Count: 1, NetMinor: 98_000},
		batch.Blocked[driversettlement.ReleaseBlockHeld])
	assert.Equal(t, settlementreleaseports.Tally{Count: 3, NetMinor: 239_250}, batch.BlockedTotal())
}
