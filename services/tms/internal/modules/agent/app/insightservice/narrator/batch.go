package narrator

import (
	"context"
	"errors"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// ErrBatchUnavailable is a deployment whose narration provider has no batch
// API; the caller narrates the findings one request at a time instead.
var ErrBatchUnavailable = errors.New("insight narration cannot be batched here")

// Submit sends the findings to the batch API, eight to a request, so a sweep
// across every organization pays the batch rate for wording nobody is waiting
// on.
func (s *Service) Submit(
	ctx context.Context,
	req *NarrateRequest,
) (*llmports.BatchHandle, error) {
	items := make([]llmports.NovaBatchItem, 0, len(req.Findings)/maxFindingsPerCall+1)
	for batch := range chunk(req.Findings, maxFindingsPerCall) {
		items = append(items, llmports.NovaBatchItem{
			CustomID: "narration-" + strconv.Itoa(len(items)),
			Task:     narrationTask(req.TenantInfo, batch, req.WindowDays),
		})
	}

	handle, err := s.batches.SubmitTasks(ctx, &llmports.NovaBatch{
		TenantInfo: req.TenantInfo,
		Task:       aitask.TaskOperationalInsights,
		Items:      items,
	})
	if errors.Is(err, llmports.ErrBatchUnsupported) ||
		errors.Is(err, llmports.ErrNoProviderConfigured) {
		return nil, ErrBatchUnavailable
	}

	return handle, err
}

// Ended reports whether the provider has finished the batch.
func (s *Service) Ended(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	handle *llmports.BatchHandle,
) (bool, error) {
	status, err := s.batches.BatchStatus(ctx, &llmports.BatchStatusRequest{
		TenantInfo: tenantInfo,
		ProviderID: handle.ProviderID,
		BatchID:    handle.BatchID,
	})
	if err != nil {
		return false, err
	}

	return status.Ended, nil
}

// Collect reads an ended batch and returns the wording that survived the
// number guard, keyed by dedupe key. A finding the batch said nothing about,
// or whose request failed, is left out and keeps its own wording.
func (s *Service) Collect(
	ctx context.Context,
	req *NarrateRequest,
	handle *llmports.BatchHandle,
) (map[string]Narration, error) {
	attributions := make(map[string]aiusageports.AIUsageAttribution, len(handle.CustomIDs))
	for _, custom := range handle.CustomIDs {
		attributions[custom] = aiusageports.AIUsageAttribution{
			Feature: aiusage.FeatureInsightNarration,
		}
	}
	results, err := s.batches.BatchResults(ctx, &llmports.BatchResultsRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aitask.TaskOperationalInsights,
		ProviderID:   handle.ProviderID,
		BatchID:      handle.BatchID,
		SubmittedAt:  handle.SubmittedAt,
		Attributions: attributions,
	})
	if err != nil {
		return nil, err
	}

	into := make(map[string]Narration, len(req.Findings))
	for custom, item := range results {
		if !item.Succeeded() {
			s.l.Warn("an insight narration request in a batch failed",
				zap.String("request", custom),
				zap.String("code", item.FailureCode),
				zap.String("message", item.FailureMessage),
			)

			continue
		}

		var envelope narrationEnvelope
		if err = llmports.DecodeAnswer(
			aitask.TaskOperationalInsights,
			item.Result.Text,
			&envelope,
		); err != nil {
			s.l.Warn("insight narration could not be parsed",
				zap.String("model", item.Result.ModelIdentifier),
				zap.Error(err),
			)

			continue
		}
		s.applyNarrations(applyParams{
			batch:      findingsNamed(req.Findings, envelope.byKey()),
			narrations: envelope.byKey(),
			completion: item.Result,
			windowDays: req.WindowDays,
			into:       into,
		})
	}

	return into, nil
}

func findingsNamed(
	findings []detector.Finding,
	narrations map[string]narratedFinding,
) []detector.Finding {
	named := make([]detector.Finding, 0, len(narrations))
	for _, finding := range findings {
		if _, ok := narrations[finding.DedupeKey]; ok {
			named = append(named, finding)
		}
	}

	return named
}
