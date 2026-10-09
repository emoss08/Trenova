package deskbench

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Event struct {
	At   time.Time `json:"at"`
	Name string    `json:"name"`
	Data any       `json:"data,omitempty"`
}

type ToolCallRecord struct {
	CallID         string                      `json:"callId"`
	Name           string                      `json:"name"`
	Arguments      map[string]any              `json:"arguments,omitempty"`
	Why            *conversation.StepRationale `json:"why,omitempty"`
	Effect         string                      `json:"effect,omitempty"`
	Verdict        string                      `json:"verdict,omitempty"`
	Failed         bool                        `json:"failed"`
	Proposed       bool                        `json:"proposed"`
	Summary        string                      `json:"summary,omitempty"`
	Result         string                      `json:"result,omitempty"`
	AgentID        pulid.ID                    `json:"agentId,omitempty"`
	DelegateCallID string                      `json:"delegateCallId,omitempty"`
	StartedAt      time.Time                   `json:"startedAt"`
	FinishedAt     time.Time                   `json:"finishedAt"`
	Finished       bool                        `json:"finished"`
}

func (c *ToolCallRecord) Refused() bool {
	switch c.Verdict {
	case verdictDenied, verdictInvalid, verdictDuplicate, verdictOverBudget, verdictFailed,
		verdictUnregistered, verdictUnknown:
		return true
	}

	return c.Failed
}

const (
	verdictRan          = "ran"
	verdictProposed     = "proposed"
	verdictSimulated    = "simulated"
	verdictDenied       = "denied"
	verdictInvalid      = "invalid"
	verdictDuplicate    = "duplicate"
	verdictOverBudget   = "over_budget"
	verdictFailed       = "failed"
	verdictUnregistered = "unregistered"
	verdictUnknown      = "unknown"
)

type Usage struct {
	ModelCalls      int              `json:"modelCalls"`
	InputTokens     int              `json:"inputTokens"`
	OutputTokens    int              `json:"outputTokens"`
	ReasoningTokens int              `json:"reasoningTokens"`
	CostUSD         *decimal.Decimal `json:"costUsd,omitempty"`
	ModelTime       time.Duration    `json:"modelTimeNs"`
}

func (u *Usage) add(other Usage) {
	u.ModelCalls += other.ModelCalls
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.ReasoningTokens += other.ReasoningTokens
	u.ModelTime += other.ModelTime
	if other.CostUSD != nil {
		total := other.CostUSD.Copy()
		if u.CostUSD != nil {
			total = u.CostUSD.Add(*other.CostUSD)
		}
		u.CostUSD = &total
	}
}

type ProposalRecord struct {
	ID        pulid.ID                   `json:"id"`
	PlanID    pulid.ID                   `json:"planId,omitempty"`
	PlanStep  int                        `json:"planStep,omitempty"`
	Tool      string                     `json:"tool"`
	Arguments map[string]any             `json:"arguments,omitempty"`
	Rationale string                     `json:"rationale,omitempty"`
	Tier      agent.AutonomyTier         `json:"tier,omitempty"`
	Status    agent.ProposalStatus       `json:"status"`
	HeldBy    []string                   `json:"heldBy,omitempty"`
	Preview   *agent.ProposalPreview     `json:"preview,omitempty"`
	Simulated *agent.ToolSimulation      `json:"simulation,omitempty"`
	Error     string                     `json:"error,omitempty"`
	Executed  *agent.ToolExecutionResult `json:"executionResult,omitempty"`
}

type TurnRecord struct {
	TurnID       pulid.ID                         `json:"turnId"`
	ThreadID     pulid.ID                         `json:"threadId"`
	TraceID      string                           `json:"traceId,omitempty"`
	Origin       conversation.AssistantTurnOrigin `json:"origin"`
	Input        string                           `json:"input,omitempty"`
	Status       conversation.AssistantTurnStatus `json:"status"`
	Refused      bool                             `json:"refused"`
	Failure      string                           `json:"failure,omitempty"`
	Reply        string                           `json:"reply"`
	StartedAt    time.Time                        `json:"startedAt"`
	FinishedAt   time.Time                        `json:"finishedAt"`
	Tools        []*ToolCallRecord                `json:"tools"`
	Events       []Event                          `json:"events"`
	StreamDeltas int                              `json:"streamDeltas"`
	ModelCalls   []*ModelCall                     `json:"-"`
	Calls        []CallSummary                    `json:"modelCalls"`
	Messages     []conversation.Message           `json:"messages,omitempty"`
	Proposals    []*ProposalRecord                `json:"proposals,omitempty"`
	Artifacts    []serviceports.AssistantArtifact `json:"artifacts,omitempty"`
	Usage        Usage                            `json:"usage"`
	Models       []string                         `json:"models,omitempty"`
	Unattributed []CallSummary                    `json:"unattributedModelCalls,omitempty"`
	CaptureError string                           `json:"captureError,omitempty"`
}

func (t *TurnRecord) Duration() time.Duration {
	return t.FinishedAt.Sub(t.StartedAt)
}

type CallSummary struct {
	Seq             int                               `json:"seq"`
	Kind            ModelCallKind                     `json:"kind"`
	File            string                            `json:"file,omitempty"`
	Model           string                            `json:"model,omitempty"`
	Provider        string                            `json:"provider,omitempty"`
	Delegate        string                            `json:"delegateCallId,omitempty"`
	Messages        int                               `json:"messages"`
	Tools           int                               `json:"tools"`
	SystemDigest    string                            `json:"systemDigest,omitempty"`
	ToolsDigest     string                            `json:"toolsDigest,omitempty"`
	InputTokens     int                               `json:"inputTokens"`
	OutputTokens    int                               `json:"outputTokens"`
	ReasoningTokens int                               `json:"reasoningTokens"`
	LatencyMs       int64                             `json:"latencyMs"`
	Truncated       bool                              `json:"truncated,omitempty"`
	CutOffCall      string                            `json:"cutOffCall,omitempty"`
	Text            string                            `json:"text,omitempty"`
	Thinking        string                            `json:"thinking,omitempty"`
	ToolCalls       []serviceports.ToolCall           `json:"toolCalls,omitempty"`
	FallbackFrom    *serviceports.ChatProviderFailure `json:"fallbackFrom,omitempty"`
	Retries         []serviceports.ChatRetryNotice    `json:"retries,omitempty"`
	Error           string                            `json:"error,omitempty"`
}

type DecisionRecord struct {
	ProposalID     pulid.ID                   `json:"proposalId,omitempty"`
	PlanID         pulid.ID                   `json:"planId,omitempty"`
	Tool           string                     `json:"tool,omitempty"`
	Decision       agent.DecisionType         `json:"decision"`
	Note           string                     `json:"note,omitempty"`
	Error          string                     `json:"error,omitempty"`
	StatusAfter    agent.ProposalStatus       `json:"statusAfter,omitempty"`
	ExecutionError string                     `json:"executionError,omitempty"`
	Executed       bool                       `json:"executed"`
	Result         *agent.ToolExecutionResult `json:"executionResult,omitempty"`
}
