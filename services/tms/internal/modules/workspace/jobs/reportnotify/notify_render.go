package reportnotify

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// Wording is a rendered notification: a title and a message.
type Wording struct {
	Title   string
	Message string
}

type Renderer struct {
	templates documenttemplateports.DocumentTemplateResolver
	l         *zap.Logger
}

func NewRenderer(
	templates documenttemplateports.DocumentTemplateResolver,
	l *zap.Logger,
) *Renderer {
	return &Renderer{templates: templates, l: l}
}

// Render renders one report notification through the organization's
// template.
//
// It falls back to the built-in, like every notification: an owner who is never
// told their schedule was disabled will find out from a customer instead. A
// failure that even the built-in cannot survive returns false and the caller skips
// the notification rather than writing one with an empty title.
func (a *Renderer) Render(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	kind documenttemplate.Kind,
	referenceID pulid.ID,
	data *notificationtemplate.ReportNotificationContext,
) (Wording, bool) {
	if a.templates == nil {
		a.l.Warn("template rendering is not configured; skipping report notification",
			zap.String("kind", string(kind)))
		return Wording{}, false
	}

	rendered, err := a.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        tenantInfo,
		Kind:              kind,
		Data:              data,
		ReferenceID:       referenceID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		a.l.Warn("failed to render report notification",
			zap.String("kind", string(kind)), zap.Error(err))
		return Wording{}, false
	}

	return Wording{Title: rendered.Subject, Message: rendered.Text}, true
}
