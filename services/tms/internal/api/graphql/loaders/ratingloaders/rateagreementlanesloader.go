package ratingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *RateAgreementLanesLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*rateagreement.RateAgreementRule] {
	return loadercore.BatchFirstFunc[*rateagreement.RateAgreementRule](
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

var rateAgreementLanesKey = loadercore.NewKey(newRateAgreementLanesLoader)

func RateAgreementLanes(
	ctx context.Context,
	agreementID string,
	first int,
) ([]*rateagreement.RateAgreementRule, error) {
	return loadercore.Load(ctx, rateAgreementLanesKey, loadercore.FirstKey(agreementID, first))
}

func (f *RateAgreementLanesLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
