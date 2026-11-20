package plannotice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventPlanChanged    = "managed_plan_changed"
	PlanPath            = "/admin/plan"
	PlanResource        = "managed_plan"
	noticeSource        = "plannotice"
	subjectKind         = "managed-plan"
	planEndSubject      = "plan-end"
	maxNoticeRecipients = 25
	changeLeft          = "left"
	changeJoined        = "joined"
)

type Notifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type Params struct {
	fx.In

	Notifications *notificationservice.Service
	Tenants       tenantports.TenantSyncRepository
	Realtime      realtimeports.RealtimeService
	Clock         timeutils.Clock
	Logger        *zap.Logger
}

type Result struct {
	fx.Out

	Observer managedfeaturesports.PlanObserver `group:"managed_plan_observers"`
}

type Notices struct {
	notifier Notifier
	tenants  tenantports.TenantPager
	realtime realtimeports.RealtimeService
	clock    timeutils.Clock
	l        *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) Result {
	return Result{Observer: &Notices{
		notifier: p.Notifications,
		tenants:  p.Tenants,
		realtime: p.Realtime,
		clock:    p.Clock,
		l:        p.Logger.Named("service.plan-notices"),
	}}
}

func (n *Notices) PlanChanged(
	ctx context.Context,
	replacement *managedfeatures.Replacement,
) error {
	if !replacement.Updated() {
		return nil
	}

	notices := n.addOnNotices(ctx, replacement)
	if end := n.planEndNotice(ctx, replacement.PlanEnd); end != nil {
		notices = append(notices, *end)
		n.l.Info("the plan's end changed; telling the administrators of the scope it concerns",
			zap.String("scope", replacement.Scope.Key()),
			zap.String("notice", string(replacement.PlanEnd.Kind)),
			zap.Int64("endsAt", replacement.PlanEnd.EndsAt),
		)
	}

	return n.eachTenant(
		ctx,
		replacement.Scope,
		func(ctx context.Context, tenant pagination.TenantInfo) error {
			return n.tell(ctx, tenant, notices)
		},
	)
}

func (n *Notices) tell(
	ctx context.Context,
	tenant pagination.TenantInfo,
	notices []notification.Notification,
) error {
	var errs []error
	for i := range notices {
		if _, err := n.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
			Tenant:       tenant,
			Resource:     permission.ResourceOrganization,
			Operation:    permission.OpUpdate,
			Limit:        maxNoticeRecipients,
			Now:          n.clock(),
			Notification: notices[i],
		}); err != nil {
			errs = append(errs, err)
		}
	}
	if err := n.realtime.PublishResourceInvalidation(
		ctx,
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Resource:       PlanResource,
			Action:         "updated",
		},
	); err != nil {
		n.l.Warn("cannot tell open pages that the plan changed", zap.Error(err))
	}
	return errors.Join(errs...)
}

func (n *Notices) addOnNotices(
	ctx context.Context,
	replacement *managedfeatures.Replacement,
) []notification.Notification {
	if !replacement.FeaturesChanged() || !replacement.Previous.HasList() {
		return []notification.Notification{}
	}
	shift := managedfeatures.Shift(replacement.Previous.Features, replacement.Current.Features)
	if shift.Empty() {
		return []notification.Notification{}
	}
	n.l.Info("the plan's add-ons changed; telling the administrators of the scope it concerns",
		zap.String("scope", replacement.Scope.Key()),
		zap.Int("left", len(shift.Left)),
		zap.Int("joined", len(shift.Joined)),
	)

	out := make([]notification.Notification, 0, len(shift.Left)+len(shift.Joined)+1)
	for _, addOn := range shift.Left {
		out = append(out, notice(addOn, changeLeft, notification.PriorityHigh,
			i18n.T(ctx, "{0} is no longer on your plan", i18n.Term(addOn.Name)),
			i18n.T(
				ctx,
				"Everything you entered is kept. It comes back if you add {0} again from your Trenova account.",
				i18n.Term(addOn.Name),
			),
		))
	}
	for _, addOn := range shift.Joined {
		out = append(out, notice(addOn, changeJoined, notification.PriorityMedium,
			i18n.T(ctx, "{0} is now on your plan", i18n.Term(addOn.Name)),
			i18n.T(ctx, "Open Your plan to see what it adds."),
		))
	}
	return out
}

func (n *Notices) planEndNotice(
	ctx context.Context,
	end *managedfeatures.PlanEndNotice,
) *notification.Notification {
	if end == nil {
		return nil
	}
	day := i18n.LongDate(i18n.FromContext(ctx), time.Unix(end.EndsAt, 0).UTC())
	var title, message string
	priority := notification.PriorityHigh
	switch end.Kind {
	case managedfeatures.PlanEndSet:
		title = i18n.T(ctx, "Your plan ends on {0}", day)
		message = i18n.T(
			ctx,
			"Trenova stays available until then. After that, this workspace is taken down and a final backup is kept for {0, plural, one {# day} other {# days}}. An owner or admin of your Trenova account can keep the plan before then.",
			end.FinalBackupDays,
		)
	case managedfeatures.PlanEndKept:
		priority = notification.PriorityMedium
		title = i18n.T(ctx, "Your plan continues")
		message = i18n.T(ctx, "It will no longer end on {0}.", day)
	case managedfeatures.PlanEndReminder:
		title = i18n.T(ctx, "Your plan ends in 3 days")
		message = i18n.T(
			ctx,
			"After {0}, this workspace is taken down and a final backup is kept for {1, plural, one {# day} other {# days}}. An owner or admin of your Trenova account can keep the plan before then.",
			day,
			end.FinalBackupDays,
		)
	default:
		return nil
	}
	return &notification.Notification{
		EventType:  EventPlanChanged,
		Priority:   priority,
		Title:      title,
		Message:    message,
		Source:     noticeSource,
		SubjectKey: notification.SubjectKey(subjectKind, planEndSubject),
		Data: map[string]any{
			"link":   PlanPath,
			"change": string(end.Kind),
			"endsAt": end.EndsAt,
		},
	}
}

func notice(
	addOn platformcatalog.AddOn,
	change string,
	priority notification.Priority,
	title, message string,
) notification.Notification {
	return notification.Notification{
		EventType:  EventPlanChanged,
		Priority:   priority,
		Title:      title,
		Message:    message,
		Source:     noticeSource,
		SubjectKey: notification.SubjectKey(subjectKind, string(addOn.Key)),
		Data: map[string]any{
			"link":   PlanPath,
			"addOn":  addOn.Name,
			"change": change,
		},
	}
}

func (n *Notices) eachTenant(
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
	return tenantports.EachTenant(ctx, n.tenants, entitlement.AnyActive(), fn)
}
