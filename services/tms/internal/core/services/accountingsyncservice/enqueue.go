package accountingsyncservice

import (
	"context"
	"fmt"

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
	Dispatcher  services.AccountingSyncDispatcher `optional:"true"`
}

type Enqueuer struct {
	l           *zap.Logger
	connections repositories.AccountingConnectionRepository
	records     repositories.AccountingSyncRecordRepository
	mappings    repositories.AccountingMappingRepository
	dispatcher  services.AccountingSyncDispatcher
}

var _ services.AccountingSyncEnqueuer = (*Enqueuer)(nil)

func NewEnqueuer(p EnqueuerParams) *Enqueuer {
	return &Enqueuer{
		l:           p.Logger.Named("service.accounting-sync-enqueuer"),
		connections: p.Connections,
		records:     p.Records,
		mappings:    p.Mappings,
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

	conns, err := e.connections.ListByTenant(ctx, req.TenantInfo)
	if err != nil {
		return fmt.Errorf("accounting sync: list connections: %w", err)
	}

	now := timeutils.NowUnix()
	records := make([]*accountingsync.AccountingSyncRecord, 0, len(conns))
	for _, conn := range conns {
		include, includeErr := e.covers(ctx, req, conn)
		if includeErr != nil {
			return includeErr
		}
		if !include {
			continue
		}
		records = append(records, NewRecordFor(conn, req, now))
	}
	if len(records) == 0 {
		return nil
	}

	result, err := e.records.Enqueue(ctx, records)
	if err != nil {
		return err
	}
	services.NoteAccountingSyncEnqueued(ctx, result.Inserted)
	if req.Operation == accountingsync.SyncOperationUpdate {
		for _, record := range result.Inserted {
			if _, err = e.records.SupersedeOlder(
				ctx,
				&repositories.SupersedeAccountingSyncRecordsRequest{
					TenantInfo:     req.TenantInfo,
					ConnectionID:   record.ConnectionID,
					ObjectType:     record.ObjectType,
					ObjectID:       record.ObjectID,
					Operation:      record.Operation,
					BeforeRevision: record.Revision,
				},
			); err != nil {
				return err
			}
		}
	}

	e.kickAfterCommit(ctx, req.TenantInfo, result.Inserted)
	return nil
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
	if !conn.IsSyncing() {
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
