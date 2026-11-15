package plannotice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventPlanChanged    = "managed_plan_changed"
	PlanPath            = "/admin/plan"
	noticeSource        = "plannotice"
	subjectKind         = "managed-plan"
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

type TenantPager interface {
	ListTenantsPage(ctx context.Context, after pulid.ID, limit int) ([]pagination.TenantInfo, error)
}

type Params struct {
	fx.In

	Notifications *notificationservice.Service
	Tenants       tenantports.TenantSyncRepository
	Clock         timeutils.Clock
	Logger        *zap.Logger
}

type Result struct {
	fx.Out

	Observer managedfeaturesports.PlanObserver `group:"managed_plan_observers"`
}

type Notices struct {
	notifier Notifier
	tenants  TenantPager
	clock    timeutils.Clock
	l        *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) Result {
	return Result{Observer: &Notices{
		notifier: p.Notifications,
		tenants:  p.Tenants,
		clock:    p.Clock,
		l:        p.Logger.Named("service.plan-notices"),
	}}
}

func (n *Notices) PlanChanged(
	ctx context.Context,
	replacement *managedfeatures.Replacement,
) error {
	if !replacement.Changed() || replacement.Change.Starting || replacement.Previous == nil {
		return nil
	}

	shift := managedfeatures.Shift(replacement.Previous.Features, replacement.Current.Features)
	if shift.Empty() {
		return nil
	}
	notices := n.notices(ctx, shift)
	n.l.Info("the plan's add-ons changed; telling each organization's administrators",
		zap.Int("left", len(shift.Left)),
		zap.Int("joined", len(shift.Joined)),
	)

	return n.eachTenant(ctx, func(ctx context.Context, tenant pagination.TenantInfo) error {
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
		return errors.Join(errs...)
	})
}

func (n *Notices) notices(
	ctx context.Context,
	shift managedfeatures.AddOnShift,
) []notification.Notification {
	out := make([]notification.Notification, 0, len(shift.Left)+len(shift.Joined))
	for _, addOn := range shift.Left {
		out = append(out, notice(addOn, changeLeft, notification.PriorityHigh,
			i18n.T(ctx, "{0} is no longer on your plan", addOn.Name),
			i18n.T(
				ctx,
				"Everything you entered is kept. It comes back if you add {0} again from your Trenova account.",
				addOn.Name,
			),
		))
	}
	for _, addOn := range shift.Joined {
		out = append(out, notice(addOn, changeJoined, notification.PriorityMedium,
			i18n.T(ctx, "{0} is now on your plan", addOn.Name),
			i18n.T(ctx, "Open Your plan to see what it adds."),
		))
	}
	return out
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
	fn func(context.Context, pagination.TenantInfo) error,
) error {
	return tenantports.EachTenant(ctx, n.tenants, fn)
}
