package supportaccessnotice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	source           = "supportaccessnotice"
	EventGranted     = "support_access.granted"
	EventExtended    = "support_access.extended"
	EventFirstEntry  = "support_access.first_entry"
	EventEnded       = "support_access.ended"
	SupportAccessURL = "/admin/support-access"
	maxRecipients    = 200
	endsLayout       = "Jan 2, 15:04 UTC"
)

type RecipientLister interface {
	ListUsersWithPermission(
		ctx context.Context,
		req permissionports.ListUsersWithPermissionRequest,
	) ([]permissionports.PermittedUser, error)
}

type NotificationCreator interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type Params struct {
	fx.In

	Roles         permissionports.RoleRepository
	Notifications *notificationservice.Service
}

type Notifier struct {
	recipients    RecipientLister
	notifications NotificationCreator
}

var _ supportaccessports.Notifier = (*Notifier)(nil)

func New(p Params) *Notifier {
	return &Notifier{recipients: p.Roles, notifications: p.Notifications}
}

type content struct {
	event    string
	priority notification.Priority
	title    string
	message  string
}

func (n *Notifier) SupportAccessChanged(
	ctx context.Context,
	notice *supportaccessports.Notice,
) error {
	grant := notice.Grant
	if grant == nil {
		return nil
	}
	recipients, err := n.recipients.ListUsersWithPermission(
		ctx,
		permissionports.ListUsersWithPermissionRequest{
			OrganizationID: notice.TenantInfo.OrgID,
			BusinessUnitID: notice.TenantInfo.BuID,
			Resource:       permission.ResourceSupportAccess,
			Operation:      permission.OpRead,
			Now:            timeutils.NowUnix(),
		},
	)
	if err != nil {
		return fmt.Errorf("list who manages support access: %w", err)
	}
	if len(recipients) > maxRecipients {
		recipients = recipients[:maxRecipients]
	}

	said := describe(ctx, notice)
	correlation := grant.ID.String()
	buID := notice.TenantInfo.BuID
	for _, recipient := range recipients {
		if recipient.UserID.IsNil() || supportaccess.IsPrincipalID(recipient.UserID) ||
			recipient.UserID == notice.ActorID {
			continue
		}
		userID := recipient.UserID
		if _, err = n.notifications.Create(ctx, &notification.Notification{
			OrganizationID: notice.TenantInfo.OrgID,
			BusinessUnitID: &buID,
			TargetUserID:   &userID,
			Channel:        notification.ChannelUser,
			EventType:      said.event,
			Priority:       said.priority,
			Title:          said.title,
			Message:        said.message,
			Source:         source,
			CorrelationID:  &correlation,
			Data: map[string]any{
				"link":       SupportAccessURL,
				"staffEmail": grant.StaffEmail,
				"scope":      grant.Scope.String(),
				"endsAt":     grant.ExpiresAt,
			},
		}); err != nil {
			return fmt.Errorf("notify an administrator about support access: %w", err)
		}
	}
	return nil
}

func describe(ctx context.Context, notice *supportaccessports.Notice) content {
	grant := notice.Grant
	ends := time.Unix(grant.ExpiresAt, 0).UTC().Format(endsLayout)
	full := grant.Scope == supportaccess.ScopeFull
	switch notice.Kind {
	case supportaccessports.NoticeGranted:
		return content{
			event:    EventGranted,
			priority: notification.PriorityMedium,
			title:    i18n.T(ctx, "Trenova support was given access"),
			message:  grantedMessage(ctx, full, grant.StaffEmail, ends),
		}
	case supportaccessports.NoticeExtended:
		return content{
			event:    EventExtended,
			priority: notification.PriorityMedium,
			title:    i18n.T(ctx, "Trenova support access was extended"),
			message:  i18n.T(ctx, "{0} now has access until {1}.", grant.StaffEmail, ends),
		}
	case supportaccessports.NoticeFirstEntry:
		return content{
			event:    EventFirstEntry,
			priority: notification.PriorityHigh,
			title:    i18n.T(ctx, "Trenova support entered this organization"),
			message:  enteredMessage(ctx, full, notice.StaffName, grant.StaffEmail, ends),
		}
	case supportaccessports.NoticeRevoked:
		return content{
			event:    EventEnded,
			priority: notification.PriorityMedium,
			title:    i18n.T(ctx, "Trenova support access ended"),
			message: i18n.T(
				ctx,
				"Access for {0} was ended: {1}",
				grant.StaffEmail,
				grant.RevokeReason,
			),
		}
	case supportaccessports.NoticeExpired:
		return endedOnItsOwn(ctx, grant.StaffEmail, ends)
	}
	return endedOnItsOwn(ctx, grant.StaffEmail, ends)
}

func endedOnItsOwn(ctx context.Context, staffEmail, ends string) content {
	return content{
		event:    EventEnded,
		priority: notification.PriorityLow,
		title:    i18n.T(ctx, "Trenova support access ended"),
		message:  i18n.T(ctx, "Access for {0} ended on its own at {1}.", staffEmail, ends),
	}
}

func grantedMessage(ctx context.Context, full bool, staffEmail, ends string) string {
	if full {
		return i18n.T(ctx, "{0} has full access to this organization until {1}.", staffEmail, ends)
	}
	return i18n.T(ctx, "{0} has read-only access to this organization until {1}.", staffEmail, ends)
}

func enteredMessage(ctx context.Context, full bool, staffName, staffEmail, ends string) string {
	if full {
		return i18n.T(ctx, "{0} ({1}) is in this organization with full access. It ends by {2}.",
			staffName, staffEmail, ends)
	}
	return i18n.T(ctx, "{0} ({1}) is in this organization with read-only access. It ends by {2}.",
		staffName, staffEmail, ends)
}
