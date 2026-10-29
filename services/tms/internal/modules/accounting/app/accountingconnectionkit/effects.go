package accountingconnectionkit

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweraccounting"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (k *Kit) AfterHealthChange(
	ctx context.Context,
	before accountingconnection.ConnectionStatus,
	conn *accountingconnection.AccountingConnection,
	now int64,
) {
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}

	if item, open := watchtoweraccounting.DescribeAccountingConnectionHealth(conn); open {
		if k.watchtower != nil {
			k.watchtower.Upsert(ctx, item)
		}
		if worsened(before, conn.Status) {
			agenteventports.PublishAgentEvent(ctx, k.publisher, agenteventports.AgentEvent{
				Kind:       agenttrigger.EventAccountingConnectionDegraded,
				SubjectID:  conn.ID,
				TenantInfo: tenant,
			})
		}
	} else if k.watchtower != nil {
		k.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtoweraccounting.AccountingConnectionHealthSourceID(conn),
		)
	}

	if k.watchtower != nil {
		if item, open := watchtoweraccounting.DescribeAccountingReconnect(conn, now); open {
			k.watchtower.Upsert(ctx, item)
		} else {
			k.watchtower.Resolve(
				ctx,
				tenant,
				watchtower.SourceAccountingSync,
				watchtoweraccounting.AccountingReconnectSourceID(conn),
			)
		}
	}

	if before != conn.Status {
		k.SyncIntegrationFlag(ctx, conn)
		k.PublishInvalidation(ctx, conn, pulid.Nil)
	}
}

func (k *Kit) ResolveWatchtower(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	if k.watchtower == nil {
		return
	}
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	k.watchtower.Resolve(
		ctx,
		tenant,
		watchtower.SourceAccountingSync,
		watchtoweraccounting.AccountingConnectionHealthSourceID(conn),
	)
	k.watchtower.Resolve(
		ctx,
		tenant,
		watchtower.SourceAccountingSync,
		watchtoweraccounting.AccountingReconnectSourceID(conn),
	)
}

func worsened(before, after accountingconnection.ConnectionStatus) bool {
	return severityRank(after) > severityRank(before)
}

func severityRank(status accountingconnection.ConnectionStatus) int {
	switch status { //nolint:exhaustive // connected and disconnected rank lowest
	case accountingconnection.ConnectionStatusDegraded:
		return 1
	case accountingconnection.ConnectionStatusFailing:
		return 2
	case accountingconnection.ConnectionStatusRevoked:
		return 3
	default:
		return 0
	}
}

func (k *Kit) SyncIntegrationFlag(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) {
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	enabled := conn.IsActive()

	record, err := k.integrations.GetByType(ctx, tenant, conn.IntegrationType)
	switch {
	case err == nil && record.Enabled == enabled:
		return
	case err == nil:
		record.Enabled = enabled
		if enabled {
			record.EnabledByID = conn.ConnectedByID
		}
	case errortypes.IsNotFoundError(err):
		if !enabled {
			return
		}
		record = &integration.Integration{
			OrganizationID: conn.OrganizationID,
			BusinessUnitID: conn.BusinessUnitID,
			Type:           conn.IntegrationType,
			Name:           accountingprovider.ProviderName(conn.IntegrationType),
			Category:       integration.CategoryAccounting,
			Enabled:        true,
			EnabledByID:    conn.ConnectedByID,
		}
	default:
		k.l.Warn("could not read integration record", zap.Error(err))
		return
	}

	if record.Enabled && record.EnabledByID.IsNil() {
		record.EnabledByID = conn.ConnectedByID
	}
	if _, err = k.integrations.Upsert(ctx, record); err != nil {
		k.l.Warn("could not update integration record", zap.Error(err))
	}
}

func (k *Kit) LogAudit(
	conn *accountingconnection.AccountingConnection,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     conn.ID.String(),
		Operation:      permission.OpManage,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(conn),
		PreviousState:  previous,
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		Critical:       true,
	}
	if err := k.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		k.l.Error("failed to log accounting connection audit", zap.Error(err))
	}
}

func (k *Kit) PublishInvalidation(
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
