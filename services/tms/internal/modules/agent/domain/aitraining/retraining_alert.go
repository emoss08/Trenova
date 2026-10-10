package aitraining

import (
	"fmt"
	"strings"
)

const retrainingEventPrefix = "retraining."

func (s RetrainingStatus) EventName() string {
	return retrainingEventPrefix + strings.ToLower(string(s))
}

func (c *RetrainingCycle) AlertSummary() string {
	id := c.ID.String()
	switch c.Status {
	case RetrainingStatusSkipped:
		return fmt.Sprintf("Retraining skipped (%s): %s. %d new corrections, %d needed.",
			id, c.SkipReason.Message(), c.NewExamples, c.MinNewExamples)
	case RetrainingStatusExporting:
		return fmt.Sprintf(
			"Retraining %s started (%s) with %d new corrections; exporting training data.",
			id,
			strings.ToLower(c.Trigger.String()),
			c.NewExamples,
		)
	case RetrainingStatusReady:
		return fmt.Sprintf("Retraining %s is ready to train; its export finished.", id)
	case RetrainingStatusTraining:
		return fmt.Sprintf(
			"Retraining %s is training on %s (attempt %d).",
			id,
			c.Trainer,
			c.Attempts,
		)
	case RetrainingStatusPassed:
		return fmt.Sprintf(
			"Retraining %s passed: model %s against production %s on %d validation examples. Model: %s",
			id,
			formatShare(c.ModelCorrect, c.ModelScored),
			formatShare(c.BaselineCorrect, c.BaselineScored),
			c.Examples,
			c.ModelDirectory,
		)
	case RetrainingStatusRejected:
		return fmt.Sprintf("Retraining %s was rejected by its gate: %s", id, c.GateMessage)
	case RetrainingStatusFailed:
		return fmt.Sprintf("Retraining %s failed: %s", id, c.FailureMessage)
	case RetrainingStatusCanceled:
		return fmt.Sprintf("Retraining %s was canceled.", id)
	default:
		return fmt.Sprintf("Retraining %s is %s.", id, c.Status)
	}
}
