package aidocumentservice

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// structuredCall is one schema-constrained request, in the terms the router
// takes rather than the terms one vendor's HTTP API took.
type structuredCall struct {
	tenant     pagination.TenantInfo
	documentID pulid.ID
	feature    aiusage.Feature
	task       aiprovider.Task
	metric     string
	system     string
	schemaName string
	context    serviceports.DelimitedContext
	schema     map[string]any
	providerID pulid.ID
	evaluation bool
}

func (c *structuredCall) request() *serviceports.StructuredCompletionRequest {
	request := &serviceports.StructuredCompletionRequest{
		TenantInfo:   c.tenant,
		Task:         c.task,
		System:       c.system,
		Context:      c.context,
		OutputSchema: c.schema,
		SchemaName:   c.schemaName,
		Attribution:  documentAttribution(c.tenant, c.documentID, c.feature),
	}
	if c.evaluation {
		request.PreferredProviderID = c.providerID
		request.RequireProvider = true
		request.Attribution.Purpose = serviceports.AIUsagePurposeEvaluation
	}

	return request
}

func documentAttribution(
	tenant pagination.TenantInfo,
	documentID pulid.ID,
	feature aiusage.Feature,
) serviceports.AIUsageAttribution {
	attribution := serviceports.AIUsageAttribution{
		UserID:  tenant.UserID,
		Feature: feature,
	}
	if documentID.IsNotNil() {
		attribution.Subject = aiusage.Subject{
			Type: aiusage.SubjectTypeDocument,
			ID:   documentID.String(),
		}
	}

	return attribution
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

func (s *Service) recordAIUsage(operation string, success bool, outcome string) {
	s.metrics.Document.RecordAIOutcome(operation, success, outcome)
}
