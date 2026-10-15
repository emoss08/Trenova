package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type FactoringCompanyByIDLoaderFactoryParams struct {
	fx.In

	FactoringCompanyRepo factoringcompanyports.FactoringCompanyRepository
}

type FactoringCompanyByIDLoaderFactory struct {
	factors factoringcompanyports.FactoringCompanyRepository
}

func NewFactoringCompanyByIDLoaderFactory(
	p FactoringCompanyByIDLoaderFactoryParams,
) *FactoringCompanyByIDLoaderFactory {
	return &FactoringCompanyByIDLoaderFactory{factors: p.FactoringCompanyRepo}
}

func newFactoringCompanyByIDLoader(
	f *FactoringCompanyByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoringcompany.FactoringCompany] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*factoringcompany.FactoringCompany, error) {
			return f.factors.GetByIDs(ctx, factoringcompanyports.GetFactoringCompaniesByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Factoring company not found",
	))
}

type FactoringStateByInvoiceIDLoaderFactoryParams struct {
	fx.In

	States factoringports.InvoiceStateRepository
}

type FactoringStateByInvoiceIDLoaderFactory struct {
	states factoringports.InvoiceStateRepository
}

func NewFactoringStateByInvoiceIDLoaderFactory(
	p FactoringStateByInvoiceIDLoaderFactoryParams,
) *FactoringStateByInvoiceIDLoaderFactory {
	return &FactoringStateByInvoiceIDLoaderFactory{states: p.States}
}

func newFactoringStateByInvoiceIDLoader(
	f *FactoringStateByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoring.InvoiceState] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*factoring.InvoiceState, error) {
			return f.states.ListByInvoiceIDs(
				ctx,
				&factoringports.ListInvoiceStatesByInvoiceIDsRequest{
					TenantInfo: tenantInfo,
					InvoiceIDs: ids,
				},
			)
		},
	))
}

type FactoringAgreementsByCompanyIDLoaderFactoryParams struct {
	fx.In

	Agreements factoringports.AgreementRepository
}

type FactoringAgreementsByCompanyIDLoaderFactory struct {
	agreements factoringports.AgreementRepository
}

func NewFactoringAgreementsByCompanyIDLoaderFactory(
	p FactoringAgreementsByCompanyIDLoaderFactoryParams,
) *FactoringAgreementsByCompanyIDLoaderFactory {
	return &FactoringAgreementsByCompanyIDLoaderFactory{agreements: p.Agreements}
}

func newFactoringAgreementsByCompanyIDLoader(
	f *FactoringAgreementsByCompanyIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*factoring.Agreement] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*factoring.Agreement, error) {
			return f.agreements.ListByFactoringCompanyIDs(
				ctx,
				&factoringports.ListAgreementsByFactoringCompanyIDsRequest{
					TenantInfo:          tenantInfo,
					FactoringCompanyIDs: ids,
				},
			)
		},
	))
}

type FactoringSubmissionByIDLoaderFactoryParams struct {
	fx.In

	Submissions factoringports.SubmissionRepository
}

type FactoringSubmissionByIDLoaderFactory struct {
	submissions factoringports.SubmissionRepository
}

func NewFactoringSubmissionByIDLoaderFactory(
	p FactoringSubmissionByIDLoaderFactoryParams,
) *FactoringSubmissionByIDLoaderFactory {
	return &FactoringSubmissionByIDLoaderFactory{submissions: p.Submissions}
}

func newFactoringSubmissionByIDLoader(
	f *FactoringSubmissionByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoring.Submission] {
	return newLoader(batchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*factoring.Submission, error) {
			return f.submissions.GetByIDs(ctx, factoringports.GetSubmissionsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Factoring schedule not found",
	))
}

type FactoringSubmissionItemsLoaderFactoryParams struct {
	fx.In

	Submissions factoringports.SubmissionRepository
}

type FactoringSubmissionItemsLoaderFactory struct {
	submissions factoringports.SubmissionRepository
}

func NewFactoringSubmissionItemsLoaderFactory(
	p FactoringSubmissionItemsLoaderFactoryParams,
) *FactoringSubmissionItemsLoaderFactory {
	return &FactoringSubmissionItemsLoaderFactory{submissions: p.Submissions}
}

func newFactoringSubmissionItemsLoader(
	f *FactoringSubmissionItemsLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*factoring.SubmissionItem] {
	return newLoader(batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*factoring.SubmissionItem, error) {
			return f.submissions.ListItemsBySubmissionIDs(
				ctx,
				&factoringports.ListItemsBySubmissionIDsRequest{
					TenantInfo:    tenantInfo,
					SubmissionIDs: ids,
				},
			)
		},
	))
}

type FactoringLiveItemByInvoiceIDLoaderFactoryParams struct {
	fx.In

	Submissions factoringports.SubmissionRepository
}

type FactoringLiveItemByInvoiceIDLoaderFactory struct {
	submissions factoringports.SubmissionRepository
}

func NewFactoringLiveItemByInvoiceIDLoaderFactory(
	p FactoringLiveItemByInvoiceIDLoaderFactoryParams,
) *FactoringLiveItemByInvoiceIDLoaderFactory {
	return &FactoringLiveItemByInvoiceIDLoaderFactory{submissions: p.Submissions}
}

func newFactoringLiveItemByInvoiceIDLoader(
	f *FactoringLiveItemByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoringports.LiveItem] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*factoringports.LiveItem, error) {
			return f.submissions.ListLiveItems(ctx, &factoringports.ListLiveItemsRequest{
				TenantInfo: tenantInfo,
				InvoiceIDs: ids,
			})
		},
	))
}
