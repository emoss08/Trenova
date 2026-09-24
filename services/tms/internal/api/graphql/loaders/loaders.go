package loaders

import (
	"context"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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
	AgentDefinitionStatsByID                  *AgentDefinitionStatsLoaderFactory
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
	AgentDecisionsByProposalID                *AgentDecisionsByProposalIDLoaderFactory
	AgentRunByID                              *AgentRunByIDLoaderFactory
	AgentDefinitionByID                       *AgentDefinitionByIDLoaderFactory
	UsableAgentByID                           *UsableAgentByIDLoaderFactory
	AccessRolesByAgentID                      *AccessRolesByAgentIDLoaderFactory
	AgentsByRoleID                            *AgentsByRoleIDLoaderFactory
	ToolTrustByAgentID                        *ToolTrustByAgentIDLoaderFactory
	DocumentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
	IFTAJurisdictionByID                      *IFTAJurisdictionByIDLoaderFactory
	IFTAReturnByID                            *IFTAReturnByIDLoaderFactory
	FuelCardByID                              *FuelCardByIDLoaderFactory
	TractorByID                               *TractorByIDLoaderFactory
	ShipmentMoveJurisdictionMilesByMoveID     *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory
	UserByID                                  *UserByIDLoaderFactory
	DocumentByID                              *DocumentByIDLoaderFactory
	UsStateByID                               *UsStateByIDLoaderFactory
	InvoicesByShipmentID                      *InvoicesByShipmentIDLoaderFactory
	ChargeAllocationsByShipmentID             *ChargeAllocationsByShipmentIDLoaderFactory
	ChargeAllocationsByOrderChargeID          *ChargeAllocationsByOrderChargeIDLoaderFactory
	CustomerPaymentApplicationsByInvoiceID    *CustomerPaymentApplicationsByInvoiceIDLoaderFactory
	CreditMemoApplicationsByInvoiceID         *CreditMemoApplicationsByInvoiceIDLoaderFactory
	InvoiceDisputesByInvoiceID                *InvoiceDisputesByInvoiceIDLoaderFactory
	LateChargeAssessmentsByInvoiceID          *LateChargeAssessmentsByInvoiceIDLoaderFactory
	InvoiceEDISendPlanByInvoiceID             *InvoiceEDISendPlanByInvoiceIDLoaderFactory
	CarrierIntelSnapshotByCarrierID           *CarrierIntelSnapshotByCarrierIDLoaderFactory
	CarrierIntelSnapshotByCustomerID          *CarrierIntelSnapshotByCustomerIDLoaderFactory
	CarrierIntelOpenEventCount                *CarrierIntelOpenEventCountLoaderFactory
	InboundAttachmentCount                    *InboundAttachmentCountLoaderFactory
	ShipmentSummaryByID                       *ShipmentSummaryByIDLoaderFactory
	CarrierMonitoringEnrollmentByCarrierID    *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory
}

type Factory struct {
	organizationByID                          *OrganizationByIDLoaderFactory
	locationByID                              *LocationByIDLoaderFactory
	orderByID                                 *OrderByIDLoaderFactory
	shipmentProfitabilityByID                 *ShipmentProfitabilityLoaderFactory
	ediPartnerByCustomerID                    *EDIPartnerByCustomerIDLoaderFactory
	formulaTemplateStatsByID                  *FormulaTemplateStatsLoaderFactory
	agentDefinitionStatsByID                  *AgentDefinitionStatsLoaderFactory
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
	agentDecisionsByProposalID                *AgentDecisionsByProposalIDLoaderFactory
	agentRunByID                              *AgentRunByIDLoaderFactory
	agentDefinitionByID                       *AgentDefinitionByIDLoaderFactory
	usableAgentByID                           *UsableAgentByIDLoaderFactory
	accessRolesByAgentID                      *AccessRolesByAgentIDLoaderFactory
	agentsByRoleID                            *AgentsByRoleIDLoaderFactory
	toolTrustByAgentID                        *ToolTrustByAgentIDLoaderFactory
	documentTemplateKindByTemplateID          *DocumentTemplateKindByTemplateIDLoaderFactory
	iFTAJurisdictionByID                      *IFTAJurisdictionByIDLoaderFactory
	iFTAReturnByID                            *IFTAReturnByIDLoaderFactory
	fuelCardByID                              *FuelCardByIDLoaderFactory
	tractorByID                               *TractorByIDLoaderFactory
	shipmentMoveJurisdictionMilesByMoveID     *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory
	userByID                                  *UserByIDLoaderFactory
	documentByID                              *DocumentByIDLoaderFactory
	usStateByID                               *UsStateByIDLoaderFactory
	invoicesByShipmentID                      *InvoicesByShipmentIDLoaderFactory
	chargeAllocationsByShipmentID             *ChargeAllocationsByShipmentIDLoaderFactory
	chargeAllocationsByOrderChargeID          *ChargeAllocationsByOrderChargeIDLoaderFactory
	customerPaymentApplicationsByInvoiceID    *CustomerPaymentApplicationsByInvoiceIDLoaderFactory
	creditMemoApplicationsByInvoiceID         *CreditMemoApplicationsByInvoiceIDLoaderFactory
	invoiceDisputesByInvoiceID                *InvoiceDisputesByInvoiceIDLoaderFactory
	lateChargeAssessmentsByInvoiceID          *LateChargeAssessmentsByInvoiceIDLoaderFactory
	invoiceEDISendPlanByInvoiceID             *InvoiceEDISendPlanByInvoiceIDLoaderFactory
	carrierIntelSnapshotByCarrierID           *CarrierIntelSnapshotByCarrierIDLoaderFactory
	carrierIntelSnapshotByCustomerID          *CarrierIntelSnapshotByCustomerIDLoaderFactory
	carrierIntelOpenEventCount                *CarrierIntelOpenEventCountLoaderFactory
	inboundAttachmentCount                    *InboundAttachmentCountLoaderFactory
	shipmentSummaryByID                       *ShipmentSummaryByIDLoaderFactory
	carrierMonitoringEnrollmentByCarrierID    *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory
}

type Loaders struct {
	OrganizationByID                          *dataloadgen.Loader[string, *tenant.Organization]
	LocationByID                              *dataloadgen.Loader[string, *location.Location]
	OrderByID                                 *dataloadgen.Loader[string, *order.Order]
	ShipmentProfitabilityByID                 *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate]
	EDIPartnerByCustomerID                    *dataloadgen.Loader[string, *edi.EDIPartner]
	FormulaTemplateStatsByID                  *dataloadgen.Loader[string, repositories.TemplateStats]
	AgentDefinitionStatsByID                  *dataloadgen.Loader[string, repositories.AgentDefinitionStats]
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
	AgentDecisionsByProposalID                *dataloadgen.Loader[string, []*agent.AgentDecision]
	AgentRunByID                              *dataloadgen.Loader[string, *agent.AgentRun]
	AgentDefinitionByID                       *dataloadgen.Loader[string, *agentdefinition.Definition]
	UsableAgentByID                           *dataloadgen.Loader[string, *agentdefinition.Definition]
	AccessRolesByAgentID                      *dataloadgen.Loader[string, []*permission.Role]
	AgentsByRoleID                            *dataloadgen.Loader[string, []*agentdefinition.Definition]
	ToolTrustByAgentID                        *dataloadgen.Loader[string, []*agent.ToolTrust]
	DocumentTemplateKindByTemplateID          *dataloadgen.Loader[string, documenttemplate.Kind]
	IFTAJurisdictionByID                      *dataloadgen.Loader[string, *ifta.Jurisdiction]
	IFTAReturnByID                            *dataloadgen.Loader[string, *ifta.Return]
	FuelCardByID                              *dataloadgen.Loader[string, *fuelpurchase.FuelCard]
	TractorByID                               *dataloadgen.Loader[string, *tractor.Tractor]
	ShipmentMoveJurisdictionMilesByMoveID     *dataloadgen.Loader[string, []*shipment.ShipmentMoveJurisdictionMile]
	UserByID                                  *dataloadgen.Loader[string, *tenant.User]
	DocumentByID                              *dataloadgen.Loader[string, *document.Document]
	UsStateByID                               *dataloadgen.Loader[string, *usstate.UsState]
	InvoicesByShipmentID                      *dataloadgen.Loader[string, []*invoice.Invoice]
	ChargeAllocationsByShipmentID             *dataloadgen.Loader[string, []*shipment.ChargeAllocation]
	ChargeAllocationsByOrderChargeID          *dataloadgen.Loader[string, []*shipment.ChargeAllocation]
	CustomerPaymentApplicationsByInvoiceID    *dataloadgen.Loader[string, []*customerpayment.Application]
	CreditMemoApplicationsByInvoiceID         *dataloadgen.Loader[string, []*customerpayment.CreditMemoApplication]
	InvoiceDisputesByInvoiceID                *dataloadgen.Loader[string, []*invoice.InvoiceDispute]
	LateChargeAssessmentsByInvoiceID          *dataloadgen.Loader[string, []*latecharge.LateChargeAssessment]
	InvoiceEDISendPlanByInvoiceID             *dataloadgen.Loader[string, *services.InvoiceEDISendPlan]
	CarrierIntelSnapshotByCarrierID           *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot]
	CarrierIntelSnapshotByCustomerID          *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot]
	CarrierIntelOpenEventCount                *dataloadgen.Loader[string, int]
	InboundAttachmentCount                    *dataloadgen.Loader[string, int]
	ShipmentSummaryByID                       *dataloadgen.Loader[string, *repositories.ShipmentSummary]
	CarrierMonitoringEnrollmentByCarrierID    *dataloadgen.Loader[string, []*carrierintel.CarrierMonitoringEnrollment]
}

func NewFactory(p FactoryParams) *Factory {
	return &Factory{
		organizationByID:                          p.OrganizationByID,
		locationByID:                              p.LocationByID,
		orderByID:                                 p.OrderByID,
		shipmentProfitabilityByID:                 p.ShipmentProfitabilityByID,
		ediPartnerByCustomerID:                    p.EDIPartnerByCustomerID,
		formulaTemplateStatsByID:                  p.FormulaTemplateStatsByID,
		agentDefinitionStatsByID:                  p.AgentDefinitionStatsByID,
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
		agentDecisionsByProposalID:                p.AgentDecisionsByProposalID,
		agentRunByID:                              p.AgentRunByID,
		agentDefinitionByID:                       p.AgentDefinitionByID,
		usableAgentByID:                           p.UsableAgentByID,
		accessRolesByAgentID:                      p.AccessRolesByAgentID,
		agentsByRoleID:                            p.AgentsByRoleID,
		toolTrustByAgentID:                        p.ToolTrustByAgentID,
		documentTemplateKindByTemplateID:          p.DocumentTemplateKindByTemplateID,
		iFTAJurisdictionByID:                      p.IFTAJurisdictionByID,
		iFTAReturnByID:                            p.IFTAReturnByID,
		fuelCardByID:                              p.FuelCardByID,
		tractorByID:                               p.TractorByID,
		shipmentMoveJurisdictionMilesByMoveID:     p.ShipmentMoveJurisdictionMilesByMoveID,
		userByID:                                  p.UserByID,
		documentByID:                              p.DocumentByID,
		usStateByID:                               p.UsStateByID,
		invoicesByShipmentID:                      p.InvoicesByShipmentID,
		chargeAllocationsByShipmentID:             p.ChargeAllocationsByShipmentID,
		chargeAllocationsByOrderChargeID:          p.ChargeAllocationsByOrderChargeID,
		customerPaymentApplicationsByInvoiceID:    p.CustomerPaymentApplicationsByInvoiceID,
		creditMemoApplicationsByInvoiceID:         p.CreditMemoApplicationsByInvoiceID,
		invoiceDisputesByInvoiceID:                p.InvoiceDisputesByInvoiceID,
		lateChargeAssessmentsByInvoiceID:          p.LateChargeAssessmentsByInvoiceID,
		invoiceEDISendPlanByInvoiceID:             p.InvoiceEDISendPlanByInvoiceID,
		carrierIntelSnapshotByCarrierID:           p.CarrierIntelSnapshotByCarrierID,
		carrierIntelSnapshotByCustomerID:          p.CarrierIntelSnapshotByCustomerID,
		carrierIntelOpenEventCount:                p.CarrierIntelOpenEventCount,
		inboundAttachmentCount:                    p.InboundAttachmentCount,
		shipmentSummaryByID:                       p.ShipmentSummaryByID,
		carrierMonitoringEnrollmentByCarrierID:    p.CarrierMonitoringEnrollmentByCarrierID,
	}
}

func (f *Factory) NewForTenant(tenantInfo pagination.TenantInfo) *Loaders {
	return &Loaders{
		OrganizationByID: f.organizationByID.NewForTenant(tenantInfo),
		LocationByID:     f.locationByID.NewForTenant(tenantInfo),
		OrderByID:        f.orderByID.NewForTenant(tenantInfo),
		ShipmentProfitabilityByID: f.shipmentProfitabilityByID.NewForTenant(
			tenantInfo,
		),
		EDIPartnerByCustomerID: f.ediPartnerByCustomerID.NewForTenant(
			tenantInfo,
		),
		FormulaTemplateStatsByID: f.formulaTemplateStatsByID.NewForTenant(
			tenantInfo,
		),
		AgentDefinitionStatsByID: f.agentDefinitionStatsByID.NewForTenant(
			tenantInfo,
		),
		PayProfileActiveAssignmentCount: f.payProfileActiveAssignmentCount.NewForTenant(
			tenantInfo,
		),
		PTOPolicyOpenAssignmentCount: f.pTOPolicyOpenAssignmentCount.NewForTenant(
			tenantInfo,
		),
		ShiftTemplateActiveAssignmentCount: f.shiftTemplateActiveAssignmentCount.NewForTenant(
			tenantInfo,
		),
		WorkerChecklistTemplateOpenChecklistCount: f.workerChecklistTemplateOpenChecklistCount.NewForTenant(
			tenantInfo,
		),
		WorkerCredentialTypeActiveCredentialCount: f.workerCredentialTypeActiveCredentialCount.NewForTenant(
			tenantInfo,
		),
		TrainingCourseOpenRecordCount: f.trainingCourseOpenRecordCount.NewForTenant(
			tenantInfo,
		),
		PerformanceReviewTemplateOpenReviewCount: f.performanceReviewTemplateOpenReviewCount.NewForTenant(
			tenantInfo,
		),
		CustomerByID:  f.customerByID.NewForTenant(tenantInfo),
		InvoiceByID:   f.invoiceByID.NewForTenant(tenantInfo),
		GLAccountByID: f.gLAccountByID.NewForTenant(tenantInfo),
		RoutingGuideWithEntriesByID: f.routingGuideWithEntriesByID.NewForTenant(
			tenantInfo,
		),
		WorkerDQFVerificationsByWorkerID: f.workerDQFVerificationsByWorkerID.NewForTenant(
			tenantInfo,
		),
		WorkerLeaveEntriesByCaseID: f.workerLeaveEntriesByCaseID.NewForTenant(
			tenantInfo,
		),
		FiscalPeriodsByFiscalYearID: f.fiscalPeriodsByFiscalYearID.NewForTenant(
			tenantInfo,
		),
		AgentDecisionsByProposalID: f.agentDecisionsByProposalID.NewForTenant(
			tenantInfo,
		),
		AgentRunByID:         f.agentRunByID.NewForTenant(tenantInfo),
		AgentDefinitionByID:  f.agentDefinitionByID.NewForTenant(tenantInfo),
		UsableAgentByID:      f.usableAgentByID.NewForTenant(tenantInfo),
		AccessRolesByAgentID: f.accessRolesByAgentID.NewForTenant(tenantInfo),
		AgentsByRoleID:       f.agentsByRoleID.NewForTenant(tenantInfo),
		ToolTrustByAgentID:   f.toolTrustByAgentID.NewForTenant(tenantInfo),
		DocumentTemplateKindByTemplateID: f.documentTemplateKindByTemplateID.NewForTenant(
			tenantInfo,
		),
		IFTAJurisdictionByID: f.iFTAJurisdictionByID.NewForTenant(tenantInfo),
		IFTAReturnByID:       f.iFTAReturnByID.NewForTenant(tenantInfo),
		FuelCardByID:         f.fuelCardByID.NewForTenant(tenantInfo),
		TractorByID:          f.tractorByID.NewForTenant(tenantInfo),
		ShipmentMoveJurisdictionMilesByMoveID: f.shipmentMoveJurisdictionMilesByMoveID.NewForTenant(
			tenantInfo,
		),
		UserByID:             f.userByID.NewForTenant(tenantInfo),
		DocumentByID:         f.documentByID.NewForTenant(tenantInfo),
		UsStateByID:          f.usStateByID.NewForTenant(tenantInfo),
		InvoicesByShipmentID: f.invoicesByShipmentID.NewForTenant(tenantInfo),
		ChargeAllocationsByShipmentID: f.chargeAllocationsByShipmentID.NewForTenant(
			tenantInfo,
		),
		ChargeAllocationsByOrderChargeID: f.chargeAllocationsByOrderChargeID.NewForTenant(
			tenantInfo,
		),
		CustomerPaymentApplicationsByInvoiceID: f.customerPaymentApplicationsByInvoiceID.NewForTenant(
			tenantInfo,
		),
		CreditMemoApplicationsByInvoiceID: f.creditMemoApplicationsByInvoiceID.NewForTenant(
			tenantInfo,
		),
		InvoiceDisputesByInvoiceID: f.invoiceDisputesByInvoiceID.NewForTenant(
			tenantInfo,
		),
		LateChargeAssessmentsByInvoiceID: f.lateChargeAssessmentsByInvoiceID.NewForTenant(
			tenantInfo,
		),
		InvoiceEDISendPlanByInvoiceID: f.invoiceEDISendPlanByInvoiceID.NewForTenant(
			tenantInfo,
		),
		CarrierIntelSnapshotByCarrierID: f.carrierIntelSnapshotByCarrierID.NewForTenant(
			tenantInfo,
		),
		CarrierIntelSnapshotByCustomerID: f.carrierIntelSnapshotByCustomerID.NewForTenant(
			tenantInfo,
		),
		CarrierIntelOpenEventCount: f.carrierIntelOpenEventCount.NewForTenant(
			tenantInfo,
		),
		InboundAttachmentCount: f.inboundAttachmentCount.NewForTenant(
			tenantInfo,
		),
		ShipmentSummaryByID: f.shipmentSummaryByID.NewForTenant(tenantInfo),
		CarrierMonitoringEnrollmentByCarrierID: f.carrierMonitoringEnrollmentByCarrierID.NewForTenant(
			tenantInfo,
		),
	}
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, contextKey{}, loaders)
}

func FromContext(ctx context.Context) (*Loaders, bool) {
	loaders, ok := ctx.Value(contextKey{}).(*Loaders)
	return loaders, ok
}
