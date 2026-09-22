package recordversionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
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
	// Likewise the carrier intelligence tools act on one event.
	permission.ResourceCarrierIntelligence: {
		model:   func() versioned { return new(carrierintel.CarrierIntelEvent) },
		scope:   buncolgen.CarrierIntelEventScopeTenant,
		idEq:    buncolgen.CarrierIntelEventColumns.ID.Eq(),
		version: func(v versioned) int64 { return v.(*carrierintel.CarrierIntelEvent).Version },
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
