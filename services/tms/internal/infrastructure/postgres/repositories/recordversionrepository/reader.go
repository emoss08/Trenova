package recordversionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// Params wires the reader.
type Params struct {
	fx.In

	DB *postgres.Connection
}

// Reader answers "what version is this record now" for the records agent
// tools act on.
//
// One reader rather than a method per tool: each tool holds only the narrow
// service it acts through — holds, comments, assignments — and none of those
// hands back a version. The tool names its target; this resolves it. Adding a
// resource is one line in the table below, and a resource not in it is
// reported rather than guessed.
type Reader struct {
	db *postgres.Connection
}

func New(p Params) services.RecordVersionReader {
	return &Reader{db: p.DB}
}

// lookup is one resource's query: which model, how to scope it to the tenant,
// which column is the id, and how to read the version off the row.
type lookup struct {
	model   func() versioned
	scope   func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery
	idEq    string
	version func(versioned) int64
}

type versioned any

var lookups = map[permission.Resource]lookup{
	permission.ResourceShipment: {
		model:   func() versioned { return new(shipment.Shipment) },
		scope:   buncolgen.ShipmentScopeTenant,
		idEq:    buncolgen.ShipmentColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*shipment.Shipment).Version },
	},
	permission.ResourceShipmentMove: {
		model:   func() versioned { return new(shipment.ShipmentMove) },
		scope:   buncolgen.ShipmentMoveScopeTenant,
		idEq:    buncolgen.ShipmentMoveColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*shipment.ShipmentMove).Version },
	},
	permission.ResourceDocument: {
		model:   func() versioned { return new(document.Document) },
		scope:   buncolgen.DocumentScopeTenant,
		idEq:    buncolgen.DocumentColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*document.Document).Version },
	},
	permission.ResourceDashboard: {
		model:   func() versioned { return new(report.Dashboard) },
		scope:   buncolgen.DashboardScopeTenant,
		idEq:    buncolgen.DashboardColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*report.Dashboard).Version },
	},
	permission.ResourceWorkerPTO: {
		model:   func() versioned { return new(worker.WorkerPTO) },
		scope:   buncolgen.WorkerPTOScopeTenant,
		idEq:    buncolgen.WorkerPTOColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*worker.WorkerPTO).Version },
	},
	permission.ResourceBillingQueue: {
		model:   func() versioned { return new(billingqueue.BillingQueueItem) },
		scope:   buncolgen.BillingQueueItemScopeTenant,
		idEq:    buncolgen.BillingQueueItemColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*billingqueue.BillingQueueItem).Version },
	},
	permission.ResourceBankReceipt: {
		model:   func() versioned { return new(bankreceipt.BankReceipt) },
		scope:   buncolgen.BankReceiptScopeTenant,
		idEq:    buncolgen.BankReceiptColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*bankreceipt.BankReceipt).Version },
	},
	permission.ResourceBankReceiptWorkItem: {
		model:   func() versioned { return new(bankreceiptworkitem.WorkItem) },
		scope:   buncolgen.WorkItemScopeTenant,
		idEq:    buncolgen.WorkItemColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*bankreceiptworkitem.WorkItem).Version },
	},
	permission.ResourceInsight: {
		model:   func() versioned { return new(insight.Insight) },
		scope:   buncolgen.InsightScopeTenant,
		idEq:    buncolgen.InsightColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*insight.Insight).Version },
	},
	permission.ResourceInboundMessage: {
		model:   func() versioned { return new(inboundmessage.InboundMessage) },
		scope:   buncolgen.InboundMessageScopeTenant,
		idEq:    buncolgen.InboundMessageColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*inboundmessage.InboundMessage).Version },
	},
	permission.ResourceWorker: {
		model:   func() versioned { return new(worker.Worker) },
		scope:   buncolgen.WorkerScopeTenant,
		idEq:    buncolgen.WorkerColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*worker.Worker).Version },
	},
	permission.ResourceServiceFailure: {
		model:   func() versioned { return new(servicefailure.ServiceFailure) },
		scope:   buncolgen.ServiceFailureScopeTenant,
		idEq:    buncolgen.ServiceFailureColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*servicefailure.ServiceFailure).Version },
	},
	permission.ResourceReport: {
		model:   func() versioned { return new(report.ReportDefinition) },
		scope:   buncolgen.ReportDefinitionScopeTenant,
		idEq:    buncolgen.ReportDefinitionColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*report.ReportDefinition).Version },
	},
	// The detention tools act on an occurrence under the detention policy
	// resource, which is the permission a person needs to act on one.
	permission.ResourceDetentionPolicy: {
		model:   func() versioned { return new(detention.DetentionOccurrence) },
		scope:   buncolgen.DetentionOccurrenceScopeTenant,
		idEq:    buncolgen.DetentionOccurrenceColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*detention.DetentionOccurrence).Version },
	},
	// The sync tools act on one ledger record under the accounting sync
	// resource; its version moves each time it is claimed, sent or failed.
	permission.ResourceAccountingSync: {
		model: func() versioned { return new(accountingsync.AccountingSyncRecord) },
		scope: buncolgen.AccountingSyncRecordScopeTenant,
		idEq:  buncolgen.AccountingSyncRecordColumns.ID.Eq(),
		version: func(v versioned) int64 {
			record, ok := v.(*accountingsync.AccountingSyncRecord)
			if !ok {
				return 0
			}
			return record.Version
		},
	},
	// A tender is withdrawn as a whole; its version moves as the tender
	// advances or ends.
	permission.ResourceTender: {
		model: func() versioned { return new(tender.Tender) },
		scope: buncolgen.TenderScopeTenant,
		idEq:  buncolgen.TenderColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*tender.Tender)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceRateConfirmation: {
		model: func() versioned { return new(rateconfirmation.RateConfirmation) },
		scope: buncolgen.RateConfirmationScopeTenant,
		idEq:  buncolgen.RateConfirmationColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*rateconfirmation.RateConfirmation)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	// An invoice is posted or sent as a whole; its version moves with each
	// change to it, so an approval refuses one edited since it was proposed.
	permission.ResourceInvoice: {
		model: func() versioned { return new(invoice.Invoice) },
		scope: buncolgen.InvoiceScopeTenant,
		idEq:  buncolgen.InvoiceColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*invoice.Invoice)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	// Likewise the carrier intelligence tools act on one event.
	permission.ResourceCarrierIntelligence: {
		model:   func() versioned { return new(carrierintel.CarrierIntelEvent) },
		scope:   buncolgen.CarrierIntelEventScopeTenant,
		idEq:    buncolgen.CarrierIntelEventColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*carrierintel.CarrierIntelEvent).Version },
	},
	permission.ResourceDriverSettlement: {
		model: func() versioned { return new(driversettlement.Settlement) },
		scope: buncolgen.SettlementScopeTenant,
		idEq:  buncolgen.SettlementColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driversettlement.Settlement)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceCarrierSettlement: {
		model: func() versioned { return new(carriersettlement.CarrierSettlement) },
		scope: buncolgen.CarrierSettlementScopeTenant,
		idEq:  buncolgen.CarrierSettlementColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*carriersettlement.CarrierSettlement)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceCarrierInvoiceMatch: {
		model: func() versioned { return new(carriersettlement.InvoiceMatch) },
		scope: buncolgen.InvoiceMatchScopeTenant,
		idEq:  buncolgen.InvoiceMatchColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*carriersettlement.InvoiceMatch)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceEscrowAccount: {
		model: func() versioned { return new(driverpay.EscrowAccount) },
		scope: buncolgen.EscrowAccountScopeTenant,
		idEq:  buncolgen.EscrowAccountColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.EscrowAccount)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourcePayAdvance: {
		model: func() versioned { return new(driverpay.PayAdvance) },
		scope: buncolgen.PayAdvanceScopeTenant,
		idEq:  buncolgen.PayAdvanceColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.PayAdvance)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceRecurringDeduction: {
		model: func() versioned { return new(driverpay.RecurringDeduction) },
		scope: buncolgen.RecurringDeductionScopeTenant,
		idEq:  buncolgen.RecurringDeductionColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.RecurringDeduction)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
	permission.ResourceRecurringEarning: {
		model: func() versioned { return new(driverpay.RecurringEarning) },
		scope: buncolgen.RecurringEarningScopeTenant,
		idEq:  buncolgen.RecurringEarningColumns.ID.Eq(),
		version: func(v versioned) int64 {
			entity, ok := v.(*driverpay.RecurringEarning)
			if !ok {
				return 0
			}
			return entity.Version
		},
	},
}

// Version reads the record's current version inside the tenant. A record that
// is not there is an error, not a zero: a proposal against a deleted record
// must not compare equal to anything.
func (r *Reader) Version(
	ctx context.Context,
	tenant pagination.TenantInfo,
	target services.ToolTarget,
) (int64, error) {
	entry, ok := lookups[target.Resource]
	if !ok {
		return 0, fmt.Errorf("%w: %s", services.ErrRecordVersionUnsupported, target.Resource)
	}

	record := entry.model()
	query := r.db.DBForContext(ctx).NewSelect().Model(record)
	query = entry.scope(query, tenant).Where(entry.idEq, target.ID).Limit(1)
	if err := query.Scan(ctx); err != nil {
		return 0, fmt.Errorf("read %s %s version: %w", target.Resource, target.ID, err)
	}

	return entry.version(record), nil
}
