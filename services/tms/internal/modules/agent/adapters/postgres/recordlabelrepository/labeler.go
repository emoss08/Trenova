// Package recordlabelrepository reads the words records are known by, for a
// preview that names a customer rather than its id. One query per resource,
// inside the tenant, through the caller's transaction when there is one.
package recordlabelrepository

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/accessorialchargecol"
	"github.com/emoss08/trenova/pkg/buncolgen/bankreceiptcol"
	"github.com/emoss08/trenova/pkg/buncolgen/billingqueuecol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/carriersettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/commoditycol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customerpaymentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/driversettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/equipmentmanufacturercol"
	"github.com/emoss08/trenova/pkg/buncolgen/equipmenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"
	"github.com/emoss08/trenova/pkg/buncolgen/fiscalperiodcol"
	"github.com/emoss08/trenova/pkg/buncolgen/fleetcodecol"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/glaccountcol"
	"github.com/emoss08/trenova/pkg/buncolgen/hazardousmaterialcol"
	"github.com/emoss08/trenova/pkg/buncolgen/holdreasoncol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/insightcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/journalentrycol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcategorycol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/ordercol"
	"github.com/emoss08/trenova/pkg/buncolgen/reportcol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicefailurecol"
	"github.com/emoss08/trenova/pkg/buncolgen/servicetypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tendercol"
	"github.com/emoss08/trenova/pkg/buncolgen/tractorcol"
	"github.com/emoss08/trenova/pkg/buncolgen/trailercol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	tenantdomain "github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// MaxIDsPerResource bounds one resource's query: every record the longest
// record subset may offer a person to untick. A preview names a few dozen;
// anything past this is left unlabelled.
const MaxIDsPerResource = agentports.MaxRecordLabelsPerResource

const labelAlias = " AS label"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Labeler struct {
	db *postgres.Connection
}

func New(p Params) agentports.RecordLabeler {
	return &Labeler{db: p.DB}
}

type labelRow struct {
	ID    pulid.ID `bun:"id"`
	Label string   `bun:"label"`
}

type ownedRow struct {
	ID      pulid.ID `bun:"id"`
	OwnerID pulid.ID `bun:"owner_id"`
	Label   string   `bun:"label"`
}

type labelReader func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error)

// source is one resource's label: which table, how it is scoped to the
// tenant, which column is the id and what expression names a row.
type source struct {
	model func() any
	scope func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery
	id    buncolgen.Column
	label string
}

// ownedSource is a record named by its owner and, when label is set, what it
// is of that owner.
type ownedSource struct {
	source
	ownerID buncolgen.Column
	owner   permission.Resource
}

//nolint:exhaustive // only the resources whose records a preview names by label are listed
var sources = map[permission.Resource]source{
	permission.ResourceShipment: {
		model: func() any { return (*shipment.Shipment)(nil) },
		scope: shipmentcol.ShipmentScopeTenant,
		id:    shipmentcol.ShipmentColumns.ID,
		label: shipmentcol.ShipmentColumns.ProNumber.Qualified(),
	},
	permission.ResourceCustomer: {
		model: func() any { return (*customer.Customer)(nil) },
		scope: customercol.CustomerScopeTenant,
		id:    customercol.CustomerColumns.ID,
		label: customercol.CustomerColumns.Name.Qualified(),
	},
	permission.ResourceLocation: {
		model: func() any { return (*location.Location)(nil) },
		scope: locationcol.LocationScopeTenant,
		id:    locationcol.LocationColumns.ID,
		label: locationcol.LocationColumns.Name.Qualified(),
	},
	permission.ResourceWorker: {
		model: func() any { return (*worker.Worker)(nil) },
		scope: workercol.WorkerScopeTenant,
		id:    workercol.WorkerColumns.ID,
		label: buncolgen.Expr(
			"CONCAT_WS(' ', {0}, {1})",
			workercol.WorkerColumns.FirstName,
			workercol.WorkerColumns.LastName,
		),
	},
	permission.ResourceTractor: {
		model: func() any { return (*tractor.Tractor)(nil) },
		scope: tractorcol.TractorScopeTenant,
		id:    tractorcol.TractorColumns.ID,
		label: tractorcol.TractorColumns.Code.Qualified(),
	},
	permission.ResourceTrailer: {
		model: func() any { return (*trailer.Trailer)(nil) },
		scope: trailercol.TrailerScopeTenant,
		id:    trailercol.TrailerColumns.ID,
		label: trailercol.TrailerColumns.Code.Qualified(),
	},
	permission.ResourceCarrier: {
		model: func() any { return (*carrier.Carrier)(nil) },
		scope: carriercol.CarrierScopeTenant,
		id:    carriercol.CarrierColumns.ID,
		label: carriercol.CarrierColumns.Name.Qualified(),
	},
	permission.ResourceCommodity: {
		model: func() any { return (*commodity.Commodity)(nil) },
		scope: commoditycol.CommodityScopeTenant,
		id:    commoditycol.CommodityColumns.ID,
		label: commoditycol.CommodityColumns.Name.Qualified(),
	},
	permission.ResourceEquipmentType: {
		model: func() any { return (*equipmenttype.EquipmentType)(nil) },
		scope: equipmenttypecol.EquipmentTypeScopeTenant,
		id:    equipmenttypecol.EquipmentTypeColumns.ID,
		label: equipmenttypecol.EquipmentTypeColumns.Code.Qualified(),
	},
	permission.ResourceFleetCode: {
		model: func() any { return (*fleetcode.FleetCode)(nil) },
		scope: fleetcodecol.FleetCodeScopeTenant,
		id:    fleetcodecol.FleetCodeColumns.ID,
		label: fleetcodecol.FleetCodeColumns.Code.Qualified(),
	},
	permission.ResourceAccessorialCharge: {
		model: func() any { return (*accessorialcharge.AccessorialCharge)(nil) },
		scope: accessorialchargecol.AccessorialChargeScopeTenant,
		id:    accessorialchargecol.AccessorialChargeColumns.ID,
		label: accessorialchargecol.AccessorialChargeColumns.Code.Qualified(),
	},
	permission.ResourceFactoringCompany: {
		model: func() any { return (*factoringcompany.FactoringCompany)(nil) },
		scope: factoringcompanycol.FactoringCompanyScopeTenant,
		id:    factoringcompanycol.FactoringCompanyColumns.ID,
		label: factoringcompanycol.FactoringCompanyColumns.Name.Qualified(),
	},
	permission.ResourceFactoringSubmission: {
		model: func() any { return (*factoring.Submission)(nil) },
		scope: factoringcol.SubmissionScopeTenant,
		id:    factoringcol.SubmissionColumns.ID,
		label: factoringcol.SubmissionColumns.Number.Qualified(),
	},
	permission.ResourceInvoice: {
		model: func() any { return (*invoice.Invoice)(nil) },
		scope: invoicecol.InvoiceScopeTenant,
		id:    invoicecol.InvoiceColumns.ID,
		label: invoicecol.InvoiceColumns.Number.Qualified(),
	},
	permission.ResourceDocument: {
		model: func() any { return (*document.Document)(nil) },
		scope: documentcol.DocumentScopeTenant,
		id:    documentcol.DocumentColumns.ID,
		label: buncolgen.Expr(
			"COALESCE(NULLIF({0}, ''), {1})",
			documentcol.DocumentColumns.OriginalName,
			documentcol.DocumentColumns.FileName,
		),
	},
	permission.ResourceDashboard: {
		model: func() any { return (*report.Dashboard)(nil) },
		scope: reportcol.DashboardScopeTenant,
		id:    reportcol.DashboardColumns.ID,
		label: reportcol.DashboardColumns.Name.Qualified(),
	},
	permission.ResourceReport: {
		model: func() any { return (*report.ReportDefinition)(nil) },
		scope: reportcol.ReportDefinitionScopeTenant,
		id:    reportcol.ReportDefinitionColumns.ID,
		label: reportcol.ReportDefinitionColumns.Name.Qualified(),
	},
	permission.ResourceBillingQueue: {
		model: func() any { return (*billingqueue.BillingQueueItem)(nil) },
		scope: billingqueuecol.BillingQueueItemScopeTenant,
		id:    billingqueuecol.BillingQueueItemColumns.ID,
		label: billingqueuecol.BillingQueueItemColumns.Number.Qualified(),
	},
	permission.ResourceDriverSettlement: {
		model: func() any { return (*driversettlement.Settlement)(nil) },
		scope: driversettlementcol.SettlementScopeTenant,
		id:    driversettlementcol.SettlementColumns.ID,
		label: driversettlementcol.SettlementColumns.SettlementNumber.Qualified(),
	},
	permission.ResourceCarrierSettlement: {
		model: func() any { return (*carriersettlement.CarrierSettlement)(nil) },
		scope: carriersettlementcol.CarrierSettlementScopeTenant,
		id:    carriersettlementcol.CarrierSettlementColumns.ID,
		label: carriersettlementcol.CarrierSettlementColumns.SettlementNumber.Qualified(),
	},
	permission.ResourceBankReceipt: {
		model: func() any { return (*bankreceipt.BankReceipt)(nil) },
		scope: bankreceiptcol.BankReceiptScopeTenant,
		id:    bankreceiptcol.BankReceiptColumns.ID,
		label: bankreceiptcol.BankReceiptColumns.ReferenceNumber.Qualified(),
	},
	permission.ResourceInsight: {
		model: func() any { return (*insight.Insight)(nil) },
		scope: insightcol.InsightScopeTenant,
		id:    insightcol.InsightColumns.ID,
		label: insightcol.InsightColumns.Headline.Qualified(),
	},
	permission.ResourceInboundMessage: {
		model: func() any { return (*inboundmessage.InboundMessage)(nil) },
		scope: inboundmessagecol.InboundMessageScopeTenant,
		id:    inboundmessagecol.InboundMessageColumns.ID,
		label: inboundmessagecol.InboundMessageColumns.Subject.Qualified(),
	},
	permission.ResourceServiceFailure: {
		model: func() any { return (*servicefailure.ServiceFailure)(nil) },
		scope: servicefailurecol.ServiceFailureScopeTenant,
		id:    servicefailurecol.ServiceFailureColumns.ID,
		label: servicefailurecol.ServiceFailureColumns.Number.Qualified(),
	},
	permission.ResourceHoldReason: {
		model: func() any { return (*holdreason.HoldReason)(nil) },
		scope: holdreasoncol.HoldReasonScopeTenant,
		id:    holdreasoncol.HoldReasonColumns.ID,
		label: holdreasoncol.HoldReasonColumns.Label.Qualified(),
	},
	permission.ResourceCustomerPayment: {
		model: func() any { return (*customerpayment.Payment)(nil) },
		scope: customerpaymentcol.PaymentScopeTenant,
		id:    customerpaymentcol.PaymentColumns.ID,
		label: buncolgen.Expr(
			"COALESCE(NULLIF({0}, ''), 'Payment of ' || "+
				"TO_CHAR(TO_TIMESTAMP({1}) AT TIME ZONE 'UTC', 'YYYY-MM-DD'))",
			customerpaymentcol.PaymentColumns.ReferenceNumber,
			customerpaymentcol.PaymentColumns.PaymentDate,
		),
	},
	permission.ResourceDocumentType: {
		model: func() any { return (*documenttype.DocumentType)(nil) },
		scope: documenttypecol.DocumentTypeScopeTenant,
		id:    documenttypecol.DocumentTypeColumns.ID,
		label: documenttypecol.DocumentTypeColumns.Name.Qualified(),
	},
	permission.ResourceEquipmentManufacturer: {
		model: func() any { return (*equipmentmanufacturer.EquipmentManufacturer)(nil) },
		scope: equipmentmanufacturercol.EquipmentManufacturerScopeTenant,
		id:    equipmentmanufacturercol.EquipmentManufacturerColumns.ID,
		label: equipmentmanufacturercol.EquipmentManufacturerColumns.Name.Qualified(),
	},
	permission.ResourceFiscalPeriod: {
		model: func() any { return (*fiscalperiod.FiscalPeriod)(nil) },
		scope: fiscalperiodcol.FiscalPeriodScopeTenant,
		id:    fiscalperiodcol.FiscalPeriodColumns.ID,
		label: fiscalperiodcol.FiscalPeriodColumns.Name.Qualified(),
	},
	permission.ResourceFormulaTemplate: {
		model: func() any { return (*formulatemplate.FormulaTemplate)(nil) },
		scope: formulatemplatecol.FormulaTemplateScopeTenant,
		id:    formulatemplatecol.FormulaTemplateColumns.ID,
		label: formulatemplatecol.FormulaTemplateColumns.Name.Qualified(),
	},
	permission.ResourceGeneralLedgerAccount: {
		model: func() any { return (*glaccount.GLAccount)(nil) },
		scope: glaccountcol.GLAccountScopeTenant,
		id:    glaccountcol.GLAccountColumns.ID,
		label: buncolgen.Expr(
			"CONCAT_WS(' ', {0}, {1})",
			glaccountcol.GLAccountColumns.AccountCode,
			glaccountcol.GLAccountColumns.Name,
		),
	},
	permission.ResourceHazardousMaterial: {
		model: func() any { return (*hazardousmaterial.HazardousMaterial)(nil) },
		scope: hazardousmaterialcol.HazardousMaterialScopeTenant,
		id:    hazardousmaterialcol.HazardousMaterialColumns.ID,
		label: hazardousmaterialcol.HazardousMaterialColumns.Name.Qualified(),
	},
	permission.ResourceJournalEntry: {
		model: func() any { return (*journalentry.JournalEntry)(nil) },
		scope: journalentrycol.JournalEntryScopeTenant,
		id:    journalentrycol.JournalEntryColumns.ID,
		label: journalentrycol.JournalEntryColumns.EntryNumber.Qualified(),
	},
	permission.ResourceLocationCategory: {
		model: func() any { return (*locationcategory.LocationCategory)(nil) },
		scope: locationcategorycol.LocationCategoryScopeTenant,
		id:    locationcategorycol.LocationCategoryColumns.ID,
		label: locationcategorycol.LocationCategoryColumns.Name.Qualified(),
	},
	permission.ResourceOrder: {
		model: func() any { return (*order.Order)(nil) },
		scope: ordercol.OrderScopeTenant,
		id:    ordercol.OrderColumns.ID,
		label: ordercol.OrderColumns.OrderNumber.Qualified(),
	},
	permission.ResourceRoutingGuide: {
		model: func() any { return (*tender.RoutingGuide)(nil) },
		scope: tendercol.RoutingGuideScopeTenant,
		id:    tendercol.RoutingGuideColumns.ID,
		label: tendercol.RoutingGuideColumns.Name.Qualified(),
	},
	permission.ResourceServiceFailureReasonCode: {
		model: func() any { return (*servicefailure.ReasonCode)(nil) },
		scope: servicefailurecol.ReasonCodeScopeTenant,
		id:    servicefailurecol.ReasonCodeColumns.ID,
		label: servicefailurecol.ReasonCodeColumns.Label.Qualified(),
	},
	permission.ResourceServiceType: {
		model: func() any { return (*servicetype.ServiceType)(nil) },
		scope: servicetypecol.ServiceTypeScopeTenant,
		id:    servicetypecol.ServiceTypeColumns.ID,
		label: servicetypecol.ServiceTypeColumns.Code.Qualified(),
	},
	permission.ResourceShipmentType: {
		model: func() any { return (*shipmenttype.ShipmentType)(nil) },
		scope: shipmenttypecol.ShipmentTypeScopeTenant,
		id:    shipmenttypecol.ShipmentTypeColumns.ID,
		label: shipmenttypecol.ShipmentTypeColumns.Code.Qualified(),
	},
	permission.ResourceShiftTemplate: {
		model: func() any { return (*worker.ShiftTemplate)(nil) },
		scope: workercol.ShiftTemplateScopeTenant,
		id:    workercol.ShiftTemplateColumns.ID,
		label: workercol.ShiftTemplateColumns.Name.Qualified(),
	},
	permission.ResourceTrainingCourse: {
		model: func() any { return (*worker.TrainingCourse)(nil) },
		scope: workercol.TrainingCourseScopeTenant,
		id:    workercol.TrainingCourseColumns.ID,
		label: workercol.TrainingCourseColumns.Name.Qualified(),
	},
	permission.ResourceWorkerChecklistTemplate: {
		model: func() any { return (*worker.WorkerChecklistTemplate)(nil) },
		scope: workercol.WorkerChecklistTemplateScopeTenant,
		id:    workercol.WorkerChecklistTemplateColumns.ID,
		label: workercol.WorkerChecklistTemplateColumns.Name.Qualified(),
	},
	permission.ResourceWorkerCredentialType: {
		model: func() any { return (*worker.WorkerCredentialType)(nil) },
		scope: workercol.WorkerCredentialTypeScopeTenant,
		id:    workercol.WorkerCredentialTypeColumns.ID,
		label: workercol.WorkerCredentialTypeColumns.Name.Qualified(),
	},
}

// ownedSources are records known by the record they belong to and what they
// are of it: a driver's leave case as "Jane Doe · Medical", a tender offer
// by the carrier it went to. The owner is named through sources.
//
//nolint:exhaustive // only the resources whose records a preview names by label are listed
var ownedSources = map[permission.Resource]ownedSource{
	permission.ResourceTender: {
		source: source{
			model: func() any { return (*tender.TenderOffer)(nil) },
			scope: tendercol.TenderOfferScopeTenant,
			id:    tendercol.TenderOfferColumns.ID,
		},
		ownerID: tendercol.TenderOfferColumns.CarrierID,
		owner:   permission.ResourceCarrier,
	},
	permission.ResourceFactoringAgreement: {
		source: source{
			model: func() any { return (*factoring.Agreement)(nil) },
			scope: factoringcol.AgreementScopeTenant,
			id:    factoringcol.AgreementColumns.ID,
			label: factoringcol.AgreementColumns.Status.Qualified(),
		},
		ownerID: factoringcol.AgreementColumns.FactoringCompanyID,
		owner:   permission.ResourceFactoringCompany,
	},
	permission.ResourceWorkerLeave: {
		source: source{
			model: func() any { return (*worker.WorkerLeaveCase)(nil) },
			scope: workercol.WorkerLeaveCaseScopeTenant,
			id:    workercol.WorkerLeaveCaseColumns.ID,
			label: workercol.WorkerLeaveCaseColumns.LeaveType.Qualified(),
		},
		ownerID: workercol.WorkerLeaveCaseColumns.WorkerID,
		owner:   permission.ResourceWorker,
	},
	permission.ResourceWorkerPTO: {
		source: source{
			model: func() any { return (*worker.WorkerPTO)(nil) },
			scope: workercol.WorkerPTOScopeTenant,
			id:    workercol.WorkerPTOColumns.ID,
			label: workercol.WorkerPTOColumns.Type.Qualified(),
		},
		ownerID: workercol.WorkerPTOColumns.WorkerID,
		owner:   permission.ResourceWorker,
	},
	permission.ResourceWorkerSafetyEvent: {
		source: source{
			model: func() any { return (*worker.WorkerSafetyEvent)(nil) },
			scope: workercol.WorkerSafetyEventScopeTenant,
			id:    workercol.WorkerSafetyEventColumns.ID,
			label: workercol.WorkerSafetyEventColumns.Kind.Qualified(),
		},
		ownerID: workercol.WorkerSafetyEventColumns.WorkerID,
		owner:   permission.ResourceWorker,
	},
}

func (l *Labeler) Labels(
	ctx context.Context,
	tenant pagination.TenantInfo,
	refs map[permission.Resource][]pulid.ID,
) (agentports.RecordLabels, error) {
	labels := make(agentports.RecordLabels, len(refs))

	resources := make([]permission.Resource, 0, len(refs))
	for resource := range refs {
		resources = append(resources, resource)
	}
	slices.Sort(resources)

	for _, resource := range resources {
		ids := distinctIDs(refs[resource])
		if len(ids) == 0 {
			continue
		}

		read := l.reader(resource)
		if read == nil {
			continue
		}
		found, err := read(ctx, tenant, ids)
		if err != nil {
			return nil, fmt.Errorf("read %s labels: %w", resource, err)
		}
		if len(found) > 0 {
			labels[resource] = found
		}
	}

	return labels, nil
}

// reader is how one resource's labels are read; nil for a resource with no
// label.
func (l *Labeler) reader(resource permission.Resource) labelReader {
	if resource == permission.ResourceShipmentMove {
		return l.moveLabels
	}
	if resource == permission.ResourceUser {
		return l.userLabels
	}
	if src, ok := sources[resource]; ok {
		return func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]string, error) {
			return l.read(ctx, tenant, &src, ids)
		}
	}
	if src, ok := ownedSources[resource]; ok {
		return func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]string, error) {
			return l.readOwned(ctx, tenant, &src, ids)
		}
	}

	return nil
}

func (l *Labeler) read(
	ctx context.Context,
	tenant pagination.TenantInfo,
	src *source,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows := make([]labelRow, 0, len(ids))
		if err := labelQuery(
			l.db.DBForContext(ctx),
			tenant,
			src,
			ids,
		).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return collect(rows), nil
	})
}

func labelQuery(
	db bun.IDB,
	tenant pagination.TenantInfo,
	src *source,
	ids []pulid.ID,
) *bun.SelectQuery {
	return db.NewSelect().
		Model(src.model()).
		ColumnExpr(src.id.As("id")).
		ColumnExpr(src.label+labelAlias).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return src.scope(q, tenant).Where(src.id.In(), bun.List(ids))
		})
}

// readOwned names each record by its owner's label and its own, "Jane Doe ·
// Medical"; a record whose owner cannot be named keeps its own label alone.
func (l *Labeler) readOwned(
	ctx context.Context,
	tenant pagination.TenantInfo,
	src *ownedSource,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows := make([]ownedRow, 0, len(ids))
		if err := ownedQuery(
			l.db.DBForContext(ctx),
			tenant,
			src,
			ids,
		).Scan(ctx, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return map[pulid.ID]string{}, nil
		}

		ownerIDs := make([]pulid.ID, 0, len(rows))
		for i := range rows {
			ownerIDs = append(ownerIDs, rows[i].OwnerID)
		}
		owner := sources[src.owner]
		owners, err := l.read(ctx, tenant, &owner, distinctIDs(ownerIDs))
		if err != nil {
			return nil, err
		}

		labels := make(map[pulid.ID]string, len(rows))
		for i := range rows {
			if label := joinLabel(owners[rows[i].OwnerID], rows[i].Label); label != "" {
				labels[rows[i].ID] = label
			}
		}

		return labels, nil
	})
}

func ownedQuery(
	db bun.IDB,
	tenant pagination.TenantInfo,
	src *ownedSource,
	ids []pulid.ID,
) *bun.SelectQuery {
	label := src.label
	if label == "" {
		label = "''"
	}

	return db.NewSelect().
		Model(src.model()).
		ColumnExpr(src.id.As("id")).
		ColumnExpr(src.ownerID.As("owner_id")).
		ColumnExpr(label+labelAlias).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return src.scope(q, tenant).Where(src.id.In(), bun.List(ids))
		})
}

func joinLabel(owner, own string) string {
	owner, own = strings.TrimSpace(owner), strings.TrimSpace(own)
	switch {
	case owner == "":
		return own
	case own == "":
		return owner
	default:
		return owner + " · " + own
	}
}

// userLabels names the people of the tenant: a user is not scoped to one
// organization, so only one who holds, or held, a membership in it, or works
// in it now, is named. A membership that has since lapsed still names the
// person on the records they left behind.
func (l *Labeler) userLabels(
	ctx context.Context,
	ti pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows := make([]labelRow, 0, len(ids))
		if err := userQuery(l.db.DBForContext(ctx), ti, ids).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return collect(rows), nil
	})
}

func userQuery(db bun.IDB, ti pagination.TenantInfo, ids []pulid.ID) *bun.SelectQuery {
	user := tenantcol.UserColumns
	membership := tenantcol.OrganizationMembershipColumns

	member := db.NewSelect().
		Model((*tenantdomain.OrganizationMembership)(nil)).
		ColumnExpr("1").
		Where(membership.UserID.EqColumn(user.ID)).
		Where(membership.OrganizationID.Eq(), ti.OrgID).
		Where(membership.BusinessUnitID.Eq(), ti.BuID)

	return db.NewSelect().
		Model((*tenantdomain.User)(nil)).
		ColumnExpr(user.ID.As("id")).
		ColumnExpr(user.Name.Qualified()+labelAlias).
		Where(user.ID.In(), bun.List(ids)).
		Where(user.BusinessUnitID.Eq(), ti.BuID).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("EXISTS (?)", member).
				WhereOr(user.CurrentOrganizationID.Eq(), ti.OrgID)
		})
}

// moveLabels names a move by its shipment and its place in it: "PRO-100
// move 2".
func (l *Labeler) moveLabels(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Write(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		moves := make([]*shipment.ShipmentMove, 0, len(ids))
		cols := shipmentcol.ShipmentMoveColumns
		if err := l.db.DBForContext(ctx).NewSelect().
			Model(&moves).
			Column(cols.ID.String(), cols.ShipmentID.String(), cols.Sequence.String()).
			WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentMoveScopeTenant(q, tenant).
					Where(cols.ID.In(), bun.List(ids))
			}).
			Scan(ctx); err != nil {
			return nil, err
		}
		if len(moves) == 0 {
			return map[pulid.ID]string{}, nil
		}

		shipmentIDs := make([]pulid.ID, 0, len(moves))
		for _, move := range moves {
			shipmentIDs = append(shipmentIDs, move.ShipmentID)
		}
		shipmentSource := sources[permission.ResourceShipment]
		pros, err := l.read(ctx, tenant, &shipmentSource, distinctIDs(shipmentIDs))
		if err != nil {
			return nil, err
		}

		labels := make(map[pulid.ID]string, len(moves))
		for _, move := range moves {
			pro := pros[move.ShipmentID]
			if pro == "" {
				continue
			}
			labels[move.ID] = pro + " move " + strconv.FormatInt(move.Sequence+1, 10)
		}

		return labels, nil
	})
}

func collect(rows []labelRow) map[pulid.ID]string {
	labels := make(map[pulid.ID]string, len(rows))
	for _, row := range rows {
		if label := strings.TrimSpace(row.Label); label != "" {
			labels[row.ID] = label
		}
	}

	return labels
}

func distinctIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, min(len(ids), MaxIDsPerResource))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) == MaxIDsPerResource {
			break
		}
	}

	return out
}
