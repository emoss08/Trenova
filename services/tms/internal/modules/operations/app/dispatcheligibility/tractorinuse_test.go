package dispatcheligibility

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tractorInUseFindings(eval *Evaluation) []dispatchfinding.Finding {
	out := make([]dispatchfinding.Finding, 0)
	for _, finding := range eval.Findings {
		if finding.Code == dispatchfinding.CodeTractorInUse {
			out = append(out, finding)
		}
	}

	return out
}

// The ranker put drivers at the top whose tractor was still on another load,
// and agents learned to check every one by hand. A tractor on a move in
// progress is now a finding: it blocks when that move ends after this one has
// to start, warns when either time is unknown, and says nothing when the
// tractor is free in time.
func TestEvaluateEquipment_ATractorOnAnotherMoveInProgress(t *testing.T) {
	t.Parallel()

	thisMove, otherMove := pulid.MustNew("sm_"), pulid.MustNew("sm_")
	due := int64(1_800_000_000)
	busy := func() *tractor.Tractor {
		return &tractor.Tractor{
			Code:                "T-101",
			Status:              domaintypes.EquipmentStatusAvailable,
			InProgressMoveID:    otherMove,
			InProgressProNumber: "PRO-77",
			InProgressDueAt:     &due,
		}
	}

	blocked := tractorInUseFindings(EvaluateEquipment(EquipmentInput{
		Tractor:    busy(),
		MoveID:     thisMove,
		MoveWindow: TimeWindow{Start: due - 3600},
	}))
	require.Len(t, blocked, 1)
	assert.Equal(t, dispatchfinding.SeverityBlock, blocked[0].Severity)
	assert.Contains(t, blocked[0].Message, "T-101 is on PRO-77")

	assert.Empty(t, tractorInUseFindings(EvaluateEquipment(EquipmentInput{
		Tractor:    busy(),
		MoveID:     thisMove,
		MoveWindow: TimeWindow{Start: due + 3600},
	})), "free before this move starts: ordinary back-to-back planning")

	unknownDue := busy()
	unknownDue.InProgressDueAt = nil
	unknown := tractorInUseFindings(EvaluateEquipment(EquipmentInput{
		Tractor:    unknownDue,
		MoveID:     thisMove,
		MoveWindow: TimeWindow{Start: due + 3600},
	}))
	require.Len(t, unknown, 1)
	assert.Equal(t, dispatchfinding.SeverityWarn, unknown[0].Severity,
		"with no due time it cannot say, so it warns rather than blocks")

	own := busy()
	own.InProgressMoveID = thisMove
	assert.Empty(t, tractorInUseFindings(EvaluateEquipment(EquipmentInput{
		Tractor: own, MoveID: thisMove, MoveWindow: TimeWindow{Start: due - 3600},
	})), "the move it is on is this one")

	free := busy()
	free.InProgressMoveID = pulid.Nil
	assert.Empty(t, tractorInUseFindings(EvaluateEquipment(EquipmentInput{
		Tractor: free, MoveID: thisMove, MoveWindow: TimeWindow{Start: due - 3600},
	})))
}
