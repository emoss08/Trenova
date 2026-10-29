package billingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
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
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
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
) *dataloadgen.Loader[string, *factoringreadiness.InvoiceState] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*factoringreadiness.InvoiceState, error) {
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
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
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

	Submissions factoringsubmissionports.SubmissionRepository
}

type FactoringSubmissionByIDLoaderFactory struct {
	submissions factoringsubmissionports.SubmissionRepository
}

func NewFactoringSubmissionByIDLoaderFactory(
	p FactoringSubmissionByIDLoaderFactoryParams,
) *FactoringSubmissionByIDLoaderFactory {
	return &FactoringSubmissionByIDLoaderFactory{submissions: p.Submissions}
}

func newFactoringSubmissionByIDLoader(
	f *FactoringSubmissionByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoringsubmission.Submission] {
	return loadercore.NewLoader(loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*factoringsubmission.Submission, error) {
			return f.submissions.GetByIDs(ctx, factoringsubmissionports.GetSubmissionsByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Factoring schedule not found",
	))
}

type FactoringSubmissionItemsLoaderFactoryParams struct {
	fx.In

	Submissions factoringsubmissionports.SubmissionRepository
}

type FactoringSubmissionItemsLoaderFactory struct {
	submissions factoringsubmissionports.SubmissionRepository
}

func NewFactoringSubmissionItemsLoaderFactory(
	p FactoringSubmissionItemsLoaderFactoryParams,
) *FactoringSubmissionItemsLoaderFactory {
	return &FactoringSubmissionItemsLoaderFactory{submissions: p.Submissions}
}

func newFactoringSubmissionItemsLoader(
	f *FactoringSubmissionItemsLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*factoringsubmission.SubmissionItem] {
	return loadercore.NewLoader(loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error) {
			return f.submissions.ListItemsBySubmissionIDs(
				ctx,
				&factoringsubmissionports.ListItemsBySubmissionIDsRequest{
					TenantInfo:    tenantInfo,
					SubmissionIDs: ids,
				},
			)
		},
	))
}

type FactoringLiveItemByInvoiceIDLoaderFactoryParams struct {
	fx.In

	Submissions factoringsubmissionports.SubmissionRepository
}

type FactoringLiveItemByInvoiceIDLoaderFactory struct {
	submissions factoringsubmissionports.SubmissionRepository
}

func NewFactoringLiveItemByInvoiceIDLoaderFactory(
	p FactoringLiveItemByInvoiceIDLoaderFactoryParams,
) *FactoringLiveItemByInvoiceIDLoaderFactory {
	return &FactoringLiveItemByInvoiceIDLoaderFactory{submissions: p.Submissions}
}

func newFactoringLiveItemByInvoiceIDLoader(
	f *FactoringLiveItemByInvoiceIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *factoringsubmissionports.LiveItem] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
			return f.submissions.ListLiveItems(ctx, &factoringsubmissionports.ListLiveItemsRequest{
				TenantInfo: tenantInfo,
				InvoiceIDs: ids,
			})
		},
	))
}

var (
	factoringCompanyByIDKey           = loadercore.NewKey(newFactoringCompanyByIDLoader)
	factoringStateByInvoiceIDKey      = loadercore.NewKey(newFactoringStateByInvoiceIDLoader)
	factoringAgreementsByCompanyIDKey = loadercore.NewKey(newFactoringAgreementsByCompanyIDLoader)
	factoringSubmissionItemsKey       = loadercore.NewKey(newFactoringSubmissionItemsLoader)
	factoringSubmissionByIDKey        = loadercore.NewKey(newFactoringSubmissionByIDLoader)
	factoringLiveItemByInvoiceIDKey   = loadercore.NewKey(newFactoringLiveItemByInvoiceIDLoader)
)

func FactoringCompanyByID(
	ctx context.Context,
	key string,
) (*factoringcompany.FactoringCompany, error) {
	return loadercore.Load(ctx, factoringCompanyByIDKey, key)
}

func (f *FactoringCompanyByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func FactoringStateByInvoiceID(
	ctx context.Context,
	key string,
) (*factoringreadiness.InvoiceState, error) {
	return loadercore.Load(ctx, factoringStateByInvoiceIDKey, key)
}

func (f *FactoringStateByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func FactoringAgreementsByCompanyID(
	ctx context.Context,
	key string,
) ([]*factoring.Agreement, error) {
	return loadercore.Load(ctx, factoringAgreementsByCompanyIDKey, key)
}

func (f *FactoringAgreementsByCompanyIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func FactoringSubmissionItems(
	ctx context.Context,
	key string,
) ([]*factoringsubmission.SubmissionItem, error) {
	return loadercore.Load(ctx, factoringSubmissionItemsKey, key)
}

func (f *FactoringSubmissionItemsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func FactoringSubmissionByID(
	ctx context.Context,
	key string,
) (*factoringsubmission.Submission, error) {
	return loadercore.Load(ctx, factoringSubmissionByIDKey, key)
}

func (f *FactoringSubmissionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func FactoringLiveItemByInvoiceID(
	ctx context.Context,
	key string,
) (*factoringsubmissionports.LiveItem, error) {
	return loadercore.Load(ctx, factoringLiveItemByInvoiceIDKey, key)
}

func (f *FactoringLiveItemByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
