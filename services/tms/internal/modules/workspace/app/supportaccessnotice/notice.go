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
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	source        = "supportaccessnotice"
	eventType     = "support_access.session_started"
	maxRecipients = 200
)

type RecipientLister interface {
	ListUsersWithPermission(
		ctx context.Context,
		req permissionports.ListUsersWithPermissionRequest,
	) ([]permissionports.PermittedUser, error)
}

type NotificationCreator interface {
	Create(ctx context.Context, entity *notification.Notification) (*notification.Notification, error)
}

type Params struct {
	fx.In

	Roles         permissionports.RoleRepository
	Notifications *notificationservice.Service
}

// Notifier tells the people who manage support access in an organization that a Trenova support
// session started there. It notifies on the start of a session, never for each action.
type Notifier struct {
	recipients    RecipientLister
	notifications NotificationCreator
}

var _ supportaccessports.Notifier = (*Notifier)(nil)

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Notifier {
	return &Notifier{recipients: p.Roles, notifications: p.Notifications}
}

func (n *Notifier) SupportSessionStarted(
	ctx context.Context,
	notice *supportaccessports.SessionStartedNotice,
) error {
	recipients, err := n.recipients.ListUsersWithPermission(ctx, permissionports.ListUsersWithPermissionRequest{
		OrganizationID: notice.TenantInfo.OrgID,
		BusinessUnitID: notice.TenantInfo.BuID,
		Resource:       permission.ResourceSupportAccess,
		Operation:      permission.OpRead,
		Now:            timeutils.NowUnix(),
	})
	if err != nil {
		return fmt.Errorf("list who manages support access: %w", err)
	}
	if len(recipients) > maxRecipients {
		recipients = recipients[:maxRecipients]
	}

	scope := "read-only"
	if notice.Scope == supportaccess.ScopeFull {
		scope = "full"
	}
	message := fmt.Sprintf(
		"%s (%s) started a Trenova support session with %s access. It ends by %s UTC.",
		notice.StaffName,
		notice.StaffEmail,
		scope,
		time.Unix(notice.EndsAt, 0).UTC().Format("Jan 2, 15:04"),
	)
	correlation := notice.GrantID.String()
	buID := notice.TenantInfo.BuID

	for _, recipient := range recipients {
		if recipient.UserID.IsNil() || supportaccess.IsPrincipalID(recipient.UserID) {
			continue
		}
		userID := recipient.UserID
		if _, err = n.notifications.Create(ctx, &notification.Notification{
			OrganizationID: notice.TenantInfo.OrgID,
			BusinessUnitID: &buID,
			TargetUserID:   &userID,
			Channel:        notification.ChannelUser,
			EventType:      eventType,
			Priority:       notification.PriorityHigh,
			Title:          "Trenova support session started",
			Message:        message,
			Source:         source,
			CorrelationID:  &correlation,
			Data: map[string]any{
				"link":       "/admin/support-access?grant=" + correlation,
				"grantId":    correlation,
				"staffEmail": notice.StaffEmail,
				"staffName":  notice.StaffName,
				"scope":      notice.Scope.String(),
				"endsAt":     notice.EndsAt,
			},
		}); err != nil {
			return fmt.Errorf("notify %s: %w", userID, err)
		}
	}
	return nil
}
