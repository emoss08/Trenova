package agentsubjectrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.AgentSubjectRepository {
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
		tenant: buncolgen.BillingQueueItemApplyTenant,
		id:     buncolgen.BillingQueueItemColumns.ID,
	},
	agent.SubjectShipmentMove: {
		model:  func() any { return (*shipment.ShipmentMove)(nil) },
		tenant: buncolgen.ShipmentMoveApplyTenant,
		id:     buncolgen.ShipmentMoveColumns.ID,
	},
	agent.SubjectAssistantThread: {
		model:  func() any { return (*conversation.Thread)(nil) },
		tenant: buncolgen.ThreadApplyTenant,
		id:     buncolgen.ThreadColumns.ID,
	},
	agent.SubjectShipment: {
		model:  func() any { return (*shipment.Shipment)(nil) },
		tenant: buncolgen.ShipmentApplyTenant,
		id:     buncolgen.ShipmentColumns.ID,
	},
	agent.SubjectDocument: {
		model:  func() any { return (*document.Document)(nil) },
		tenant: buncolgen.DocumentApplyTenant,
		id:     buncolgen.DocumentColumns.ID,
	},
	agent.SubjectInsight: {
		model:  func() any { return (*insight.Insight)(nil) },
		tenant: buncolgen.InsightApplyTenant,
		id:     buncolgen.InsightColumns.ID,
	},
	agent.SubjectBankReceipt: {
		model:  func() any { return (*bankreceipt.BankReceipt)(nil) },
		tenant: buncolgen.BankReceiptApplyTenant,
		id:     buncolgen.BankReceiptColumns.ID,
	},
	agent.SubjectDetentionOccurrence: {
		model:  func() any { return (*detention.DetentionOccurrence)(nil) },
		tenant: buncolgen.DetentionOccurrenceApplyTenant,
		id:     buncolgen.DetentionOccurrenceColumns.ID,
	},
	agent.SubjectWorker: {
		model:  func() any { return (*worker.Worker)(nil) },
		tenant: buncolgen.WorkerApplyTenant,
		id:     buncolgen.WorkerColumns.ID,
	},
	agent.SubjectCarrierIntelEvent: {
		model:  func() any { return (*carrierintel.CarrierIntelEvent)(nil) },
		tenant: buncolgen.CarrierIntelEventApplyTenant,
		id:     buncolgen.CarrierIntelEventColumns.ID,
	},
	agent.SubjectEDIInboundFile: {
		model:  func() any { return (*edi.EDIInboundFile)(nil) },
		tenant: buncolgen.EDIInboundFileApplyTenant,
		id:     buncolgen.EDIInboundFileColumns.ID,
	},
	agent.SubjectInboundMessage: {
		model:  func() any { return (*inboundmessage.InboundMessage)(nil) },
		tenant: buncolgen.InboundMessageApplyTenant,
		id:     buncolgen.InboundMessageColumns.ID,
	},
	agent.SubjectReport: {
		model:  func() any { return (*report.ReportDefinition)(nil) },
		tenant: buncolgen.ReportDefinitionApplyTenant,
		id:     buncolgen.ReportDefinitionColumns.ID,
	},
	agent.SubjectDashboard: {
		model:  func() any { return (*report.Dashboard)(nil) },
		tenant: buncolgen.DashboardApplyTenant,
		id:     buncolgen.DashboardColumns.ID,
	},
	agent.SubjectFormulaTemplate: {
		model:  func() any { return (*formulatemplate.FormulaTemplate)(nil) },
		tenant: buncolgen.FormulaTemplateApplyTenant,
		id:     buncolgen.FormulaTemplateColumns.ID,
	},
}

func (r *repository) Exists(
	ctx context.Context,
	req repositories.AgentSubjectExistsRequest,
) (bool, error) {
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
}
