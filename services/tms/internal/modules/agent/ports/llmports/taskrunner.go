package llmports

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NovaTask is one job Nova does in the background: what to do, the shape of
// the answer, and what to read, including files the model reads as themselves.
// It is not a conversation; the Desk has its own runtime.
type NovaTask struct {
	Name         aitask.Task
	Feature      aiusage.Feature
	Instructions string
	SchemaName   string
	Schema       map[string]any
	Context      DelimitedContext
	Files        []MessageFile
	MaxTokens    int
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	// Attribution overrides the usage record when a task bills against more
	// than a user and its feature; Feature and UserID fill it otherwise.
	Attribution *aiusageports.AIUsageAttribution
	// Reasoning overrides how hard Nova's model thinks for this task; empty
	// keeps Nova's own.
	Reasoning aireasoning.ReasoningEffort
	// Reads are tools the task may call before it answers. The answer is then
	// itself a call, so a task that needs no read still answers in one model
	// call; MaxCalls bounds them, and the last answers with what was read.
	// Reads run only on a worker; on a request path the task answers from its
	// context alone.
	Reads    []NovaTool
	MaxCalls int
}

// ErrOutsideSchema is a task whose answer did not decode into the shape it
// asked for; the result is still returned so the caller can record the call.
var ErrOutsideSchema = errors.New("nova answered outside its schema")

// DecodeAnswer reads a task's answer into out, failing with ErrOutsideSchema
// when the answer is not the shape the task asked for.
func DecodeAnswer(task aitask.Task, text string, out any) error {
	if out == nil {
		return nil
	}
	if err := sonic.UnmarshalString(text, out); err != nil {
		return fmt.Errorf("%w (%s): %w", ErrOutsideSchema, task, err)
	}

	return nil
}

// TaskRunner runs Nova's background tasks with Nova's identity in front of
// each task's instructions, on the provider the organization chose for Nova
// and with Nova's own model. The agent module's harness implements it.
type TaskRunner interface {
	// Available reports whether Nova can run for the tenant: AI is on and the
	// organization has chosen an enabled provider for Nova.
	Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	// Run does the task and decodes its answer into out, returning what the
	// call cost and which model answered.
	Run(ctx context.Context, task *NovaTask, out any) (*StructuredCompletionResult, error)
}
