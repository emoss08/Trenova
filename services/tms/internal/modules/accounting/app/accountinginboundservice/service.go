package accountinginboundservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnlookup"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweraccounting"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultEvaluateLimit = 100
	summaryWindow        = 7 * 24 * time.Hour
	driftEventsPerPoll   = 20
)

var permissionResource = permission.ResourceAccountingSync.String()

type Params struct {
	fx.In

	Logger           *zap.Logger
	DB               kernelports.Transactor
	Connections      accountingconnectionports.AccountingConnectionRepository
	Sessions         accountingconnectionports.AccountingSessionProvider
	Records          accountingsyncports.AccountingSyncRecordRepository
	Changes          accountinginboundports.AccountingInboundChangeRepository
	References       accountingmappingports.AccountingReferenceObjectRepository
	Mappings         accountingmappingports.AccountingMappingRepository
	Invoices         invoiceports.InvoiceRepository
	Payables         accountingpayablesports.AccountingPayablesSource
	FiscalPeriods    fiscalperiodports.FiscalPeriodRepository
	Organizations    tenantports.OrganizationRepository
	Users            tenantports.UserRepository
	CustomerPayments customerpaymentports.CustomerPaymentService
	CarrierPayer     carriersettlementports.CarrierSettlementPayer
	DriverPayer      carriersettlementports.DriverSettlementPayer
	AuditService     auditports.AuditService
	Permissions      permissionports.PermissionEngine
	Refresher        accountingconnectionports.AccountingReferenceRefresher `optional:"true"`
	Drift            accountingdriftports.AccountingDriftRechecker          `optional:"true"`
	Publisher        agenteventports.AgentEventPublisher                    `optional:"true"`
	Watchtower       watchtowerports.WatchtowerProjector                    `optional:"true"`
	Realtime         realtimeports.RealtimeService                          `optional:"true"`
}

type Service struct {
	l                *zap.Logger
	db               kernelports.Transactor
	connections      accountingconnectionports.AccountingConnectionRepository
	sessions         accountingconnectionports.AccountingSessionProvider
	records          accountingsyncports.AccountingSyncRecordRepository
	changes          accountinginboundports.AccountingInboundChangeRepository
	references       accountingmappingports.AccountingReferenceObjectRepository
	mappings         accountingmappingports.AccountingMappingRepository
	invoices         invoiceports.InvoiceRepository
	payables         accountingpayablesports.AccountingPayablesSource
	fiscalPeriods    fiscalperiodports.FiscalPeriodRepository
	organizations    tenantports.OrganizationRepository
	users            tenantports.UserRepository
	customerPayments customerpaymentports.CustomerPaymentService
	carrierPayer     carriersettlementports.CarrierSettlementPayer
	driverPayer      carriersettlementports.DriverSettlementPayer
	audit            auditports.AuditService
	permissions      permissionports.PermissionEngine
	refresher        accountingconnectionports.AccountingReferenceRefresher
	drift            accountingdriftports.AccountingDriftRechecker
	publisher        agenteventports.AgentEventPublisher
	watchtower       watchtowerports.WatchtowerProjector
	realtime         realtimeports.RealtimeService
	now              func() time.Time
}

var _ accountinginboundports.AccountingInboundService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.accounting-inbound"),
		db:               p.DB,
		connections:      p.Connections,
		sessions:         p.Sessions,
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
		drift:            p.Drift,
		publisher:        p.Publisher,
		watchtower:       p.Watchtower,
		realtime:         p.Realtime,
		now:              time.Now,
	}
}

func tenantOf(conn *accountingconnection.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (s *Service) nowUnix() int64 {
	return s.now().Unix()
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

func (s *Service) connectionFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnection.AccountingConnection, error) {
	return accountingconnlookup.ByType(ctx, s.connections, tenantInfo, typ)
}

func (s *Service) List(
	ctx context.Context,
	req *accountinginboundports.ListAccountingInboundChangesRequest,
) (*pagination.CursorListResult[*accountinginbound.AccountingInboundChange], error) {
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
		&accountinginboundports.ListAccountingInboundChangesConnectionRequest{
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
	req *accountinginboundports.GetAccountingInboundChangeRequest,
) (*accountinginbound.AccountingInboundChange, error) {
	return s.changes.GetByID(ctx, accountinginboundports.GetAccountingInboundChangeQuery{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
}

func (s *Service) Overview(
	ctx context.Context,
	req *accountinginboundports.AccountingInboundOverviewRequest,
) (*accountinginboundports.AccountingInboundOverview, error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	summary, err := s.changes.Summarize(
		ctx,
		&accountinginboundports.SummarizeAccountingInboundChangesRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: conn.ID,
			DecidedSince: s.now().Add(-summaryWindow).Unix(),
		},
	)
	if err != nil {
		return nil, err
	}
	return &accountinginboundports.AccountingInboundOverview{
		ConnectionID:  conn.ID,
		Policy:        conn.PaymentPolicy(),
		ChangesReadAt: conn.ChangesReadAt,
		ChangesError:  conn.ChangesErrorMessage,
		Summary:       summary,
	}, nil
}

func (s *Service) Ignore(
	ctx context.Context,
	req *accountinginboundports.DecideAccountingInboundChangeRequest,
	actor *permissionports.RequestActor,
) (*accountinginbound.AccountingInboundChange, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Ignoring a payment from the accounting system requires an authenticated user",
		)
	}

	var updated *accountinginbound.AccountingInboundChange
	var previous map[string]any
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		change, err := s.changes.GetByID(
			txCtx,
			accountinginboundports.GetAccountingInboundChangeQuery{
				TenantInfo: req.TenantInfo,
				ID:         req.ID,
				ForUpdate:  true,
			},
		)
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
	if err == accountinginbound.ErrInboundNoteRequired { //nolint:errorlint // sentinel from the domain
		field = "note"
	}
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, err.Error())
}

func (s *Service) logAudit(
	change *accountinginbound.AccountingInboundChange,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	if s.audit == nil {
		return
	}
	params := &auditports.LogActionParams{
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
	change *accountinginbound.AccountingInboundChange,
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

func (s *Service) refreshAttention(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if s.watchtower == nil {
		return
	}
	tenant := tenantOf(conn)
	groups, err := s.changes.ListAttention(
		ctx,
		&accountinginboundports.ListAccountingInboundAttentionRequest{
			TenantInfo:     tenant,
			ConnectionID:   conn.ID,
			DetectedBefore: s.now().Add(-watchtoweraccounting.AccountingInboundAfter).Unix(),
		},
	)
	if err != nil {
		s.l.Warn("failed to read accounting inbound attention", zap.Error(err))
		return
	}
	byReason := make(
		map[accountinginbound.InboundChangeReason]*accountinginboundports.AccountingInboundAttentionGroup,
		len(groups),
	)
	for idx := range groups {
		byReason[groups[idx].Reason] = &groups[idx]
	}
	for _, reason := range watchtoweraccounting.AccountingInboundAttentionReasons() {
		if item, open := watchtoweraccounting.DescribeAccountingInboundAttention(
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
			watchtoweraccounting.AccountingInboundAttentionSourceID(conn, reason),
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
