package notificationservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var errNoRoleDirectory = errors.New("notifications cannot find who holds a permission")

// NotifyPermittedRequest tells the people who may act on a resource about one
// thing. Notification is the template; the tenant, the recipient and the
// channel are set per person.
type NotifyPermittedRequest struct {
	Tenant    pagination.TenantInfo
	Resource  permission.Resource
	Operation permission.Operation
	// Limit bounds who is told, so one event cannot page a whole company.
	Limit int
	Now   int64
	// DedupeSince, when set with a correlation id on the template, skips the
	// whole notice if the same event and correlation went out since then.
	DedupeSince  int64
	Notification notification.Notification
}

// NotifyPermitted sends one user notification to each person who holds the
// permission, and reports how many went. A failure to reach one person is
// logged and does not stop the rest.
func (s *Service) NotifyPermitted(ctx context.Context, req NotifyPermittedRequest) (int, error) {
	if s.roles == nil {
		return 0, errNoRoleDirectory
	}

	correlation := ""
	if req.Notification.CorrelationID != nil {
		correlation = *req.Notification.CorrelationID
	}
	if req.DedupeSince > 0 && correlation != "" {
		recent, err := s.repo.ExistsRecent(ctx, repositories.ExistsRecentNotificationRequest{
			OrganizationID: req.Tenant.OrgID,
			BusinessUnitID: req.Tenant.BuID,
			EventType:      req.Notification.EventType,
			CorrelationID:  correlation,
			Since:          req.DedupeSince,
		})
		if err != nil {
			return 0, err
		}
		if recent {
			return 0, nil
		}
	}

	recipients, err := s.roles.ListUsersWithPermission(
		ctx,
		repositories.ListUsersWithPermissionRequest{
			OrganizationID: req.Tenant.OrgID,
			BusinessUnitID: req.Tenant.BuID,
			Resource:       req.Resource,
			Operation:      req.Operation,
			Now:            req.Now,
		},
	)
	if err != nil {
		return 0, err
	}
	if req.Limit > 0 && len(recipients) > req.Limit {
		recipients = recipients[:req.Limit]
	}

	sent := 0
	for _, recipient := range recipients {
		entity := req.Notification
		buID := req.Tenant.BuID
		userID := recipient.UserID
		entity.OrganizationID = req.Tenant.OrgID
		entity.BusinessUnitID = &buID
		entity.TargetUserID = &userID
		entity.Channel = notification.ChannelUser

		if _, err = s.Create(ctx, &entity); err != nil {
			s.l.Warn("notification to a permitted user lost",
				zap.String("eventType", entity.EventType),
				zap.String("userId", userID.String()),
				zap.Error(err),
			)

			continue
		}
		sent++
	}

	return sent, nil
}
