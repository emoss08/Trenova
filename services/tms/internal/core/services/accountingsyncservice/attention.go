package accountingsyncservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func tenantOf(conn *accountingsync.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (s *Service) refreshAttention(ctx context.Context, conn *accountingsync.AccountingConnection) {
	if s.watchtower == nil {
		return
	}
	tenant := tenantOf(conn)
	groups, err := s.records.ListAttention(ctx, repositories.ListAccountingSyncAttentionRequest{
		TenantInfo:   tenant,
		ConnectionID: conn.ID,
		Limit:        attentionGroups,
	})
	if err != nil {
		s.l.Warn("failed to read accounting sync attention", zap.Error(err))
		return
	}

	byKey := watchtowersources.GroupAccountingSyncAttention(groups)
	for _, key := range watchtowersources.AccountingSyncAttentionKeys() {
		if item, open := watchtowersources.DescribeAccountingSyncAttention(conn, key, byKey[key]); open {
			s.watchtower.Upsert(ctx, item)
			continue
		}
		s.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtowersources.AccountingSyncAttentionSourceID(conn, key),
		)
	}
}

func (s *Service) refreshPaused(ctx context.Context, conn *accountingsync.AccountingConnection) {
	if s.watchtower == nil {
		return
	}
	if item, open := watchtowersources.DescribeAccountingSyncPaused(conn, timeutils.NowUnix()); open {
		s.watchtower.Upsert(ctx, item)
		return
	}
	s.watchtower.Resolve(
		ctx,
		tenantOf(conn),
		watchtower.SourceAccountingSync,
		watchtowersources.AccountingSyncPausedSourceID(conn),
	)
}

func (s *Service) reportSafetyNet(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	found int,
) {
	if s.watchtower == nil || found == 0 {
		return
	}
	if item, open := watchtowersources.DescribeAccountingSafetyNet(
		conn,
		found,
		timeutils.NowUnix(),
	); open {
		s.watchtower.Upsert(ctx, item)
	}
}

func (s *Service) publishOutcome(
	ctx context.Context,
	record *accountingsync.AccountingSyncRecord,
	outcome accountingsync.SyncAttemptOutcome,
) {
	var kind agent.EventKind
	switch outcome {
	case accountingsync.SyncAttemptBlocked:
		kind = agent.EventAccountingSyncBlocked
	case accountingsync.SyncAttemptDeadLettered:
		kind = agent.EventAccountingSyncFailed
	case accountingsync.SyncAttemptSynced,
		accountingsync.SyncAttemptRetrying,
		accountingsync.SyncAttemptWaiting:
		return
	default:
		return
	}
	services.PublishAgentEvent(ctx, s.publisher, services.AgentEvent{
		Kind:       kind,
		SubjectID:  record.ID,
		TenantInfo: record.TenantInfo(),
	})
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
		s.l.Warn("failed to publish accounting sync invalidation", zap.Error(err))
	}
}
