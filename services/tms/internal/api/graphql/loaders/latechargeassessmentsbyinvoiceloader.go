package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type lateChargeAssessmentsBySourceInvoiceIDsLister interface {
	ListBySourceInvoiceIDs(
		ctx context.Context,
		req *repositories.ListLateChargeAssessmentsByInvoiceIDsRequest,
	) (map[pulid.ID][]*latecharge.LateChargeAssessment, error)
}

type LateChargeAssessmentsByInvoiceIDLoaderFactoryParams struct {
	fx.In

	LateChargeRepo repositories.LateChargeRepository
}

type LateChargeAssessmentsByInvoiceIDLoaderFactory struct {
	assessments lateChargeAssessmentsBySourceInvoiceIDsLister
}

func NewLateChargeAssessmentsByInvoiceIDLoaderFactory(
	p LateChargeAssessmentsByInvoiceIDLoaderFactoryParams,
) *LateChargeAssessmentsByInvoiceIDLoaderFactory {
	return &LateChargeAssessmentsByInvoiceIDLoaderFactory{assessments: p.LateChargeRepo}
}

func (f *LateChargeAssessmentsByInvoiceIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*latecharge.LateChargeAssessment] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *LateChargeAssessmentsByInvoiceIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*latecharge.LateChargeAssessment] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*latecharge.LateChargeAssessment, error) {
			return f.assessments.ListBySourceInvoiceIDs(ctx, &repositories.ListLateChargeAssessmentsByInvoiceIDsRequest{
				TenantInfo: tenantInfo,
				InvoiceIDs: ids,
			})
		},
	)
}
