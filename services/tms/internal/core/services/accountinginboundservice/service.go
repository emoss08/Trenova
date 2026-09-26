package accountinginboundservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultEvaluateLimit = 100
	summaryWindow        = 7 * 24 * time.Hour
)

var permissionResource = permission.ResourceAccountingSync.String()

type Params struct {
	fx.In

	Logger            *zap.Logger
	DB                ports.DBConnection
	Connections       repositories.AccountingConnectionRepository
	ConnectionService services.AccountingConnectionService
	Records           repositories.AccountingSyncRecordRepository
	Changes           repositories.AccountingInboundChangeRepository
	References        repositories.AccountingReferenceObjectRepository
	Mappings          repositories.AccountingMappingRepository
	Invoices          repositories.InvoiceRepository
	Payables          repositories.AccountingPayablesSource
	FiscalPeriods     repositories.FiscalPeriodRepository
	Organizations     repositories.OrganizationRepository
	Users             repositories.UserRepository
	CustomerPayments  services.CustomerPaymentService
	CarrierPayer      services.CarrierSettlementPayer
	DriverPayer       services.DriverSettlementPayer
	AuditService      services.AuditService
	Permissions       services.PermissionEngine
	Refresher         services.AccountingReferenceRefresher `optional:"true"`
	Publisher         services.AgentEventPublisher          `optional:"true"`
	Watchtower        services.WatchtowerProjector          `optional:"true"`
	Realtime          services.RealtimeService              `optional:"true"`
}

type Service struct {
	l                *zap.Logger
	db               ports.DBConnection
	connections      repositories.AccountingConnectionRepository
	connService      services.AccountingConnectionService
	records          repositories.AccountingSyncRecordRepository
	changes          repositories.AccountingInboundChangeRepository
	references       repositories.AccountingReferenceObjectRepository
	mappings         repositories.AccountingMappingRepository
	invoices         repositories.InvoiceRepository
	payables         repositories.AccountingPayablesSource
	fiscalPeriods    repositories.FiscalPeriodRepository
	organizations    repositories.OrganizationRepository
	users            repositories.UserRepository
	customerPayments services.CustomerPaymentService
	carrierPayer     services.CarrierSettlementPayer
	driverPayer      services.DriverSettlementPayer
	audit            services.AuditService
	permissions      services.PermissionEngine
	refresher        services.AccountingReferenceRefresher
	publisher        services.AgentEventPublisher
	watchtower       services.WatchtowerProjector
	realtime         services.RealtimeService
	now              func() time.Time
}

var _ services.AccountingInboundService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.accounting-inbound"),
		db:               p.DB,
		connections:      p.Connections,
		connService:      p.ConnectionService,
		records:          p.Records,
		changes:          p.Changes,
		references:       p.References,
		mappings:         p.Mappings,
		invoices:         p.Invoices,
		payables:         p.Payables,
		fiscalPeriods:    p.FiscalPeriods,
		organizations:    p.Organizations,
		users:            p.Users,
		customerPayments: p.CustomerPayments,
		carrierPayer:     p.CarrierPayer,
		driverPayer:      p.DriverPayer,
		audit:            p.AuditService,
		permissions:      p.Permissions,
		refresher:        p.Refresher,
		publisher:        p.Publisher,
		watchtower:       p.Watchtower,
		realtime:         p.Realtime,
		now:              time.Now,
	}
}

func tenantOf(conn *accountingsync.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (s *Service) nowUnix() int64 {
	return s.now().Unix()
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

func (s *Service) List(
	ctx context.Context,
	req *services.ListAccountingInboundChangesRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingInboundChange], error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	if filter == nil {
		filter = &pagination.QueryOptions{}
	}
	filter.TenantInfo = req.TenantInfo
	return s.changes.ListConnection(
		ctx,
		&repositories.ListAccountingInboundChangesConnectionRequest{
			Filter:       filter,
			Cursor:       req.Cursor,
			ConnectionID: conn.ID,
			Statuses:     req.Statuses,
			Kinds:        req.Kinds,
			Reasons:      req.Reasons,
			Search:       req.Search,
		},
	)
}

func (s *Service) Get(
	ctx context.Context,
	req *services.GetAccountingInboundChangeRequest,
) (*accountingsync.AccountingInboundChange, error) {
	return s.changes.GetByID(ctx, repositories.GetAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
}

func (s *Service) Overview(
	ctx context.Context,
	req *services.AccountingInboundOverviewRequest,
) (*services.AccountingInboundOverview, error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	summary, err := s.changes.Summarize(ctx, &repositories.SummarizeAccountingInboundChangesRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		DecidedSince: s.now().Add(-summaryWindow).Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &services.AccountingInboundOverview{
		ConnectionID:  conn.ID,
		Policy:        conn.PaymentPolicy(),
		ChangesReadAt: conn.ChangesReadAt,
		ChangesError:  conn.ChangesErrorMessage,
		Summary:       summary,
	}, nil
}

func (s *Service) Ignore(
	ctx context.Context,
	req *services.DecideAccountingInboundChangeRequest,
	actor *services.RequestActor,
) (*accountingsync.AccountingInboundChange, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Ignoring a payment from the accounting system requires an authenticated user",
		)
	}

	var updated *accountingsync.AccountingInboundChange
	var previous map[string]any
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		change, err := s.changes.GetByID(txCtx, repositories.GetAccountingInboundChangeRequest{
			TenantInfo: req.TenantInfo,
			ID:         req.ID,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		previous = jsonutils.MustToJSON(change)
		if err = change.Dismiss(actor.UserID, req.Note, s.nowUnix()); err != nil {
			return decisionError(err)
		}
		updated, err = s.changes.Update(txCtx, change)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, actor.UserID, previous, "Ignored a payment recorded in the books")
	s.afterDecision(ctx, updated, actor.UserID)
	return updated, nil
}

func decisionError(err error) error {
	field := "status"
	if err == accountingsync.ErrInboundNoteRequired { //nolint:errorlint // sentinel from the domain
		field = "note"
	}
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, err.Error())
}

func (s *Service) logAudit(
	change *accountingsync.AccountingInboundChange,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	if s.audit == nil {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceAccountingSync,
		ResourceID:     change.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(change),
		PreviousState:  previous,
		OrganizationID: change.OrganizationID,
		BusinessUnitID: change.BusinessUnitID,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log accounting inbound audit", zap.Error(err))
	}
}

func (s *Service) afterDecision(
	ctx context.Context,
	change *accountingsync.AccountingInboundChange,
	actorID pulid.ID,
) {
	s.publishInvalidation(ctx, change.TenantInfo(), actorID, change.ID)
	conn, err := s.connectionByID(ctx, change.TenantInfo(), change.ConnectionID)
	if err != nil {
		s.l.Warn("failed to reload the connection after an inbound decision", zap.Error(err))
		return
	}
	s.refreshAttention(ctx, conn)
}

func (s *Service) refreshAttention(ctx context.Context, conn *accountingsync.AccountingConnection) {
	if s.watchtower == nil {
		return
	}
	tenant := tenantOf(conn)
	groups, err := s.changes.ListAttention(ctx, &repositories.ListAccountingInboundAttentionRequest{
		TenantInfo:     tenant,
		ConnectionID:   conn.ID,
		DetectedBefore: s.now().Add(-watchtowersources.AccountingInboundAfter).Unix(),
	})
	if err != nil {
		s.l.Warn("failed to read accounting inbound attention", zap.Error(err))
		return
	}
	byReason := make(
		map[accountingsync.InboundChangeReason]*repositories.AccountingInboundAttentionGroup,
		len(groups),
	)
	for idx := range groups {
		byReason[groups[idx].Reason] = &groups[idx]
	}
	for _, reason := range watchtowersources.AccountingInboundAttentionReasons() {
		if item, open := watchtowersources.DescribeAccountingInboundAttention(
			conn,
			byReason[reason],
		); open {
			s.watchtower.Upsert(ctx, item)
			continue
		}
		s.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtowersources.AccountingInboundAttentionSourceID(conn, reason),
		)
	}
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
		s.l.Warn("failed to publish accounting inbound invalidation", zap.Error(err))
	}
}
