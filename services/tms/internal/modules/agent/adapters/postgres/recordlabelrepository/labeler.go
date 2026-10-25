package recordlabelrepository

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge/accessorialchargedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const MaxIDsPerResource = agentports.MaxRecordLabelsPerResource

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
	ID    pulid.ID
	Label string
}

type ownedRow struct {
	ID      pulid.ID
	OwnerID pulid.ID
	Label   string
}

type moveRow struct {
	ID         pulid.ID
	ShipmentID pulid.ID
	Sequence   int64
}

type labelReader func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error)

type projection interface {
	keel.Projection
	ToSQL(ctx context.Context, db *keel.DB) (string, []any, error)
}

type source func(ids []pulid.ID) projection

type ownedSource struct {
	query source
	owner permission.Resource
}

func labelText(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

//nolint:exhaustive // only the resources whose records a preview names by label are listed
var sources = map[permission.Resource]source{
	permission.ResourceShipment: func(ids []pulid.ID) projection {
		t := shipmentdb.Shipments
		return t.Pick(t.ID, labelText(t.ProNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceCustomer: func(ids []pulid.ID) projection {
		t := customerdb.Customers
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceLocation: func(ids []pulid.ID) projection {
		t := locationdb.Locations
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceWorker: func(ids []pulid.ID) projection {
		t := workerdb.Workers
		return t.Pick(t.ID, labelText(keel.ConcatWS(" ", t.FirstName, t.LastName))).
			Where(t.ID.In(ids...))
	},
	permission.ResourceTractor: func(ids []pulid.ID) projection {
		t := tractordb.Tractors
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceTrailer: func(ids []pulid.ID) projection {
		t := trailerdb.Trailers
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceCarrier: func(ids []pulid.ID) projection {
		t := carrierdb.Carriers
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceCommodity: func(ids []pulid.ID) projection {
		t := commoditydb.Commodities
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceEquipmentType: func(ids []pulid.ID) projection {
		t := equipmenttypedb.EquipmentTypes
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceFleetCode: func(ids []pulid.ID) projection {
		t := fleetcodedb.FleetCodes
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceAccessorialCharge: func(ids []pulid.ID) projection {
		t := accessorialchargedb.AccessorialCharges
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceFactoringCompany: func(ids []pulid.ID) projection {
		t := factoringcompanydb.FactoringCompanies
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceFactoringSubmission: func(ids []pulid.ID) projection {
		t := factoringdb.Submissions
		return t.Pick(t.ID, labelText(t.Number)).Where(t.ID.In(ids...))
	},
	permission.ResourceInvoice: func(ids []pulid.ID) projection {
		t := invoicedb.Invoices
		return t.Pick(t.ID, labelText(t.Number)).Where(t.ID.In(ids...))
	},
	permission.ResourceDocument: func(ids []pulid.ID) projection {
		t := documentdb.Documents
		return t.Pick(t.ID, labelText(t.OriginalName.NullIf("").Coalesce(t.FileName))).
			Where(t.ID.In(ids...))
	},
	permission.ResourceDashboard: func(ids []pulid.ID) projection {
		t := reportdb.Dashboards
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceReport: func(ids []pulid.ID) projection {
		t := reportdb.ReportDefinitions
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceBillingQueue: func(ids []pulid.ID) projection {
		t := billingqueuedb.BillingQueueItems
		return t.Pick(t.ID, labelText(t.Number)).Where(t.ID.In(ids...))
	},
	permission.ResourceDriverSettlement: func(ids []pulid.ID) projection {
		t := driversettlementdb.Settlements
		return t.Pick(t.ID, labelText(t.SettlementNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceCarrierSettlement: func(ids []pulid.ID) projection {
		t := carriersettlementdb.CarrierSettlements
		return t.Pick(t.ID, labelText(t.SettlementNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceBankReceipt: func(ids []pulid.ID) projection {
		t := bankreceiptdb.BankReceipts
		return t.Pick(t.ID, labelText(t.ReferenceNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceInsight: func(ids []pulid.ID) projection {
		t := insightdb.Insights
		return t.Pick(t.ID, labelText(t.Headline)).Where(t.ID.In(ids...))
	},
	permission.ResourceInboundMessage: func(ids []pulid.ID) projection {
		t := inboundmessagedb.InboundMessages
		return t.Pick(t.ID, labelText(t.Subject)).Where(t.ID.In(ids...))
	},
	permission.ResourceServiceFailure: func(ids []pulid.ID) projection {
		t := servicefailuredb.ServiceFailures
		return t.Pick(t.ID, labelText(t.Number)).Where(t.ID.In(ids...))
	},
	permission.ResourceHoldReason: func(ids []pulid.ID) projection {
		t := holdreasondb.HoldReasons
		return t.Pick(t.ID, labelText(t.Label)).Where(t.ID.In(ids...))
	},
	permission.ResourceCustomerPayment: func(ids []pulid.ID) projection {
		t := customerpaymentdb.Payments
		paidOn := keel.ToTimestamp(t.PaymentDate).AtTimeZone(keel.Text("UTC")).ToChar("YYYY-MM-DD")
		label := t.ReferenceNumber.
			NullIf("").
			Coalesce(keel.DynLit("Payment of ").Cast(keel.TypeString).Concat(paidOn))
		return t.Pick(t.ID, labelText(label)).
			Where(t.ID.In(ids...))
	},
	permission.ResourceDocumentType: func(ids []pulid.ID) projection {
		t := documenttypedb.DocumentTypes
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceEquipmentManufacturer: func(ids []pulid.ID) projection {
		t := equipmentmanufacturerdb.EquipmentManufacturers
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceFiscalPeriod: func(ids []pulid.ID) projection {
		t := fiscalperioddb.FiscalPeriods
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceFormulaTemplate: func(ids []pulid.ID) projection {
		t := formulatemplatedb.FormulaTemplates
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceGeneralLedgerAccount: func(ids []pulid.ID) projection {
		t := glaccountdb.GLAccounts
		return t.Pick(t.ID, labelText(keel.ConcatWS(" ", t.AccountCode, t.Name))).
			Where(t.ID.In(ids...))
	},
	permission.ResourceHazardousMaterial: func(ids []pulid.ID) projection {
		t := hazardousmaterialdb.HazardousMaterials
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceJournalEntry: func(ids []pulid.ID) projection {
		t := journalentrydb.JournalEntries
		return t.Pick(t.ID, labelText(t.EntryNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceLocationCategory: func(ids []pulid.ID) projection {
		t := locationcategorydb.LocationCategories
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceOrder: func(ids []pulid.ID) projection {
		t := orderdb.Orders
		return t.Pick(t.ID, labelText(t.OrderNumber)).Where(t.ID.In(ids...))
	},
	permission.ResourceRoutingGuide: func(ids []pulid.ID) projection {
		t := tenderdb.RoutingGuides
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceServiceFailureReasonCode: func(ids []pulid.ID) projection {
		t := servicefailuredb.ReasonCodes
		return t.Pick(t.ID, labelText(t.Label)).Where(t.ID.In(ids...))
	},
	permission.ResourceServiceType: func(ids []pulid.ID) projection {
		t := servicetypedb.ServiceTypes
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceShipmentType: func(ids []pulid.ID) projection {
		t := shipmenttypedb.ShipmentTypes
		return t.Pick(t.ID, labelText(t.Code)).Where(t.ID.In(ids...))
	},
	permission.ResourceShiftTemplate: func(ids []pulid.ID) projection {
		t := workerdb.ShiftTemplates
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceTrainingCourse: func(ids []pulid.ID) projection {
		t := workerdb.TrainingCourses
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceWorkerChecklistTemplate: func(ids []pulid.ID) projection {
		t := workerdb.WorkerChecklistTemplates
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
	permission.ResourceWorkerCredentialType: func(ids []pulid.ID) projection {
		t := workerdb.WorkerCredentialTypes
		return t.Pick(t.ID, labelText(t.Name)).Where(t.ID.In(ids...))
	},
}

//nolint:exhaustive // only the resources whose records a preview names by label are listed
var ownedSources = map[permission.Resource]ownedSource{
	permission.ResourceTender: {
		query: func(ids []pulid.ID) projection {
			t := tenderdb.TenderOffers
			return t.Pick(t.ID, t.CarrierID, keel.Text("")).Where(t.ID.In(ids...))
		},
		owner: permission.ResourceCarrier,
	},
	permission.ResourceFactoringAgreement: {
		query: func(ids []pulid.ID) projection {
			t := factoringdb.Agreements
			return t.Pick(t.ID, t.FactoringCompanyID, labelText(t.Status)).Where(t.ID.In(ids...))
		},
		owner: permission.ResourceFactoringCompany,
	},
	permission.ResourceWorkerLeave: {
		query: func(ids []pulid.ID) projection {
			t := workerdb.WorkerLeaveCases
			return t.Pick(t.ID, t.WorkerID, labelText(t.LeaveType)).Where(t.ID.In(ids...))
		},
		owner: permission.ResourceWorker,
	},
	permission.ResourceWorkerPTO: {
		query: func(ids []pulid.ID) projection {
			t := workerdb.WorkerPTOs
			return t.Pick(t.ID, t.WorkerID, labelText(t.Type)).Where(t.ID.In(ids...))
		},
		owner: permission.ResourceWorker,
	},
	permission.ResourceWorkerSafetyEvent: {
		query: func(ids []pulid.ID) projection {
			t := workerdb.WorkerSafetyEvents
			return t.Pick(t.ID, t.WorkerID, labelText(t.Kind)).Where(t.ID.In(ids...))
		},
		owner: permission.ResourceWorker,
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
			_ pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]string, error) {
			return l.read(ctx, src, ids)
		}
	}
	if src, ok := ownedSources[resource]; ok {
		return func(
			ctx context.Context,
			_ pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]string, error) {
			return l.readOwned(ctx, &src, ids)
		}
	}

	return nil
}

func (l *Labeler) read(
	ctx context.Context,
	src source,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows, err := keel.Into[labelRow](ctx, l.db.Keel(ctx), src(ids))
		if err != nil {
			return nil, err
		}

		return collect(rows), nil
	})
}

func (l *Labeler) readOwned(
	ctx context.Context,
	src *ownedSource,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows, err := keel.Into[ownedRow](ctx, l.db.Keel(ctx), src.query(ids))
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return map[pulid.ID]string{}, nil
		}

		ownerIDs := make([]pulid.ID, 0, len(rows))
		for i := range rows {
			ownerIDs = append(ownerIDs, rows[i].OwnerID)
		}
		owners, err := l.read(ctx, sources[src.owner], distinctIDs(ownerIDs))
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

func (l *Labeler) userLabels(
	ctx context.Context,
	ti pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		rows, err := keel.Into[labelRow](ctx, l.db.Keel(ctx), userQuery(ti, ids))
		if err != nil {
			return nil, err
		}

		return collect(rows), nil
	})
}

func userQuery(ti pagination.TenantInfo, ids []pulid.ID) projection {
	u := tenantdb.Users
	m := tenantdb.OrganizationMemberships

	return u.Pick(u.ID, labelText(u.Name)).Where(
		u.ID.In(ids...),
		keel.Or(
			keel.Exists(m.Where(m.UserID.EqCol(u.ID), m.OrganizationID.Eq(ti.OrgID))),
			u.CurrentOrganizationID.Eq(ti.OrgID),
		),
	)
}

func movesQuery(ids []pulid.ID) projection {
	sm := shipmentdb.ShipmentMoves

	return sm.Pick(sm.ID, sm.ShipmentID, sm.Sequence).Where(sm.ID.In(ids...))
}

func (l *Labeler) moveLabels(
	ctx context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Write(ctx, l.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		moves, err := keel.Into[moveRow](ctx, l.db.Keel(ctx), movesQuery(ids))
		if err != nil {
			return nil, err
		}
		if len(moves) == 0 {
			return map[pulid.ID]string{}, nil
		}

		shipmentIDs := make([]pulid.ID, 0, len(moves))
		for _, move := range moves {
			shipmentIDs = append(shipmentIDs, move.ShipmentID)
		}
		pros, err := l.read(ctx, sources[permission.ResourceShipment], distinctIDs(shipmentIDs))
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
