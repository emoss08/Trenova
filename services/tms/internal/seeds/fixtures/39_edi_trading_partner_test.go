package fixtures

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEDISeed_Writes214sThatParseCleanly(t *testing.T) {
	t.Parallel()

	load := &shipment.Shipment{ProNumber: "SEED-SHP-001", BOL: "BOL-1001"}
	raw := seed214(time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC), 3, load, "X3")

	inspection, err := edix12inspect.InspectX12(
		t.Context(),
		&edix12inspect.InspectX12Request{RawX12: raw},
	)
	require.NoError(t, err)
	for _, diagnostic := range inspection.Diagnostics {
		assert.NotEqualf(t, edidocument.ValidationSeverityError, diagnostic.Severity,
			"%s at %s: %s", diagnostic.Code, diagnostic.SegmentID, diagnostic.Message)
	}
	require.Len(t, inspection.Transactions, 1)
	assert.Equal(t, "214", inspection.Transactions[0].TransactionSet)
	assert.Equal(t, "000005003", inspection.Envelope.ISAControlNumber)
}

func TestEDISeed_CoversEveryAcknowledgmentState(t *testing.T) {
	t.Parallel()

	seen := map[edimessage.MessageAcknowledgmentStatus]int{}
	overdue := 0
	for _, entry := range seedEDIMessages {
		seen[entry.ack]++
		if entry.ack == edimessage.MessageAcknowledgmentStatusPending && entry.age > 4*time.Hour {
			overdue++
		}
	}

	assert.Equal(t, 3, overdue, "three 214s wait past the overdue age, one does not yet")
	for _, status := range []edimessage.MessageAcknowledgmentStatus{
		edimessage.MessageAcknowledgmentStatusPending,
		edimessage.MessageAcknowledgmentStatusAccepted,
		edimessage.MessageAcknowledgmentStatusAcceptedWithErrors,
		edimessage.MessageAcknowledgmentStatusRejected,
	} {
		assert.Positivef(t, seen[status], "no seeded 214 is %s", status)
	}
}
