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

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Config     *config.Config
	Metrics    *metrics.Registry
	Completion serviceports.CompletionService
	AILogRepo  repositories.AILogRepository
}

type Service struct {
	logger     *zap.Logger
	cfg        *config.AIConfig
	metrics    *metrics.Registry
	completion serviceports.CompletionService
	aiLogRepo  repositories.AILogRepository
}

func New(p Params) serviceports.AIDocumentService {
	return &Service{
		logger:     p.Logger.Named("service.ai-document"),
		cfg:        p.Config.GetAIConfig(),
		metrics:    p.Metrics,
		completion: p.Completion,
		aiLogRepo:  p.AILogRepo,
	}
}

var errDisabled = errortypes.NewBusinessError("AI document intelligence is disabled")

func (s *Service) RouteDocument(
	ctx context.Context,
	req *serviceports.AIRouteRequest,
) (*serviceports.AIRouteResult, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	var parsed routeResponse
	if _, err := s.runStructured(ctx, &structuredCall{
		tenant:     req.TenantInfo,
		documentID: req.DocumentID,
		operation:  ailog.OperationDocumentIntelligenceRoute,
		task:       aiprovider.TaskDocumentClassification,
		metric:     "route",
		system:     routeSystemPrompt,
		schemaName: schemaNameRoute,
		context:    buildRouteContext(req),
		schema:     buildRouteSchema(),
	}, &parsed); err != nil {
		return nil, err
	}

	s.recordAIUsage("route", true, "success")

	return &serviceports.AIRouteResult{
		ShouldExtract:       parsed.ShouldExtract,
		DocumentKind:        strings.TrimSpace(parsed.DocumentKind),
		Confidence:          clampAIConfidence(parsed.Confidence),
		Signals:             parsed.Signals,
		ReviewStatus:        normalizeReviewStatus(parsed.ReviewStatus),
		ClassifierSource:    strings.TrimSpace(parsed.ClassifierSource),
		ProviderFingerprint: strings.TrimSpace(parsed.ProviderFingerprint),
		Reason:              strings.TrimSpace(parsed.Reason),
	}, nil
}

func (s *Service) ExtractRateConfirmation(
	ctx context.Context,
	req *serviceports.AIExtractRequest,
) (*serviceports.AIExtractResult, error) {
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

func (s *Service) extractCall(req *serviceports.AIExtractRequest) *structuredCall {
	return &structuredCall{
		tenant:     req.TenantInfo,
		documentID: req.DocumentID,
		operation:  ailog.OperationDocumentIntelligenceExtract,
		task:       aiprovider.TaskDocumentExtraction,
		metric:     "extract",
		system:     extractSystemPrompt,
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
it. Where no configured provider can defer — an install running only Ollama, for
instance — the router runs the call inline and returns the answer on the
submission. That is why ExtractResult exists on the submission: the alternative
was for those installs to lose document extraction entirely over a protocol
feature they do not have.
*/
func (s *Service) SubmitRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *serviceports.AIExtractRequest,
) (*serviceports.AIBackgroundExtractSubmission, error) {
	if !s.cfg.DocumentExtractionEnabled() {
		return nil, errDisabled
	}

	call := s.extractCall(req)
	call.metric = "extract_background_submit"

	request := call.request()
	request.MaxTokens = s.cfg.GetExtractionMaxTokens()

	submission, err := s.completion.SubmitBackground(ctx, request)
	if err != nil {
		s.recordAIUsage(call.metric, false, failureOutcome(err))
		return nil, err
	}

	result := &serviceports.AIBackgroundExtractSubmission{
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

	s.logInteraction(ctx, call, submission.Result)
	s.recordAIUsage(call.metric, true, "inline")

	result.Model = submission.Result.ModelIdentifier
	result.Status = string(serviceports.AIBackgroundExtractionStatusCompleted)
	result.ExtractResult = convertExtractResponse(parsed)

	return result, nil
}

func (s *Service) PollRateConfirmationBackgroundExtraction(
	ctx context.Context,
	req *serviceports.AIBackgroundExtractPollRequest,
) (*serviceports.AIBackgroundExtractPollResult, error) {
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

	outcome, err := s.completion.PollBackground(ctx, &serviceports.BackgroundPollRequest{
		TenantInfo: req.TenantInfo,
		ProviderID: req.ProviderID,
		Handle:     req.ResponseID,
	})
	if err != nil {
		s.recordAIUsage("extract_background_poll", false, failureOutcome(err))
		return nil, err
	}

	return s.describePoll(ctx, req, outcome), nil
}

func (s *Service) describePoll(
	ctx context.Context,
	req *serviceports.AIBackgroundExtractPollRequest,
	outcome *serviceports.BackgroundOutcome,
) *serviceports.AIBackgroundExtractPollResult {
	result := &serviceports.AIBackgroundExtractPollResult{
		ResponseID: req.ResponseID,
		Model:      outcome.ModelIdentifier,
		RawStatus:  strings.TrimSpace(outcome.RawStatus),
	}

	switch outcome.State {
	case serviceports.BackgroundPending:
		result.Status = serviceports.AIBackgroundExtractionStatusPending
		s.recordAIUsage("extract_background_poll", true, "pending")

		return result
	case serviceports.BackgroundFailed:
		result.Status = serviceports.AIBackgroundExtractionStatusFailed
		result.FailureCode = firstNonEmpty(outcome.FailureCode, result.RawStatus, "failed")
		result.FailureMessage = firstNonEmpty(
			outcome.FailureMessage,
			"The AI provider ended the extraction without a result",
		)
		s.recordAIUsage("extract_background_poll", true, "terminal_failure")

		return result
	case serviceports.BackgroundCompleted:
	}

	if outcome.Result == nil || strings.TrimSpace(outcome.Result.Text) == "" {
		result.Status = serviceports.AIBackgroundExtractionStatusFailed
		result.FailureCode = "empty_output"
		result.FailureMessage = "AI background extraction completed without structured output"
		s.recordAIUsage("extract_background_poll", false, "empty_output")

		return result
	}

	parsed := new(extractResponse)
	if err := decodeStructured(outcome.Result.Text, parsed); err != nil {
		result.Status = serviceports.AIBackgroundExtractionStatusFailed
		result.FailureCode = "invalid_output"
		result.FailureMessage = err.Error()
		s.recordAIUsage("extract_background_poll", false, "invalid_output")

		return result
	}

	s.logInteraction(ctx, &structuredCall{
		tenant:     req.TenantInfo,
		documentID: req.DocumentID,
		operation:  ailog.OperationDocumentIntelligenceExtract,
		system:     extractSystemPrompt,
	}, outcome.Result)

	result.Status = serviceports.AIBackgroundExtractionStatusCompleted
	result.ExtractResult = convertExtractResponse(parsed)
	s.recordAIUsage("extract_background_poll", true, "completed")

	return result
}
