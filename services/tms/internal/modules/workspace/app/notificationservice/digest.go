package notificationservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	DigestHour           = 8
	digestLookback       = 24 * time.Hour
	digestMinimumSpacing = 20 * time.Hour
	digestIdempotency    = "notification-digest-"
)

type DigestRunResult struct {
	Subscribers int `json:"subscribers"`
	Due         int `json:"due"`
	Sent        int `json:"sent"`
	Empty       int `json:"empty"`
	Failed      int `json:"failed"`
}

func DigestDue(pref *notification.Preference, now time.Time, loc *time.Location) bool {
	if pref.EmailMode != notification.DeliveryModeAll || !pref.EmailDigest {
		return false
	}
	if loc == nil {
		loc = time.UTC
	}
	if now.In(loc).Hour() != DigestHour {
		return false
	}
	if pref.LastDigestAt == nil {
		return true
	}
	return now.Unix()-*pref.LastDigestAt >= int64(digestMinimumSpacing/time.Second)
}

func (s *Service) SendDueDigests(ctx context.Context, now time.Time) (*DigestRunResult, error) {
	result := new(DigestRunResult)
	if s.preferences == nil || s.users == nil || s.email == nil {
		return result, nil
	}

	subscribers, err := s.preferences.ListDigestSubscribers(ctx)
	if err != nil {
		return nil, err
	}
	result.Subscribers = len(subscribers)

	for _, pref := range subscribers {
		tenantInfo := pagination.TenantInfo{
			OrgID:  pref.OrganizationID,
			BuID:   pref.BusinessUnitID,
			UserID: pref.UserID,
		}
		tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())

		sent, sendErr := s.sendDigest(tenantCtx, tenantInfo, pref, now, result)
		switch {
		case sendErr != nil:
			result.Failed++
			s.l.Warn("failed to send notification digest",
				zap.String("userID", pref.UserID.String()),
				zap.Error(sendErr),
			)
		case sent:
			result.Sent++
		}
	}

	return result, nil
}

func (s *Service) sendDigest(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	pref *notification.Preference,
	now time.Time,
	result *DigestRunResult,
) (bool, error) {
	user, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     tenantInfo.UserID,
		TenantInfo: tenantInfo,
		Now:        now.Unix(),
	})
	if err != nil {
		return false, err
	}

	location, locErr := time.LoadLocation(user.Timezone)
	if locErr != nil || user.Timezone == "" {
		location = time.UTC
	}
	if !DigestDue(pref, now, location) {
		return false, nil
	}
	result.Due++

	if pref.IsPaused(now.Unix()) {
		return false, nil
	}

	since := now.Add(-digestLookback).Unix()
	if pref.LastDigestAt != nil && *pref.LastDigestAt > since {
		since = *pref.LastDigestAt
	}

	items, err := s.preferences.ListDigestNotifications(ctx, notificationports.ListDigestNotificationsRequest{
		TenantInfo: tenantInfo,
		Since:      since,
		Until:      now.Unix(),
	})
	if err != nil {
		return false, err
	}

	markSent := func() error {
		return s.preferences.MarkDigestSent(ctx, notificationports.MarkDigestSentRequest{
			TenantInfo: tenantInfo,
			SentAt:     now.Unix(),
		})
	}

	address := strings.TrimSpace(user.EmailAddress)
	if len(items) == 0 || address == "" {
		result.Empty++
		return false, markSent()
	}

	digest := &digestEmail{
		RecipientName: user.Name,
		Items:         make([]digestItem, 0, len(items)),
		SettingsURL:   s.settingsURL(),
	}
	if s.webBaseURL != "" {
		digest.BellURL = s.webBaseURL + "/"
	}
	for _, item := range items {
		digest.Items = append(digest.Items, digestItem{
			Title:   item.Title,
			Message: item.Message,
			Link:    s.absoluteLink(item),
		})
	}

	rendered, err := renderDigestEmail(digest)
	if err != nil {
		return false, err
	}

	day := now.In(location).Format(time.DateOnly)
	if _, err = s.email.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeNotifications,
		To:             []string{address},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: digestIdempotency + tenantInfo.UserID.String() + "-" + day,
	}); err != nil {
		return false, err
	}

	return true, markSent()
}
