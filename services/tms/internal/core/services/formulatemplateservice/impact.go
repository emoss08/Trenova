package formulatemplateservice

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula/effectiveversioncache"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const impactDefaultLimit = 100

type ApprovalImpactRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	Limit      int
}

// ApprovalImpact answers the question a reviewer actually has before
// approving: what would this template's pending content have done to the
// shipments it already priced? Each shipment is rated twice — once with the
// version that was effective when it shipped, once with the row as it stands
// — and the deltas are returned biggest-movers-first.
func (s *Service) ApprovalImpact(
	ctx context.Context,
	req *ApprovalImpactRequest,
) (*BacktestResponse, error) {
	log := s.l.With(
		zap.String("operation", "ApprovalImpact"),
		zap.String("templateID", req.TemplateID.String()),
	)

	if req.Limit > backtestMaxLimit {
		return nil, errortypes.NewValidationError(
			"limit",
			errortypes.ErrInvalid,
			"Limit cannot exceed 500",
		)
	}

	ctx = ratetablecache.With(ctx)
	ctx = effectiveversioncache.With(ctx)

	limit := req.Limit
	if limit <= 0 {
		limit = impactDefaultLimit
	}

	template, err := s.getTemplateByIDWithTenant(ctx, req.TemplateID, req.TenantInfo)
	if err != nil {
		log.Error("failed to get formula template", zap.Error(err))
		return nil, err
	}

	shipments, err := s.shipmentRepo.ListRatedByFormulaTemplate(
		ctx,
		&repositories.ListRatedByFormulaTemplateRequest{
			TemplateID: req.TemplateID,
			TenantInfo: req.TenantInfo,
			Limit:      limit,
		},
	)
	if err != nil {
		log.Error("failed to list rated shipments", zap.Error(err))
		return nil, err
	}

	results := make([]*BacktestResult, 0, len(shipments))
	for _, entity := range shipments {
		results = append(
			results,
			s.backtestShipment(ctx, template, template, entity, req.TenantInfo),
		)
	}

	sortResultsByImpact(results)

	return &BacktestResponse{
		Results: results,
		Summary: buildBacktestSummary(results),
	}, nil
}

// sortResultsByImpact puts the biggest movers first and shipments that failed
// to evaluate last, so a truncated view still shows what matters.
func sortResultsByImpact(results []*BacktestResult) {
	sort.SliceStable(results, func(i, j int) bool {
		iFailed := results[i].CurrentError != "" || results[i].CandidateError != ""
		jFailed := results[j].CurrentError != "" || results[j].CandidateError != ""
		if iFailed != jFailed {
			return jFailed
		}
		return results[i].Delta.Abs().GreaterThan(results[j].Delta.Abs())
	})
}
