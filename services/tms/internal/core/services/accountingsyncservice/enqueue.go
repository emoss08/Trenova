package accountingsyncservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EnqueuerParams struct {
	fx.In

	Logger      *zap.Logger
	Connections repositories.AccountingConnectionRepository
	Records     repositories.AccountingSyncRecordRepository
	Mappings    repositories.AccountingMappingRepository
	Orgs        repositories.OrganizationRepository
	Ledger      repositories.AccountingLedgerSource
	Dispatcher  services.AccountingSyncDispatcher `optional:"true"`
}

type Enqueuer struct {
	l           *zap.Logger
	connections repositories.AccountingConnectionRepository
	records     repositories.AccountingSyncRecordRepository
	mappings    repositories.AccountingMappingRepository
	orgs        repositories.OrganizationRepository
	ledger      repositories.AccountingLedgerSource
	dispatcher  services.AccountingSyncDispatcher
}

var (
	_ services.AccountingSyncEnqueuer   = (*Enqueuer)(nil)
	_ services.AccountingSyncPlanner    = (*Enqueuer)(nil)
	_ services.AccountingLedgerEnqueuer = (*Enqueuer)(nil)
)

func NewEnqueuer(p EnqueuerParams) *Enqueuer {
	return &Enqueuer{
		l:           p.Logger.Named("service.accounting-sync-enqueuer"),
		connections: p.Connections,
		records:     p.Records,
		mappings:    p.Mappings,
		orgs:        p.Orgs,
		ledger:      p.Ledger,
		dispatcher:  p.Dispatcher,
	}
}

func validateEnqueue(req *services.AccountingSyncEnqueueRequest) error {
	switch {
	case req == nil:
		return errortypes.NewBusinessError("An accounting sync request is required")
	case !req.ObjectType.IsValid():
		return fmt.Errorf("accounting sync: unknown object type %q", req.ObjectType)
	case !req.Operation.IsValid():
		return fmt.Errorf("accounting sync: unknown operation %q", req.Operation)
	case !req.SourceEvent.IsValid():
		return fmt.Errorf("accounting sync: unknown source event %q", req.SourceEvent)
	case req.ObjectID.IsNil():
		return fmt.Errorf("accounting sync: %s has no id", req.ObjectType)
	default:
		return nil
	}
}

func (e *Enqueuer) Enqueue(ctx context.Context, req *services.AccountingSyncEnqueueRequest) error {
	if err := validateEnqueue(req); err != nil {
		return err
	}

	conns, err := e.coveringConnections(ctx, req)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	records := make([]*accountingsync.AccountingSyncRecord, 0, len(conns))
	for _, conn := range conns {
		records = append(records, NewRecordFor(conn, req, now))
	}
	_, err = e.EnqueueRecords(ctx, req.TenantInfo, records)
	return err
}

// EnqueueRecords queues the records and wakes the dispatcher once the
// transaction commits.
func (e *Enqueuer) EnqueueRecords(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	records []*accountingsync.AccountingSyncRecord,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	result, err := e.QueueRecords(ctx, tenantInfo, records)
	if err != nil {
		return nil, err
	}
	e.kickAfterCommit(ctx, tenantInfo, result.Inserted)
	return result, nil
}

// QueueRecords inserts the records, queues a fresh update for a ledger day
// that was already queued, and retires the updates a newer one replaces. It
// leaves waking the dispatcher to the caller.
func (e *Enqueuer) QueueRecords(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	records []*accountingsync.AccountingSyncRecord,
) (*repositories.EnqueueAccountingSyncRecordsResult, error) {
	if len(records) == 0 {
		return &repositories.EnqueueAccountingSyncRecordsResult{}, nil
	}

	result, err := e.records.Enqueue(ctx, records)
	if err != nil {
		return nil, err
	}
	if followUps := dayUpdatesFor(records, result.Inserted); len(followUps) > 0 {
		updated, updateErr := e.records.Enqueue(ctx, followUps)
		if updateErr != nil {
			return nil, updateErr
		}
		result.Inserted = append(result.Inserted, updated.Inserted...)
		result.Existing += updated.Existing
	}
	services.NoteAccountingSyncEnqueued(ctx, result.Inserted)

	for _, record := range result.Inserted {
		if record.Operation != accountingsync.SyncOperationUpdate {
			continue
		}
		if _, err = e.records.SupersedeOlder(
			ctx,
			&repositories.SupersedeAccountingSyncRecordsRequest{
				TenantInfo:     tenantInfo,
				ConnectionID:   record.ConnectionID,
				ObjectType:     record.ObjectType,
				ObjectID:       record.ObjectID,
				Operation:      record.Operation,
				BeforeRevision: record.Revision,
			},
		); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func dayUpdatesFor(
	records, inserted []*accountingsync.AccountingSyncRecord,
) []*accountingsync.AccountingSyncRecord {
	insertedIDs := make(map[pulid.ID]struct{}, len(inserted))
	for _, record := range inserted {
		insertedIDs[record.ID] = struct{}{}
	}
	revision := time.Now().UnixNano()
	updates := make([]*accountingsync.AccountingSyncRecord, 0, 1)
	for _, record := range records {
		if _, ok := insertedIDs[record.ID]; ok ||
			record.ObjectType != accountingsync.SyncObjectJournalSummary ||
			record.Operation != accountingsync.SyncOperationCreate {
			continue
		}
		updates = append(updates, record.DayUpdate(revision))
	}
	return updates
}

func (e *Enqueuer) EnqueueJournal(
	ctx context.Context,
	posted *services.AccountingJournalPosted,
) error {
	if posted == nil || posted.EntryID.IsNil() ||
		!accountingsync.JournalSendable(posted.EntryType, "") {
		return nil
	}
	conns, err := e.connections.ListByTenant(ctx, posted.TenantInfo)
	if err != nil {
		return fmt.Errorf("accounting sync: list connections: %w", err)
	}

	now := timeutils.NowUnix()
	var loc *time.Location
	records := make([]*accountingsync.AccountingSyncRecord, 0, len(conns))
	for _, conn := range conns {
		if !conn.IsSyncing() || !conn.SendsLedger() || !conn.Covers(posted.AccountingDate) {
			continue
		}
		req := &services.AccountingSyncEnqueueRequest{
			TenantInfo:   posted.TenantInfo,
			ObjectType:   accountingsync.SyncObjectJournalEntry,
			ObjectID:     posted.EntryID,
			ObjectNumber: posted.EntryNumber,
			Operation:    accountingsync.SyncOperationCreate,
			Revision:     1,
			SourceEvent:  accountingsync.SyncSourceJournalPosted,
			DocumentDate: posted.AccountingDate,
		}
		if conn.SumsByDay() {
			if loc == nil {
				if loc, err = e.location(ctx, posted.TenantInfo); err != nil {
					return err
				}
			}
			req = JournalDayRequest(posted.TenantInfo, posted.AccountingDate, loc)
			req.SourceEvent = accountingsync.SyncSourceJournalPosted
		}
		records = append(records, NewRecordFor(conn, req, now))
	}
	if len(records) == 0 {
		return nil
	}
	sendable, err := e.reversalSendable(ctx, posted)
	if err != nil || !sendable {
		return err
	}

	_, err = e.EnqueueRecords(ctx, posted.TenantInfo, records)
	return err
}

func (e *Enqueuer) reversalSendable(
	ctx context.Context,
	posted *services.AccountingJournalPosted,
) (bool, error) {
	if posted.ReversalOfID.IsNil() {
		return true, nil
	}
	reverses, err := e.ledger.GetEntryType(ctx, &repositories.GetLedgerJournalRequest{
		TenantInfo: posted.TenantInfo,
		ID:         posted.ReversalOfID,
	})
	if err != nil {
		return false, fmt.Errorf("accounting sync: load reversed journal: %w", err)
	}
	return accountingsync.JournalSendable(posted.EntryType, reverses), nil
}

func JournalDayRequest(
	tenantInfo pagination.TenantInfo,
	accountingDate int64,
	loc *time.Location,
) *services.AccountingSyncEnqueueRequest {
	return &services.AccountingSyncEnqueueRequest{
		TenantInfo:   tenantInfo,
		ObjectType:   accountingsync.SyncObjectJournalSummary,
		ObjectID:     pulid.ID(accountingsync.JournalDayID(accountingDate, loc)),
		ObjectNumber: timeutils.FormatCalendarDate(accountingDate, loc),
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		DocumentDate: timeutils.DayStart(accountingDate, loc),
	}
}

func (e *Enqueuer) location(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	org, err := e.orgs.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if org == nil {
		return time.UTC, nil
	}
	return timeutils.LoadLocation(org.Timezone), nil
}

// Destinations is where Enqueue would queue the record: the connections that
// cover it, decided by the same rule. It writes nothing.
func (e *Enqueuer) Destinations(
	ctx context.Context,
	req *services.AccountingSyncEnqueueRequest,
) ([]services.AccountingSyncDestination, error) {
	if err := validateEnqueue(req); err != nil {
		return nil, err
	}

	conns, err := e.coveringConnections(ctx, req)
	if err != nil {
		return nil, err
	}

	destinations := make([]services.AccountingSyncDestination, 0, len(conns))
	for _, conn := range conns {
		destinations = append(destinations, services.AccountingSyncDestination{
			ConnectionID: conn.ID,
			Integration:  string(conn.IntegrationType),
			Company:      conn.ExternalCompanyName,
			AwaitsRelease: !conn.AutoSync &&
				req.SourceEvent != accountingsync.SyncSourceDependencyOf,
		})
	}

	return destinations, nil
}

func (e *Enqueuer) coveringConnections(
	ctx context.Context,
	req *services.AccountingSyncEnqueueRequest,
) ([]*accountingsync.AccountingConnection, error) {
	conns, err := e.connections.ListByTenant(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("accounting sync: list connections: %w", err)
	}

	covering := make([]*accountingsync.AccountingConnection, 0, len(conns))
	for _, conn := range conns {
		include, includeErr := e.covers(ctx, req, conn)
		if includeErr != nil {
			return nil, includeErr
		}
		if include {
			covering = append(covering, conn)
		}
	}

	return covering, nil
}

func NewRecordFor(
	conn *accountingsync.AccountingConnection,
	req *services.AccountingSyncEnqueueRequest,
	now int64,
) *accountingsync.AccountingSyncRecord {
	var documentDate *int64
	if req.DocumentDate > 0 {
		date := req.DocumentDate
		documentDate = &date
	}
	return accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: req.ObjectType,
			ObjectID:   req.ObjectID,
			Operation:  req.Operation,
			Revision:   req.Revision,
		},
		ObjectNumber: req.ObjectNumber,
		SourceEvent:  req.SourceEvent,
		DocumentDate: documentDate,
		AwaitRelease: !conn.AutoSync && req.SourceEvent != accountingsync.SyncSourceDependencyOf,
		At:           now,
	})
}

func (e *Enqueuer) covers(
	ctx context.Context,
	req *services.AccountingSyncEnqueueRequest,
	conn *accountingsync.AccountingConnection,
) (bool, error) {
	if !conn.IsSyncing() || !conn.Sends(req.ObjectType) {
		return false, nil
	}
	if req.ObjectType.NeedsDriverSettlements() && !conn.SyncsDriverSettlements() {
		return false, nil
	}
	target, isParty := req.ObjectType.PartyTarget()
	if !isParty {
		return conn.Covers(req.DocumentDate), nil
	}
	if req.SourceEvent == accountingsync.SyncSourceDependencyOf {
		return true, nil
	}

	mapping, err := e.mappings.GetByTarget(ctx, &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    conn.ID,
		TargetType:      target,
		TrenovaObjectID: req.ObjectID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return mapping.State == accountingsync.MappingStateConfirmed, nil
}

func (e *Enqueuer) kickAfterCommit(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inserted []*accountingsync.AccountingSyncRecord,
) {
	if e.dispatcher == nil || len(inserted) == 0 {
		return
	}
	connectionIDs := make(map[pulid.ID]struct{}, 1)
	for _, record := range inserted {
		if record.Status == accountingsync.SyncStatusQueued {
			connectionIDs[record.ConnectionID] = struct{}{}
		}
	}
	for connectionID := range connectionIDs {
		ports.AfterCommit(ctx, func(committedCtx context.Context) {
			if err := e.dispatcher.Kick(committedCtx, tenantInfo, connectionID); err != nil {
				e.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
					zap.String("connectionId", connectionID.String()), zap.Error(err))
			}
		})
	}
}
