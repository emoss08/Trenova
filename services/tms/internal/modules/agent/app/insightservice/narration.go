package insightservice

import (
	"context"
	"errors"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/narrator"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// PendingNarration is a deferred refresh's wording still being written: the
// batch writing it and the stored findings it words.
type PendingNarration struct {
	Handle     llmports.BatchHandle `json:"handle"`
	InsightIDs []pulid.ID           `json:"insightIds"`
	WindowDays int                  `json:"windowDays"`
}

// carriedNarrations keeps the wording of every finding whose figures have not
// moved since a model wrote it, and returns the findings that still need
// words. Six-hourly refreshes of an unchanged operation then cost nothing.
func (s *Service) carriedNarrations(
	ctx context.Context,
	p refreshParams,
	findings []detector.Finding,
) (map[string]narrator.Narration, []detector.Finding) {
	narrations := make(map[string]narrator.Narration, len(findings))
	if len(findings) == 0 {
		return narrations, nil
	}

	worded, err := s.repo.ListNarrated(ctx, insightports.ListNarratedInsightsRequest{
		TenantInfo:  p.params.TenantInfo,
		DetectorKey: p.detector.Key(),
	})
	if err != nil {
		s.l.Warn("could not read earlier insight wording; narrating afresh",
			zap.String("detector", p.detector.Key()), zap.Error(err))

		return narrations, findings
	}

	byKey := make(map[string]*insight.Insight, len(worded))
	for _, entity := range worded {
		byKey[entity.DedupeKey] = entity
	}

	fresh := make([]detector.Finding, 0, len(findings))
	for idx := range findings {
		finding := &findings[idx]
		previous := byKey[finding.DedupeKey]
		if previous == nil || !sameFigures(previous, finding) {
			fresh = append(fresh, *finding)

			continue
		}
		narrations[finding.DedupeKey] = narrator.Narration{
			Headline:        previous.Headline,
			Narrative:       previous.Narrative,
			Recommendation:  previous.Recommendation,
			Narrated:        true,
			ModelIdentifier: previous.ModelIdentifier,
			ProviderID:      previous.ProviderID,
		}
	}

	return narrations, fresh
}

// sameFigures reports whether a stored finding says what a new one says. A
// wording checked against these figures stays true while they hold.
func sameFigures(previous *insight.Insight, finding *detector.Finding) bool {
	if previous.Subject != finding.Subject || previous.Severity != finding.Severity {
		return false
	}
	before := figures(previous.Metrics, previous.Links)

	return before != "" && before == figures(finding.Metrics, finding.Links)
}

func figures(metrics []insight.Metric, links []insight.Link) string {
	encoded, err := sonic.MarshalString(struct {
		Metrics []insight.Metric `json:"metrics"`
		Links   []insight.Link   `json:"links"`
	}{metrics, links})
	if err != nil {
		return ""
	}

	return encoded
}

func unworded(entities []*insight.Insight) []*insight.Insight {
	out := make([]*insight.Insight, 0, len(entities))
	for _, entity := range entities {
		if !entity.Narrated && entity.ID.IsNotNil() {
			out = append(out, entity)
		}
	}

	return out
}

func findingsOf(entities []*insight.Insight) []detector.Finding {
	findings := make([]detector.Finding, 0, len(entities))
	for _, entity := range entities {
		findings = append(findings, detector.Finding{
			DedupeKey: entity.DedupeKey,
			Subject:   entity.Subject,
			Headline:  entity.Headline,
			Severity:  entity.Severity,
			Metrics:   entity.Metrics,
			Links:     entity.Links,
		})
	}

	return findings
}

// deferNarration sends the refresh's unworded findings to the batch API. A
// provider with no batch API words them now, one request at a time.
func (s *Service) deferNarration(
	ctx context.Context,
	params detector.Params,
	entities []*insight.Insight,
	result *RefreshResult,
) *PendingNarration {
	if len(entities) == 0 {
		return nil
	}

	req := &narrator.NarrateRequest{
		TenantInfo: params.TenantInfo,
		Findings:   findingsOf(entities),
		WindowDays: params.WindowDays(),
	}
	handle, err := s.narrator.Submit(ctx, req)
	switch {
	case errors.Is(err, narrator.ErrBatchUnavailable):
		narrations := s.narrator.Narrate(ctx, req)
		worded, storeErr := s.storeNarrations(ctx, params.TenantInfo, entities, narrations)
		if storeErr != nil {
			s.l.Warn("could not store insight wording", zap.Error(storeErr))
		}
		result.Narrated += worded

		return nil
	case err != nil:
		s.l.Warn("could not send insight narration to the batch API; findings keep their own wording",
			zap.String("organization", params.TenantInfo.OrgID.String()),
			zap.Error(err))

		return nil
	}

	ids := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.ID)
	}

	return &PendingNarration{Handle: *handle, InsightIDs: ids, WindowDays: req.WindowDays}
}

// NarrationEnded reports whether a deferred refresh's batch has finished.
func (s *Service) NarrationEnded(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pending *PendingNarration,
) (bool, error) {
	return s.narrator.Ended(ctx, tenantInfo, &pending.Handle)
}

// ApplyNarration words the findings of an ended batch that are still active
// and still unworded, and returns how many it worded.
func (s *Service) ApplyNarration(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pending *PendingNarration,
) (int, error) {
	stored, err := s.repo.GetByIDs(ctx, insightports.GetInsightsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        pending.InsightIDs,
	})
	if err != nil {
		return 0, err
	}

	waiting := make([]*insight.Insight, 0, len(stored))
	for _, entity := range stored {
		if entity.Status == insight.StatusActive && !entity.Narrated {
			waiting = append(waiting, entity)
		}
	}
	if len(waiting) == 0 {
		return 0, nil
	}

	narrations, err := s.narrator.Collect(ctx, &narrator.NarrateRequest{
		TenantInfo: tenantInfo,
		Findings:   findingsOf(waiting),
		WindowDays: pending.WindowDays,
	}, &pending.Handle)
	if err != nil {
		return 0, err
	}

	return s.storeNarrations(ctx, tenantInfo, waiting, narrations)
}

func (s *Service) storeNarrations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entities []*insight.Insight,
	narrations map[string]narrator.Narration,
) (int, error) {
	updates := make([]insightports.InsightNarration, 0, len(entities))
	worded := make([]*insight.Insight, 0, len(entities))
	for _, entity := range entities {
		narration, ok := narrations[entity.DedupeKey]
		if !ok || !narration.Narrated {
			continue
		}
		updates = append(updates, insightports.InsightNarration{
			ID:              entity.ID,
			Headline:        narration.Headline,
			Narrative:       narration.Narrative,
			Recommendation:  narration.Recommendation,
			ModelIdentifier: narration.ModelIdentifier,
			ProviderID:      narration.ProviderID,
		})
		entity.Headline = narration.Headline
		entity.Narrative = narration.Narrative
		entity.Recommendation = narration.Recommendation
		entity.Narrated = true
		entity.ModelIdentifier = narration.ModelIdentifier
		entity.ProviderID = narration.ProviderID
		worded = append(worded, entity)
	}

	applied, err := s.repo.ApplyNarrations(ctx, insightports.ApplyInsightNarrationsRequest{
		TenantInfo: tenantInfo,
		Narrations: updates,
	})
	if err != nil {
		return 0, err
	}
	s.project(ctx, worded)

	return applied, nil
}
