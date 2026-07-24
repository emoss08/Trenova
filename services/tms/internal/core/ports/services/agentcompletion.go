package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
)

var ErrModelSchemaValidation = errors.New("model output failed schema validation")

type ContextSection struct {
	Title   string
	Trusted bool
	Content string
}

type DelimitedContext struct {
	Sections []ContextSection
}

type ProposedAction struct {
	ToolName   string              `json:"toolName"`
	ToolParams map[string]any      `json:"toolParams"`
	Confidence float64             `json:"confidence"`
	Rationale  string              `json:"rationale"`
	Evidence   []agent.EvidenceRef `json:"evidence"`
}

type RaisedException struct {
	Category       string              `json:"category"`
	Severity       string              `json:"severity"`
	AttemptSummary string              `json:"attemptSummary"`
	Evidence       []agent.EvidenceRef `json:"evidence"`
	BlastRadius    int                 `json:"blastRadius"`
}

type DiagnoseRequest struct {
	TenantInfo    pagination.TenantInfo
	PromptVersion string
	SystemPrompt  string
	Context       DelimitedContext
	ToolSchemas   []AgentToolDescriptor
}

type DiagnoseResult struct {
	Proposals       []ProposedAction
	Exceptions      []RaisedException
	ModelIdentifier string
}

type CompletionService interface {
	Diagnose(ctx context.Context, req *DiagnoseRequest) (*DiagnoseResult, error)
}
