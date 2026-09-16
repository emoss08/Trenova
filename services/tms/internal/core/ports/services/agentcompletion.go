package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrModelSchemaValidation = errors.New("model output failed schema validation")
	// ErrNoProviderConfigured reports that no enabled provider is assigned to the
	// requested task. It is distinct from a call failure: nothing was attempted.
	ErrNoProviderConfigured = errors.New("no AI provider is configured for this task")
)

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
	ProviderID      pulid.ID
	ProviderKind    aiprovider.Kind
}

type StructuredCompletionRequest struct {
	TenantInfo pagination.TenantInfo
	// Task selects which configured providers may serve this call. Leaving it
	// empty routes to the general-purpose pool.
	Task         aiprovider.Task
	System       string
	Context      DelimitedContext
	OutputSchema map[string]any
	// SchemaName labels the schema for the protocols that require one.
	SchemaName string
	MaxTokens  int
}

type StructuredCompletionResult struct {
	Text            string
	ModelIdentifier string
	InputTokens     int
	OutputTokens    int
	// ProviderID and ProviderKind attribute the call to the endpoint that served
	// it, which is what makes per-provider spend and reliability reportable once
	// several are configured.
	ProviderID   pulid.ID
	ProviderKind aiprovider.Kind
}

type CompletionService interface {
	Diagnose(ctx context.Context, req *DiagnoseRequest) (*DiagnoseResult, error)
	CompleteStructured(
		ctx context.Context,
		req *StructuredCompletionRequest,
	) (*StructuredCompletionResult, error)
}
