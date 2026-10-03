package watchtowersources

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeadlines_ComeFromWhatTheRecordHolds(t *testing.T) {
	t.Parallel()

	t.Run("the soonest paper's expiry", func(t *testing.T) {
		t.Parallel()
		in := DescribeExpiringCredentials(ExpiringCredentials{
			WorkerID:   pulid.MustNew("wrk_"),
			WorkerName: "Dana Ortiz",
			Papers: []ExpiringPaper{
				{Name: "CDL", DaysLeft: 30, ExpiresAt: 3_000},
				{Name: "Medical card", DaysLeft: 12, ExpiresAt: 2_000},
			},
		})
		require.NotNil(t, in.DueAt)
		assert.EqualValues(t, 2_000, *in.DueAt)
		assert.Equal(t, "Medical card expires", in.DueLabel)
	})

	t.Run("a paper with no recorded expiry has no deadline", func(t *testing.T) {
		t.Parallel()
		in := DescribeExpiringCredentials(ExpiringCredentials{
			WorkerID: pulid.MustNew("wrk_"),
			Papers:   []ExpiringPaper{{Name: "CDL", DaysLeft: 3}},
		})
		assert.Nil(t, in.DueAt)
	})

	t.Run("a move's start", func(t *testing.T) {
		t.Parallel()
		in := DescribeMoveCoverageRisk(UncoveredMove{MoveID: pulid.MustNew("smv_"), StartsAt: 5_000})
		require.NotNil(t, in.DueAt)
		assert.EqualValues(t, 5_000, *in.DueAt)
		assert.Equal(t, "Move starts", in.DueLabel)
	})

	t.Run("detention: free time, then the clock running", func(t *testing.T) {
		t.Parallel()
		waiting := DescribeDetentionOccurrence(&detention.DetentionOccurrence{
			ID: pulid.MustNew("dto_"), ClockStartAt: 100, FreeTimeExpiresAt: 7_300,
		})
		require.NotNil(t, waiting.DueAt)
		assert.EqualValues(t, 7_300, *waiting.DueAt)
		assert.Equal(t, "Free time ends", waiting.DueLabel)

		billing := DescribeDetentionOccurrence(&detention.DetentionOccurrence{
			ID: pulid.MustNew("dto_"), ClockStartAt: 100, FreeTimeExpiresAt: 7_300, BillableMinutes: 40,
		})
		assert.Equal(t, "Detention is accruing", billing.DueLabel)
	})

	t.Run("a proposal's expiry, and none when it never expires", func(t *testing.T) {
		t.Parallel()
		expiring := DescribeProposal(&agent.AgentProposal{ID: pulid.MustNew("ap_"), ExpiresAt: 9_000}, "Billing")
		require.NotNil(t, expiring.DueAt)
		assert.EqualValues(t, 9_000, *expiring.DueAt)
		assert.Nil(t, DescribeProposal(&agent.AgentProposal{ID: pulid.MustNew("ap_")}, "Billing").DueAt)
	})
}
