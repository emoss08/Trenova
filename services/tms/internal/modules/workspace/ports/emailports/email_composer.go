package emailports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
)

type ComposedEmail struct {
	Subject string
	HTML    string
	Text    string
}

type NotificationEmail struct {
	TenantInfo    pagination.TenantInfo
	Locale        i18n.Locale
	RecipientName string
	Title         string
	Message       string
	Link          string
	SettingsURL   string
}

type NotificationDigestEmail struct {
	TenantInfo    pagination.TenantInfo
	Locale        i18n.Locale
	RecipientName string
	Items         []NotificationDigestItem
	BellURL       string
	SettingsURL   string
}

type NotificationDigestItem struct {
	Title   string
	Message string
	Link    string
}

type ProfileTestEmail struct {
	TenantInfo   pagination.TenantInfo
	ProfileName  string
	ProviderName string
	SenderEmail  string
}

type Composer interface {
	ComposeNotification(ctx context.Context, req *NotificationEmail) (*ComposedEmail, error)
	ComposeNotificationDigest(
		ctx context.Context,
		req *NotificationDigestEmail,
	) (*ComposedEmail, error)
	ComposeProfileTest(ctx context.Context, req *ProfileTestEmail) (*ComposedEmail, error)
}
