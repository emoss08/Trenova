package accountingmappingkit

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (k *Kit) LogAudit(
	row *accountingmapping.AccountingMapping,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     row.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(row),
		PreviousState:  previous,
		OrganizationID: row.OrganizationID,
		BusinessUnitID: row.BusinessUnitID,
	}
	if err := k.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		k.l.Error("failed to log accounting mapping audit", zap.Error(err))
	}
}

func (k *Kit) PublishInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	rows []*accountingmapping.AccountingMapping,
) {
	if k.realtime == nil || len(rows) == 0 {
		return
	}
	recordID := rows[0].ID
	if len(rows) > 1 {
		recordID = pulid.Nil
	}
	if err := realtimeinvalidation.Publish(ctx, k.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       recordID,
	}); err != nil {
		k.l.Warn("failed to publish accounting mapping invalidation", zap.Error(err))
	}
}

func (k *Kit) PublishConnectionInvalidation(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	userID pulid.ID,
) {
	if k.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, k.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       conn.ID,
	}); err != nil {
		k.l.Warn("failed to publish accounting connection invalidation", zap.Error(err))
	}
}

func (k *Kit) historyUse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
) (int, error) {
	if k.syncRecords == nil || row.State != accountingmapping.MappingStateConfirmed {
		return 0, nil
	}
	usage, err := k.syncRecords.MappingUsage(
		ctx,
		&accountingsyncports.AccountingSyncMappingUsageRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: row.ConnectionID,
			MappingIDs:   []pulid.ID{row.ID},
		},
	)
	if err != nil {
		return 0, err
	}
	return usage[row.ID], nil
}

func (k *Kit) GuardHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingmapping.AccountingMapping,
	acknowledged bool,
) (int, error) {
	used, err := k.historyUse(ctx, tenantInfo, row)
	if err != nil || used == 0 || acknowledged {
		return used, err
	}
	return used, errortypes.NewValidationError(
		"acknowledgeHistory",
		errortypes.ErrInvalidOperation,
		"{0} was used by {1} documents already sent to the accounting system. "+
			"They keep the record they were sent with; confirm that only new documents change",
		row.TargetLabel,
		used,
	)
}

func HistoryNote(used int) string {
	if used == 0 {
		return ""
	}
	return " (" + strconv.Itoa(used) + " documents already sent keep the previous record)"
}

func (k *Kit) RequeueMappingBlocked(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionIDs ...pulid.ID,
) {
	if k.syncRecords == nil {
		return
	}
	seen := make(map[pulid.ID]struct{}, len(connectionIDs))
	for _, connectionID := range connectionIDs {
		if _, ok := seen[connectionID]; ok || connectionID.IsNil() {
			continue
		}
		seen[connectionID] = struct{}{}

		requeued, err := k.syncRecords.Requeue(
			ctx,
			&accountingsyncports.RequeueAccountingSyncRecordsRequest{
				TenantInfo:   tenantInfo,
				ConnectionID: connectionID,
				ErrorCategories: []accountingsync.SyncErrorCategory{
					accountingsync.SyncErrorMapping,
				},
				At: timeutils.NowUnix(),
			},
		)
		if err != nil {
			k.l.Warn("failed to requeue records blocked on a mapping",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
			continue
		}
		if requeued == 0 || k.dispatcher == nil {
			continue
		}
		if err = k.dispatcher.Kick(ctx, tenantInfo, connectionID); err != nil {
			k.l.Warn("failed to wake the accounting dispatcher",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
		}
	}
}

type Deps struct {
	Logger      *zap.Logger
	Audit       auditports.AuditService
	Realtime    realtimeports.RealtimeService
	SyncRecords accountingsyncports.AccountingSyncRecordRepository
	Dispatcher  accountingsyncports.AccountingSyncDispatcher
}

type Kit struct {
	l           *zap.Logger
	audit       auditports.AuditService
	realtime    realtimeports.RealtimeService
	syncRecords accountingsyncports.AccountingSyncRecordRepository
	dispatcher  accountingsyncports.AccountingSyncDispatcher
}

func New(d Deps) *Kit {
	return &Kit{
		l:           d.Logger,
		audit:       d.Audit,
		realtime:    d.Realtime,
		syncRecords: d.SyncRecords,
		dispatcher:  d.Dispatcher,
	}
}
