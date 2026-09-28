package extractionshadowservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	defaultReportWindowDays = 30
	maxReportWindowDays     = 365
	reportSampleLimit       = 5000
)

func (s *Service) Report(
	ctx context.Context,
	req *services.ExtractionShadowReportRequest,
) (*services.ExtractionShadowReport, error) {
	window := req.WindowDays
	if window == 0 {
		window = defaultReportWindowDays
	}
	if window < 1 || window > maxReportWindowDays {
		return nil, errortypes.NewValidationError(
			"windowDays",
			errortypes.ErrInvalid,
			"The window must be between 1 and 365 days",
		)
	}

	providerID := req.ProviderID
	if providerID.IsNil() {
		settings, err := s.GetSettings(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		if settings.ProviderID != nil {
			providerID = *settings.ProviderID
		}
	}

	since := s.now() - int64(window)*timeutils.SecondsPerDay
	report := &services.ExtractionShadowReport{
		WindowDays: window,
		Since:      since,
		ProviderID: pulid.PtrOrNil(providerID),
		CostUSD:    decimal.Zero,
	}
	if providerID.IsNil() {
		return report, nil
	}

	if err := s.addTotals(ctx, report, req.TenantInfo, providerID, since); err != nil {
		return nil, err
	}

	scored, err := s.results.ListScored(ctx, &repositories.ListScoredExtractionShadowResultsRequest{
		TenantInfo: req.TenantInfo,
		ProviderID: providerID,
		Since:      since,
		Limit:      reportSampleLimit,
	})
	if err != nil {
		return nil, err
	}
	addScored(report, scored)
	report.Truncated = len(scored) >= reportSampleLimit
	if report.ProviderName == "" {
		report.ProviderName = s.providerName(ctx, req.TenantInfo, providerID)
	}

	return report, nil
}

func (s *Service) addTotals(
	ctx context.Context,
	report *services.ExtractionShadowReport,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	since int64,
) error {
	totals, err := s.results.TotalsByStatus(ctx, repositories.TotalExtractionShadowResultsRequest{
		TenantInfo: tenant,
		ProviderID: providerID,
		Since:      since,
	})
	if err != nil {
		return err
	}

	var latency int64
	var timed int
	for _, total := range totals {
		report.Sampled += total.Count
		report.CostUSD = report.CostUSD.Add(total.CostUSD)
		latency += total.LatencyMsSum
		timed += total.Timed
		switch total.Status {
		case extractionshadow.ResultStatusPending:
			report.Pending += total.Count
		case extractionshadow.ResultStatusCompleted:
			report.Completed += total.Count
		case extractionshadow.ResultStatusFailed:
			report.Failed += total.Count
		case extractionshadow.ResultStatusSkipped:
			report.Skipped += total.Count
		}
	}
	if timed > 0 {
		report.AvgLatencyMs = latency / int64(timed)
	}

	return nil
}

func addScored(report *services.ExtractionShadowReport, scored []*extractionshadow.ShadowResult) {
	candidate := aicorrection.NewAccuracyAggregator()
	production := aicorrection.NewAccuracyAggregator()
	for _, result := range scored {
		if report.ProviderName == "" {
			report.ProviderName = result.ProviderName
		}
		report.Scored++
		switch result.Verdict {
		case extractionshadow.VerdictBetter:
			report.Better++
		case extractionshadow.VerdictWorse:
			report.Worse++
		case extractionshadow.VerdictSame:
			report.Same++
		}
		addSide(
			&report.Candidate,
			result.ScoredCount,
			result.CorrectCount,
			result.CorrectedCount,
			result.MissedCount,
		)
		addSide(
			&report.Production,
			result.BaselineScoredCount,
			result.BaselineCorrectCount,
			result.BaselineCorrectedCount,
			result.BaselineMissedCount,
		)
		candidate.Add(result.FieldResults)
		production.Add(result.BaselineFieldResults)
	}

	report.Candidate.Accuracy = aicorrection.Accuracy(
		report.Candidate.Correct,
		report.Candidate.Scored,
	)
	report.Production.Accuracy = aicorrection.Accuracy(
		report.Production.Correct,
		report.Production.Scored,
	)
	report.Fields = compareFields(candidate.Fields(), production.Fields())
}

func addSide(side *services.ExtractionShadowSide, scored, correct, corrected, missed int) {
	side.Scored += scored
	side.Correct += correct
	side.Corrected += corrected
	side.Missed += missed
}

func compareFields(
	candidate, production []aicorrection.FieldAccuracy,
) []services.ExtractionShadowFieldComparison {
	byKey := make(
		map[string]*services.ExtractionShadowFieldComparison,
		len(candidate)+len(production),
	)
	entry := func(key string) *services.ExtractionShadowFieldComparison {
		row, ok := byKey[key]
		if !ok {
			row = &services.ExtractionShadowFieldComparison{Key: key}
			byKey[key] = row
		}
		return row
	}
	for i := range candidate {
		row := entry(candidate[i].Key)
		row.CandidateScored = candidate[i].Scored
		row.CandidateCorrect = candidate[i].Correct
		row.CandidateAccuracy = candidate[i].Accuracy
	}
	for i := range production {
		row := entry(production[i].Key)
		row.ProductionScored = production[i].Scored
		row.ProductionCorrect = production[i].Correct
		row.ProductionAccuracy = production[i].Accuracy
	}

	out := make([]services.ExtractionShadowFieldComparison, 0, len(byKey))
	for _, row := range byKey {
		if row.CandidateScored == 0 && row.ProductionScored == 0 {
			continue
		}
		out = append(out, *row)
	}
	slices.SortFunc(out, func(a, b services.ExtractionShadowFieldComparison) int {
		if c := cmp.Compare(
			a.CandidateAccuracy-a.ProductionAccuracy,
			b.CandidateAccuracy-b.ProductionAccuracy,
		); c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})

	return out
}

func (s *Service) providerName(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
) string {
	provider, err := s.providers.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn(
				"failed to read shadow provider name",
				zap.String("providerId", providerID.String()),
				zap.Error(err),
			)
		}
		return ""
	}

	return provider.Name
}

func (s *Service) ListResults(
	ctx context.Context,
	req *repositories.ListExtractionShadowResultConnectionRequest,
) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
	return s.results.ListConnection(ctx, req)
}

func (s *Service) GetResult(
	ctx context.Context,
	req repositories.GetExtractionShadowResultRequest,
) (*extractionshadow.ShadowResult, error) {
	return s.results.GetByID(ctx, req)
}
