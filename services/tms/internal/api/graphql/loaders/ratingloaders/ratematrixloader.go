package ratingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type RateMatrixByIDLoaderFactoryParams struct {
	fx.In

	MatrixRepo ratematrixports.RateMatrixRepository
}

type RateMatrixByIDLoaderFactory struct {
	matrices ratematrixports.RateMatrixRepository
}

func NewRateMatrixByIDLoaderFactory(
	p RateMatrixByIDLoaderFactoryParams,
) *RateMatrixByIDLoaderFactory {
	return &RateMatrixByIDLoaderFactory{matrices: p.MatrixRepo}
}

func newRateMatrixByIDLoader(f *RateMatrixByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *ratematrix.RateMatrix] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *RateMatrixByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*ratematrix.RateMatrix] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*ratematrix.RateMatrix, error) {
			return f.matrices.GetByIDs(ctx, &ratematrixports.GetRateMatricesByIDsRequest{
				TenantInfo:    tenantInfo,
				RateMatrixIDs: ids,
			})
		},
		"Rate matrix not found within your organization",
	)
}

var rateMatrixByIDKey = loadercore.NewKey(newRateMatrixByIDLoader)

func RateMatrixByID(ctx context.Context, key string) (*ratematrix.RateMatrix, error) {
	return loadercore.Load(ctx, rateMatrixByIDKey, key)
}

func (f *RateMatrixByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
