package accountingsyncreads

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnlookup"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const MaxStateObjects = 500

const maxAttemptsListed = 50

const maxBackfillsListed = 20

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingsyncports.AccountingSyncSummary, error) {
	summary := &accountingsyncports.AccountingSyncSummary{
		IntegrationType: integrationType,
		ProviderName:    accountingprovider.ProviderName(integrationType),
		Counts:          []accountingsyncports.AccountingSyncStatusCount{},
		Attention:       []accountingsyncports.AccountingSyncAttentionGroup{},
	}
	conn, err := accountingconnlookup.ByType(ctx, s.connections, tenantInfo, integrationType)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return summary, nil
		}
		return nil, err
	}
	summary.Connection = conn
	ref := accountingsyncports.AccountingSyncConnectionRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
	}

	if summary.Counts, err = s.records.CountByStatus(ctx, ref); err != nil {
		return nil, err
	}
	if summary.Attention, err = s.records.ListAttention(
		ctx,
		accountingsyncports.ListAccountingSyncAttentionRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: conn.ID,
			Limit:        accountingsynceffects.AttentionGroups,
		},
	); err != nil {
		return nil, err
	}
	active, err := s.backfills.GetActive(ctx, ref)
	switch {
	case err == nil:
		summary.ActiveBackfill = active
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}
	return summary, nil
}

func (s *Service) ListRecords(
	ctx context.Context,
	req *accountingsyncports.ListAccountingSyncRecordsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	conn, err := accountingconnlookup.ByType(
		ctx,
		s.connections,
		req.TenantInfo,
		req.IntegrationType,
	)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	if filter == nil {
		filter = &pagination.QueryOptions{}
	}
	filter.TenantInfo = req.TenantInfo
	return s.records.ListConnection(
		ctx,
		&accountingsyncports.ListAccountingSyncRecordsConnectionRequest{
			Filter:          filter,
			Cursor:          req.Cursor,
			ConnectionID:    conn.ID,
			Statuses:        req.Statuses,
			ObjectTypes:     req.ObjectTypes,
			ErrorCategories: req.ErrorCategories,
			ObjectID:        req.ObjectID,
			Search:          req.Search,
		},
	)
}

func (s *Service) GetRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.records.GetByID(ctx, accountingsyncports.GetAccountingSyncRecordRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) ListAttempts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	if _, err := s.GetRecord(ctx, tenantInfo, recordID); err != nil {
		return nil, err
	}
	return s.records.ListAttempts(ctx, accountingsyncports.ListAccountingSyncAttemptsRequest{
		TenantInfo:   tenantInfo,
		SyncRecordID: recordID,
		Limit:        maxAttemptsListed,
	})
}

func stateRank(record *accountingsync.AccountingSyncRecord) int {
	switch {
	case record.Status.NeedsAttention():
		return 4
	case record.Status == accountingsync.SyncStatusInFlight ||
		record.Status.Dispatchable() ||
		record.Status == accountingsync.SyncStatusAwaitingApproval:
		return 3
	case record.Status == accountingsync.SyncStatusSynced:
		return 2
	case record.Status == accountingsync.SyncStatusSkipped:
		return 1
	default:
		return 0
	}
}

func outranks(record, current *accountingsync.AccountingSyncRecord) bool {
	rank, currentRank := stateRank(record), stateRank(current)
	if rank != currentRank {
		return rank > currentRank
	}
	return record.QueuedAt > current.QueuedAt
}

func (s *Service) ObjectStates(
	ctx context.Context,
	req *accountingsyncports.AccountingSyncObjectStatesRequest,
) (map[pulid.ID]*accountingsyncports.AccountingSyncObjectState, error) {
	objectIDs := req.ObjectIDs
	states := make(map[pulid.ID]*accountingsyncports.AccountingSyncObjectState, len(objectIDs))
	if len(objectIDs) == 0 {
		return states, nil
	}
	if len(objectIDs) > MaxStateObjects {
		objectIDs = objectIDs[:MaxStateObjects]
	}

	conns, err := s.connections.ListByTenant(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	providers := make(map[pulid.ID]string, len(conns))
	for _, conn := range conns {
		if !conn.IsSyncing() ||
			(req.IntegrationType != "" && conn.IntegrationType != req.IntegrationType) {
			continue
		}
		providers[conn.ID] = accountingprovider.ProviderName(conn.IntegrationType)
	}
	if len(providers) == 0 {
		return states, nil
	}

	records, err := s.records.ListByObjects(
		ctx,
		&accountingsyncports.ListAccountingSyncRecordsByObjectsRequest{
			TenantInfo: req.TenantInfo,
			ObjectIDs:  objectIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		provider, ok := providers[record.ConnectionID]
		if !ok || record.Status == accountingsync.SyncStatusSuperseded {
			continue
		}
		current, seen := states[record.ObjectID]
		if seen && !outranks(record, current.Record) {
			continue
		}
		states[record.ObjectID] = &accountingsyncports.AccountingSyncObjectState{
			ObjectType:   record.ObjectType,
			ObjectID:     record.ObjectID,
			ProviderName: provider,
			Record:       record,
		}
	}
	return states, nil
}

func (s *Service) ListBackfills(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) ([]*accountingbackfill.AccountingBackfill, error) {
	conn, err := accountingconnlookup.ByType(ctx, s.connections, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	return s.backfills.ListByConnection(ctx, accountingsyncports.ListAccountingBackfillsRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
		Limit:        maxBackfillsListed,
	})
}

type Params struct {
	fx.In

	Connections accountingconnectionports.AccountingConnectionRepository
	Records     accountingsyncports.AccountingSyncRecordRepository
	Backfills   accountingsyncports.AccountingBackfillRepository
}

type Service struct {
	connections accountingconnectionports.AccountingConnectionRepository
	records     accountingsyncports.AccountingSyncRecordRepository
	backfills   accountingsyncports.AccountingBackfillRepository
}

func New(p Params) *Service {
	return &Service{connections: p.Connections, records: p.Records, backfills: p.Backfills}
}

var _ accountingsyncports.AccountingSyncReader = (*Service)(nil)
