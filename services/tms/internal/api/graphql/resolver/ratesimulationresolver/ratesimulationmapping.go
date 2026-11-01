package ratesimulationresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultSimulationPage = 20
	defaultResultPage     = 50
)

func boundedFirst(first *int, fallback int) int {
	if first == nil {
		return fallback
	}

	return min(max(*first, 1), pagination.MaxLimit)
}

func rateSimulationToModel(entity *ratesimulation.RateSimulation) *gqlmodel.RateSimulation {
	model := &gqlmodel.RateSimulation{
		ID:              entity.ID.String(),
		BusinessUnitID:  entity.BusinessUnitID.String(),
		OrganizationID:  entity.OrganizationID.String(),
		RateAgreementID: entity.RateAgreementID.String(),
		Name:            entity.Name,
		Description:     entity.Description,
		Status:          entity.Status,
		PartyType:       gqlmodel.RateAgreementPartyType(entity.PartyType),
		SampleFrom:      int(entity.SampleFrom),
		SampleTo:        int(entity.SampleTo),
		SampleLimit:     entity.SampleLimit,
		RuleCoverage:    make([]*gqlmodel.RateSimulationRuleCoverage, 0, len(entity.RuleCoverage)),
		Error:           entity.Error,
		StartedAt:       gqlscalar.Int64PtrToIntPtr(entity.StartedAt),
		CompletedAt:     gqlscalar.Int64PtrToIntPtr(entity.CompletedAt),
		RequestedBy:     gqlid.IDPtrFromPulidPtr(entity.RequestedBy),
		WorkflowID:      entity.WorkflowID,
		Version:         int(entity.Version),
		CreatedAt:       int(entity.CreatedAt),
		UpdatedAt:       int(entity.UpdatedAt),
	}
	if summary := entity.Summary; summary != nil {
		model.Summary = &gqlmodel.RateSimulationSummary{
			ShipmentCount:  summary.ShipmentCount,
			EvaluatedCount: summary.EvaluatedCount,
			ChangedCount:   summary.ChangedCount,
			IncreasedCount: summary.IncreasedCount,
			DecreasedCount: summary.DecreasedCount,
			ErrorCount:     summary.ErrorCount,
			BeforeTotal:    summary.BeforeTotal.String(),
			AfterTotal:     summary.AfterTotal.String(),
			TotalDelta:     summary.TotalDelta.String(),
			TotalDeltaPct:  summary.TotalDeltaPct.String(),
			MaxIncrease:    summary.MaxIncrease.String(),
			MaxDecrease:    summary.MaxDecrease.String(),
		}
	}
	for _, coverage := range entity.RuleCoverage {
		if coverage == nil {
			continue
		}
		model.RuleCoverage = append(model.RuleCoverage, &gqlmodel.RateSimulationRuleCoverage{
			RuleID:      coverage.RuleID.String(),
			Label:       coverage.Label,
			LaneKey:     coverage.LaneKey,
			Outcome:     coverage.Outcome,
			WonCount:    coverage.WonCount,
			LostCount:   coverage.LostCount,
			LostTo:      gqlid.IDPtrFromPulidPtr(coverage.LostTo),
			LostToLabel: coverage.LostToLabel,
		})
	}

	return model
}

func rateSimulationResultsToModel(
	results []*ratesimulation.RateSimulationResult,
) []*gqlmodel.RateSimulationResult {
	out := make([]*gqlmodel.RateSimulationResult, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		out = append(out, &gqlmodel.RateSimulationResult{
			ID:               result.ID.String(),
			RateSimulationID: result.RateSimulationID.String(),
			ShipmentID:       result.ShipmentID.String(),
			ProNumber:        result.ProNumber,
			CustomerID:       gqlid.IDPtrFromPulidPtr(result.CustomerID),
			LaneKey:          result.LaneKey,
			EquipmentTypeID:  gqlid.IDPtrFromPulidPtr(result.EquipmentTypeID),
			BeforeAmount:     result.BeforeAmount.String(),
			AfterAmount:      result.AfterAmount.String(),
			Delta:            result.Delta.String(),
			DeltaPercent:     result.DeltaPercent.String(),
			Outcome:          gqlmodel.RateQuoteOutcome(result.Outcome),
			BeforeRuleID:     gqlid.IDPtrFromPulidPtr(result.BeforeRuleID),
			AfterRuleID:      gqlid.IDPtrFromPulidPtr(result.AfterRuleID),
			Error:            result.Error,
			CreatedAt:        int(result.CreatedAt),
		})
	}

	return out
}
