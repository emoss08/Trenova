package agentsubjectrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight/insightdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary/formulalibrarydb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentSubjectRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-subject-repository"),
	}
}

// subjectExists asks whether the subject's record exists in the tenant on the
// context; each table answers through its own generated handle.
type subjectExists func(id pulid.ID, found *bool) keel.BatchItem

var subjectTables = map[agent.SubjectType]subjectExists{
	agent.SubjectBillingQueueItem: func(id pulid.ID, found *bool) keel.BatchItem {
		t := billingqueuedb.BillingQueueItems
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectShipmentMove: func(id pulid.ID, found *bool) keel.BatchItem {
		t := shipmentdb.ShipmentMoves
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectAssistantThread: func(id pulid.ID, found *bool) keel.BatchItem {
		t := conversationdb.Threads
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectShipment: func(id pulid.ID, found *bool) keel.BatchItem {
		t := shipmentdb.Shipments
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectDocument: func(id pulid.ID, found *bool) keel.BatchItem {
		t := documentdb.Documents
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectInsight: func(id pulid.ID, found *bool) keel.BatchItem {
		t := insightdb.Insights
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectBankReceipt: func(id pulid.ID, found *bool) keel.BatchItem {
		t := bankreceiptdb.BankReceipts
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectDetentionOccurrence: func(id pulid.ID, found *bool) keel.BatchItem {
		t := detentiondb.DetentionOccurrences
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectWorker: func(id pulid.ID, found *bool) keel.BatchItem {
		t := workerdb.Workers
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectCarrierIntelEvent: func(id pulid.ID, found *bool) keel.BatchItem {
		t := carrierinteldb.CarrierIntelEvents
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectEDIInboundFile: func(id pulid.ID, found *bool) keel.BatchItem {
		t := edidb.EDIInboundFiles
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectInboundMessage: func(id pulid.ID, found *bool) keel.BatchItem {
		t := inboundmessagedb.InboundMessages
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectReport: func(id pulid.ID, found *bool) keel.BatchItem {
		t := reportdb.ReportDefinitions
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectDashboard: func(id pulid.ID, found *bool) keel.BatchItem {
		t := reportdb.Dashboards
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectAccountingConnection: func(id pulid.ID, found *bool) keel.BatchItem {
		t := accountingsyncdb.AccountingConnections
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectAccountingSyncRecord: func(id pulid.ID, found *bool) keel.BatchItem {
		t := accountingsyncdb.AccountingSyncRecords
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectAccountingInbound: func(id pulid.ID, found *bool) keel.BatchItem {
		t := accountingsyncdb.AccountingInboundChanges
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectAccountingDrift: func(id pulid.ID, found *bool) keel.BatchItem {
		t := accountingsyncdb.AccountingDriftFindings
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectFormulaTemplate: func(id pulid.ID, found *bool) keel.BatchItem {
		t := formulatemplatedb.FormulaTemplates
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectFormulaLibrary: func(id pulid.ID, found *bool) keel.BatchItem {
		t := formulalibrarydb.FormulaLibraries
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectInvoice: func(id pulid.ID, found *bool) keel.BatchItem {
		t := invoicedb.Invoices
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectInvoiceDispute: func(id pulid.ID, found *bool) keel.BatchItem {
		t := invoicedb.InvoiceDisputes
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectFactoringNoa: func(id pulid.ID, found *bool) keel.BatchItem {
		t := factoringdb.Noas
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectFactoringSubmission: func(id pulid.ID, found *bool) keel.BatchItem {
		t := factoringdb.Submissions
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
	agent.SubjectFactoringRemittance: func(id pulid.ID, found *bool) keel.BatchItem {
		t := factoringdb.Remittances
		return keel.ExistsInto(t.Where(t.ID.Eq(id)), found)
	},
}

func (r *repository) Exists(
	ctx context.Context,
	req agentports.AgentSubjectExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		if req.SubjectID.IsNil() {
			return false, nil
		}
		if req.SubjectType == agent.SubjectOrganization {
			return req.SubjectID == req.TenantInfo.OrgID, nil
		}

		exists, ok := subjectTables[req.SubjectType]
		if !ok {
			return false, fmt.Errorf("no table holds subject type %q", req.SubjectType)
		}

		var found bool
		if err := keel.Batch(ctx, r.db.Keel(ctx), exists(req.SubjectID, &found)); err != nil {
			r.l.Error(
				"failed to check agent subject",
				zap.String("subjectType", string(req.SubjectType)),
				zap.Error(err),
			)
			return false, err
		}

		return found, nil
	})
}
