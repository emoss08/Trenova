package reflectionjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	QuietPeriod  = 10 * time.Minute
	LongestWait  = time.Hour
	RoundsPerRun = 20
)

type FinishReflectionInput struct {
	Plan   *agentreflectionports.ReflectionPlan `json:"plan"`
	Result *llmports.StructuredCompletionResult `json:"result"`
}

type ReflectionResult struct {
	Rounds  int `json:"rounds"`
	Kept    int `json:"kept"`
	Offered int `json:"offered"`
	Refused int `json:"refused"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func (r *ReflectionResult) absorb(outcome *agentreflectionports.ReflectionOutcome) {
	if outcome == nil {
		return
	}
	r.Kept += outcome.Kept
	r.Offered += outcome.Offered
	r.Refused += outcome.Refused
}
