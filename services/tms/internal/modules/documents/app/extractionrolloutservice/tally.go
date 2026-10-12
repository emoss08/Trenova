package extractionrolloutservice

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
)

type tally struct {
	candidate         extractionrolloutports.ExtractionRolloutArmReport
	control           extractionrolloutports.ExtractionRolloutArmReport
	candidateOutcomes extractionrollout.ArmOutcomes
	controlOutcomes   extractionrollout.ArmOutcomes
}

func tallyAssignments(totals []extractionrolloutports.RolloutAssignmentTotal) tally {
	var t tally
	for _, total := range totals {
		arm, outcomes := &t.control, &t.controlOutcomes
		if total.Arm == extractionrollout.ArmCandidate {
			arm, outcomes = &t.candidate, &t.candidateOutcomes
		}

		arm.Assigned += total.Count
		switch total.Outcome {
		case extractionrollout.OutcomePending:
			arm.Pending += total.Count
		case extractionrollout.OutcomeAccepted:
			arm.Accepted += total.Count
		case extractionrollout.OutcomeRejected:
			arm.Rejected += total.Count
		case extractionrollout.OutcomeFailed:
			arm.Failed += total.Count
		case extractionrollout.OutcomeSuperseded:
			arm.Superseded += total.Count
		}

		if !counts(total) {
			continue
		}
		if total.Arm == extractionrollout.ArmCandidate &&
			total.ServedBy == extractionrollout.ServedByOther {
			arm.FellBack += total.Count
			continue
		}
		outcomes.Settled += total.Count
		if total.Outcome != extractionrollout.OutcomeAccepted {
			outcomes.Rejected += total.Count
		}
	}

	t.candidate.RejectionRate = t.candidateOutcomes.Rate()
	t.control.RejectionRate = t.controlOutcomes.Rate()

	return t
}

func counts(total extractionrolloutports.RolloutAssignmentTotal) bool {
	return total.Outcome != extractionrollout.OutcomePending &&
		total.Outcome != extractionrollout.OutcomeSuperseded
}
