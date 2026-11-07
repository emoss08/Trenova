package aidocumentservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// structuredCall is one schema-constrained request, in the terms the router
// takes rather than the terms one vendor's HTTP API took.
type structuredCall struct {
	tenant     pagination.TenantInfo
	documentID pulid.ID
	feature    aiusage.Feature
	task       aitask.Task
	metric     string
	system     string
	schemaName string
	context    llmports.DelimitedContext
	schema     map[string]any
}

func (c *structuredCall) novaTask(maxTokens int) *llmports.NovaTask {
	attribution := documentAttribution(c.tenant, c.documentID, c.feature)

	return &llmports.NovaTask{
		Name:         c.task,
		Feature:      c.feature,
		Instructions: c.system,
		SchemaName:   c.schemaName,
		Schema:       c.schema,
		Context:      c.context,
		MaxTokens:    maxTokens,
		TenantInfo:   c.tenant,
		UserID:       c.tenant.UserID,
		Attribution:  &attribution,
	}
}

func documentAttribution(
	tenant pagination.TenantInfo,
	documentID pulid.ID,
	feature aiusage.Feature,
) aiusageports.AIUsageAttribution {
	attribution := aiusageports.AIUsageAttribution{
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
) (*llmports.StructuredCompletionResult, error) {
	if err := s.admit(ctx, call); err != nil {
		return nil, err
	}

	result, err := s.nova.Run(ctx, call.novaTask(s.cfg.GetExtractionMaxTokens()), out)
	if errors.Is(err, llmports.ErrOutsideSchema) {
		s.recordAIUsage(call.metric, false, "invalid_output")
		return nil, errors.Join(llmports.ErrModelSchemaValidation, err)
	}
	if err != nil {
		s.recordAIUsage(call.metric, false, failureOutcome(err))
		return nil, err
	}

	return result, nil
}

// decodeStructured reads a deferred extraction's answer, which comes back on a
// submission or a poll rather than from Run.
func decodeStructured(text string, out any) error {
	if err := llmports.DecodeAnswer(aitask.TaskDocumentExtraction, text, out); err != nil {
		return errors.Join(llmports.ErrModelSchemaValidation, err)
	}

	return nil
}

// failureOutcome keeps the metric label stable across the move to the router.
// "missing_config" meant the OpenAI integration was not set up; it now means no
// provider is assigned to the task, which is the same operational condition and
// the same thing to do about it.
func failureOutcome(err error) string {
	switch {
	case errors.Is(err, llmports.ErrNoProviderConfigured):
		return "missing_config"
	case errors.Is(err, llmports.ErrModelSchemaValidation),
		errors.Is(err, llmports.ErrOutsideSchema):
		return "invalid_output"
	default:
		return "error"
	}
}

func (s *Service) recordAIUsage(operation string, success bool, outcome string) {
	s.metrics.Document.RecordAIOutcome(operation, success, outcome)
}

func (s *Service) admit(ctx context.Context, call *structuredCall) error {
	if err := s.requireDocumentIntelligence(ctx, call.tenant); err != nil {
		return err
	}
	if call.feature != aiusage.FeatureDocumentIntelligenceExtract {
		return nil
	}

	return quotaservice.Preflight(ctx, s.quota, &platformplanports.QuotaRequest{
		TenantInfo: call.tenant,
		Meter:      platformcatalog.MeterDocumentAIExtractions,
		Quantity:   1,
	})
}

func (s *Service) requireDocumentIntelligence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityDocumentIntelligence,
	)
}
