package accountingdriftservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	DB                ports.DBConnection
	Connections       repositories.AccountingConnectionRepository
	ConnectionService services.AccountingConnectionService
	Records           repositories.AccountingSyncRecordRepository
	Findings          repositories.AccountingDriftFindingRepository
	Source            repositories.AccountingDriftSource
	Publisher         services.AgentEventPublisher `optional:"true"`
	Realtime          services.RealtimeService     `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	db          ports.DBConnection
	connections repositories.AccountingConnectionRepository
	connService services.AccountingConnectionService
	records     repositories.AccountingSyncRecordRepository
	findings    repositories.AccountingDriftFindingRepository
	source      repositories.AccountingDriftSource
	publisher   services.AgentEventPublisher
	realtime    services.RealtimeService
	now         func() time.Time
}

var (
	_ services.AccountingDriftReconciler = (*Service)(nil)
	_ services.AccountingDriftRechecker  = (*Service)(nil)
)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.accounting-drift"),
		db:          p.DB,
		connections: p.Connections,
		connService: p.ConnectionService,
		records:     p.Records,
		findings:    p.Findings,
		source:      p.Source,
		publisher:   p.Publisher,
		realtime:    p.Realtime,
		now:         time.Now,
	}
}

type readSession struct {
	tenant pagination.TenantInfo
	conn   *accountingsync.AccountingConnection
	reader services.AccountingDocumentReader
	writer services.AccountingDocumentWriter
	auth   services.AccountingDocumentAuth
	limit  int
}

func tenantOf(conn *accountingsync.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (s *Service) connectionByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingConnection, error) {
	return s.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) openRead(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) (*readSession, error) {
	tenant := tenantOf(conn)
	session, err := s.connService.Session(ctx, tenant, conn.ID)
	if err != nil {
		return nil, err
	}
	reader, ok := session.Connector.(services.AccountingDocumentReader)
	if !ok {
		return nil, fmt.Errorf(
			"%s cannot show Trenova the documents it holds",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	writer, _ := session.Connector.(services.AccountingDocumentWriter)
	return &readSession{
		tenant: tenant,
		conn:   session.Connection,
		reader: reader,
		writer: writer,
		auth: services.AccountingDocumentAuth{
			RealmID:     session.Connection.ExternalRealmID,
			AccessToken: session.AccessToken,
		},
		limit: max(reader.DocumentReadLimits().MaxPerRead, 1),
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
	tenant pagination.TenantInfo,
	connectionID pulid.ID,
	failure *accountingsync.SyncError,
) {
	if err := s.connections.SaveDriftCheck(ctx, &repositories.SaveAccountingDriftCheckRequest{
		TenantInfo:    tenant,
		ID:            connectionID,
		ErrorCategory: failure.Category,
		ErrorMessage:  failure.Message,
	}); err != nil {
		s.l.Warn("failed to record a drift check failure",
			zap.String("connectionId", connectionID.String()), zap.Error(err))
	}
}

func (s *Service) FinishCheck(
	ctx context.Context,
	req *services.FinishAccountingDriftCheckRequest,
) error {
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return err
	}
	if req.Failure != "" {
		conn.RecordDriftFailure(accountingsync.SyncErrorTransient, req.Failure)
	} else {
		conn.FinishDriftCheck(s.now().Unix())
	}
	if err = s.connections.SaveDriftCheck(ctx, &repositories.SaveAccountingDriftCheckRequest{
		TenantInfo:    req.TenantInfo,
		ID:            conn.ID,
		CheckedAt:     conn.DriftCheckedAt,
		ErrorCategory: conn.DriftErrorCategory,
		ErrorMessage:  conn.DriftErrorMessage,
	}); err != nil {
		return err
	}
	s.publishInvalidation(ctx, req.TenantInfo, pulid.Nil, conn.ID)
	return nil
}

func (s *Service) announce(
	ctx context.Context,
	tenant pagination.TenantInfo,
	opened []*accountingsync.AccountingDriftFinding,
	budget int,
) int {
	sent := 0
	for _, finding := range opened {
		if sent >= budget {
			break
		}
		services.PublishAgentEvent(ctx, s.publisher, services.AgentEvent{
			Kind:       agent.EventAccountingDriftDetected,
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
