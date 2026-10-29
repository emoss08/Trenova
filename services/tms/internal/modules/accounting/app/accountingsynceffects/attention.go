package accountingsynceffects

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweraccounting"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func TenantOf(conn *accountingconnection.AccountingConnection) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
}

func (e *Effects) RefreshAttention(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if e.watchtower == nil {
		return
	}
	tenant := TenantOf(conn)
	groups, err := e.records.ListAttention(
		ctx,
		accountingsyncports.ListAccountingSyncAttentionRequest{
			TenantInfo:   tenant,
			ConnectionID: conn.ID,
			Limit:        AttentionGroups,
		},
	)
	if err != nil {
		e.l.Warn("failed to read accounting sync attention", zap.Error(err))
		return
	}

	byKey := watchtoweraccounting.GroupAccountingSyncAttention(groups)
	for _, key := range watchtoweraccounting.AccountingSyncAttentionKeys() {
		if item, open := watchtoweraccounting.DescribeAccountingSyncAttention(
			conn,
			key,
			byKey[key],
		); open {
			e.watchtower.Upsert(ctx, item)
			continue
		}
		e.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtoweraccounting.AccountingSyncAttentionSourceID(conn, key),
		)
	}
}

func (e *Effects) RefreshPaused(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if e.watchtower == nil {
		return
	}
	if item, open := watchtoweraccounting.DescribeAccountingSyncPaused(
		conn,
		timeutils.NowUnix(),
	); open {
		e.watchtower.Upsert(ctx, item)
		return
	}
	e.watchtower.Resolve(
		ctx,
		TenantOf(conn),
		watchtower.SourceAccountingSync,
		watchtoweraccounting.AccountingSyncPausedSourceID(conn),
	)
}

func (e *Effects) ReportSafetyNet(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	found int,
) {
	if e.watchtower == nil || found == 0 {
		return
	}
	if item, open := watchtoweraccounting.DescribeAccountingSafetyNet(
		conn,
		found,
		timeutils.NowUnix(),
	); open {
		e.watchtower.Upsert(ctx, item)
	}
}

func (e *Effects) PublishOutcome(
	ctx context.Context,
	record *accountingsync.AccountingSyncRecord,
	outcome accountingsync.SyncAttemptOutcome,
) {
	var kind agenttrigger.EventKind
	switch outcome {
	case accountingsync.SyncAttemptBlocked:
		kind = agenttrigger.EventAccountingSyncBlocked
	case accountingsync.SyncAttemptDeadLettered:
		kind = agenttrigger.EventAccountingSyncFailed
	case accountingsync.SyncAttemptSynced,
		accountingsync.SyncAttemptRetrying,
		accountingsync.SyncAttemptWaiting:
		return
	default:
		return
	}
	agentports.PublishAgentEvent(ctx, e.publisher, agentports.AgentEvent{
		Kind:       kind,
		SubjectID:  record.ID,
		TenantInfo: record.TenantInfo(),
	})
}

func (e *Effects) PublishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
	recordID pulid.ID,
) {
	if e.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, e.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    actorID,
		Resource:       permissionResource,
		Action:         "updated",
		RecordID:       recordID,
	}); err != nil {
		e.l.Warn("failed to publish accounting sync invalidation", zap.Error(err))
	}
}

var permissionResource = permission.ResourceAccountingSync.String()
