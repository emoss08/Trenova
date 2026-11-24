package driversettlementboardreader

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAHeldSettlementSaysItIsOnHoldAndWhyBeforeAnythingElse(t *testing.T) {
	t.Parallel()

	reading := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status:      driversettlement.StatusApproved,
		NetPayMinor: 141_250,
		Currency:    "USD",
		PayDate:     payDay(),
		Block:       driversettlement.ReleaseBlockHeld,
		HoldReason:  driversettlement.HoldReasonRateDispute,
		HoldNote:    "Waiting on the broker's corrected rate confirmation",
		Exceptions: []driversettlement.Exception{{
			Code:     driversettlement.ExceptionCodeNegativeNet,
			Severity: driversettlement.ExceptionSeverityCritical,
			Message:  "Deductions exceed earnings",
		}},
	}, pacific)

	assert.Equal(t, "1412.5", reading.Value.Number.String(), "a held settlement still adds its net")
	assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
	require.NotNil(t, reading.Fact)
	assert.Equal(t, "{0}", reading.Fact.Text)
	require.NotNil(t, reading.Fact.Value)
	assert.Equal(t,
		"On hold: Rate dispute, Waiting on the broker's corrected rate confirmation",
		reading.Fact.Value.Text)

	bare := netPay(context.Background(), &settlementreleaseports.Settlement{
		Status: driversettlement.StatusDraft, Currency: "USD", PayDate: payDay(),
		Block:      driversettlement.ReleaseBlockHeld,
		HoldReason: driversettlement.HoldReasonMissingPaperwork,
	}, pacific)
	require.NotNil(t, bare.Fact)
	assert.Equal(t, "On hold: Missing paperwork", bare.Fact.Value.Text)
}

func TestTheBatchSplitPutsWhatIsOnHoldFirstAmongWhatCannotBeReleased(t *testing.T) {
	t.Parallel()

	batch := openBatch()
	batch.Ready = settlementreleaseports.Tally{Count: 173, NetMinor: 15_963_606}
	batch.Blocked[driversettlement.ReleaseBlockHeld] = settlementreleaseports.Tally{
		Count: 1, NetMinor: 141_250,
	}
	read := batchRead(context.Background(), batch, pacific, 0)

	require.NotNil(t, read.Split)
	labels := make([]string, 0, len(read.Split.Parts))
	var sum int64
	for _, part := range read.Split.Parts {
		labels = append(labels, part.Label)
		sum += part.Value.Number.Shift(2).IntPart()
	}
	assert.Equal(t, []string{
		"Ready to release", "On hold", "Has a critical exception", "Dispute open", "Deducted",
	}, labels)
	assert.Equal(t, int64(18_768_646), sum, "the parts still make up the whole")

	figures := read.Figures
	assert.Equal(t, "1", figures[settlementboard.FigureHeldSettlements].Number.String())
	assert.Equal(t, "10", figures[settlementboard.FigureBlockedSettlements].Number.String())
	assert.Equal(t, "2825", figures[settlementboard.FigureBlockedNet].Number.String())
}
