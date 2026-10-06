package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

const maxFacetsPerRequest = 8

func (s *Service) scope(
	ctx context.Context,
	req *services.ShipmentBoardScopeRequest,
	needAll bool,
) (*repositories.ShipmentBoardScope, error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewBusinessError("A shipment board scope is required")
	}

	multiErr := errortypes.NewMultiError()
	shipment.ValidateQuickFilters(req.QuickFilters, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	margin, detention := shipment.QuickFiltersNeed(req.QuickFilters)
	basis, err := s.quickFilters.Resolve(ctx, &services.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: req.Filter.TenantInfo,
		Timezone:   req.Timezone,
		Margin:     margin || needAll,
		Detention:  detention || needAll,
	})
	if err != nil {
		return nil, err
	}

	return &repositories.ShipmentBoardScope{
		Filter: req.Filter,
		Options: repositories.ShipmentOptions{
			QuickFilters:     req.QuickFilters,
			Timezone:         req.Timezone,
			QuickFilterBasis: basis,
		},
	}, nil
}

func (s *Service) StageSummary(
	ctx context.Context,
	req *services.ShipmentBoardScopeRequest,
) ([]*services.ShipmentStageSummary, error) {
	scope, err := s.scope(ctx, req, false)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.StageSummary(ctx, scope)
	if err != nil {
		return nil, err
	}

	return ZeroFilledStageSummary(rows), nil
}

func ZeroFilledStageSummary(
	rows []*repositories.ShipmentStageSummaryRow,
) []*services.ShipmentStageSummary {
	byRank := make(map[int16]*repositories.ShipmentStageSummaryRow, len(rows))
	for _, row := range rows {
		byRank[row.StageRank] = row
	}

	stages := shipment.Stages()
	out := make([]*services.ShipmentStageSummary, 0, len(stages))
	for _, stage := range stages {
		summary := &services.ShipmentStageSummary{
			Stage:   stage,
			Rank:    int(stage.Rank()),
			Revenue: decimal.Zero,
		}
		if row, ok := byRank[stage.Rank()]; ok {
			summary.Count = row.Count
			summary.Revenue = row.Revenue
		}
		out = append(out, summary)
	}

	return out
}

func (s *Service) QuickFilterCounts(
	ctx context.Context,
	req *services.ShipmentBoardScopeRequest,
) ([]*services.ShipmentQuickFilterCount, error) {
	scope, err := s.scope(ctx, req, true)
	if err != nil {
		return nil, err
	}

	filters := shipment.CountableQuickFilters()
	specs := make([]shipment.QuickFilterSpec, 0, len(filters))
	for _, filter := range filters {
		specs = append(specs, shipment.Quick(filter))
	}

	counts, err := s.repo.QuickFilterCounts(ctx, &repositories.CountShipmentQuickFiltersRequest{
		Scope:   scope,
		Filters: specs,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*services.ShipmentQuickFilterCount, 0, len(filters))
	for _, filter := range filters {
		out = append(out, &services.ShipmentQuickFilterCount{Filter: filter, Count: counts[filter]})
	}

	return out, nil
}

func (s *Service) FacetCounts(
	ctx context.Context,
	req *services.ShipmentBoardScopeRequest,
	facets []repositories.ShipmentFacet,
) ([]*repositories.ShipmentFacetCounts, error) {
	if len(facets) > maxFacetsPerRequest {
		return nil, errortypes.NewValidationError(
			"facets",
			errortypes.ErrInvalid,
			"Too many facets requested",
		)
	}
	for i, facet := range facets {
		if !facet.IsValid() {
			return nil, errortypes.NewValidationError(
				"facets",
				errortypes.ErrInvalid,
				"Unknown facet at position {0}",
				i,
			)
		}
	}

	scope, err := s.scope(ctx, req, false)
	if err != nil {
		return nil, err
	}

	out := make([]*repositories.ShipmentFacetCounts, 0, len(facets))
	for _, facet := range facets {
		counts, fErr := s.repo.FacetCounts(ctx, &repositories.CountShipmentFacetRequest{
			Scope: scope,
			Facet: facet,
		})
		if fErr != nil {
			return nil, fErr
		}
		out = append(out, counts)
	}

	return out, nil
}
