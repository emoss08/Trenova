package dispatchquerytools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type weatherReader interface {
	GetActiveAlerts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*weatheralertports.WeatherAlertFeatureCollection, error)
}

const (
	weatherExtreme  = "Extreme"
	weatherSevere   = "Severe"
	weatherModerate = "Moderate"
	weatherMinor    = "Minor"
	weatherUnknown  = "Unknown"
)

var weatherSeverities = []string{
	weatherExtreme, weatherSevere, weatherModerate, weatherMinor, weatherUnknown,
}

type weatherAlertRow struct {
	ID          string `json:"id"`
	Event       string `json:"event"`
	Severity    string `json:"severity,omitempty"`
	Urgency     string `json:"urgency,omitempty"`
	Category    string `json:"category"`
	Headline    string `json:"headline,omitempty"`
	Area        string `json:"area,omitempty"`
	Effective   string `json:"effective,omitempty"`
	Expires     string `json:"expires,omitempty"`
	Instruction string `json:"instruction,omitempty"`
}

type listWeatherAlertsTool struct {
	weather weatherReader
}

func NewListWeatherAlertsTool(weather weatherReader) agenttoolports.AgentQueryTool {
	return &listWeatherAlertsTool{weather: weather}
}

func (t *listWeatherAlertsTool) Name() string { return "insight_list_weather_alerts" }

func (t *listWeatherAlertsTool) Description() string {
	return "Active National Weather Service alerts: the event, its severity, the area " +
		"it covers and when it expires. By default only Severe and Extreme alerts are " +
		"returned. The alerts are not matched to lanes or trucks; compare the area " +
		"against a shipment's stops and the tractor's position yourself, and say " +
		"which alert you matched to which load."
}

func (t *listWeatherAlertsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"severity": agenttoolschema.Enum(
				"Return alerts at this severity and worse. Default Severe.", weatherSeverityLevels,
			),
			"query": map[string]any{
				"type": "string",
				"description": "Optional text matched against the event, headline and area, " +
					"such as a state or county.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listWeatherAlertsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceShipment,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceWeather,
		Rationale: "Reads National Weather Service alert text; nothing changes and nothing is " +
			"sent.",
	})
}

// severityRank orders NWS severities so "Severe and worse" is a comparison.
var severityRank = map[string]int{
	weatherExtreme: 4, weatherSevere: 3, weatherModerate: 2, weatherMinor: 1, weatherUnknown: 0,
}

func (t *listWeatherAlertsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	floor := querytool.OptionalString(params.Params, "severity")
	if _, known := severityRank[floor]; !known {
		floor = weatherSevere
	}
	query := strings.ToLower(querytool.OptionalString(params.Params, "query"))

	criteria := filtercatalog.NewCriteria("weather alerts").At(querytool.ClockFor(params))
	criteria.Field("severity at least", floor)
	criteria.Text(query)

	collection, err := t.weather.GetActiveAlerts(ctx, querytool.TenantOf(params))
	if err != nil {
		return nil, err
	}

	rows := make([]weatherAlertRow, 0, len(collection.Features))
	for _, feature := range collection.Features {
		if feature == nil {
			continue
		}
		p := feature.Properties
		if severityRank[p.Severity] < severityRank[floor] {
			continue
		}
		if query != "" && !strings.Contains(
			strings.ToLower(p.Event+" "+p.Headline+" "+p.AreaDesc), query,
		) {
			continue
		}
		row := weatherAlertRow{
			ID:          p.ID.String(),
			Event:       p.Event,
			Severity:    p.Severity,
			Urgency:     p.Urgency,
			Category:    string(p.AlertCategory),
			Headline:    p.Headline,
			Area:        p.AreaDesc,
			Instruction: p.Instruction,
		}
		if p.Effective != nil {
			row.Effective = timeutils.FormatUnixDateTimeIn(*p.Effective, params.Timezone)
		}
		if p.Expires != nil {
			row.Expires = timeutils.FormatUnixDateTimeIn(*p.Expires, params.Timezone)
		}
		rows = append(rows, row)
	}

	return querytool.SearchResult(criteria, rows, len(rows)), nil
}
