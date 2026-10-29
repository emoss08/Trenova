package accountingsynceffects

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnlookup"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (e *Effects) Connection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnection.AccountingConnection, error) {
	return accountingconnlookup.ByType(ctx, e.connections, tenantInfo, typ)
}

func (e *Effects) ConnectionByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingconnection.AccountingConnection, error) {
	return e.connections.GetByID(ctx, accountingconnectionports.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (e *Effects) Kick(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) {
	if e.dispatcher == nil {
		return
	}
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		if err := e.dispatcher.Kick(committedCtx, tenantInfo, connectionID); err != nil {
			e.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
		}
	})
}

func (e *Effects) OrgLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	org, err := e.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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

func (e *Effects) RedatesToNextOpen(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	control, err := e.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return false, err
	}
	return control.ClosedPeriodPostingPolicy == accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen, nil
}

func (e *Effects) SyncingConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := e.Connection(ctx, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		return nil, errortypes.NewBusinessError(
			"{0} is not sending documents yet; finish its setup first",
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}
	return conn, nil
}
