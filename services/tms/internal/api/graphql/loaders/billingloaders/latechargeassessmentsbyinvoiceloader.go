package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type lateChargeAssessmentsBySourceInvoiceIDsLister interface {
	ListBySourceInvoiceIDs(
		ctx context.Context,
		req *latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest,
	) (map[pulid.ID][]*latecharge.LateChargeAssessment, error)
}

type LateChargeAssessmentsByInvoiceIDLoaderFactoryParams struct {
	fx.In

	LateChargeRepo latechargeports.LateChargeRepository
}

type LateChargeAssessmentsByInvoiceIDLoaderFactory struct {
	assessments lateChargeAssessmentsBySourceInvoiceIDsLister
}

func NewLateChargeAssessmentsByInvoiceIDLoaderFactory(
	p LateChargeAssessmentsByInvoiceIDLoaderFactoryParams,
) *LateChargeAssessmentsByInvoiceIDLoaderFactory {
	return &LateChargeAssessmentsByInvoiceIDLoaderFactory{assessments: p.LateChargeRepo}
}

func newLateChargeAssessmentsByInvoiceIDLoader(f *LateChargeAssessmentsByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*latecharge.LateChargeAssessment] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *LateChargeAssessmentsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*latecharge.LateChargeAssessment] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
			return f.assessments.ListBySourceInvoiceIDs(
				ctx,
				&latechargeports.ListLateChargeAssessmentsByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	)
}

var lateChargeAssessmentsByInvoiceIDKey = loadercore.NewKey(
	newLateChargeAssessmentsByInvoiceIDLoader,
)

func LateChargeAssessmentsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*latecharge.LateChargeAssessment, error) {
	return loadercore.Load(ctx, lateChargeAssessmentsByInvoiceIDKey, key)
}

func (f *LateChargeAssessmentsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
