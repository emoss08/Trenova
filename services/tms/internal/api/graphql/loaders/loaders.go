package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/domain/worker"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type contextKey struct{}

type FactoryParams struct {
	fx.In

	OrganizationByID                          *OrganizationByIDLoaderFactory
	LocationByID                              *LocationByIDLoaderFactory
	OrderByID                                 *OrderByIDLoaderFactory
	ShipmentProfitabilityByID                 *ShipmentProfitabilityLoaderFactory
	EDIPartnerByCustomerID                    *EDIPartnerByCustomerIDLoaderFactory
	FormulaTemplateStatsByID                  *FormulaTemplateStatsLoaderFactory
	PayProfileActiveAssignmentCount           *PayProfileActiveAssignmentCountLoaderFactory
	PTOPolicyOpenAssignmentCount              *PTOPolicyOpenAssignmentCountLoaderFactory
	ShiftTemplateActiveAssignmentCount        *ShiftTemplateActiveAssignmentCountLoaderFactory
	WorkerChecklistTemplateOpenChecklistCount *WorkerChecklistTemplateOpenChecklistCountLoaderFactory
	WorkerCredentialTypeActiveCredentialCount *WorkerCredentialTypeActiveCredentialCountLoaderFactory
	TrainingCourseOpenRecordCount             *TrainingCourseOpenRecordCountLoaderFactory
	PerformanceReviewTemplateOpenReviewCount  *PerformanceReviewTemplateOpenReviewCountLoaderFactory
	CustomerByID                              *CustomerByIDLoaderFactory
	InvoiceByID                               *InvoiceByIDLoaderFactory
	GLAccountByID                             *GLAccountByIDLoaderFactory
	RoutingGuideWithEntriesByID               *RoutingGuideWithEntriesByIDLoaderFactory
	WorkerDQFVerificationsByWorkerID          *WorkerDQFVerificationsByWorkerIDLoaderFactory
	WorkerLeaveEntriesByCaseID                *WorkerLeaveEntriesByCaseIDLoaderFactory
	FiscalPeriodsByFiscalYearID               *FiscalPeriodsByFiscalYearIDLoaderFactory
	DocumentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
}

type Factory struct {
	organizationByID                          *OrganizationByIDLoaderFactory
	locationByID                              *LocationByIDLoaderFactory
	orderByID                                 *OrderByIDLoaderFactory
	shipmentProfitabilityByID                 *ShipmentProfitabilityLoaderFactory
	ediPartnerByCustomerID                    *EDIPartnerByCustomerIDLoaderFactory
	formulaTemplateStatsByID                  *FormulaTemplateStatsLoaderFactory
	payProfileActiveAssignmentCount           *PayProfileActiveAssignmentCountLoaderFactory
	pTOPolicyOpenAssignmentCount              *PTOPolicyOpenAssignmentCountLoaderFactory
	shiftTemplateActiveAssignmentCount        *ShiftTemplateActiveAssignmentCountLoaderFactory
	workerChecklistTemplateOpenChecklistCount *WorkerChecklistTemplateOpenChecklistCountLoaderFactory
	workerCredentialTypeActiveCredentialCount *WorkerCredentialTypeActiveCredentialCountLoaderFactory
	trainingCourseOpenRecordCount             *TrainingCourseOpenRecordCountLoaderFactory
	performanceReviewTemplateOpenReviewCount  *PerformanceReviewTemplateOpenReviewCountLoaderFactory
	customerByID                              *CustomerByIDLoaderFactory
	invoiceByID                               *InvoiceByIDLoaderFactory
	gLAccountByID                             *GLAccountByIDLoaderFactory
	routingGuideWithEntriesByID               *RoutingGuideWithEntriesByIDLoaderFactory
	workerDQFVerificationsByWorkerID          *WorkerDQFVerificationsByWorkerIDLoaderFactory
	workerLeaveEntriesByCaseID                *WorkerLeaveEntriesByCaseIDLoaderFactory
	fiscalPeriodsByFiscalYearID               *FiscalPeriodsByFiscalYearIDLoaderFactory
	documentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
}

type Loaders struct {
	OrganizationByID                          *dataloadgen.Loader[string, *tenant.Organization]
	LocationByID                              *dataloadgen.Loader[string, *location.Location]
	OrderByID                                 *dataloadgen.Loader[string, *order.Order]
	ShipmentProfitabilityByID                 *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate]
	EDIPartnerByCustomerID                    *dataloadgen.Loader[string, *edi.EDIPartner]
	FormulaTemplateStatsByID                  *dataloadgen.Loader[string, repositories.TemplateStats]
	PayProfileActiveAssignmentCount           *dataloadgen.Loader[string, int]
	PTOPolicyOpenAssignmentCount              *dataloadgen.Loader[string, int]
	ShiftTemplateActiveAssignmentCount        *dataloadgen.Loader[string, int]
	WorkerChecklistTemplateOpenChecklistCount *dataloadgen.Loader[string, int]
	WorkerCredentialTypeActiveCredentialCount *dataloadgen.Loader[string, int]
	TrainingCourseOpenRecordCount             *dataloadgen.Loader[string, int]
	PerformanceReviewTemplateOpenReviewCount  *dataloadgen.Loader[string, int]
	CustomerByID                              *dataloadgen.Loader[string, *customer.Customer]
	InvoiceByID                               *dataloadgen.Loader[string, *invoice.Invoice]
	GLAccountByID                             *dataloadgen.Loader[string, *glaccount.GLAccount]
	RoutingGuideWithEntriesByID               *dataloadgen.Loader[string, *tender.RoutingGuide]
	WorkerDQFVerificationsByWorkerID          *dataloadgen.Loader[string, []*worker.WorkerEmploymentVerification]
	WorkerLeaveEntriesByCaseID                *dataloadgen.Loader[string, []*worker.WorkerLeaveEntry]
	FiscalPeriodsByFiscalYearID               *dataloadgen.Loader[string, []*fiscalperiod.FiscalPeriod]
	DocumentTemplateKindByTemplateID          *dataloadgen.Loader[string, documenttemplate.Kind]
}

func NewFactory(p FactoryParams) *Factory {
	return &Factory{
		organizationByID:                          p.OrganizationByID,
		locationByID:                              p.LocationByID,
		orderByID:                                 p.OrderByID,
		shipmentProfitabilityByID:                 p.ShipmentProfitabilityByID,
		ediPartnerByCustomerID:                    p.EDIPartnerByCustomerID,
		formulaTemplateStatsByID:                  p.FormulaTemplateStatsByID,
		payProfileActiveAssignmentCount:           p.PayProfileActiveAssignmentCount,
		pTOPolicyOpenAssignmentCount:              p.PTOPolicyOpenAssignmentCount,
		shiftTemplateActiveAssignmentCount:        p.ShiftTemplateActiveAssignmentCount,
		workerChecklistTemplateOpenChecklistCount: p.WorkerChecklistTemplateOpenChecklistCount,
		workerCredentialTypeActiveCredentialCount: p.WorkerCredentialTypeActiveCredentialCount,
		trainingCourseOpenRecordCount:             p.TrainingCourseOpenRecordCount,
		performanceReviewTemplateOpenReviewCount:  p.PerformanceReviewTemplateOpenReviewCount,
		customerByID:                              p.CustomerByID,
		invoiceByID:                               p.InvoiceByID,
		gLAccountByID:                             p.GLAccountByID,
		routingGuideWithEntriesByID:               p.RoutingGuideWithEntriesByID,
		workerDQFVerificationsByWorkerID:          p.WorkerDQFVerificationsByWorkerID,
		workerLeaveEntriesByCaseID:                p.WorkerLeaveEntriesByCaseID,
		fiscalPeriodsByFiscalYearID:               p.FiscalPeriodsByFiscalYearID,
		documentTemplateKindByTemplateID:          p.DocumentTemplateKindByTemplateID,
	}
}

func (f *Factory) NewForTenant(tenantInfo pagination.TenantInfo) *Loaders {
	return &Loaders{
		OrganizationByID:                          f.organizationByID.NewForTenant(tenantInfo),
		LocationByID:                              f.locationByID.NewForTenant(tenantInfo),
		OrderByID:                                 f.orderByID.NewForTenant(tenantInfo),
		ShipmentProfitabilityByID:                 f.shipmentProfitabilityByID.NewForTenant(tenantInfo),
		EDIPartnerByCustomerID:                    f.ediPartnerByCustomerID.NewForTenant(tenantInfo),
		FormulaTemplateStatsByID:                  f.formulaTemplateStatsByID.NewForTenant(tenantInfo),
		PayProfileActiveAssignmentCount:           f.payProfileActiveAssignmentCount.NewForTenant(tenantInfo),
		PTOPolicyOpenAssignmentCount:              f.pTOPolicyOpenAssignmentCount.NewForTenant(tenantInfo),
		ShiftTemplateActiveAssignmentCount:        f.shiftTemplateActiveAssignmentCount.NewForTenant(tenantInfo),
		WorkerChecklistTemplateOpenChecklistCount: f.workerChecklistTemplateOpenChecklistCount.NewForTenant(tenantInfo),
		WorkerCredentialTypeActiveCredentialCount: f.workerCredentialTypeActiveCredentialCount.NewForTenant(tenantInfo),
		TrainingCourseOpenRecordCount:             f.trainingCourseOpenRecordCount.NewForTenant(tenantInfo),
		PerformanceReviewTemplateOpenReviewCount:  f.performanceReviewTemplateOpenReviewCount.NewForTenant(tenantInfo),
		CustomerByID:                              f.customerByID.NewForTenant(tenantInfo),
		InvoiceByID:                               f.invoiceByID.NewForTenant(tenantInfo),
		GLAccountByID:                             f.gLAccountByID.NewForTenant(tenantInfo),
		RoutingGuideWithEntriesByID:               f.routingGuideWithEntriesByID.NewForTenant(tenantInfo),
		WorkerDQFVerificationsByWorkerID:          f.workerDQFVerificationsByWorkerID.NewForTenant(tenantInfo),
		WorkerLeaveEntriesByCaseID:                f.workerLeaveEntriesByCaseID.NewForTenant(tenantInfo),
		FiscalPeriodsByFiscalYearID:               f.fiscalPeriodsByFiscalYearID.NewForTenant(tenantInfo),
		DocumentTemplateKindByTemplateID:          f.documentTemplateKindByTemplateID.NewForTenant(tenantInfo),
	}
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, contextKey{}, loaders)
}

func FromContext(ctx context.Context) (*Loaders, bool) {
	loaders, ok := ctx.Value(contextKey{}).(*Loaders)
	return loaders, ok
}
