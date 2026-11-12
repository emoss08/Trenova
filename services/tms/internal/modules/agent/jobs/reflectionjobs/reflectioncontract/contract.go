package reflectioncontract

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ThreadReflectionWorkflowName = "AgentThreadReflectionWorkflow"
	RunReflectionWorkflowName    = "AgentRunReflectionWorkflow"
	TurnFinishedSignalName       = "turn-finished"
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
