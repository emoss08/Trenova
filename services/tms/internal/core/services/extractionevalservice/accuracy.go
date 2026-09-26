package extractionevalservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultAccuracyWindowDays = 30
	maxAccuracyWindowDays     = 365
	accuracySampleLimit       = 5000
	recentRunLimit            = 10
)

type groupTotals struct {
	corrections int
	scored      int
	correct     int
}

func (s *Service) Accuracy(
	ctx context.Context,
	req *services.ExtractionAccuracyRequest,
) (*services.ExtractionAccuracy, error) {
	window := req.WindowDays
	if window == 0 {
		window = defaultAccuracyWindowDays
	}
	if window < 1 || window > maxAccuracyWindowDays {
		return nil, errortypes.NewValidationError(
			"windowDays",
			errortypes.ErrInvalid,
			"The window must be between 1 and 365 days",
		)
	}

	task := aicorrection.TaskShipmentDraftExtraction
	since := s.now() - int64(window)*timeutils.SecondsPerDay
	corrections, err := s.corrections.ListForAccuracy(ctx, repositories.ListAICorrectionsForAccuracyRequest{
		TenantInfo: req.TenantInfo,
		Task:       task,
		Since:      since,
		Limit:      accuracySampleLimit,
	})
	if err != nil {
		return nil, err
	}

	counts, err := s.cases.Counts(ctx, repositories.CountExtractionEvalCasesRequest{
		TenantInfo: req.TenantInfo,
		Task:       task,
	})
	if err != nil {
		return nil, err
	}

	runs, err := s.runs.ListRecentFinished(ctx, req.TenantInfo, task, recentRunLimit)
	if err != nil {
		return nil, err
	}

	report := &services.ExtractionAccuracy{
		WindowDays:  window,
		Since:       since,
		Corrections: len(corrections),
		Sampled:     len(corrections) >= accuracySampleLimit,
		Cases:       *counts,
		RecentRuns:  runs,
	}

	aggregator := aicorrection.NewAccuracyAggregator()
	byModel := map[string]*groupTotals{}
	byKind := map[string]*groupTotals{}
	for _, correction := range corrections {
		aggregator.Add(correction.FieldResults)
		report.Scored += correction.ScoredCount
		report.Correct += correction.CorrectCount
		report.Corrected += correction.CorrectedCount
		report.Missed += correction.MissedCount
		report.Unconfirmed += correction.UnconfirmedCount
		addGroup(byModel, correction.ExtractionModel, correction)
		addGroup(byKind, correction.DocumentKind, correction)
	}
	report.Accuracy = aicorrection.Accuracy(report.Correct, report.Scored)
	report.Fields = aggregator.Fields()
	report.ByModel = groups(byModel)
	report.ByKind = groups(byKind)

	return report, nil
}

func addGroup(totals map[string]*groupTotals, key string, correction *aicorrection.Correction) {
	entry, ok := totals[key]
	if !ok {
		entry = &groupTotals{}
		totals[key] = entry
	}
	entry.corrections++
	entry.scored += correction.ScoredCount
	entry.correct += correction.CorrectCount
}

func groups(totals map[string]*groupTotals) []services.ExtractionAccuracyGroup {
	out := make([]services.ExtractionAccuracyGroup, 0, len(totals))
	for key, entry := range totals {
		out = append(out, services.ExtractionAccuracyGroup{
			Key:         key,
			Corrections: entry.corrections,
			Scored:      entry.scored,
			Correct:     entry.correct,
			Accuracy:    aicorrection.Accuracy(entry.correct, entry.scored),
		})
	}
	slices.SortFunc(out, func(a, b services.ExtractionAccuracyGroup) int {
		if c := cmp.Compare(b.Corrections, a.Corrections); c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return out
}
