package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/pagination"
)

type DraftAgentRequest struct {
	TenantInfo  pagination.TenantInfo
	Description string
}

type TightenAgentInstructionsRequest struct {
	TenantInfo   pagination.TenantInfo
	Instructions string
}

type AgentDraftNote struct {
	Field  string `json:"field"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

type AgentDraft struct {
	Name                   string                            `json:"name"`
	Description            string                            `json:"description"`
	Icon                   string                            `json:"icon"`
	Accent                 string                            `json:"accent"`
	Instructions           string                            `json:"instructions"`
	Guardrails             []string                          `json:"guardrails"`
	TriggerMode            agentdefinition.TriggerMode       `json:"triggerMode"`
	CronExpression         string                            `json:"cronExpression"`
	CronTimezone           string                            `json:"cronTimezone"`
	EventKinds             []agent.EventKind                 `json:"eventKinds"`
	IntervalSeconds        int                               `json:"intervalSeconds"`
	ToolNames              []string                          `json:"toolNames"`
	ToolTiers              map[string]agent.AutonomyTier     `json:"toolTiers"`
	AutonomyCeiling        agent.AutonomyTier                `json:"autonomyCeiling"`
	DataAccessCeiling      agentdefinition.DataAccessCeiling `json:"dataAccessCeiling"`
	OutputMode             agentdefinition.OutputMode        `json:"outputMode"`
	Enabled                bool                              `json:"enabled"`
	ShadowMode             bool                              `json:"shadowMode"`
	DecisionTimeoutSeconds int                               `json:"decisionTimeoutSeconds"`
	RunTimeoutSeconds      int                               `json:"runTimeoutSeconds"`
	MaxToolCalls           int                               `json:"maxToolCalls"`
	MaxConcurrentRuns      int                               `json:"maxConcurrentRuns"`
	Notes                  []AgentDraftNote                  `json:"notes"`
}

type TightenedAgentInstructions struct {
	Instructions string `json:"instructions"`
	Changed      bool   `json:"changed"`
}

type AgentDraftingService interface {
	Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	DraftFromDescription(ctx context.Context, req *DraftAgentRequest) (*AgentDraft, error)
	TightenInstructions(
		ctx context.Context,
		req *TightenAgentInstructionsRequest,
	) (*TightenedAgentInstructions, error)
}
