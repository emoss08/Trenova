package homequerytools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
func homeRequest(params *agenttoolports.QueryToolParams) *homelayoutservice.Request {
	return &homelayoutservice.Request{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		Principal: kernelports.PrincipalInfo{
			Type:     params.Actor.PrincipalType,
			ID:       params.Actor.PrincipalID,
			UserID:   params.Actor.UserID,
			APIKeyID: params.Actor.APIKeyID,
		},
	}
}

const (
	homeLayoutNoteLocked = "The person's administrator manages this home page, so it cannot " +
		"be changed from here. Say so rather than proposing a change."
	homeLayoutNoteEdit = "Pass version to home_add_widget, home_remove_widget or " +
		"home_arrange_layout. Widget ids are the id field."
)

// HomeLayoutGetSpec reads the person's own home page as the page resolves it.
func HomeLayoutGetSpec() readspec.Get {
	return readspec.Get{
		Name:   "home_get_layout",
		Entity: "home page",
		Summary: "Read what is on the person's own home page: each widget, its size and " +
			"settings, and whether they may change it. Use it for \"my dashboard\", \"my home " +
			"page\" or \"what am I looking at\", and before home_add_widget, home_remove_widget " +
			"or home_arrange_layout, which need its version. Report dashboards under Reports " +
			"are separate; report_list_dashboards reads those.",
		SearchTerms: homelayout.SpokenNames(),
		Resource:    permission.ResourceHomeLayoutPreset,
		Scope:       agenttooltrust.ToolScopeSelf,
		Rationale:   "Reads the caller's own home page; nothing changes and nothing is sent.",
		Shape:       labelHomeLayout,
	}
}

// labelHomeLayout names each widget as the page does and says whether the
// person may change the page, which the layout's own fields leave the model
// to work out.
func labelHomeLayout(record map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	widgets, _ := record["widgets"].([]any)
	for _, raw := range widgets {
		widget, _ := raw.(map[string]any)
		key, _ := widget["key"].(string)
		widget["label"] = key
		if definition, ok := homelayout.WidgetDefinitionFor(key); ok {
			widget["label"] = definition.Label
			if definition.ConfigKind == homelayout.ConfigKindNone {
				delete(widget, "config")
			}
		}
	}
	locked, _ := record["locked"].(bool)
	customizable, _ := record["canCustomize"].(bool)
	record[graphqlread.OutcomeNote] = homeLayoutNoteEdit
	if locked || !customizable {
		record[graphqlread.OutcomeNote] = homeLayoutNoteLocked
	}
}

type listHomeWidgetsTool struct {
	layouts homeLayoutReader
}

func NewListHomeWidgetsTool(layouts *homelayoutservice.Service) agenttoolports.AgentQueryTool {
	return &listHomeWidgetsTool{layouts: layouts}
}

func (t *listHomeWidgetsTool) Name() string { return "home_list_widgets" }

func (t *listHomeWidgetsTool) Description() string {
	return "List the widgets the person may put on their home page, with what each needs set, " +
		"and the metrics a KPI widget can show. Use it before home_add_widget to pick a real " +
		"widget key and metric; never invent one."
}

func (t *listHomeWidgetsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			querytool.ParamCategory: agenttoolschema.Enum(
				"Optional group: work (queues), pulse (figures), insight, orientation or comms.",
				homeWidgetCategories,
			),
		},
		"additionalProperties": false,
	}
}

func (t *listHomeWidgetsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceHomeLayoutPreset,
		Scope:    agenttooltrust.ToolScopeSelf,
		Rationale: "Lists the widgets the caller's home page can show; nothing changes and " +
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
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	catalog, err := t.layouts.GetCatalog(ctx, homeRequest(params))
	if err != nil {
		return nil, err
	}

	category := querytool.OptionalString(params.Params, "category")
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
