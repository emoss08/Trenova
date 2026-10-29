package ratingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type RateAgreementLaneCountLoaderFactoryParams struct {
	fx.In

	AgreementRepo rateagreementports.RateAgreementRepository
}

type RateAgreementLaneCountLoaderFactory struct {
	agreements rateagreementports.RateAgreementRepository
}

func NewRateAgreementLaneCountLoaderFactory(
	p RateAgreementLaneCountLoaderFactoryParams,
) *RateAgreementLaneCountLoaderFactory {
	return &RateAgreementLaneCountLoaderFactory{agreements: p.AgreementRepo}
}

func newRateAgreementLaneCountLoader(
	f *RateAgreementLaneCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, rateagreementports.LaneCount] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *RateAgreementLaneCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[rateagreementports.LaneCount] {
	return loadercore.BatchValueFunc[rateagreementports.LaneCount](
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID]rateagreementports.LaneCount, error) {
			return f.agreements.CountLiveLanes(ctx, &rateagreementports.CountLiveLanesRequest{
				TenantInfo:       tenantInfo,
				RateAgreementIDs: ids,
			})
		},
	)
}

var rateAgreementLaneCountKey = loadercore.NewKey(newRateAgreementLaneCountLoader)

func RateAgreementLaneCount(ctx context.Context, key string) (rateagreementports.LaneCount, error) {
	return loadercore.Load(ctx, rateAgreementLaneCountKey, key)
}

func (f *RateAgreementLaneCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
