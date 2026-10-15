package loaders

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/vikstrous/dataloadgen"
)

var ErrNotAttached = errors.New("graphql loaders are not attached to this request")

func (f *Factory) Attach(ctx context.Context, tenantInfo pagination.TenantInfo) context.Context {
	return context.WithValue(ctx, contextKey{}, newSet(f, tenantInfo))
}

func Available(ctx context.Context) bool {
	_, ok := fromContext(ctx)
	return ok
}

func FieldCeilings(ctx context.Context) aiauditports.FieldCeilings {
	s, ok := fromContext(ctx)
	if !ok {
		return nil
	}

	return s.fieldCeilings
}

func fromContext(ctx context.Context) (*set, bool) {
	s, ok := ctx.Value(contextKey{}).(*set)
	return s, ok && s != nil
}

func attach(ctx context.Context, apply func(*set)) context.Context {
	next := &set{}
	if current, ok := fromContext(ctx); ok {
		*next = *current
	}
	apply(next)

	return context.WithValue(ctx, contextKey{}, next)
}

func loaderFrom[V any](
	ctx context.Context,
	pick func(*set) *dataloadgen.Loader[string, V],
) (*dataloadgen.Loader[string, V], bool) {
	s, ok := fromContext(ctx)
	if !ok {
		return nil, false
	}
	loader := pick(s)

	return loader, loader != nil
}

func load[V any](
	ctx context.Context,
	key string,
	pick func(*set) *dataloadgen.Loader[string, V],
) (V, error) {
	loader, ok := loaderFrom(ctx, pick)
	if !ok {
		var zero V
		return zero, ErrNotAttached
	}

	return loader.Load(ctx, key)
}

func loadAll[V any](
	ctx context.Context,
	keys []string,
	pick func(*set) *dataloadgen.Loader[string, V],
) ([]V, error) {
	loader, ok := loaderFrom(ctx, pick)
	if !ok {
		return nil, ErrNotAttached
	}

	return loader.LoadAll(ctx, keys)
}

func loadThunk[V any](
	ctx context.Context,
	key string,
	pick func(*set) *dataloadgen.Loader[string, V],
) func() (V, error) {
	loader, ok := loaderFrom(ctx, pick)
	if !ok {
		return func() (V, error) {
			var zero V
			return zero, ErrNotAttached
		}
	}

	return loader.LoadThunk(ctx, key)
}

func OrganizationByID(ctx context.Context, key string) (*tenant.Organization, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *tenant.Organization] {
		return s.organizationByID
	})
}

func (f *OrganizationByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.organizationByID = newOrganizationByIDLoader(f, tenantInfo)
	})
}

func LocationByID(ctx context.Context, key string) (*location.Location, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *location.Location] {
		return s.locationByID
	})
}

func (f *LocationByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.locationByID = newLocationByIDLoader(f, tenantInfo)
	})
}

func OrderByID(ctx context.Context, key string) (*order.Order, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *order.Order] {
		return s.orderByID
	})
}

func (f *OrderByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.orderByID = newOrderByIDLoader(f, tenantInfo)
	})
}

func ShipmentProfitabilityByID(
	ctx context.Context,
	key string,
) (*costingservice.ShipmentProfitabilityEstimate, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *costingservice.ShipmentProfitabilityEstimate] {
			return s.shipmentProfitabilityByID
		},
	)
}

func (f *ShipmentProfitabilityLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.shipmentProfitabilityByID = newShipmentProfitabilityLoader(f, tenantInfo)
	})
}

func ShipmentEtaByID(ctx context.Context, key string) (*shipmenttrackingports.ShipmentEta, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *shipmenttrackingports.ShipmentEta] {
			return s.shipmentEtaByID
		},
	)
}

func (f *ShipmentEtaLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.shipmentEtaByID = newShipmentEtaLoader(f, tenantInfo)
	})
}

func EDIPartnerByCustomerID(ctx context.Context, key string) (*edi.EDIPartner, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *edi.EDIPartner] {
		return s.ediPartnerByCustomerID
	})
}

func (f *EDIPartnerByCustomerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.ediPartnerByCustomerID = newEDIPartnerByCustomerIDLoader(f, tenantInfo)
	})
}

func FormulaTemplateStatsByID(
	ctx context.Context,
	key string,
) (formulatemplateports.TemplateStats, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, formulatemplateports.TemplateStats] {
			return s.formulaTemplateStatsByID
		},
	)
}

func (f *FormulaTemplateStatsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.formulaTemplateStatsByID = newFormulaTemplateStatsLoader(f, tenantInfo)
	})
}

func AgentDefinitionStatsByID(
	ctx context.Context,
	key string,
) (agentdefinitionports.AgentDefinitionStats, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, agentdefinitionports.AgentDefinitionStats] {
			return s.agentDefinitionStatsByID
		},
	)
}

func (f *AgentDefinitionStatsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentDefinitionStatsByID = newAgentDefinitionStatsLoader(f, tenantInfo)
	})
}

func PayProfileActiveAssignmentCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.payProfileActiveAssignmentCount
	})
}

func (f *PayProfileActiveAssignmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.payProfileActiveAssignmentCount = newPayProfileActiveAssignmentCountLoader(f, tenantInfo)
	})
}

func PTOPolicyOpenAssignmentCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.ptoPolicyOpenAssignmentCount
	})
}

func (f *PTOPolicyOpenAssignmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.ptoPolicyOpenAssignmentCount = newPTOPolicyOpenAssignmentCountLoader(f, tenantInfo)
	})
}

func ShiftTemplateActiveAssignmentCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.shiftTemplateActiveAssignmentCount
	})
}

func (f *ShiftTemplateActiveAssignmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.shiftTemplateActiveAssignmentCount = newShiftTemplateActiveAssignmentCountLoader(
			f,
			tenantInfo,
		)
	})
}

func AIProviderMonthSpend(ctx context.Context, key string) (decimal.Decimal, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, decimal.Decimal] {
		return s.aiProviderMonthSpend
	})
}

func (f *AIProviderMonthSpendLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.aiProviderMonthSpend = newAIProviderMonthSpendLoader(f, tenantInfo)
	})
}

func WorkerChecklistTemplateOpenChecklistCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.workerChecklistTemplateOpenChecklistCount
	})
}

func (f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.workerChecklistTemplateOpenChecklistCount = newWorkerChecklistTemplateOpenChecklistCountLoader(
			f,
			tenantInfo,
		)
	})
}

func WorkerCredentialTypeActiveCredentialCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.workerCredentialTypeActiveCredentialCount
	})
}

func (f *WorkerCredentialTypeActiveCredentialCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.workerCredentialTypeActiveCredentialCount = newWorkerCredentialTypeActiveCredentialCountLoader(
			f,
			tenantInfo,
		)
	})
}

func TrainingCourseOpenRecordCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.trainingCourseOpenRecordCount
	})
}

func (f *TrainingCourseOpenRecordCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.trainingCourseOpenRecordCount = newTrainingCourseOpenRecordCountLoader(f, tenantInfo)
	})
}

func PerformanceReviewTemplateOpenReviewCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.performanceReviewTemplateOpenReviewCount
	})
}

func (f *PerformanceReviewTemplateOpenReviewCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.performanceReviewTemplateOpenReviewCount = newPerformanceReviewTemplateOpenReviewCountLoader(
			f,
			tenantInfo,
		)
	})
}

func CustomerByID(ctx context.Context, key string) (*customer.Customer, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *customer.Customer] {
		return s.customerByID
	})
}

func (f *CustomerByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.customerByID = newCustomerByIDLoader(f, tenantInfo)
	})
}

func InvoiceByID(ctx context.Context, key string) (*invoice.Invoice, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *invoice.Invoice] {
		return s.invoiceByID
	})
}

func (f *InvoiceByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.invoiceByID = newInvoiceByIDLoader(f, tenantInfo)
	})
}

func GLAccountByID(ctx context.Context, key string) (*glaccount.GLAccount, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *glaccount.GLAccount] {
		return s.glAccountByID
	})
}

func (f *GLAccountByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.glAccountByID = newGLAccountByIDLoader(f, tenantInfo)
	})
}

func RoutingGuideWithEntriesByID(ctx context.Context, key string) (*tender.RoutingGuide, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *tender.RoutingGuide] {
		return s.routingGuideWithEntriesByID
	})
}

func (f *RoutingGuideWithEntriesByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.routingGuideWithEntriesByID = newRoutingGuideWithEntriesByIDLoader(f, tenantInfo)
	})
}

func WorkerDQFVerificationsByWorkerID(
	ctx context.Context,
	key string,
) ([]*worker.WorkerEmploymentVerification, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*worker.WorkerEmploymentVerification] {
			return s.workerDQFVerificationsByWorkerID
		},
	)
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.workerDQFVerificationsByWorkerID = newWorkerDQFVerificationsByWorkerIDLoader(
			f,
			tenantInfo,
		)
	})
}

func WorkerLeaveEntriesByCaseID(
	ctx context.Context,
	key string,
) ([]*worker.WorkerLeaveEntry, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*worker.WorkerLeaveEntry] {
		return s.workerLeaveEntriesByCaseID
	})
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.workerLeaveEntriesByCaseID = newWorkerLeaveEntriesByCaseIDLoader(f, tenantInfo)
	})
}

func FiscalPeriodsByFiscalYearID(
	ctx context.Context,
	key string,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*fiscalperiod.FiscalPeriod] {
		return s.fiscalPeriodsByFiscalYearID
	})
}

func (f *FiscalPeriodsByFiscalYearIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.fiscalPeriodsByFiscalYearID = newFiscalPeriodsByFiscalYearIDLoader(f, tenantInfo)
	})
}

func AgentDecisionsByProposalID(ctx context.Context, key string) ([]*agent.AgentDecision, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*agent.AgentDecision] {
		return s.agentDecisionsByProposalID
	})
}

func (f *AgentDecisionsByProposalIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentDecisionsByProposalID = newAgentDecisionsByProposalIDLoader(f, tenantInfo)
	})
}

func AgentRunByID(ctx context.Context, key string) (*agent.AgentRun, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agent.AgentRun] {
		return s.agentRunByID
	})
}

func (f *AgentRunByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentRunByID = newAgentRunByIDLoader(f, tenantInfo)
	})
}

func AgentRunTranscriptByID(ctx context.Context, key string) (*agent.AgentRun, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agent.AgentRun] {
		return s.agentRunTranscriptByID
	})
}

func (f *AgentRunTranscriptByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentRunTranscriptByID = newAgentRunTranscriptByIDLoader(f, tenantInfo)
	})
}

func AgentMemoryByID(ctx context.Context, key string) (*agent.Memory, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agent.Memory] {
		return s.agentMemoryByID
	})
}

func (f *AgentMemoryByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentMemoryByID = newAgentMemoryByIDLoader(f, tenantInfo)
	})
}

func AgentMemoryReplacement(ctx context.Context, key string) (*agent.Memory, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agent.Memory] {
		return s.agentMemoryReplacement
	})
}

func (f *AgentMemoryReplacementLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentMemoryReplacement = newAgentMemoryReplacementLoader(f, tenantInfo)
	})
}

func AgentDefinitionByID(ctx context.Context, key string) (*agentdefinition.Definition, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agentdefinition.Definition] {
		return s.agentDefinitionByID
	})
}

func AgentDefinitionByIDThunk(
	ctx context.Context,
	key string,
) func() (*agentdefinition.Definition, error) {
	return loadThunk(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *agentdefinition.Definition] {
			return s.agentDefinitionByID
		},
	)
}

func (f *AgentDefinitionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentDefinitionByID = newAgentDefinitionByIDLoader(f, tenantInfo)
	})
}

func ThreadAgentByID(ctx context.Context, key string) (*conversation.Thread, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *conversation.Thread] {
		return s.threadAgentByID
	})
}

func (f *ThreadAgentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.threadAgentByID = newThreadAgentByIDLoader(f, tenantInfo)
	})
}

func UsableAgentByID(ctx context.Context, key string) (*agentdefinition.Definition, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *agentdefinition.Definition] {
		return s.usableAgentByID
	})
}

func UsableAgentByIDThunk(
	ctx context.Context,
	key string,
) func() (*agentdefinition.Definition, error) {
	return loadThunk(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *agentdefinition.Definition] {
			return s.usableAgentByID
		},
	)
}

func (f *UsableAgentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.usableAgentByID = newUsableAgentByIDLoader(f, tenantInfo)
	})
}

func AccessRolesByAgentID(ctx context.Context, key string) ([]*permission.Role, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*permission.Role] {
		return s.accessRolesByAgentID
	})
}

func (f *AccessRolesByAgentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.accessRolesByAgentID = newAccessRolesByAgentIDLoader(f, tenantInfo)
	})
}

func AgentsByRoleID(ctx context.Context, key string) ([]*agentdefinition.Definition, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*agentdefinition.Definition] {
		return s.agentsByRoleID
	})
}

func (f *AgentsByRoleIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.agentsByRoleID = newAgentsByRoleIDLoader(f, tenantInfo)
	})
}

func ToolTrustByAgentID(ctx context.Context, key string) ([]*agent.ToolTrust, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*agent.ToolTrust] {
		return s.toolTrustByAgentID
	})
}

func (f *ToolTrustByAgentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.toolTrustByAgentID = newToolTrustByAgentIDLoader(f, tenantInfo)
	})
}

func DocumentTemplateKindByTemplateID(
	ctx context.Context,
	key string,
) (documenttemplate.Kind, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, documenttemplate.Kind] {
		return s.documentTemplateKindByTemplateID
	})
}

func (f *DocumentTemplateKindByTemplateIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.documentTemplateKindByTemplateID = newDocumentTemplateKindByTemplateIDLoader(
			f,
			tenantInfo,
		)
	})
}

func IFTAJurisdictionByID(ctx context.Context, key string) (*ifta.Jurisdiction, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *ifta.Jurisdiction] {
		return s.iftaJurisdictionByID
	})
}

func (f *IFTAJurisdictionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.iftaJurisdictionByID = newIFTAJurisdictionByIDLoader(f, tenantInfo)
	})
}

func IFTAReturnByID(ctx context.Context, key string) (*ifta.Return, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *ifta.Return] {
		return s.iftaReturnByID
	})
}

func (f *IFTAReturnByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.iftaReturnByID = newIFTAReturnByIDLoader(f, tenantInfo)
	})
}

func FuelCardByID(ctx context.Context, key string) (*fuelpurchase.FuelCard, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *fuelpurchase.FuelCard] {
		return s.fuelCardByID
	})
}

func (f *FuelCardByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.fuelCardByID = newFuelCardByIDLoader(f, tenantInfo)
	})
}

func TractorByID(ctx context.Context, key string) (*tractor.Tractor, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *tractor.Tractor] {
		return s.tractorByID
	})
}

func (f *TractorByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.tractorByID = newTractorByIDLoader(f, tenantInfo)
	})
}

func ShipmentMoveJurisdictionMilesByMoveID(
	ctx context.Context,
	key string,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*shipment.ShipmentMoveJurisdictionMile] {
			return s.shipmentMoveJurisdictionMilesByMoveID
		},
	)
}

func (f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.shipmentMoveJurisdictionMilesByMoveID = newShipmentMoveJurisdictionMilesByMoveIDLoader(
			f,
			tenantInfo,
		)
	})
}

func UserByID(ctx context.Context, key string) (*tenant.User, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *tenant.User] {
		return s.userByID
	})
}

func (f *UserByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.userByID = newUserByIDLoader(f, tenantInfo)
	})
}

func DocumentByID(ctx context.Context, key string) (*document.Document, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *document.Document] {
		return s.documentByID
	})
}

func (f *DocumentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.documentByID = newDocumentByIDLoader(f, tenantInfo)
	})
}

func UsStateByID(ctx context.Context, key string) (*usstate.UsState, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *usstate.UsState] {
		return s.usStateByID
	})
}

func (f *UsStateByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.usStateByID = newUsStateByIDLoader(f, tenantInfo)
	})
}

func InvoicesByShipmentID(ctx context.Context, key string) ([]*invoice.Invoice, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*invoice.Invoice] {
		return s.invoicesByShipmentID
	})
}

func InvoicesByShipmentIDAll(ctx context.Context, keys []string) ([][]*invoice.Invoice, error) {
	return loadAll(ctx, keys, func(s *set) *dataloadgen.Loader[string, []*invoice.Invoice] {
		return s.invoicesByShipmentID
	})
}

func (f *InvoicesByShipmentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.invoicesByShipmentID = newInvoicesByShipmentIDLoader(f, tenantInfo)
	})
}

func ChargeAllocationsByShipmentID(
	ctx context.Context,
	key string,
) ([]*shipment.ChargeAllocation, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*shipment.ChargeAllocation] {
		return s.chargeAllocationsByShipmentID
	})
}

func (f *ChargeAllocationsByShipmentIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.chargeAllocationsByShipmentID = newChargeAllocationsByShipmentIDLoader(f, tenantInfo)
	})
}

func ChargeAllocationsByOrderChargeID(
	ctx context.Context,
	key string,
) ([]*shipment.ChargeAllocation, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*shipment.ChargeAllocation] {
		return s.chargeAllocationsByOrderChargeID
	})
}

func (f *ChargeAllocationsByOrderChargeIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.chargeAllocationsByOrderChargeID = newChargeAllocationsByOrderChargeIDLoader(
			f,
			tenantInfo,
		)
	})
}

func CustomerPaymentApplicationsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*customerpayment.Application, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*customerpayment.Application] {
		return s.customerPaymentApplicationsByInvoiceID
	})
}

func (f *CustomerPaymentApplicationsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.customerPaymentApplicationsByInvoiceID = newCustomerPaymentApplicationsByInvoiceIDLoader(
			f,
			tenantInfo,
		)
	})
}

func CreditMemoApplicationsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*customerpayment.CreditMemoApplication, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*customerpayment.CreditMemoApplication] {
			return s.creditMemoApplicationsByInvoiceID
		},
	)
}

func (f *CreditMemoApplicationsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.creditMemoApplicationsByInvoiceID = newCreditMemoApplicationsByInvoiceIDLoader(
			f,
			tenantInfo,
		)
	})
}

func InvoiceDisputesByInvoiceID(
	ctx context.Context,
	key string,
) ([]*invoice.InvoiceDispute, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*invoice.InvoiceDispute] {
		return s.invoiceDisputesByInvoiceID
	})
}

func (f *InvoiceDisputesByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.invoiceDisputesByInvoiceID = newInvoiceDisputesByInvoiceIDLoader(f, tenantInfo)
	})
}

func LateChargeAssessmentsByInvoiceID(
	ctx context.Context,
	key string,
) ([]*latecharge.LateChargeAssessment, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*latecharge.LateChargeAssessment] {
			return s.lateChargeAssessmentsByInvoiceID
		},
	)
}

func (f *LateChargeAssessmentsByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.lateChargeAssessmentsByInvoiceID = newLateChargeAssessmentsByInvoiceIDLoader(
			f,
			tenantInfo,
		)
	})
}

func InvoiceEDISendPlanByInvoiceID(
	ctx context.Context,
	key string,
) (*invoiceports.InvoiceEDISendPlan, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *invoiceports.InvoiceEDISendPlan] {
			return s.invoiceEDISendPlanByInvoiceID
		},
	)
}

func (f *InvoiceEDISendPlanByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.invoiceEDISendPlanByInvoiceID = newInvoiceEDISendPlanByInvoiceIDLoader(f, tenantInfo)
	})
}

func CarrierIntelSnapshotByCarrierID(
	ctx context.Context,
	key string,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot] {
			return s.carrierIntelSnapshotByCarrierID
		},
	)
}

func (f *CarrierIntelSnapshotByCarrierIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.carrierIntelSnapshotByCarrierID = newCarrierIntelSnapshotByCarrierIDLoader(f, tenantInfo)
	})
}

func CarrierIntelSnapshotByCustomerID(
	ctx context.Context,
	key string,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*carrierintel.CarrierIntelSnapshot] {
			return s.carrierIntelSnapshotByCustomerID
		},
	)
}

func (f *CarrierIntelSnapshotByCustomerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.carrierIntelSnapshotByCustomerID = newCarrierIntelSnapshotByCustomerIDLoader(
			f,
			tenantInfo,
		)
	})
}

func CarrierIntelOpenEventCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.carrierIntelOpenEventCount
	})
}

func (f *CarrierIntelOpenEventCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.carrierIntelOpenEventCount = newCarrierIntelOpenEventCountLoader(f, tenantInfo)
	})
}

func InboundAttachmentCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.inboundAttachmentCount
	})
}

func (f *InboundAttachmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundAttachmentCount = newInboundAttachmentCountLoader(f, tenantInfo)
	})
}

func InboundThreadCount(ctx context.Context, threadKey string) (int, error) {
	return load(ctx, threadKey, func(s *set) *dataloadgen.Loader[string, int] {
		return s.inboundThreadCount
	})
}

func (f *InboundThreadCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundThreadCount = newInboundThreadCountLoader(f, tenantInfo)
	})
}

func InboundActivitiesByMessageID(ctx context.Context, key string) ([]*triage.Activity, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*triage.Activity] {
		return s.inboundActivitiesByMessageID
	})
}

func (f *InboundActivitiesByMessageIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundActivitiesByMessageID = newInboundActivitiesByMessageIDLoader(f, tenantInfo)
	})
}

func DocumentTypeNameByID(ctx context.Context, key string) (string, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, string] {
		return s.documentTypeNameByID
	})
}

func (f *DocumentTypeNameByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.documentTypeNameByID = newDocumentTypeNameByIDLoader(f, tenantInfo)
	})
}

func TriageViewWaitingCount(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.triageViewWaitingCount
	})
}

func (f *TriageViewWaitingCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.triageViewWaitingCount = newTriageViewWaitingCountLoader(f, tenantInfo)
	})
}

func ShipmentSummaryByID(ctx context.Context, key string) (*shipmentports.ShipmentSummary, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *shipmentports.ShipmentSummary] {
		return s.shipmentSummaryByID
	})
}

func (f *ShipmentSummaryByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.shipmentSummaryByID = newShipmentSummaryByIDLoader(f, tenantInfo)
	})
}

func CaptureRecordLabel(ctx context.Context, key string) (*captureports.CaptureRecordLabel, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *captureports.CaptureRecordLabel] {
			return s.captureRecordLabel
		},
	)
}

func (f *CaptureRecordLabelLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.captureRecordLabel = newCaptureRecordLabelLoader(f, tenantInfo)
	})
}

func CarrierMonitoringEnrollmentByCarrierID(
	ctx context.Context,
	key string,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*carrierintel.CarrierMonitoringEnrollment] {
			return s.carrierMonitoringEnrollmentByCarrierID
		},
	)
}

func (f *CarrierMonitoringEnrollmentByCarrierIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.carrierMonitoringEnrollmentByCarrierID = newCarrierMonitoringEnrollmentByCarrierIDLoader(
			f,
			tenantInfo,
		)
	})
}

func AuditEntriesByAIAuditEventID(ctx context.Context, key string) ([]*audit.Entry, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*audit.Entry] {
		return s.auditEntriesByAIAuditEventID
	})
}

func (f *AuditEntriesByAIAuditEventIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.auditEntriesByAIAuditEventID = newAuditEntriesByAIAuditEventIDLoader(f, tenantInfo)
	})
}

func SubsetLabels(ctx context.Context, key string) (agentports.RecordLabels, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, agentports.RecordLabels] {
		return s.subsetLabels
	})
}

func (f *SubsetLabelsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.subsetLabels = newSubsetLabelsLoader(f, tenantInfo)
	})
}

func TriageRuleMonthMatches(ctx context.Context, key string) (int, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, int] {
		return s.triageRuleMonthMatches
	})
}

func (f *TriageRuleMonthMatchesLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.triageRuleMonthMatches = newTriageRuleMonthMatchesLoader(f, tenantInfo)
	})
}

func TriageSampleByMessageID(ctx context.Context, key string) (*inboundmessageports.Sample, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *inboundmessageports.Sample] {
		return s.triageSampleByMessageID
	})
}

func (f *TriageSampleByMessageIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.triageSampleByMessageID = newTriageSampleByMessageIDLoader(f, tenantInfo)
	})
}

func InboundMailboxByID(ctx context.Context, key string) (*inboundmessage.Mailbox, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *inboundmessage.Mailbox] {
		return s.inboundMailboxByID
	})
}

func (f *InboundMailboxByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundMailboxByID = newInboundMailboxByIDLoader(f, tenantInfo)
	})
}

func InboundAttachmentsByMessageID(
	ctx context.Context,
	key string,
) ([]*inboundmessage.InboundAttachment, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, []*inboundmessage.InboundAttachment] {
			return s.inboundAttachmentsByMessageID
		},
	)
}

func (f *InboundAttachmentsByMessageIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundAttachmentsByMessageID = newInboundAttachmentsByMessageIDLoader(f, tenantInfo)
	})
}

func InboundAttachmentByID(
	ctx context.Context,
	key string,
) (*inboundmessage.InboundAttachment, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *inboundmessage.InboundAttachment] {
			return s.inboundAttachmentByID
		},
	)
}

func (f *InboundAttachmentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundAttachmentByID = newInboundAttachmentByIDLoader(f, tenantInfo)
	})
}

func InboundLatestReplyByMessageID(ctx context.Context, key string) (*triage.Reply, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *triage.Reply] {
		return s.inboundLatestReplyByMessageID
	})
}

func InboundReplyDraftByMessageID(ctx context.Context, key string) (*triage.Reply, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *triage.Reply] {
		return s.inboundReplyDraftByMessageID
	})
}

func InboundRepliesByMessageID(ctx context.Context, key string) ([]*triage.Reply, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*triage.Reply] {
		return s.inboundRepliesByMessageID
	})
}

func InboundReplyByID(ctx context.Context, key string) (*triage.Reply, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *triage.Reply] {
		return s.inboundReplyByID
	})
}

func (f *InboundReplyLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundLatestReplyByMessageID = newInboundLatestReplyByMessageIDLoader(f, tenantInfo)
		s.inboundReplyDraftByMessageID = newInboundReplyDraftByMessageIDLoader(f, tenantInfo)
		s.inboundRepliesByMessageID = newInboundRepliesByMessageIDLoader(f, tenantInfo)
		s.inboundReplyByID = newInboundReplyByIDLoader(f, tenantInfo)
	})
}

func InboundDocumentReadingByDocumentID(
	ctx context.Context,
	key string,
) (*inboundmessageports.DocumentReading, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *inboundmessageports.DocumentReading] {
			return s.inboundDocumentReadingByDocumentID
		},
	)
}

func (f *InboundDocumentReadingLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.inboundDocumentReadingByDocumentID = newInboundDocumentReadingByDocumentIDLoader(
			f,
			tenantInfo,
		)
	})
}

func FactoringCompanyByID(
	ctx context.Context,
	key string,
) (*factoringcompany.FactoringCompany, error) {
	return load(
		ctx,
		key,
		func(s *set) *dataloadgen.Loader[string, *factoringcompany.FactoringCompany] {
			return s.factoringCompanyByID
		},
	)
}

func (f *FactoringCompanyByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringCompanyByID = newFactoringCompanyByIDLoader(f, tenantInfo)
	})
}

func FactoringStateByInvoiceID(ctx context.Context, key string) (*factoring.InvoiceState, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *factoring.InvoiceState] {
		return s.factoringStateByInvoiceID
	})
}

func (f *FactoringStateByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringStateByInvoiceID = newFactoringStateByInvoiceIDLoader(f, tenantInfo)
	})
}

func FactoringAgreementsByCompanyID(
	ctx context.Context,
	key string,
) ([]*factoring.Agreement, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*factoring.Agreement] {
		return s.factoringAgreementsByCompanyID
	})
}

func (f *FactoringAgreementsByCompanyIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringAgreementsByCompanyID = newFactoringAgreementsByCompanyIDLoader(f, tenantInfo)
	})
}

func FactoringSubmissionItems(
	ctx context.Context,
	key string,
) ([]*factoring.SubmissionItem, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*factoring.SubmissionItem] {
		return s.factoringSubmissionItems
	})
}

func (f *FactoringSubmissionItemsLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringSubmissionItems = newFactoringSubmissionItemsLoader(f, tenantInfo)
	})
}

func FactoringSubmissionByID(ctx context.Context, key string) (*factoring.Submission, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *factoring.Submission] {
		return s.factoringSubmissionByID
	})
}

func (f *FactoringSubmissionByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringSubmissionByID = newFactoringSubmissionByIDLoader(f, tenantInfo)
	})
}

func FactoringLiveItemByInvoiceID(
	ctx context.Context,
	key string,
) (*factoringports.LiveItem, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *factoringports.LiveItem] {
		return s.factoringLiveItemByInvoiceID
	})
}

func (f *FactoringLiveItemByInvoiceIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.factoringLiveItemByInvoiceID = newFactoringLiveItemByInvoiceIDLoader(f, tenantInfo)
	})
}

func PayAdvanceByID(ctx context.Context, key string) (*driverpay.PayAdvance, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *driverpay.PayAdvance] {
		return s.payAdvanceByID
	})
}

func (f *PayAdvanceByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.payAdvanceByID = newPayAdvanceByIDLoader(f, tenantInfo)
	})
}

func EscrowAccountByID(ctx context.Context, key string) (*driverpay.EscrowAccount, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, *driverpay.EscrowAccount] {
		return s.escrowAccountByID
	})
}

func (f *EscrowAccountByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.escrowAccountByID = newEscrowAccountByIDLoader(f, tenantInfo)
	})
}

func SettlementDisputesBySettlementID(
	ctx context.Context,
	key string,
) ([]*driversettlement.Dispute, error) {
	return load(ctx, key, func(s *set) *dataloadgen.Loader[string, []*driversettlement.Dispute] {
		return s.settlementDisputesBySettlementID
	})
}

func (f *SettlementDisputesBySettlementIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.settlementDisputesBySettlementID = newSettlementDisputesBySettlementIDLoader(
			f,
			tenantInfo,
		)
	})
}

func CarrierInvoiceMatchesByMoveIDs(
	ctx context.Context,
	keys []string,
) ([][]*carriersettlement.InvoiceMatch, error) {
	return loadAll(
		ctx,
		keys,
		func(s *set) *dataloadgen.Loader[string, []*carriersettlement.InvoiceMatch] {
			return s.carrierInvoiceMatchesByMoveID
		},
	)
}

func (f *CarrierInvoiceMatchesByMoveIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return attach(ctx, func(s *set) {
		s.carrierInvoiceMatchesByMoveID = newCarrierInvoiceMatchesByMoveIDLoader(f, tenantInfo)
	})
}
