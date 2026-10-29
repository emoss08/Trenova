package accountingdriftservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweraccounting"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultBatch     = 200
	maxBatch         = 500
	defaultCustomers = 50
	maxCustomers     = 200
)

var permissionResource = permission.ResourceAccountingSync.String()

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                kernelports.Transactor
	Connections       accountingconnectionports.AccountingConnectionRepository
	ConnectionService accountingconnectionports.AccountingConnectionService
	Records           accountingsyncports.AccountingSyncRecordRepository
	Findings          accountingdriftports.AccountingDriftFindingRepository
	Source            accountingdriftports.AccountingDriftSource
	Ledger            journalentryports.AccountingLedgerSource
	Mappings          accountingmappingports.AccountingMappingRepository
	References        accountingmappingports.AccountingReferenceObjectRepository
	Organizations     tenantports.OrganizationRepository
	InvoiceRepo       invoiceports.InvoiceRepository
	Controls          tenantports.AccountingControlRepository
	Invoices          invoiceports.InvoiceService
	Payments          customerpaymentports.CustomerPaymentService
	Permissions       permissionports.PermissionEngine
	AuditService      auditports.AuditService
	Dispatcher        accountingsyncports.AccountingSyncDispatcher `optional:"true"`
	Checker           accountingdriftports.AccountingDriftChecker  `optional:"true"`
	Publisher         agentports.AgentEventPublisher               `optional:"true"`
	Watchtower        watchtowerports.WatchtowerProjector          `optional:"true"`
	Realtime          realtimeports.RealtimeService                `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	connections   accountingconnectionports.AccountingConnectionRepository
	connService   accountingconnectionports.AccountingConnectionService
	records       accountingsyncports.AccountingSyncRecordRepository
	findings      accountingdriftports.AccountingDriftFindingRepository
	source        accountingdriftports.AccountingDriftSource
	ledger        journalentryports.AccountingLedgerSource
	mappings      accountingmappingports.AccountingMappingRepository
	references    accountingmappingports.AccountingReferenceObjectRepository
	organizations tenantports.OrganizationRepository
	invoiceRepo   invoiceports.InvoiceRepository
	controls      tenantports.AccountingControlRepository
	invoices      invoiceports.InvoiceService
	payments      customerpaymentports.CustomerPaymentService
	permissions   permissionports.PermissionEngine
	audit         auditports.AuditService
	dispatcher    accountingsyncports.AccountingSyncDispatcher
	checker       accountingdriftports.AccountingDriftChecker
	publisher     agentports.AgentEventPublisher
	watchtower    watchtowerports.WatchtowerProjector
	realtime      realtimeports.RealtimeService
	now           func() time.Time
}

var (
	_ accountingdriftports.AccountingDriftReconciler = (*Service)(nil)
	_ accountingdriftports.AccountingDriftRechecker  = (*Service)(nil)
	_ accountingdriftports.AccountingDriftService    = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.accounting-drift"),
		db:            p.DB,
		connections:   p.Connections,
		connService:   p.ConnectionService,
		records:       p.Records,
		findings:      p.Findings,
		source:        p.Source,
		ledger:        p.Ledger,
		mappings:      p.Mappings,
		references:    p.References,
		organizations: p.Organizations,
		invoiceRepo:   p.InvoiceRepo,
		controls:      p.Controls,
		invoices:      p.Invoices,
		payments:      p.Payments,
		permissions:   p.Permissions,
		audit:         p.AuditService,
		dispatcher:    p.Dispatcher,
		checker:       p.Checker,
		publisher:     p.Publisher,
		watchtower:    p.Watchtower,
		realtime:      p.Realtime,
		now:           time.Now,
	}
}

type readSession struct {
	tenant pagination.TenantInfo
	conn   *accountingconnection.AccountingConnection
	reader accountingconnectorports.AccountingDocumentReader
	writer accountingconnectorports.AccountingDocumentWriter
	ledger accountingconnectorports.AccountingLedgerReader
	auth   accountingconnectorports.AccountingDocumentAuth
	limit  int
}

func tenantOf(conn *accountingconnection.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (s *Service) connectionByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingconnection.AccountingConnection, error) {
	return s.connections.GetByID(ctx, accountingconnectionports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) openRead(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) (*readSession, error) {
	tenant := tenantOf(conn)
	session, err := s.connService.Session(ctx, tenant, conn.ID)
	if err != nil {
		return nil, err
	}
	reader, ok := session.Connector.(accountingconnectorports.AccountingDocumentReader)
	if !ok {
		return nil, fmt.Errorf(
			"%s cannot show Trenova the documents it holds",
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}
	writer, _ := session.Connector.(accountingconnectorports.AccountingDocumentWriter)
	ledger, _ := session.Connector.(accountingconnectorports.AccountingLedgerReader)
	return &readSession{
		tenant: tenant,
		conn:   session.Connection,
		reader: reader,
		writer: writer,
		ledger: ledger,
		auth:   accountingconnectorports.DocumentAuthFor(session.Connection, session.AccessToken),
		limit:  max(reader.DocumentReadLimits().MaxPerRead, 1),
	}, nil
}

func (s *Service) classify(sess *readSession, readErr error) *accountingsync.SyncError {
	if sess != nil && sess.writer != nil {
		if classified := sess.writer.ClassifyDocumentError(readErr); classified != nil {
			return classified
		}
	}
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorTransient,
		Message:  readErr.Error(),
	}
}

func (s *Service) recordFailure(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	failure *accountingsync.SyncError,
) {
	recorded := *conn
	recorded.RecordDriftFailure(failure.Category, failure.Message)
	if err := s.connections.SaveDriftCheck(
		ctx,
		&accountingconnectionports.SaveAccountingDriftCheckRequest{
			TenantInfo:    tenantOf(conn),
			ID:            conn.ID,
			ErrorCategory: recorded.DriftErrorCategory,
			ErrorMessage:  recorded.DriftErrorMessage,
		},
	); err != nil {
		s.l.Warn("failed to record a drift check failure",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
	}
}

func (s *Service) FinishCheck(
	ctx context.Context,
	req *accountingdriftports.FinishAccountingDriftCheckRequest,
) error {
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return err
	}
	conn.FinishDriftCheck(s.now().Unix())
	if err = s.connections.SaveDriftCheck(ctx, &accountingconnectionports.SaveAccountingDriftCheckRequest{
		TenantInfo:    req.TenantInfo,
		ID:            conn.ID,
		CheckedAt:     conn.DriftCheckedAt,
		ErrorCategory: conn.DriftErrorCategory,
		ErrorMessage:  conn.DriftErrorMessage,
	}); err != nil {
		return err
	}
	s.refreshAttention(ctx, conn)
	s.publishInvalidation(ctx, req.TenantInfo, pulid.Nil, conn.ID)
	return nil
}

func (s *Service) refreshAttention(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if s.watchtower == nil {
		return
	}
	tenant := tenantOf(conn)
	groups, err := s.findings.ListAttention(
		ctx,
		&accountingdriftports.ListAccountingDriftAttentionRequest{
			TenantInfo:     tenant,
			ConnectionID:   conn.ID,
			DetectedBefore: s.now().Add(-watchtoweraccounting.AccountingDriftAfter).Unix(),
		},
	)
	if err != nil {
		s.l.Warn("failed to read accounting drift attention", zap.Error(err))
		return
	}
	byKind := make(
		map[accountingdrift.DriftKind]*accountingdriftports.AccountingDriftAttentionGroup,
		len(groups),
	)
	for idx := range groups {
		byKind[groups[idx].Kind] = &groups[idx]
	}
	for _, kind := range accountingdrift.AllDriftKinds() {
		if item, open := watchtoweraccounting.DescribeAccountingDriftAttention(
			conn,
			byKind[kind],
		); open {
			s.watchtower.Upsert(ctx, item)
			continue
		}
		s.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtoweraccounting.AccountingDriftAttentionSourceID(conn, kind),
		)
	}
}

func (s *Service) refreshAttentionFor(
	ctx context.Context,
	finding *accountingdrift.AccountingDriftFinding,
) {
	conn, err := s.connectionByID(ctx, finding.TenantInfo(), finding.ConnectionID)
	if err != nil {
		s.l.Warn("failed to reload the connection after a drift decision", zap.Error(err))
		return
	}
	s.refreshAttention(ctx, conn)
}

func (s *Service) announce(
	ctx context.Context,
	tenant pagination.TenantInfo,
	opened []*accountingdrift.AccountingDriftFinding,
	budget int,
) int {
	sent := 0
	for _, finding := range opened {
		if sent >= budget {
			break
		}
		agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
			Kind:       agenttrigger.EventAccountingDriftDetected,
			SubjectID:  finding.ID,
			TenantInfo: tenant,
		})
		sent++
	}
	return sent
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
	recordID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    actorID,
		Resource:       permissionResource,
		Action:         "updated",
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish accounting drift invalidation", zap.Error(err))
	}
}
