package extractionrolloutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const reportSampleLimit = 5000

func (s *Service) Report(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*services.ExtractionRolloutReport, error) {
	rollout, err := s.Get(ctx, tenant)
	if err != nil {
		return nil, err
	}
	report := &services.ExtractionRolloutReport{
		Rollout:              rollout,
		MinGuardScoredFields: extractionrollout.MinGuardScoredFields,
		MinGuardExtractions:  extractionrollout.MinGuardExtractions,
	}
	if rollout.ProviderID == nil || rollout.StartedAt == nil {
		return report, nil
	}

	candidate := *rollout.ProviderID
	totals, err := s.assignments.Totals(ctx, repositories.TotalRolloutAssignmentsRequest{
		TenantInfo:          tenant,
		CandidateProviderID: candidate,
		Since:               *rollout.StartedAt,
	})
	if err != nil {
		return nil, err
	}
	t := tallyAssignments(totals)
	report.Candidate = t.candidate
	report.Control = t.control

	split, err := s.accuracy(ctx, tenant, rollout)
	if err != nil {
		return nil, err
	}
	report.CandidateAccuracy = accuracyOf(split.candidate)
	report.ProductionAccuracy = accuracyOf(split.production)

	if err = s.addFields(ctx, report, tenant, rollout); err != nil {
		return nil, err
	}
	report.ProviderName = s.providerName(ctx, tenant, candidate)

	return report, nil
}

func accuracyOf(side extractionrollout.ArmAccuracy) services.ExtractionRolloutAccuracy {
	return services.ExtractionRolloutAccuracy{
		Scored:   side.Scored,
		Correct:  side.Correct,
		Accuracy: side.Rate(),
	}
}

func (s *Service) addFields(
	ctx context.Context,
	report *services.ExtractionRolloutReport,
	tenant pagination.TenantInfo,
	rollout *extractionrollout.ExtractionRollout,
) error {
	corrections, err := s.corrections.ListForAccuracy(
		ctx,
		repositories.ListAICorrectionsForAccuracyRequest{
			TenantInfo: tenant,
			Task:       aicorrection.TaskShipmentDraftExtraction,
			Since:      *rollout.StartedAt,
			Limit:      reportSampleLimit,
		},
	)
	if err != nil {
		return err
	}

	candidate := aicorrection.NewAccuracyAggregator()
	production := aicorrection.NewAccuracyAggregator()
	for _, correction := range corrections {
		switch {
		case correction.ExtractionProviderID == nil:
			continue
		case *correction.ExtractionProviderID == *rollout.ProviderID:
			candidate.Add(correction.FieldResults)
		default:
			production.Add(correction.FieldResults)
		}
	}
	report.Fields = aicorrection.CompareFields(candidate.Fields(), production.Fields())
	report.Truncated = len(corrections) >= reportSampleLimit

	return nil
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
				"failed to read rollout provider name",
				zap.String("providerId", providerID.String()),
				zap.Error(err),
			)
		}
		return ""
	}

	return provider.Name
}
