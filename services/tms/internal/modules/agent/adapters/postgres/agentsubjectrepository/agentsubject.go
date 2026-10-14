package agentsubjectrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"
	"github.com/emoss08/trenova/pkg/buncolgen/bankreceiptcol"
	"github.com/emoss08/trenova/pkg/buncolgen/billingqueuecol"
	"github.com/emoss08/trenova/pkg/buncolgen/carrierintelcol"
	"github.com/emoss08/trenova/pkg/buncolgen/conversationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/detentioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/edicol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/insightcol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/reportcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
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

type subjectTable struct {
	model  func() any
	tenant func(pagination.TenantInfo) func(*bun.SelectQuery) *bun.SelectQuery
	id     buncolgen.Column
}

var subjectTables = map[agent.SubjectType]subjectTable{
	agent.SubjectBillingQueueItem: {
		model:  func() any { return (*billingqueue.BillingQueueItem)(nil) },
		tenant: billingqueuecol.BillingQueueItemApplyTenant,
		id:     billingqueuecol.BillingQueueItemColumns.ID,
	},
	agent.SubjectShipmentMove: {
		model:  func() any { return (*shipment.ShipmentMove)(nil) },
		tenant: shipmentcol.ShipmentMoveApplyTenant,
		id:     shipmentcol.ShipmentMoveColumns.ID,
	},
	agent.SubjectAssistantThread: {
		model:  func() any { return (*conversation.Thread)(nil) },
		tenant: conversationcol.ThreadApplyTenant,
		id:     conversationcol.ThreadColumns.ID,
	},
	agent.SubjectShipment: {
		model:  func() any { return (*shipment.Shipment)(nil) },
		tenant: shipmentcol.ShipmentApplyTenant,
		id:     shipmentcol.ShipmentColumns.ID,
	},
	agent.SubjectDocument: {
		model:  func() any { return (*document.Document)(nil) },
		tenant: documentcol.DocumentApplyTenant,
		id:     documentcol.DocumentColumns.ID,
	},
	agent.SubjectInsight: {
		model:  func() any { return (*insight.Insight)(nil) },
		tenant: insightcol.InsightApplyTenant,
		id:     insightcol.InsightColumns.ID,
	},
	agent.SubjectBankReceipt: {
		model:  func() any { return (*bankreceipt.BankReceipt)(nil) },
		tenant: bankreceiptcol.BankReceiptApplyTenant,
		id:     bankreceiptcol.BankReceiptColumns.ID,
	},
	agent.SubjectDetentionOccurrence: {
		model:  func() any { return (*detention.DetentionOccurrence)(nil) },
		tenant: detentioncol.DetentionOccurrenceApplyTenant,
		id:     detentioncol.DetentionOccurrenceColumns.ID,
	},
	agent.SubjectWorker: {
		model:  func() any { return (*worker.Worker)(nil) },
		tenant: workercol.WorkerApplyTenant,
		id:     workercol.WorkerColumns.ID,
	},
	agent.SubjectCarrierIntelEvent: {
		model:  func() any { return (*carrierintel.CarrierIntelEvent)(nil) },
		tenant: carrierintelcol.CarrierIntelEventApplyTenant,
		id:     carrierintelcol.CarrierIntelEventColumns.ID,
	},
	agent.SubjectEDIInboundFile: {
		model:  func() any { return (*edi.EDIInboundFile)(nil) },
		tenant: edicol.EDIInboundFileApplyTenant,
		id:     edicol.EDIInboundFileColumns.ID,
	},
	agent.SubjectInboundMessage: {
		model:  func() any { return (*inboundmessage.InboundMessage)(nil) },
		tenant: inboundmessagecol.InboundMessageApplyTenant,
		id:     inboundmessagecol.InboundMessageColumns.ID,
	},
	agent.SubjectReport: {
		model:  func() any { return (*report.ReportDefinition)(nil) },
		tenant: reportcol.ReportDefinitionApplyTenant,
		id:     reportcol.ReportDefinitionColumns.ID,
	},
	agent.SubjectDashboard: {
		model:  func() any { return (*report.Dashboard)(nil) },
		tenant: reportcol.DashboardApplyTenant,
		id:     reportcol.DashboardColumns.ID,
	},
	agent.SubjectAccountingConnection: {
		model:  func() any { return (*accountingsync.AccountingConnection)(nil) },
		tenant: accountingsynccol.AccountingConnectionApplyTenant,
		id:     accountingsynccol.AccountingConnectionColumns.ID,
	},
	agent.SubjectAccountingSyncRecord: {
		model:  func() any { return (*accountingsync.AccountingSyncRecord)(nil) },
		tenant: accountingsynccol.AccountingSyncRecordApplyTenant,
		id:     accountingsynccol.AccountingSyncRecordColumns.ID,
	},
	agent.SubjectAccountingInbound: {
		model:  func() any { return (*accountingsync.AccountingInboundChange)(nil) },
		tenant: accountingsynccol.AccountingInboundChangeApplyTenant,
		id:     accountingsynccol.AccountingInboundChangeColumns.ID,
	},
	agent.SubjectAccountingDrift: {
		model:  func() any { return (*accountingsync.AccountingDriftFinding)(nil) },
		tenant: accountingsynccol.AccountingDriftFindingApplyTenant,
		id:     accountingsynccol.AccountingDriftFindingColumns.ID,
	},
	agent.SubjectFormulaTemplate: {
		model:  func() any { return (*formulatemplate.FormulaTemplate)(nil) },
		tenant: formulatemplatecol.FormulaTemplateApplyTenant,
		id:     formulatemplatecol.FormulaTemplateColumns.ID,
	},
	agent.SubjectInvoice: {
		model:  func() any { return (*invoice.Invoice)(nil) },
		tenant: invoicecol.InvoiceApplyTenant,
		id:     invoicecol.InvoiceColumns.ID,
	},
	agent.SubjectInvoiceDispute: {
		model:  func() any { return (*invoice.InvoiceDispute)(nil) },
		tenant: invoicecol.InvoiceDisputeApplyTenant,
		id:     invoicecol.InvoiceDisputeColumns.ID,
	},
	agent.SubjectFactoringNoa: {
		model:  func() any { return (*factoring.Noa)(nil) },
		tenant: factoringcol.NoaApplyTenant,
		id:     factoringcol.NoaColumns.ID,
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

		table, ok := subjectTables[req.SubjectType]
		if !ok {
			return false, fmt.Errorf("no table holds subject type %q", req.SubjectType)
		}

		exists, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(table.model()).
			Apply(table.tenant(req.TenantInfo)).
			Where(table.id.Eq(), req.SubjectID).
			Exists(ctx)
		if err != nil {
			r.l.Error(
				"failed to check agent subject",
				zap.String("subjectType", string(req.SubjectType)),
				zap.Error(err),
			)
			return false, err
		}

		return exists, nil
	})
}
