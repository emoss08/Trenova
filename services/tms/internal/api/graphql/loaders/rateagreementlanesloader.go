package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type RateAgreementLanesLoaderFactoryParams struct {
	fx.In

	AgreementRepo rateagreementports.RateAgreementRepository
}

type RateAgreementLanesLoaderFactory struct {
	agreements rateagreementports.RateAgreementRepository
}

func NewRateAgreementLanesLoaderFactory(
	p RateAgreementLanesLoaderFactoryParams,
) *RateAgreementLanesLoaderFactory {
	return &RateAgreementLanesLoaderFactory{agreements: p.AgreementRepo}
}

func newRateAgreementLanesLoader(
	f *RateAgreementLanesLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*rateagreement.RateAgreementRule] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *RateAgreementLanesLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*rateagreement.RateAgreementRule] {
	return batchFirstFunc[*rateagreement.RateAgreementRule](
		func(
			ctx context.Context,
			ids []pulid.ID,
			first int,
		) (map[pulid.ID][]*rateagreement.RateAgreementRule, error) {
			return f.agreements.ListFirstLanes(ctx, &rateagreementports.ListFirstLanesRequest{
				TenantInfo:       tenantInfo,
				RateAgreementIDs: ids,
				First:            first,
			})
		},
	)
}
