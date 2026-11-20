package storagequotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventStorageNearlyFull = "document_storage_nearly_full"
	noticeSource           = "storagequotaservice"
	subjectKind            = "document-storage-allowance"
	maxNoticeRecipients    = 25
)

type Notifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
	ResolveSubject(
		ctx context.Context,
		req notificationports.ResolveNotificationSubjectRequest,
	) (int64, error)
}

type TenantPager interface {
	ListTenantsPage(ctx context.Context, after pulid.ID, limit int) ([]pagination.TenantInfo, error)
}

type WarningParams struct {
	fx.In

	Notifications *notificationservice.Service
	Tenants       tenantports.TenantSyncRepository
	Clock         timeutils.Clock
	Logger        *zap.Logger
}

type WarningResult struct {
	fx.Out

	Observer managedusageports.StateObserver `group:"managed_state_observers"`
}

type Warnings struct {
	notifier Notifier
	tenants  TenantPager
	clock    timeutils.Clock
	l        *zap.Logger
}

func NewWarnings(p WarningParams) WarningResult {
	return WarningResult{Observer: &Warnings{
		notifier: p.Notifications,
		tenants:  p.Tenants,
		clock:    p.Clock,
		l:        p.Logger.Named("service.storage-quota-warnings"),
	}}
}

func (w *Warnings) StateChanged(
	ctx context.Context,
	change *managedusageports.StateChange,
) error {
	was, _ := nearlyFull(change.Previous)
	is, meter := nearlyFull(change.Current)
	switch {
	case is && !was:
		w.l.Info("document storage crossed the warning share of its allowance",
			zap.String("scope", change.Scope.Key()),
			zap.Int64("used", meter.Quantity),
			zap.Int64("included", meter.Allowance),
		)
		return w.eachTenant(
			ctx,
			change.Scope,
			func(ctx context.Context, tenant pagination.TenantInfo) error {
				return w.warn(ctx, tenant, meter)
			},
		)
	case was && !is:
		w.l.Info("document storage dropped below the warning share of its allowance",
			zap.String("scope", change.Scope.Key()),
		)
		return w.eachTenant(ctx, change.Scope, w.clear)
	default:
		return nil
	}
}

func nearlyFull(state *managedusage.State) (bool, managedusage.Meter) {
	if state == nil {
		return false, managedusage.Meter{}
	}
	m, ok := state.Meter(platformcatalog.MeterDocumentStorageBytes)
	if !ok {
		return false, managedusage.Meter{}
	}
	return managedusage.NearStorageAllowance(m), m
}

func (w *Warnings) eachTenant(
	ctx context.Context,
	scope managedscope.Scope,
	fn func(context.Context, pagination.TenantInfo) error,
) error {
	if tenant, ok := scope.Tenant(); ok {
		return fn(
			ctx,
			pagination.TenantInfo{OrgID: tenant.OrganizationID, BuID: tenant.BusinessUnitID},
		)
	}
	return tenantports.EachTenant(ctx, w.tenants, fn)
}

func (w *Warnings) warn(
	ctx context.Context,
	tenant pagination.TenantInfo,
	meter managedusage.Meter,
) error {
	_, err := w.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:    tenant,
		Resource:  permission.ResourceOrganization,
		Operation: permission.OpUpdate,
		Limit:     maxNoticeRecipients,
		Now:       w.clock(),
		Notification: notification.Notification{
			EventType:  EventStorageNearlyFull,
			Priority:   notification.PriorityHigh,
			Title:      "Document storage is nearly full",
			Message:    storageMessage(meter),
			Source:     noticeSource,
			SubjectKey: notification.SubjectKey(subjectKind),
			Data: map[string]any{
				"used":     meter.Quantity,
				"included": meter.Allowance,
			},
		},
	})
	return err
}

func storageMessage(meter managedusage.Meter) string {
	return fileutils.HumanizeBytes(meter.Quantity) + " of the " +
		fileutils.HumanizeBytes(meter.Allowance) +
		" your plan includes is used. When it is full, uploads stop unless overage is on in your Trenova account."
}

func (w *Warnings) clear(ctx context.Context, tenant pagination.TenantInfo) error {
	_, err := w.notifier.ResolveSubject(ctx, notificationports.ResolveNotificationSubjectRequest{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		SubjectKey:     notification.SubjectKey(subjectKind),
		Resolution:     notification.ResolutionCleared,
		Now:            w.clock(),
	})
	return err
}
