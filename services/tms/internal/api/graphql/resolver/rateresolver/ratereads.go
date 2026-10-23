package rateresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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

// agreementLanes is one page of an agreement's live lanes, most specific
// first, with how many it holds.
func (r *RateAgreementResolver) agreementLanes(
	ctx context.Context,
	obj *gqlmodel.RateAgreement,
	limit int,
) (*rateagreementports.RateAgreementLanePage, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateAgreement, permission.OpRead)
	if err != nil {
		return nil, err
	}
	agreementID, err := base.ParseIDField("id", obj.ID)
	if err != nil {
		return nil, err
	}

	return r.RateAgreementService.ListLanes(ctx, &rateagreementports.ListRateAgreementLanesRequest{
		TenantInfo:      base.TenantInfo(authCtx),
		RateAgreementID: agreementID,
		Limit:           limit,
	})
}

// matrixCells is one page of a matrix's grid, with how many cells it holds.
func (r *RateMatrixResolver) matrixCells(
	ctx context.Context,
	obj *gqlmodel.RateMatrix,
	offset, limit int,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateMatrix, permission.OpRead)
	if err != nil {
		return nil, err
	}
	matrixID, err := base.ParseIDField("id", obj.ID)
	if err != nil {
		return nil, err
	}
	tenant := base.TenantInfo(authCtx)

	return r.RateMatrixService.ListCells(ctx, &ratematrixports.ListRateMatrixCellsRequest{
		TenantInfo:   tenant,
		RateMatrixID: matrixID,
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit, Offset: offset},
		},
	})
}
