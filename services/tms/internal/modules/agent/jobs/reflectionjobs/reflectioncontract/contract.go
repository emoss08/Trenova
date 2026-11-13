package reflectioncontract

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ThreadReflectionWorkflowName = "AgentThreadReflectionWorkflow"
	RunReflectionWorkflowName    = "AgentRunReflectionWorkflow"
	// CorrectionReflectionWorkflowName looks back over one habit of people
	// changing by hand what an agent had set.
	CorrectionReflectionWorkflowName = "AgentCorrectionReflectionWorkflow"
	TurnFinishedSignalName           = "turn-finished"
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

// Request is what the reflection reads for this conversation.
func (in *ThreadReflectionInput) Request() *agentreflectionports.ReflectOnThreadRequest {
	return &agentreflectionports.ReflectOnThreadRequest{
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

// CorrectionWorkflowID is one look back per habit per day: a second cue the
// same night finds the first.
func CorrectionWorkflowID(req *agentreflectionports.ReflectOnCorrectionsRequest) string {
	return fmt.Sprintf("agent-reflection:correction:%s:%d", req.PatternKey, req.Day)
}
