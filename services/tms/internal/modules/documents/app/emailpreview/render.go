package emailpreview

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templateregistry"
	"github.com/emoss08/trenova/pkg/emaildesign"
	"github.com/emoss08/trenova/pkg/templateengine"
	"github.com/emoss08/trenova/shared/i18n"
)

type Rendered struct {
	Subject string
	Content string
	HTML    string
	Text    string
}

type Renderer struct {
	engine   *templateengine.Engine
	registry *templatecatalog.Registry
}

func NewRenderer() *Renderer {
	return &Renderer{
		engine:   templateengine.MustNew(templateengine.Config{}),
		registry: templateregistry.Default(),
	}
}

func (r *Renderer) Render(ctx context.Context, c *Case, locale i18n.Locale) (*Rendered, error) {
	starter, err := starters.ForLocale(c.Kind, locale)
	if err != nil {
		return nil, err
	}

	out := &Rendered{}
	for _, channel := range []struct {
		channel documenttemplate.Channel
		target  *string
	}{
		{documenttemplate.ChannelSubject, &out.Subject},
		{documenttemplate.ChannelEmailHTML, &out.Content},
		{documenttemplate.ChannelEmailText, &out.Text},
	} {
		compiled, diags := r.engine.Parse(&templateengine.ParseRequest{
			Field:        channel.channel.Field(),
			Kind:         string(c.Kind),
			Channel:      channel.channel.Engine(),
			Source:       starter.Channel(channel.channel),
			AllowedPaths: r.registry.Paths(c.Kind),
			Strict:       true,
			Locale:       locale,
		})
		if diags.HasErrors() {
			return nil, fmt.Errorf("%s %s does not compile: %s", c.Kind, channel.channel,
				diags.Summary())
		}
		rendered, renderErr := r.engine.Render(ctx, compiled, c.Data)
		if renderErr != nil {
			return nil, fmt.Errorf("render %s %s: %w", c.Kind, channel.channel, renderErr)
		}
		*channel.target = rendered
	}

	composed, diags := emaildesign.Compose(&emaildesign.Message{
		Locale:  locale,
		Subject: out.Subject,
		Brand:   c.Brand,
		Content: out.Content,
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("%s is not a sendable email: %s", c.Kind, diags.Summary())
	}
	out.HTML = composed

	return out, nil
}
