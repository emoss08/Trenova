package detentionservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate/starters"
	"github.com/emoss08/trenova/pkg/templateengine"
)

// builtInNoticeEngine is shared because the engine owns a compilation cache that
// is worthless if every caller gets its own.
var builtInNoticeEngine = sync.OnceValue(func() *templateengine.Engine {
	return templateengine.MustNew(templateengine.Config{})
})

// RenderBuiltInNotice renders a notice from the shipped default template.
//
// It needs no database, no resolver, and no organization: it answers "what does
// the built-in say about this occurrence". Seeds use it to produce notice text
// identical to what a fresh organization would actually send, rather than
// inventing prose that would drift from the template.
//
// The send path does not use this — it goes through the resolver so a customer
// assignment or an organization default wins where one exists.
func RenderBuiltInNotice(
	ctx context.Context,
	p *BuildNoticeParams,
) (NoticeContent, error) {
	// No repositories: branding is skipped, which is right here. A built-in
	// render has no organization to take a letterhead from.
	builder := &ContextBuilder{}

	data, err := builder.BuildFrom(ctx, &buildNoticeContextParams{
		Occurrence:    p.Occurrence,
		Kind:          p.Kind,
		FacilityName:  p.FacilityName,
		ShipmentRef:   p.ShipmentRef,
		CustomerName:  p.CustomerName,
		ArrivalSource: p.ArrivalSource,
		Location:      p.Location,
	})
	if err != nil {
		return NoticeContent{}, err
	}

	starter, err := starters.For(documenttemplate.KindDetentionNoticeEmail)
	if err != nil {
		return NoticeContent{}, err
	}

	engine := builtInNoticeEngine()
	allowed := documenttemplate.NewRegistry().Paths(documenttemplate.KindDetentionNoticeEmail)

	var content NoticeContent
	for _, channel := range []struct {
		channel documenttemplate.Channel
		source  string
		assign  func(string)
	}{
		{documenttemplate.ChannelSubject, starter.Subject, func(v string) { content.Subject = v }},
		{documenttemplate.ChannelEmailText, starter.BodyText, func(v string) { content.Text = v }},
		{documenttemplate.ChannelEmailHTML, starter.BodyHTML, func(v string) { content.HTML = v }},
	} {
		if channel.source == "" {
			continue
		}

		compiled, diags := engine.Parse(&templateengine.ParseRequest{
			Field:        channel.channel.Field(),
			Kind:         string(documenttemplate.KindDetentionNoticeEmail),
			Channel:      channel.channel.Engine(),
			Source:       channel.source,
			AllowedPaths: allowed,
			// The starter is immutable and its content hash is stable, so it is
			// worth caching across the many notices a seed run produces.
			CacheKey: "builtin:detention.notice.email:" + string(channel.channel),
		})
		if diags.HasErrors() {
			return NoticeContent{}, fmt.Errorf(
				"the built-in detention notice does not compile: %s", diags.Summary(),
			)
		}

		rendered, rErr := engine.Render(ctx, compiled, data)
		if rErr != nil {
			return NoticeContent{}, rErr
		}
		channel.assign(rendered)
	}

	return content, nil
}
