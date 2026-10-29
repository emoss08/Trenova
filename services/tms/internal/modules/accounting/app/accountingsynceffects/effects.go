package accountingsynceffects

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

const AttentionGroups = 20

type Deps struct {
	Logger        *zap.Logger
	Records       accountingsyncports.AccountingSyncRecordRepository
	Connections   accountingconnectionports.AccountingConnectionRepository
	Organizations tenantports.OrganizationRepository
	Controls      tenantports.AccountingControlRepository
	Audit         auditports.AuditService
	Dispatcher    accountingsyncports.AccountingSyncDispatcher
	Publisher     agentports.AgentEventPublisher
	Watchtower    watchtowerports.WatchtowerProjector
	Realtime      realtimeports.RealtimeService
}

type Effects struct {
	l             *zap.Logger
	records       accountingsyncports.AccountingSyncRecordRepository
	connections   accountingconnectionports.AccountingConnectionRepository
	organizations tenantports.OrganizationRepository
	controls      tenantports.AccountingControlRepository
	audit         auditports.AuditService
	dispatcher    accountingsyncports.AccountingSyncDispatcher
	publisher     agentports.AgentEventPublisher
	watchtower    watchtowerports.WatchtowerProjector
	realtime      realtimeports.RealtimeService
}

//nolint:gocritic // dependency injection
func New(d Deps) *Effects {
	return &Effects{
		l:             d.Logger,
		records:       d.Records,
		connections:   d.Connections,
		organizations: d.Organizations,
		controls:      d.Controls,
		audit:         d.Audit,
		dispatcher:    d.Dispatcher,
		publisher:     d.Publisher,
		watchtower:    d.Watchtower,
		realtime:      d.Realtime,
	}
}
