package render

import (
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Registry struct {
	renderers map[report.Format]services.ReportRenderer
}

type RegistryParams struct {
	fx.In

	Renderers []services.ReportRenderer `group:"report_renderers"`
}

func NewRegistry(p RegistryParams) *Registry {
	registry := &Registry{
		renderers: make(map[report.Format]services.ReportRenderer, len(p.Renderers)),
	}
	for _, renderer := range p.Renderers {
		registry.renderers[renderer.Format()] = renderer
	}
	return registry
}

func (r *Registry) For(format report.Format) (services.ReportRenderer, error) {
	renderer, ok := r.renderers[format]
	if !ok {
		return nil, fmt.Errorf("no renderer registered for format %q", format)
	}
	return renderer, nil
}

func asRenderer(constructor any) fx.Option {
	return fx.Provide(fx.Annotate(
		constructor,
		fx.As(new(services.ReportRenderer)),
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
			fx.As(new(services.ReportRendererRegistry)),
		),
	),
)

func metaLocation(meta *services.ReportRunMeta) *time.Location {
	if meta.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(meta.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func formatCell(
	column *services.ReportResultColumn,
	value any,
	loc *time.Location,
) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case decimal.Decimal:
		return v.String()
	case int64:
		if column.Type == reportcatalog.FieldEpoch {
			return time.Unix(v, 0).In(loc).Format("2006-01-02 15:04:05")
		}
		return strconv.FormatInt(v, 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

const truncationNotice = "Results were truncated at the row limit; narrow your filters or use a larger export format."
