package deskbench

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Event struct {
	At   time.Time `json:"at"`
	Name string    `json:"name"`
	Data any       `json:"data,omitempty"`
}

type ToolCallRecord struct {
	CallID         string         `json:"callId"`
	Name           string         `json:"name"`
	Arguments      map[string]any `json:"arguments,omitempty"`
	Effect         string         `json:"effect,omitempty"`
	Verdict        string         `json:"verdict,omitempty"`
	Failed         bool           `json:"failed"`
	Proposed       bool           `json:"proposed"`
	Summary        string         `json:"summary,omitempty"`
	Result         string         `json:"result,omitempty"`
	AgentID        pulid.ID       `json:"agentId,omitempty"`
	DelegateCallID string         `json:"delegateCallId,omitempty"`
	StartedAt      time.Time      `json:"startedAt"`
	FinishedAt     time.Time      `json:"finishedAt"`
	Finished       bool           `json:"finished"`
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
	ModelCalls          int              `json:"modelCalls"`
	InputTokens         int              `json:"inputTokens"`
	CacheReadTokens     int              `json:"cacheReadTokens"`
	CacheWriteTokens    int              `json:"cacheWriteTokens"`
	OutputTokens        int              `json:"outputTokens"`
	ReasoningTokens     int              `json:"reasoningTokens"`
	AdvisorCalls        int              `json:"advisorCalls,omitempty"`
	AdvisorPromptTokens int              `json:"advisorPromptTokens,omitempty"`
	AdvisorOutputTokens int              `json:"advisorOutputTokens,omitempty"`
	CostUSD             *decimal.Decimal `json:"costUsd,omitempty"`
	EstimatedCalls      int              `json:"estimatedCalls,omitempty"`
	UnpricedCalls       int              `json:"unpricedCalls,omitempty"`
	ModelTime           time.Duration    `json:"modelTimeNs"`
}

func (u *Usage) PromptTokens() int {
	return u.InputTokens + u.CacheReadTokens + u.CacheWriteTokens
}

func (u *Usage) add(other Usage) {
	u.ModelCalls += other.ModelCalls
	u.InputTokens += other.InputTokens
	u.CacheReadTokens += other.CacheReadTokens
	u.CacheWriteTokens += other.CacheWriteTokens
	u.OutputTokens += other.OutputTokens
	u.ReasoningTokens += other.ReasoningTokens
	u.AdvisorCalls += other.AdvisorCalls
	u.AdvisorPromptTokens += other.AdvisorPromptTokens
	u.AdvisorOutputTokens += other.AdvisorOutputTokens
	u.EstimatedCalls += other.EstimatedCalls
	u.UnpricedCalls += other.UnpricedCalls
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
	ID        pulid.ID                         `json:"id"`
	PlanID    pulid.ID                         `json:"planId,omitempty"`
	PlanStep  int                              `json:"planStep,omitempty"`
	Tool      string                           `json:"tool"`
	Arguments map[string]any                   `json:"arguments,omitempty"`
	Rationale string                           `json:"rationale,omitempty"`
	Tier      agent.AutonomyTier               `json:"tier,omitempty"`
	Status    agent.ProposalStatus             `json:"status"`
	HeldBy    []string                         `json:"heldBy,omitempty"`
	Preview   *agentpreview.ProposalPreview    `json:"preview,omitempty"`
	Simulated *agent.ToolSimulation            `json:"simulation,omitempty"`
	Error     string                           `json:"error,omitempty"`
	Executed  *agentrecord.ToolExecutionResult `json:"executionResult,omitempty"`
}

type TurnRecord struct {
	TurnID       pulid.ID                             `json:"turnId"`
	RunID        pulid.ID                             `json:"runId,omitzero"`
	ThreadID     pulid.ID                             `json:"threadId"`
	TraceID      string                               `json:"traceId,omitempty"`
	Origin       conversationturn.AssistantTurnOrigin `json:"origin"`
	Input        string                               `json:"input,omitempty"`
	Status       conversation.AssistantTurnStatus     `json:"status"`
	Refused      bool                                 `json:"refused"`
	Failure      string                               `json:"failure,omitempty"`
	Reply        string                               `json:"reply"`
	StartedAt    time.Time                            `json:"startedAt"`
	FinishedAt   time.Time                            `json:"finishedAt"`
	Tools        []*ToolCallRecord                    `json:"tools"`
	Events       []Event                              `json:"events"`
	StreamDeltas int                                  `json:"streamDeltas"`
	ModelCalls   []*ModelCall                         `json:"-"`
	Calls        []CallSummary                        `json:"modelCalls"`
	Messages     []conversation.Message               `json:"messages,omitempty"`
	Proposals    []*ProposalRecord                    `json:"proposals,omitempty"`
	Artifacts    []assistantports.AssistantArtifact   `json:"artifacts,omitempty"`
	Usage        Usage                                `json:"usage"`
	FirstTokenMs *int64                               `json:"firstTokenMs,omitempty"`
	Disclosures  []Disclosure                         `json:"disclosures,omitempty"`
	Models       []string                             `json:"models,omitempty"`
	Unattributed []CallSummary                        `json:"unattributedModelCalls,omitempty"`
	CaptureError string                               `json:"captureError,omitempty"`
}

func (t *TurnRecord) Duration() time.Duration {
	return t.FinishedAt.Sub(t.StartedAt)
}

type CallSummary struct {
	Seq             int                           `json:"seq"`
	Kind            ModelCallKind                 `json:"kind"`
	File            string                        `json:"file,omitempty"`
	Model           string                        `json:"model,omitempty"`
	Provider        string                        `json:"provider,omitempty"`
	Delegate        string                        `json:"delegateCallId,omitempty"`
	Messages        int                           `json:"messages"`
	Tools           int                           `json:"tools"`
	SystemDigest    string                        `json:"systemDigest,omitempty"`
	ToolsDigest     string                        `json:"toolsDigest,omitempty"`
	InputTokens     int                           `json:"inputTokens"`
	CacheRead       int                           `json:"cacheReadTokens,omitempty"`
	CacheWrite      int                           `json:"cacheWriteTokens,omitempty"`
	OutputTokens    int                           `json:"outputTokens"`
	ReasoningTokens int                           `json:"reasoningTokens"`
	Advisor         *aipricing.AdvisorUsage       `json:"advisor,omitempty"`
	CostUSD         *decimal.Decimal              `json:"costUsd,omitempty"`
	PriceSource     PriceSource                   `json:"priceSource,omitempty"`
	Reask           string                        `json:"reask,omitempty"`
	FirstTokenMs    *int64                        `json:"firstTokenMs,omitempty"`
	LatencyMs       int64                         `json:"latencyMs"`
	Truncated       bool                          `json:"truncated,omitempty"`
	CutOffCall      string                        `json:"cutOffCall,omitempty"`
	Text            string                        `json:"text,omitempty"`
	Thinking        string                        `json:"thinking,omitempty"`
	ToolCalls       []llmports.ToolCall           `json:"toolCalls,omitempty"`
	FallbackFrom    *llmports.ChatProviderFailure `json:"fallbackFrom,omitempty"`
	Retries         []llmports.ChatRetryNotice    `json:"retries,omitempty"`
	Error           string                        `json:"error,omitempty"`
}

type DecisionRecord struct {
	ProposalID     pulid.ID                         `json:"proposalId,omitempty"`
	PlanID         pulid.ID                         `json:"planId,omitempty"`
	Tool           string                           `json:"tool,omitempty"`
	Decision       agentdecision.DecisionType       `json:"decision"`
	Note           string                           `json:"note,omitempty"`
	Error          string                           `json:"error,omitempty"`
	StatusAfter    agent.ProposalStatus             `json:"statusAfter,omitempty"`
	ExecutionError string                           `json:"executionError,omitempty"`
	Executed       bool                             `json:"executed"`
	Result         *agentrecord.ToolExecutionResult `json:"executionResult,omitempty"`
}
