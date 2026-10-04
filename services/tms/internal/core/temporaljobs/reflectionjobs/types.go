package reflectionjobs

import (
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ThreadReflectionWorkflowName = "AgentThreadReflectionWorkflow"
	RunReflectionWorkflowName    = "AgentRunReflectionWorkflow"
	TurnFinishedSignalName       = "turn-finished"

	QuietPeriod  = 10 * time.Minute
	LongestWait  = time.Hour
	RoundsPerRun = 20
)

func ThreadWorkflowID(threadID pulid.ID) string {
	return "agent-reflection:thread:" + threadID.String()
}

func RunWorkflowID(runID pulid.ID) string {
	return "agent-reflection:run:" + runID.String()
}

type ThreadReflectionInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ThreadID   pulid.ID              `json:"threadId"`
	UserID     pulid.ID              `json:"userId"`
	Rounds     int                   `json:"rounds"`
}

func (in *ThreadReflectionInput) request() *serviceports.ReflectOnThreadRequest {
	return &serviceports.ReflectOnThreadRequest{
		TenantInfo: in.TenantInfo,
		ThreadID:   in.ThreadID,
		UserID:     in.UserID,
	}
}

type RunReflectionInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	RunID      pulid.ID              `json:"runId"`
}

type TurnFinished struct {
	TurnID pulid.ID `json:"turnId"`
}

type FinishReflectionInput struct {
	Plan   *serviceports.ReflectionPlan             `json:"plan"`
	Result *serviceports.StructuredCompletionResult `json:"result"`
}

type ReflectionResult struct {
	Rounds  int `json:"rounds"`
	Kept    int `json:"kept"`
	Offered int `json:"offered"`
	Refused int `json:"refused"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func (r *ReflectionResult) absorb(outcome *serviceports.ReflectionOutcome) {
	if outcome == nil {
		return
	}
	r.Kept += outcome.Kept
	r.Offered += outcome.Offered
	r.Refused += outcome.Refused
}
