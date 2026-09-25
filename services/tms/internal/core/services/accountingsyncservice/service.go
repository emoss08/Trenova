package accountingsyncservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultDrainLimit   = 25
	dependencyWait      = time.Minute
	safetyNetPage       = 200
	backfillPage        = 200
	purgeBatch          = 1000
	maxStateObjects     = 500
	maxAttemptsListed   = 50
	maxBackfillsListed  = 20
	attentionGroups     = 20
	pausedTooLongAfter  = 24 * time.Hour
	maxRetryIDs         = 500
	systemNoteMaxLength = 500
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                ports.DBConnection
	Connections       repositories.AccountingConnectionRepository
	Records           repositories.AccountingSyncRecordRepository
	Backfills         repositories.AccountingBackfillRepository
	Mappings          repositories.AccountingMappingRepository
	MappingService    services.AccountingMappingService
	ConnectionService services.AccountingConnectionService
	Invoices          repositories.InvoiceRepository
	Adjustments       repositories.InvoiceAdjustmentRepository
	Payments          repositories.CustomerPaymentRepository
	Organizations     repositories.OrganizationRepository
	AuditService      services.AuditService
	Enqueuer          *Enqueuer
	Dispatcher        services.AccountingSyncDispatcher `optional:"true"`
	Publisher         services.AgentEventPublisher      `optional:"true"`
	Watchtower        services.WatchtowerProjector      `optional:"true"`
	Realtime          services.RealtimeService          `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	db             ports.DBConnection
	connections    repositories.AccountingConnectionRepository
	records        repositories.AccountingSyncRecordRepository
	backfills      repositories.AccountingBackfillRepository
	mappings       repositories.AccountingMappingRepository
	mappingService services.AccountingMappingService
	connService    services.AccountingConnectionService
	invoices       repositories.InvoiceRepository
	adjustments    repositories.InvoiceAdjustmentRepository
	payments       repositories.CustomerPaymentRepository
	organizations  repositories.OrganizationRepository
	audit          services.AuditService
	enqueuer       *Enqueuer
	dispatcher     services.AccountingSyncDispatcher
	publisher      services.AgentEventPublisher
	watchtower     services.WatchtowerProjector
	realtime       services.RealtimeService
}

var _ services.AccountingSyncService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.accounting-sync"),
		db:             p.DB,
		connections:    p.Connections,
		records:        p.Records,
		backfills:      p.Backfills,
		mappings:       p.Mappings,
		mappingService: p.MappingService,
		connService:    p.ConnectionService,
		invoices:       p.Invoices,
		adjustments:    p.Adjustments,
		payments:       p.Payments,
		organizations:  p.Organizations,
		audit:          p.AuditService,
		enqueuer:       p.Enqueuer,
		dispatcher:     p.Dispatcher,
		publisher:      p.Publisher,
		watchtower:     p.Watchtower,
		realtime:       p.Realtime,
	}
}

func (s *Service) connectionFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingConnection, error) {
	if !accountingsync.SupportsAccountingSync(typ) {
		return nil, errortypes.NewValidationError(
			"integrationType",
			errortypes.ErrInvalid,
			"{0} is not an accounting system Trenova can sync with",
			string(typ),
		)
	}
	conn, err := s.connections.GetByType(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenantInfo,
		IntegrationType: typ,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError(
				"{0} has not been connected yet",
				accountingsync.ProviderName(typ),
			)
		}
		return nil, err
	}
	return conn, nil
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

func (s *Service) kick(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) {
	if s.dispatcher == nil {
		return
	}
	ports.AfterCommit(ctx, func(committedCtx context.Context) {
		if err := s.dispatcher.Kick(committedCtx, tenantInfo, connectionID); err != nil {
			s.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
		}
	})
}
