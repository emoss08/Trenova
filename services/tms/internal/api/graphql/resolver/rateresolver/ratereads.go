package rateresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	// maxMatrixCellPage is the most cells one read of a matrix's grid returns.
	maxMatrixCellPage = pagination.MaxLimit
	// defaultAgreementLanes is how many lanes an agreement's card reads.
	defaultAgreementLanes = 50
)

func (r *RateAgreementResolver) firstLanes(
	ctx context.Context,
	obj *gqlmodel.RateAgreement,
	first int,
) ([]*rateagreement.RateAgreementRule, error) {
	if _, err := r.RequirePermission(
		ctx,
		permission.ResourceRateAgreement,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	return loaders.RateAgreementLanes(ctx, obj.ID, first)
}

func (r *RateAgreementResolver) laneCount(
	ctx context.Context,
	obj *gqlmodel.RateAgreement,
) (rateagreementports.LaneCount, error) {
	if _, err := r.RequirePermission(
		ctx,
		permission.ResourceRateAgreement,
		permission.OpRead,
	); err != nil {
		return rateagreementports.LaneCount{}, err
	}

	return loaders.RateAgreementLaneCount(ctx, obj.ID)
}

type cellPageRequest struct {
	key    ratematrix.VersionKey
	offset *int
	limit  *int
}

func (d *Deps) matrixCellPage(
	ctx context.Context,
	req *cellPageRequest,
) ([]*gqlmodel.RateMatrixCell, error) {
	authCtx, err := d.RequirePermission(ctx, permission.ResourceRateMatrix, permission.OpRead)
	if err != nil {
		return nil, err
	}
	from := 0
	if req.offset != nil {
		from = max(*req.offset, 0)
	}
	size := maxMatrixCellPage
	if req.limit != nil {
		size = min(max(*req.limit, 1), maxMatrixCellPage)
	}
	tenant := base.TenantInfo(authCtx)

	page, err := d.RateMatrixService.ListCells(ctx, &ratematrixports.ListRateMatrixCellsRequest{
		TenantInfo:    tenant,
		RateMatrixID:  req.key.RateMatrixID,
		VersionNumber: req.key.VersionNumber,
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: size, Offset: from},
		},
	})
	if err != nil {
		return nil, err
	}

	cells := make([]*gqlmodel.RateMatrixCell, 0, len(page.Items))
	for _, cell := range page.Items {
		if cell != nil {
			cells = append(cells, rateMatrixCellToModel(cell))
		}
	}
	return cells, nil
}
