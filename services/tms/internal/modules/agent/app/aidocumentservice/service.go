// Package aidocumentservice classifies and extracts transportation documents.
//
// It holds the prompts, the output schemas and the shaping of a result; it
// holds no HTTP client and no vendor SDK. Which endpoint serves a call, in what
// protocol, with which key, is the completion router's decision — so an install
// running a local model gets document intelligence on the same terms as one
// paying a frontier vendor, and neither is written into this package.
package aidocumentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	AIConfig *aiconfig.Config
	Metrics  *metrics.Registry
	Nova     llmports.TaskRunner
	Deferrer llmports.TaskDeferrer
	Plans    platformplanports.PlanService
	Quota    platformplanports.QuotaGuard
}

type Service struct {
	logger   *zap.Logger
	cfg      *aiconfig.Config
	metrics  *metrics.Registry
	nova     llmports.TaskRunner
	deferrer llmports.TaskDeferrer
	plans    platformplanports.PlanService
	quota    platformplanports.QuotaGuard
}

func New(p Params) documentaiextractionports.AIDocumentService {
	return &Service{
		logger:   p.Logger.Named("service.ai-document"),
		cfg:      p.AIConfig,
		metrics:  p.Metrics,
		nova:     p.Nova,
		deferrer: p.Deferrer,
		plans:    p.Plans,
		quota:    quotaservice.OrUnlimited(p.Quota),
	}
}

func (s *Service) ExtractionAvailability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (documentaiextractionports.ExtractionAvailability, error) {
	if !s.cfg.AIEnabled() || !s.cfg.DocumentExtractionEnabled() {
		return documentaiextractionports.ExtractionOffOnServer, nil
	}

	allowed, err := planservice.Allows(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityDocumentIntelligence,
	)
	if err != nil {
		return "", err
	}
	if !allowed {
		return documentaiextractionports.ExtractionNotInPlan, nil
	}

	ready, err := s.nova.Available(ctx, tenantInfo)
	if err != nil {
		return "", err
	}
	if !ready {
		return documentaiextractionports.ExtractionNoProviderAssigned, nil
	}

	return documentaiextractionports.ExtractionAvailable, nil
}

var errDisabled = errortypes.NewBusinessError("AI document intelligence is disabled")

func (s *Service) RouteDocument(
	ctx context.Context,
	req *documentaiextractionports.AIRouteRequest,
) (*documentaiextractionports.AIRouteResult, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	var parsed routeResponse
	if _, err := s.runStructured(ctx, &structuredCall{
		tenant:     req.TenantInfo,
		documentID: req.DocumentID,
		feature:    aiusage.FeatureDocumentIntelligenceRoute,
		task:       aitask.TaskDocumentClassification,
		metric:     "route",
		system:     routePrompt(req.Pages),
		schemaName: schemaNameRoute,
		context:    buildRouteContext(req),
		schema:     buildRouteSchema(),
	}, &parsed); err != nil {
		return nil, err
	}

	s.recordAIUsage("route", true, "success")

	return &documentaiextractionports.AIRouteResult{
		ShouldExtract:       parsed.ShouldExtract,
		DocumentKind:        strings.TrimSpace(parsed.DocumentKind),
		Confidence:          clampAIConfidence(parsed.Confidence),
		Signals:             parsed.Signals,
		ReviewStatus:        normalizeReviewStatus(parsed.ReviewStatus),
		ClassifierSource:    strings.TrimSpace(parsed.ClassifierSource),
		ProviderFingerprint: strings.TrimSpace(parsed.ProviderFingerprint),
		Reason:              strings.TrimSpace(parsed.Reason),
		Signature:           convertSignatureReading(parsed.Signature),
	}, nil
}

func (s *Service) ExtractRateConfirmation(
	ctx context.Context,
	req *documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIExtractResult, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	parsed := new(extractResponse)
	if _, err := s.runStructured(ctx, s.extractCall(req), parsed); err != nil {
		return nil, err
	}

	s.recordAIUsage("extract", true, "success")

	return convertExtractResponse(parsed), nil
}

func (s *Service) extractCall(req *documentaiextractionports.AIExtractRequest) *structuredCall {
	return newExtractCall(req)
}

func newExtractCall(req *documentaiextractionports.AIExtractRequest) *structuredCall {
	return &structuredCall{
		tenant:     req.TenantInfo,
		documentID: req.DocumentID,
		feature:    aiusage.FeatureDocumentIntelligenceExtract,
		task:       aitask.TaskDocumentExtraction,
		metric:     "extract",
		system:     extractPrompt(req.Pages),
		schemaName: schemaNameExtract,
		context:    buildExtractContext(req),
		schema:     buildExtractSchema(),
	}
}

/*
SubmitRateConfirmationBackgroundExtraction hands the extraction to whichever
configured provider can defer it.

A full rate confirmation can take minutes, which is longer than an activity
should hold a heartbeat open, so the work is deferred where the protocol allows
it. Where no configured provider can defer — an install running only a
self-hosted OpenAI-compatible server, for instance — the router runs the call
inline and returns the answer on the submission. That is why ExtractResult
exists on the submission: the alternative was for those installs to lose
document extraction entirely over a protocol feature they do not have.
*/
func (s *Service) SubmitRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIBackgroundExtractSubmission, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	call := s.extractCall(req)
	call.metric = "extract_background_submit"
	if err := s.admit(ctx, call); err != nil {
		return nil, err
	}

	submission, err := s.deferrer.Defer(ctx, call.novaTask(s.cfg.GetExtractionMaxTokens()))
	if err != nil {
		s.recordAIUsage(call.metric, false, failureOutcome(err))
		return nil, err
	}

	result := &documentaiextractionports.AIBackgroundExtractSubmission{
		ResponseID: submission.Handle,
		ProviderID: submission.ProviderID,
		Model:      submission.ModelIdentifier,
		Status:     strings.TrimSpace(submission.RawStatus),
	}

	if submission.Handle != "" {
		s.recordAIUsage(call.metric, true, "success")
		return result, nil
	}

	if submission.Result == nil {
		s.recordAIUsage(call.metric, false, "empty_response_id")
		return nil, errortypes.NewBusinessError(
			"The AI provider neither deferred the extraction nor returned a result",
		)
	}

	parsed := new(extractResponse)
	if err = decodeStructured(submission.Result.Text, parsed); err != nil {
		s.recordAIUsage(call.metric, false, "invalid_output")
		return nil, err
	}

	s.recordAIUsage(call.metric, true, "inline")

	result.Model = submission.Result.ModelIdentifier
	result.Status = string(documentaiextractionports.AIBackgroundExtractionStatusCompleted)
	result.ExtractResult = convertExtractResponse(parsed)

	return result, nil
}

func (s *Service) PollRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *documentaiextractionports.AIBackgroundExtractPollRequest,
) (*documentaiextractionports.AIBackgroundExtractPollResult, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	// A handle belongs to the endpoint that issued it. Without a provider there
	// is nothing safe to ask — polling a different one would at best 404, so the
	// poll fails plainly rather than guessing.
	if req.ProviderID.IsNil() {
		s.recordAIUsage("extract_background_poll", false, "missing_provider")
		return nil, errortypes.NewBusinessError(
			"This extraction has no AI provider recorded and cannot be checked — run it again",
		)
	}

	outcome, err := s.deferrer.Collect(ctx, &llmports.BackgroundPollRequest{
		TenantInfo:  req.TenantInfo,
		ProviderID:  req.ProviderID,
		Handle:      req.ResponseID,
		Task:        aitask.TaskDocumentExtraction,
		SubmittedAt: req.SubmittedAt,
		Attribution: documentAttribution(
			req.TenantInfo,
			req.DocumentID,
			aiusage.FeatureDocumentIntelligenceExtract,
		),
	})
	if err != nil {
		s.recordAIUsage("extract_background_poll", false, failureOutcome(err))
		return nil, err
	}

	return s.describePoll(req, outcome), nil
}

func (s *Service) describePoll(
	req *documentaiextractionports.AIBackgroundExtractPollRequest,
	outcome *llmports.BackgroundOutcome,
) *documentaiextractionports.AIBackgroundExtractPollResult {
	result := &documentaiextractionports.AIBackgroundExtractPollResult{
		ResponseID: req.ResponseID,
		Model:      outcome.ModelIdentifier,
		RawStatus:  strings.TrimSpace(outcome.RawStatus),
	}

	switch outcome.State {
	case llmports.BackgroundPending:
		result.Status = documentaiextractionports.AIBackgroundExtractionStatusPending
		s.recordAIUsage("extract_background_poll", true, "pending")

		return result
	case llmports.BackgroundFailed:
		result.Status = documentaiextractionports.AIBackgroundExtractionStatusFailed
		result.FailureCode = stringutils.FirstNonEmptyTrimmed(
			outcome.FailureCode,
			result.RawStatus,
			"failed",
		)
		result.FailureMessage = stringutils.FirstNonEmptyTrimmed(
			outcome.FailureMessage,
			"The AI provider ended the extraction without a result",
		)
		s.recordAIUsage("extract_background_poll", true, "terminal_failure")

		return result
	case llmports.BackgroundCompleted:
	}

	if outcome.Result == nil || strings.TrimSpace(outcome.Result.Text) == "" {
		result.Status = documentaiextractionports.AIBackgroundExtractionStatusFailed
		result.FailureCode = "empty_output"
		result.FailureMessage = "AI background extraction completed without structured output"
		s.recordAIUsage("extract_background_poll", false, "empty_output")

		return result
	}

	parsed := new(extractResponse)
	if err := decodeStructured(outcome.Result.Text, parsed); err != nil {
		result.Status = documentaiextractionports.AIBackgroundExtractionStatusFailed
		result.FailureCode = "invalid_output"
		result.FailureMessage = err.Error()
		s.recordAIUsage("extract_background_poll", false, "invalid_output")

		return result
	}

	result.Status = documentaiextractionports.AIBackgroundExtractionStatusCompleted
	result.ExtractResult = convertExtractResponse(parsed)
	s.recordAIUsage("extract_background_poll", true, "completed")

	return result
}
