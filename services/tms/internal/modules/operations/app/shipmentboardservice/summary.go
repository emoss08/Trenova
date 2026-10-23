package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func (s *Service) scope(
	ctx context.Context,
	req *shipmentports.ShipmentBoardScopeRequest,
	needAll bool,
) (*shipmentports.ShipmentBoardScope, error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewBusinessError("A shipment board scope is required")
	}

	multiErr := errortypes.NewMultiError()
	shipment.ValidateQuickFilters(req.QuickFilters, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err := shipment.AuthorizeQuickFilters(req.QuickFilters, req.Margins); err != nil {
		return nil, err
	}

	margin, detention := shipment.QuickFiltersNeed(req.QuickFilters)
	basis, err := s.quickFilters.Resolve(ctx, &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: req.Filter.TenantInfo,
		Timezone:   req.Timezone,
		Margin:     margin || (needAll && req.Margins),
		Detention:  detention || needAll,
	})
	if err != nil {
		return nil, err
	}

	return &shipmentports.ShipmentBoardScope{
		Filter: req.Filter,
		Options: shipmentports.ShipmentOptions{
			QuickFilters:     req.QuickFilters,
			Timezone:         req.Timezone,
			QuickFilterBasis: basis,
		},
	}, nil
}

func (s *Service) StageSummary(
	ctx context.Context,
	req *shipmentports.ShipmentBoardScopeRequest,
) ([]*shipmentports.ShipmentStageSummary, error) {
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

func (s *Service) GroupSummary(
	ctx context.Context,
	req *shipmentports.SummarizeShipmentBoardGroupsRequest,
) ([]*shipmentports.ShipmentBoardGroup, error) {
	if req == nil || !req.GroupBy.IsValid() {
		return nil, errortypes.NewValidationError(
			"groupBy",
			errortypes.ErrInvalid,
			"The board cannot be grouped that way",
		)
	}

	scope, err := s.scope(ctx, req.Scope, false)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.GroupSummary(ctx, &shipmentports.SummarizeShipmentBoardGroupsQuery{
		Scope:    scope,
		GroupBy:  req.GroupBy,
		Timezone: req.Scope.Timezone,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*shipmentports.ShipmentBoardGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, &shipmentports.ShipmentBoardGroup{
			Key:     row.Key,
			Label:   row.Label,
			Count:   row.Count,
			Revenue: row.Revenue,
		})
	}

	return out, nil
}

func ZeroFilledStageSummary(
	rows []*shipmentports.ShipmentStageSummaryRow,
) []*shipmentports.ShipmentStageSummary {
	byRank := make(map[int16]*shipmentports.ShipmentStageSummaryRow, len(rows))
	for _, row := range rows {
		byRank[row.StageRank] = row
	}

	stages := shipment.Stages()
	out := make([]*shipmentports.ShipmentStageSummary, 0, len(stages))
	for _, stage := range stages {
		summary := &shipmentports.ShipmentStageSummary{
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
	req *shipmentports.ShipmentBoardScopeRequest,
) ([]*shipmentports.ShipmentQuickFilterCount, error) {
	scope, err := s.scope(ctx, req, true)
	if err != nil {
		return nil, err
	}

	filters := make([]shipment.QuickFilter, 0, len(shipment.CountableQuickFilters()))
	for _, filter := range shipment.CountableQuickFilters() {
		if filter.NeedsMargin() && !req.Margins {
			continue
		}
		filters = append(filters, filter)
	}
	specs := make([]shipment.QuickFilterSpec, 0, len(filters))
	for _, filter := range filters {
		specs = append(specs, shipment.Quick(filter))
	}

	totals, err := s.repo.QuickFilterTotals(ctx, &shipmentports.CountShipmentQuickFiltersRequest{
		Scope:   scope,
		Filters: specs,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*shipmentports.ShipmentQuickFilterCount, 0, len(filters))
	for i, filter := range filters {
		out = append(
			out,
			&shipmentports.ShipmentQuickFilterCount{Filter: filter, Count: totals[i].Count},
		)
	}

	return out, nil
}
