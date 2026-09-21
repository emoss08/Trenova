package aidocumentservice

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// structuredCall is one schema-constrained request, in the terms the router
// takes rather than the terms one vendor's HTTP API took.
type structuredCall struct {
	tenant     pagination.TenantInfo
	documentID pulid.ID
	operation  ailog.Operation
	task       aiprovider.Task
	metric     string
	system     string
	schemaName string
	context    serviceports.DelimitedContext
	schema     map[string]any
}

func (c *structuredCall) request() *serviceports.StructuredCompletionRequest {
	return &serviceports.StructuredCompletionRequest{
		TenantInfo:   c.tenant,
		Task:         c.task,
		System:       c.system,
		Context:      c.context,
		OutputSchema: c.schema,
		SchemaName:   c.schemaName,
	}
}

func (s *Service) runStructured(
	ctx context.Context,
	call *structuredCall,
	out any,
) (*serviceports.StructuredCompletionResult, error) {
	request := call.request()
	request.MaxTokens = s.cfg.GetExtractionMaxTokens()

	result, err := s.completion.CompleteStructured(ctx, request)
	if err != nil {
		s.recordAIUsage(call.metric, false, failureOutcome(err))
		return nil, err
	}

	if err = decodeStructured(result.Text, out); err != nil {
		s.recordAIUsage(call.metric, false, "invalid_output")
		return nil, err
	}

	s.logInteraction(ctx, call, result)

	return result, nil
}

// decodeStructured reads the model's reply. The router already rejects a reply
// that will not decode against the requested schema and falls through to the
// next provider, so reaching a decode error here means the shape parsed for the
// router and not for us.
func decodeStructured(text string, out any) error {
	if out == nil {
		return nil
	}

	if err := sonic.Unmarshal([]byte(text), out); err != nil {
		return errors.Join(serviceports.ErrModelSchemaValidation, err)
	}

	return nil
}

// failureOutcome keeps the metric label stable across the move to the router.
// "missing_config" meant the OpenAI integration was not set up; it now means no
// provider is assigned to the task, which is the same operational condition and
// the same thing to do about it.
func failureOutcome(err error) string {
	switch {
	case errors.Is(err, serviceports.ErrNoProviderConfigured):
		return "missing_config"
	case errors.Is(err, serviceports.ErrModelSchemaValidation):
		return "invalid_output"
	default:
		return "error"
	}
}

func (s *Service) logInteraction(
	ctx context.Context,
	call *structuredCall,
	result *serviceports.StructuredCompletionResult,
) {
	// The model is recorded as the provider actually reported it. It used to be
	// the configured OpenAI model name, which was the same string every time and
	// told a reader nothing once more than one provider can serve a task.
	entry := &ailog.Log{
		OrganizationID:   call.tenant.OrgID,
		BusinessUnitID:   call.tenant.BuID,
		UserID:           call.tenant.UserID,
		Prompt:           redactPrompt(call.system, promptText(call.context)),
		Response:         redactResponse(result.Text),
		Model:            ailog.Model(result.ModelIdentifier),
		Operation:        call.operation,
		Object:           call.documentID.String(),
		PromptTokens:     result.InputTokens,
		CompletionTokens: result.OutputTokens,
		TotalTokens:      result.InputTokens + result.OutputTokens,
	}

	if _, err := s.aiLogRepo.Create(ctx, entry); err != nil {
		s.logger.Warn("failed to persist ai log", zap.Error(err))
	}
}

// promptText flattens the delimited context for the log. The router does its own
// fencing on the way out; this is only the record of what was sent.
func promptText(deliminated serviceports.DelimitedContext) string {
	var text string
	for _, section := range deliminated.Sections {
		text += section.Title + "\n" + section.Content + "\n\n"
	}

	return text
}

func (s *Service) recordAIUsage(operation string, success bool, outcome string) {
	s.metrics.Document.RecordAIOutcome(operation, success, outcome)
}
