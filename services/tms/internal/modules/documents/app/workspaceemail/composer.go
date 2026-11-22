package workspaceemail

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/mailtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

const maxSubjectRunes = 200

type Params struct {
	fx.In

	Templates documenttemplateports.DocumentTemplateResolver
}

type Composer struct {
	templates documenttemplateports.DocumentTemplateResolver
}

var _ emailports.Composer = (*Composer)(nil)

func New(p Params) *Composer {
	return &Composer{templates: p.Templates}
}

func (c *Composer) ComposeNotification(
	ctx context.Context,
	req *emailports.NotificationEmail,
) (*emailports.ComposedEmail, error) {
	return c.render(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       documenttemplate.KindNotificationEmail,
		Locale:     req.Locale,
		Data: notificationtemplate.NotificationEmailContext{
			RecipientFirstName: stringutils.FirstName(req.RecipientName),
			Title:              stringutils.OneLine(req.Title, maxSubjectRunes),
			Message:            req.Message,
			Link:               req.Link,
			SettingsURL:        req.SettingsURL,
		},
		FallbackToBuiltIn: true,
	})
}

func (c *Composer) ComposeNotificationDigest(
	ctx context.Context,
	req *emailports.NotificationDigestEmail,
) (*emailports.ComposedEmail, error) {
	items := make([]notificationtemplate.NotificationDigestItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, notificationtemplate.NotificationDigestItem{
			Title:   item.Title,
			Message: item.Message,
			Link:    item.Link,
		})
	}

	return c.render(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       documenttemplate.KindNotificationDigestEmail,
		Locale:     req.Locale,
		Data: notificationtemplate.NotificationDigestEmailContext{
			RecipientFirstName: stringutils.FirstName(req.RecipientName),
			ItemCount:          len(items),
			Items:              items,
			BellURL:            req.BellURL,
			SettingsURL:        req.SettingsURL,
		},
		FallbackToBuiltIn: true,
	})
}

func (c *Composer) ComposeProfileTest(
	ctx context.Context,
	req *emailports.ProfileTestEmail,
) (*emailports.ComposedEmail, error) {
	return c.render(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo: req.TenantInfo,
		Kind:       documenttemplate.KindEmailProfileTestEmail,
		Data: mailtemplate.EmailProfileTestEmailContext{
			ProfileName:  req.ProfileName,
			ProviderName: req.ProviderName,
			SenderEmail:  req.SenderEmail,
		},
	})
}

func (c *Composer) render(
	ctx context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*emailports.ComposedEmail, error) {
	rendered, err := c.templates.RenderMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	return &emailports.ComposedEmail{
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
		Text:    rendered.Text,
	}, nil
}
