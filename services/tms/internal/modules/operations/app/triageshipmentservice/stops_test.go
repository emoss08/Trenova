package triageshipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopTimesOf(t *testing.T) {
	t.Parallel()

	arrived := int64(1_791_000_000)
	occurrence := &detention.DetentionOccurrence{
		LocationName:   " Louisville Store ",
		ArrivedAt:      &arrived,
		LateByMinutes:  -5,
		RoundedMinutes: 90,
		BillableAmount: decimal.RequireFromString("112.5"),
		PolicySnapshot: &detention.PolicySnapshot{LateArrivalRule: detention.LateArrivalRuleForfeit},
		Evidence: []*detention.DetentionEvidence{
			{Kind: detention.EvidenceKind("Departure"), Source: detention.EvidenceSourceManual},
			{Kind: detention.EvidenceKindArrival, Source: detention.EvidenceSourceTelematics},
		},
	}

	out := stopTimesOf(occurrence)
	assert.Equal(t, "Louisville Store", out.StopName)
	assert.Equal(t, 0, out.LateMinutes)
	assert.Equal(t, 90, out.BilledDetentionMinutes)
	require.NotNil(t, out.BilledDetentionAmount)
	assert.Equal(t, "112.50", *out.BilledDetentionAmount)
	assert.False(t, out.DetentionPaysWhenLate)
	assert.Equal(t, "ELD", out.ArrivalSource)

	occurrence.BillableAmount = decimal.Zero
	occurrence.Evidence = nil
	occurrence.PolicySnapshot = nil
	out = stopTimesOf(occurrence)
	assert.Nil(t, out.BilledDetentionAmount)
	assert.True(t, out.DetentionPaysWhenLate)
	assert.Equal(t, arrivalSourceUnknown, out.ArrivalSource)
}
