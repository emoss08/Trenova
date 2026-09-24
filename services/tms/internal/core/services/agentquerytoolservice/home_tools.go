package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/pkg/pagination"
)

/*
The person's own home page, which is what most people mean by "my dashboard".

The Homepage Widget Builder was asked what was on the person's dashboard and
could only list the report dashboards under Reports, which were not what they
meant. These two read the home page the way the page itself resolves it — the
person's own layout, else their role's preset, else the organization default,
else the built-in one — narrowed to the widgets they may see.
*/

type homeLayoutReader interface {
	GetEffective(
		ctx context.Context,
		req *homelayoutservice.Request,
	) (*homelayoutservice.EffectiveLayout, error)
	GetCatalog(
		ctx context.Context,
		req *homelayoutservice.Request,
	) (*homelayoutservice.WidgetCatalog, error)
}

// homeRequest is the reader's own home page request: their tenant, and the
// principal the page resolves for.
func homeRequest(params *serviceports.QueryToolParams) *homelayoutservice.Request {
	return &homelayoutservice.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Principal: serviceports.PrincipalInfo{
			Type:     params.Actor.PrincipalType,
			ID:       params.Actor.PrincipalID,
			UserID:   params.Actor.UserID,
			APIKeyID: params.Actor.APIKeyID,
		},
	}
}

type getMyHomeLayoutTool struct {
	layouts homeLayoutReader
}

func newGetMyHomeLayoutTool(layouts *homelayoutservice.Service) serviceports.AgentQueryTool {
	return &getMyHomeLayoutTool{layouts: layouts}
}

func (t *getMyHomeLayoutTool) Name() string { return "get_my_home_layout" }

func (t *getMyHomeLayoutTool) Description() string {
	return "Read what is on the person's own home page: each widget, its size and settings, " +
		"and whether they may change it. Use it for \"my dashboard\", \"my home page\" or " +
		"\"what am I looking at\", and before add_home_widget, remove_home_widget or " +
		"arrange_home_layout, which need its version. Report dashboards under Reports are " +
		"separate; list_dashboards reads those."
}

func (t *getMyHomeLayoutTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

func (t *getMyHomeLayoutTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource:  permission.ResourceHomeLayoutPreset,
		scope:     agent.ToolScopeSelf,
		rationale: "Reads the caller's own home page; nothing changes and nothing is sent.",
	})
}

func (t *getMyHomeLayoutTool) SearchTerms() []string { return homelayout.SpokenNames() }

type homeWidgetRow struct {
	ID     string                   `json:"id"`
	Key    string                   `json:"key"`
	Label  string                   `json:"label"`
	Title  string                   `json:"title,omitempty"`
	Width  int                      `json:"width"`
	Height int                      `json:"height"`
	Config *homelayout.WidgetConfig `json:"config,omitempty"`
}

type homeLayoutView struct {
	// Source is where the layout comes from: user (their own), rolePreset,
	// orgDefault or builtIn.
	Source       string          `json:"source"`
	PresetName   string          `json:"presetName,omitempty"`
	Locked       bool            `json:"locked"`
	CanCustomize bool            `json:"canCustomize"`
	Version      int64           `json:"version"`
	Density      string          `json:"density"`
	Widgets      []homeWidgetRow `json:"widgets"`
	Note         string          `json:"note"`
}

func (t *getMyHomeLayoutTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	effective, err := t.layouts.GetEffective(ctx, homeRequest(params))
	if err != nil {
		return nil, err
	}

	view := homeLayoutView{
		Source:       string(effective.Source),
		PresetName:   effective.PresetName,
		Locked:       effective.Locked,
		CanCustomize: effective.CanCustomize,
		Version:      effective.Version,
		Density:      effective.Density,
		Widgets:      homeWidgetRows(effective.Layout),
	}
	switch {
	case effective.Locked || !effective.CanCustomize:
		view.Note = "The person's administrator manages this home page, so it cannot be " +
			"changed from here. Say so rather than proposing a change."
	default:
		view.Note = "Pass version to add_home_widget, remove_home_widget or " +
			"arrange_home_layout. Widget ids are the id field."
	}

	return view, nil
}

func homeWidgetRows(layout *homelayout.Layout) []homeWidgetRow {
	if layout == nil {
		return []homeWidgetRow{}
	}

	rows := make([]homeWidgetRow, 0, len(layout.Widgets))
	for idx := range layout.Widgets {
		widget := &layout.Widgets[idx]
		row := homeWidgetRow{
			ID:     widget.ID,
			Key:    widget.Key,
			Label:  widget.Key,
			Title:  widget.Title,
			Width:  widget.W,
			Height: widget.H,
		}
		if definition, ok := homelayout.WidgetDefinitionFor(widget.Key); ok {
			row.Label = definition.Label
			if definition.ConfigKind != homelayout.ConfigKindNone {
				config := widget.Config
				row.Config = &config
			}
		}
		rows = append(rows, row)
	}

	return rows
}

type listHomeWidgetsTool struct {
	layouts homeLayoutReader
}

func newListHomeWidgetsTool(layouts *homelayoutservice.Service) serviceports.AgentQueryTool {
	return &listHomeWidgetsTool{layouts: layouts}
}

func (t *listHomeWidgetsTool) Name() string { return "list_home_widgets" }

func (t *listHomeWidgetsTool) Description() string {
	return "List the widgets the person may put on their home page, with what each needs set, " +
		"and the metrics a KPI widget can show. Use it before add_home_widget to pick a real " +
		"widget key and metric; never invent one."
}

func (t *listHomeWidgetsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category": map[string]any{
				"type": "string",
				"enum": []string{
					homelayout.CategoryWork,
					homelayout.CategoryPulse,
					homelayout.CategoryInsight,
					homelayout.CategoryOrientation,
					homelayout.CategoryComms,
				},
				"description": "Optional group: work (queues), pulse (figures), insight, " +
					"orientation or comms.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listHomeWidgetsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceHomeLayoutPreset,
		scope:    agent.ToolScopeSelf,
		rationale: "Lists the widgets the caller's home page can show; nothing changes and " +
			"nothing is sent.",
	})
}

func (t *listHomeWidgetsTool) SearchTerms() []string { return homelayout.SpokenNames() }

type homeWidgetOption struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
	// Needs is what the widget's config must carry: none, queue (limit),
	// metric, metricRow (metrics), trend (windowDays), report, dashboard or
	// text.
	Needs        string `json:"needs"`
	DefaultWidth int    `json:"defaultWidth"`
}

type homeMetricOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type homeWidgetCatalogView struct {
	Widgets    []homeWidgetOption `json:"widgets"`
	Metrics    []homeMetricOption `json:"metrics"`
	MaxWidgets int                `json:"maxWidgets"`
}

func (t *listHomeWidgetsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	catalog, err := t.layouts.GetCatalog(ctx, homeRequest(params))
	if err != nil {
		return nil, err
	}

	category := optionalString(params.Params, "category")
	view := homeWidgetCatalogView{
		Widgets:    make([]homeWidgetOption, 0, len(catalog.Widgets)),
		Metrics:    make([]homeMetricOption, 0, len(catalog.Metrics)),
		MaxWidgets: catalog.MaxWidgets,
	}
	for idx := range catalog.Widgets {
		widget := &catalog.Widgets[idx]
		if category != "" && widget.Category != category {
			continue
		}
		view.Widgets = append(view.Widgets, homeWidgetOption{
			Key:          widget.Key,
			Label:        widget.Label,
			Description:  widget.Description,
			Category:     widget.Category,
			Needs:        string(widget.ConfigKind),
			DefaultWidth: widget.DefaultW,
		})
	}
	for idx := range catalog.Metrics {
		view.Metrics = append(view.Metrics, homeMetricOption{
			Key:   catalog.Metrics[idx].Key,
			Label: catalog.Metrics[idx].Label,
		})
	}

	return view, nil
}
