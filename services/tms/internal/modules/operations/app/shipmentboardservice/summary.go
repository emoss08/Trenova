package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func (s *Service) scope(
	ctx context.Context,
	req *shipmentboardports.ShipmentBoardScopeRequest,
	needAll bool,
) (*shipmentboardports.ShipmentBoardScope, error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewBusinessError("A shipment board scope is required")
	}

	multiErr := errortypes.NewMultiError()
	quickfilter.ValidateQuickFilters(req.QuickFilters, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err := quickfilter.AuthorizeQuickFilters(req.QuickFilters, req.Margins); err != nil {
		return nil, err
	}

	margin, detention := quickfilter.QuickFiltersNeed(req.QuickFilters)
	basis, err := s.quickFilters.Resolve(ctx, &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		TenantInfo: req.Filter.TenantInfo,
		Timezone:   req.Timezone,
		Margin:     margin || (needAll && req.Margins),
		Detention:  detention || needAll,
	})
	if err != nil {
		return nil, err
	}

	return &shipmentboardports.ShipmentBoardScope{
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
	req *shipmentboardports.ShipmentBoardScopeRequest,
) ([]*shipmentboardports.ShipmentStageSummary, error) {
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
	req *shipmentboardports.SummarizeShipmentBoardGroupsRequest,
) ([]*shipmentboardports.ShipmentBoardGroup, error) {
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

	rows, err := s.repo.GroupSummary(ctx, &shipmentboardports.SummarizeShipmentBoardGroupsQuery{
		Scope:    scope,
		GroupBy:  req.GroupBy,
		Timezone: req.Scope.Timezone,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*shipmentboardports.ShipmentBoardGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, &shipmentboardports.ShipmentBoardGroup{
			Key:     row.Key,
			Label:   row.Label,
			Count:   row.Count,
			Revenue: row.Revenue,
		})
	}

	return out, nil
}

func ZeroFilledStageSummary(
	rows []*shipmentboardports.ShipmentStageSummaryRow,
) []*shipmentboardports.ShipmentStageSummary {
	byRank := make(map[int16]*shipmentboardports.ShipmentStageSummaryRow, len(rows))
	for _, row := range rows {
		byRank[row.StageRank] = row
	}

	stages := shipmentvocab.Stages()
	out := make([]*shipmentboardports.ShipmentStageSummary, 0, len(stages))
	for _, stage := range stages {
		summary := &shipmentboardports.ShipmentStageSummary{
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
	req *shipmentboardports.ShipmentBoardScopeRequest,
) ([]*shipmentboardports.ShipmentQuickFilterCount, error) {
	scope, err := s.scope(ctx, req, true)
	if err != nil {
		return nil, err
	}

	filters := make([]quickfilter.QuickFilter, 0, len(quickfilter.CountableQuickFilters()))
	for _, filter := range quickfilter.CountableQuickFilters() {
		if filter.NeedsMargin() && !req.Margins {
			continue
		}
		filters = append(filters, filter)
	}
	specs := make([]quickfilter.QuickFilterSpec, 0, len(filters))
	for _, filter := range filters {
		specs = append(specs, quickfilter.Quick(filter))
	}

	totals, err := s.repo.QuickFilterTotals(
		ctx,
		&shipmentboardports.CountShipmentQuickFiltersRequest{
			Scope:   scope,
			Filters: specs,
		},
	)
	if err != nil {
		return nil, err
	}

	out := make([]*shipmentboardports.ShipmentQuickFilterCount, 0, len(filters))
	for i, filter := range filters {
		out = append(
			out,
			&shipmentboardports.ShipmentQuickFilterCount{Filter: filter, Count: totals[i].Count},
		)
	}

	return out, nil
}
