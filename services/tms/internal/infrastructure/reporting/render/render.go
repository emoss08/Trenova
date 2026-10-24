package render

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/i18n"
	"go.uber.org/fx"
	"golang.org/x/text/language"
)

type Registry struct {
	renderers map[report.Format]reportports.ReportRenderer
}

type RegistryParams struct {
	fx.In

	Renderers []reportports.ReportRenderer `group:"report_renderers"`
}

func NewRegistry(p RegistryParams) *Registry {
	registry := &Registry{
		renderers: make(map[report.Format]reportports.ReportRenderer, len(p.Renderers)),
	}
	for _, renderer := range p.Renderers {
		registry.renderers[renderer.Format()] = renderer
	}
	return registry
}

func (r *Registry) For(format report.Format) (reportports.ReportRenderer, error) {
	renderer, ok := r.renderers[format]
	if !ok {
		return nil, fmt.Errorf("no renderer registered for format %q", format)
	}
	return renderer, nil
}

func asRenderer(constructor any) fx.Option {
	return fx.Provide(fx.Annotate(
		constructor,
		fx.As(new(reportports.ReportRenderer)),
		fx.ResultTags(`group:"report_renderers"`),
	))
}

var Module = fx.Module("report-renderers",
	asRenderer(NewCSV),
	asRenderer(NewJSON),
	asRenderer(NewXLSX),
	asRenderer(NewPDF),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(reportports.ReportRendererRegistry)),
		),
	),
)

func Location(timezone string) *time.Location {
	if timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func Language(locale string) language.Tag {
	if locale == "" {
		return language.AmericanEnglish
	}
	tag, err := language.Parse(locale)
	if err != nil {
		return language.AmericanEnglish
	}
	return tag
}

func Text(ctx context.Context) export.Text {
	locale := i18n.FromContext(ctx)
	return export.Text{
		Total:    i18n.Translate(locale, "Total"),
		Subtotal: i18n.Translate(locale, "Subtotal"),
		RowsTruncated: i18n.Translate(
			locale,
			"Results were truncated at the row limit; narrow your filters or use a larger export format.",
		),
		BytesTruncated: i18n.Translate(
			locale,
			"Results were truncated at the size limit; narrow your filters or choose fewer columns.",
		),
		DataSheet:   i18n.Translate(locale, "Data"),
		InfoSheet:   i18n.Translate(locale, "Report"),
		Report:      i18n.Translate(locale, "Report"),
		Description: i18n.Translate(locale, "Description"),
		GeneratedAt: i18n.Translate(locale, "Generated At"),
		RequestedBy: i18n.Translate(locale, "Requested By"),
		Parameter:   i18n.Translate(locale, "Parameter"),
	}
}

func Options(ctx context.Context, meta *reportports.ReportRunMeta, maxBytes int64) export.Options {
	params := make([]export.Param, 0, len(meta.Params))
	for _, p := range meta.Params {
		params = append(params, export.Param{Name: p.Name, Value: p.Value})
	}
	return export.Options{
		Meta: export.Meta{
			Title:       meta.Title,
			Description: meta.Description,
			GeneratedAt: time.Unix(meta.GeneratedAtUnix, 0).UTC(),
			RequestedBy: meta.RequestedBy,
			Params:      params,
		},
		Locale:   Language(meta.Locale),
		Location: Location(meta.Timezone),
		MaxBytes: maxBytes,
		Text:     Text(ctx),
	}
}

func stats(s export.Stats) *reportports.ReportRenderStats {
	return &reportports.ReportRenderStats{
		Rows:      s.Rows,
		Bytes:     s.Bytes,
		Truncated: s.Truncated != export.NotTruncated,
	}
}
