package alerttools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/homelayoutservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	toolReportList    = "report_list"
	toolHomeGetLayout = "home_get_layout"
)

type homeLayoutWriter interface {
	GetEffective(
		ctx context.Context,
		req *homelayoutservice.Request,
	) (*homelayoutservice.EffectiveLayout, error)
	GetCatalog(
		ctx context.Context,
		req *homelayoutservice.Request,
	) (*homelayoutservice.WidgetCatalog, error)
	Update(
		ctx context.Context,
		req *homelayoutservice.UpdateRequest,
	) (*homelayoutservice.EffectiveLayout, error)
}

// homeTargetChecker confirms that a report or dashboard a widget points at is
// one the person can open.
type homeTargetChecker interface {
	UnavailableTileTargets(
		ctx context.Context,
		tenant pagination.TenantInfo,
		tiles []report.DashboardTile,
	) []reportdashboardservice.UnavailableTile
	GetDashboard(
		ctx context.Context,
		req *reportdashboardservice.GetDashboardRequest,
	) (*report.Dashboard, error)
}

// homeEdit changes a layout in place, adding a field error for anything it
// cannot do.
type homeEdit func(
	ctx context.Context,
	layout *homelayout.Layout,
	catalog *homelayoutservice.WidgetCatalog,
	multiErr *errortypes.MultiError,
)

type homeEditor struct {
	layouts homeLayoutWriter
	targets homeTargetChecker
}

func homeRequestFor(params *agenttoolports.ToolExecuteParams) *homelayoutservice.Request {
	return &homelayoutservice.Request{
		TenantInfo: toolparams.TenantFrom(*params),
		Principal: kernelports.PrincipalInfo{
			Type:     params.Actor.PrincipalType,
			ID:       params.Actor.PrincipalID,
			UserID:   params.Actor.UserID,
			APIKeyID: params.Actor.APIKeyID,
		},
	}
}

// plan reads the person's home page, applies an edit, and returns the save
// that would make it. Validate, Preview and Execute all go through it, so
// what a person approves is what runs.
func (e homeEditor) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	edit homeEdit,
) (*homePlan, error) {
	if params.Actor == nil || params.Actor.PrincipalType != permissionports.PrincipalTypeUser {
		return nil, errortypes.NewAuthorizationError("Only a person has a home page to change")
	}
	owner := toolparams.OptionalString(params.Params, agenttoolports.SelfScopeOwnerParam)
	if owner == "" || owner != params.Actor.UserID.String() {
		return nil, errortypes.NewAuthorizationError(
			"This change is to someone else's home page; only they can approve it",
		)
	}

	request := homeRequestFor(params)
	effective, err := e.layouts.GetEffective(ctx, request)
	if err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	if effective.Locked || !effective.CanCustomize {
		multiErr.Add("layout", errortypes.ErrForbidden,
			"The person's administrator manages this home page, so it cannot be changed here")
		return nil, multiErr
	}
	if version := toolparams.OptionalInt64(params.Params, "version"); version != effective.Version {
		multiErr.Add("version", errortypes.ErrVersionMismatch, fmt.Sprintf(
			"The home page has changed since version %d was read; call home_get_layout "+
				"for the current one", version,
		))
		return nil, multiErr
	}

	catalog, err := e.layouts.GetCatalog(ctx, request)
	if err != nil {
		return nil, err
	}

	layout := effective.Layout.Clone()
	edit(ctx, layout, catalog, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	document := &homelayout.Document{
		SchemaVersion: homelayout.DocumentSchemaVersion,
		Mode:          homelayout.ModeCustom,
		Layout:        layout,
		Density:       effective.Density,
	}
	document.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &homePlan{
		current: effective,
		update: &homelayoutservice.UpdateRequest{
			Request:  *request,
			Document: document,
			Version:  effective.Version,
		},
	}, nil
}

func (e homeEditor) execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	edit homeEdit,
) error {
	plan, err := e.plan(ctx, params, edit)
	if err != nil {
		return err
	}

	_, err = e.layouts.Update(ctx, plan.update)

	return err
}

func widgetLabel(widget *homelayout.Widget) string {
	if widget.Title != "" {
		return widget.Title
	}
	if definition, ok := homelayout.WidgetDefinitionFor(widget.Key); ok {
		return definition.Label
	}

	return widget.Key
}

// homeToolBase is what the three edit tools share.
type homeToolBase struct {
	editor homeEditor
}

func (homeToolBase) SearchTerms() []string { return homelayout.SpokenNames() }

func versionProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyDescription: "The version home_get_layout returned, so a newer edit is not overwritten.",
	}
}

type AddHomeWidgetTool struct {
	homeToolBase
}

func NewAddHomeWidgetTool(
	layouts *homelayoutservice.Service,
	dashboards *reportdashboardservice.Service,
) agenttoolports.AgentTool {
	return &AddHomeWidgetTool{
		homeToolBase{editor: homeEditor{layouts: layouts, targets: dashboards}},
	}
}

func (t *AddHomeWidgetTool) Name() string { return "home_add_widget" }

func (t *AddHomeWidgetTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *AddHomeWidgetTool) Description() string {
	return "Add a widget, a tile on the person's own home page. Use it when they want " +
		"something on \"my dashboard\" or home page; pick the key and metric from " +
		"home_list_widgets and pass the version from home_get_layout. A report widget " +
		"takes a definitionId or cannedKey from report_list, a dashboard widget a " +
		"dashboardId from report_list_dashboards. Not for report dashboards under Reports; " +
		"report_create_dashboard builds those."
}

func (t *AddHomeWidgetTool) Prerequisites() []string {
	return []string{
		toolHomeGetLayout,
		"home_list_widgets",
		toolReportList,
		"report_list_dashboards",
	}
}

func (t *AddHomeWidgetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "key"},
		"properties": map[string]any{
			"version": versionProperty(),
			"key": map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The widget, by the key home_list_widgets gives.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Heading to show instead of the widget's own name.",
			},
			"width": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"maximum":     homelayout.GridColumns,
				"description": "Columns out of 12. Defaults to the widget's own width.",
			},
			"height": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"maximum":     homelayout.MaxWidgetHeight,
				"description": "Rows. Defaults to the widget's own height.",
			},
			"position": map[string]any{
				"type":        "integer",
				"minimum":     0,
				"description": "Where to put it, counting from 0. Defaults to the end.",
			},
			"config": map[string]any{
				"type": "object",
				"description": "What the widget needs, by its needs in home_list_widgets: metric, " +
					"metrics, windowDays, limit, text, a report or a dashboard.",
				"properties": map[string]any{
					"metric": map[string]any{
						"type":        "string",
						"description": "A metric key from home_list_widgets, for a kpi widget.",
					},
					"metrics": map[string]any{
						"type":        "array",
						schemaItems:   map[string]any{"type": "string"},
						"description": "Metric keys from home_list_widgets, for a kpi row.",
					},
					"definitionId": agenttoolschema.RecordIDText(permission.ResourceReport,
						"A saved report's id, from report_list."),
					"cannedKey": map[string]any{
						"type":        "string",
						"description": "A built-in report's key, from report_list.",
					},
					"chartId": map[string]any{
						"type":        "string",
						"description": "Which of the report's charts; report_get lists them.",
					},
					"columnId": map[string]any{
						"type":        "string",
						"description": "Which of the report's columns; report_get lists them.",
					},
					"dashboardId": agenttoolschema.RecordIDText(permission.ResourceDashboard,
						"A report dashboard's id, from report_list_dashboards."),
					"text": map[string]any{
						"type":        "string",
						"description": "What an announcement widget says.",
					},
					"limit": toolschema.Clamped(map[string]any{
						"type":        "integer",
						"minimum":     1,
						"maximum":     homelayout.MaxWidgetLimit,
						"description": "How many rows a queue widget shows.",
					}),
					"windowDays": map[string]any{
						"type":        "integer",
						"description": "How many days a trend widget covers.",
					},
				},
				"additionalProperties": false,
			},
		},
		"additionalProperties": false,
	}
}

func (t *AddHomeWidgetTool) edit(params *agenttoolports.ToolExecuteParams) homeEdit {
	return func(
		ctx context.Context,
		layout *homelayout.Layout,
		catalog *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		key := toolparams.OptionalString(params.Params, "key")
		definition, offered := offeredWidget(catalog, key)
		if !offered {
			multiErr.Add("key", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a widget this person can add; home_list_widgets names them", key,
			))
			return
		}
		if len(layout.Widgets) >= homelayout.MaxWidgets {
			multiErr.Add("key", errortypes.ErrInvalid, fmt.Sprintf(
				"The home page already holds the most it can, %d widgets; remove one first",
				homelayout.MaxWidgets,
			))
			return
		}

		config := widgetConfigFrom(params.Params, multiErr)
		checkWidgetMetrics(catalog, definition.ConfigKind, config, multiErr)
		t.editor.checkWidgetTargets(ctx, params, definition.ConfigKind, config, multiErr)
		if multiErr.HasErrors() {
			return
		}

		widget := homelayout.Widget{
			ID:     homelayout.NextWidgetID(layout.Widgets),
			Key:    key,
			Title:  strings.TrimSpace(toolparams.OptionalString(params.Params, "title")),
			W:      definition.DefaultW,
			H:      definition.DefaultH,
			Config: config,
		}
		if width := int(toolparams.OptionalInt64(params.Params, "width")); width > 0 {
			widget.W = width
		}
		if height := int(toolparams.OptionalInt64(params.Params, "height")); height > 0 {
			widget.H = height
		}

		position := len(layout.Widgets)
		if _, given := params.Params["position"]; given {
			position = min(
				max(int(toolparams.OptionalInt64(params.Params, "position")), 0),
				len(layout.Widgets),
			)
		}
		layout.Widgets = slices.Insert(layout.Widgets, position, widget)
	}
}

func (t *AddHomeWidgetTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.editor.plan(ctx, &params, t.edit(&params))
	return err
}

func (t *AddHomeWidgetTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, &params, t.edit(&params))
}

func offeredWidget(
	catalog *homelayoutservice.WidgetCatalog,
	key string,
) (homelayout.WidgetDefinition, bool) {
	for idx := range catalog.Widgets {
		if catalog.Widgets[idx].Key == key {
			return catalog.Widgets[idx], true
		}
	}

	return homelayout.WidgetDefinition{}, false
}

// widgetConfigFrom reads the config the model sent. An id that is not an id
// is refused here, by name, rather than stored as a widget that renders an
// error forever.
func widgetConfigFrom(
	params map[string]any,
	multiErr *errortypes.MultiError,
) homelayout.WidgetConfig {
	raw, _ := params["config"].(map[string]any)
	config := homelayout.WidgetConfig{
		Metric:     toolparams.OptionalString(raw, "metric"),
		Metrics:    toolparams.StringSliceParam(raw, "metrics"),
		CannedKey:  toolparams.OptionalString(raw, "cannedKey"),
		ChartID:    toolparams.OptionalString(raw, "chartId"),
		ColumnID:   toolparams.OptionalString(raw, "columnId"),
		Text:       toolparams.OptionalString(raw, "text"),
		Limit:      int(toolparams.OptionalInt64(raw, "limit")),
		WindowDays: int(toolparams.OptionalInt64(raw, "windowDays")),
	}
	if len(config.Metrics) == 0 {
		config.Metrics = nil
	}
	for field, target := range map[string]*pulid.ID{
		"definitionId": &config.DefinitionID,
		"dashboardId":  &config.DashboardID,
	} {
		value := toolparams.OptionalString(raw, field)
		if value == "" {
			continue
		}
		parsed, err := pulid.Parse(value)
		if err != nil {
			multiErr.Add("config."+field, errortypes.ErrInvalid,
				fmt.Sprintf("%q is not an id; %s", value, idSourceFor(field)))
			continue
		}
		*target = parsed
	}

	return config
}

func idSourceFor(field string) string {
	if field == "dashboardId" {
		return "report_list_dashboards gives a dashboard's id"
	}

	return "report_list gives a report's id"
}

// checkWidgetMetrics refuses a metric the person cannot see, which the
// document's own validation would accept because the metric exists.
func checkWidgetMetrics(
	catalog *homelayoutservice.WidgetCatalog,
	kind homelayout.ConfigKind,
	config homelayout.WidgetConfig,
	multiErr *errortypes.MultiError,
) {
	offered := func(key string) bool {
		for idx := range catalog.Metrics {
			if catalog.Metrics[idx].Key == key {
				return true
			}
		}
		return false
	}

	switch kind {
	case homelayout.ConfigKindMetric:
		if config.Metric != "" && !offered(config.Metric) {
			multiErr.Add("config.metric", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a metric this person can show; home_list_widgets names them",
				config.Metric,
			))
		}
	case homelayout.ConfigKindMetricRow:
		for idx, metric := range config.Metrics {
			if !offered(metric) {
				multiErr.Add(fmt.Sprintf("config.metrics[%d]", idx), errortypes.ErrInvalid,
					fmt.Sprintf("%q is not a metric this person can show; home_list_widgets "+
						"names them", metric))
			}
		}
	}
}

// checkWidgetTargets refuses a report or dashboard the person cannot open.
func (e homeEditor) checkWidgetTargets(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	kind homelayout.ConfigKind,
	config homelayout.WidgetConfig,
	multiErr *errortypes.MultiError,
) {
	tenant := toolparams.TenantFrom(*params)
	switch kind {
	case homelayout.ConfigKindReport:
		if config.DefinitionID.IsNil() && config.CannedKey == "" {
			return
		}
		tile := report.DashboardTile{
			Kind:         report.TileKindTable,
			DefinitionID: config.DefinitionID,
			CannedKey:    config.CannedKey,
		}
		for _, missing := range e.targets.UnavailableTileTargets(ctx, tenant, []report.DashboardTile{tile}) {
			if missing.Canned {
				multiErr.Add("config.cannedKey", errortypes.ErrInvalid, fmt.Sprintf(
					"%q is not a built-in report; report_list names the ones that exist",
					config.CannedKey,
				))
				continue
			}
			multiErr.Add("config.definitionId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no report %s that this person can open; find it with report_list",
				config.DefinitionID,
			))
		}
	case homelayout.ConfigKindDashboard:
		if config.DashboardID.IsNil() {
			return
		}
		if _, err := e.targets.GetDashboard(ctx, &reportdashboardservice.GetDashboardRequest{
			Request:     reporting.Request{TenantInfo: tenant},
			DashboardID: config.DashboardID,
		}); err != nil {
			multiErr.Add("config.dashboardId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no dashboard %s that this person can open; find it with report_list_dashboards",
				config.DashboardID,
			))
		}
	}
}

type RemoveHomeWidgetTool struct {
	homeToolBase
}

func NewRemoveHomeWidgetTool(layouts *homelayoutservice.Service) agenttoolports.AgentTool {
	return &RemoveHomeWidgetTool{homeToolBase{editor: homeEditor{layouts: layouts}}}
}

func (t *RemoveHomeWidgetTool) Name() string { return "home_remove_widget" }

func (t *RemoveHomeWidgetTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *RemoveHomeWidgetTool) Description() string {
	return "Take one widget off the person's own home page. Use it when they want something " +
		"off \"my dashboard\"; the widget id and version come from home_get_layout."
}

func (t *RemoveHomeWidgetTool) Prerequisites() []string {
	return []string{toolHomeGetLayout}
}

func (t *RemoveHomeWidgetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "widgetId"},
		"properties": map[string]any{
			"version": versionProperty(),
			"widgetId": map[string]any{
				"type":        "string",
				"description": "The widget's id, from home_get_layout.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *RemoveHomeWidgetTool) edit(params *agenttoolports.ToolExecuteParams) homeEdit {
	return func(
		_ context.Context,
		layout *homelayout.Layout,
		_ *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		id := toolparams.OptionalString(params.Params, "widgetId")
		index := slices.IndexFunc(layout.Widgets, func(widget homelayout.Widget) bool {
			return widget.ID == id
		})
		if index < 0 {
			multiErr.Add("widgetId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no widget %q on this home page; home_get_layout lists them", id,
			))
			return
		}
		layout.Widgets = slices.Delete(layout.Widgets, index, index+1)
	}
}

func (t *RemoveHomeWidgetTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.editor.plan(ctx, &params, t.edit(&params))
	return err
}

func (t *RemoveHomeWidgetTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, &params, t.edit(&params))
}

type ArrangeHomeLayoutTool struct {
	homeToolBase
}

func NewArrangeHomeLayoutTool(layouts *homelayoutservice.Service) agenttoolports.AgentTool {
	return &ArrangeHomeLayoutTool{homeToolBase{editor: homeEditor{layouts: layouts}}}
}

func (t *ArrangeHomeLayoutTool) Name() string { return "home_arrange_layout" }

func (t *ArrangeHomeLayoutTool) SearchTerms() []string {
	return append(homelayout.SpokenNames(), "reorder", "rearrange")
}

func (t *ArrangeHomeLayoutTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *ArrangeHomeLayoutTool) Description() string {
	return "Put the widgets on the person's own home page in a new order. Use it when they " +
		"want something moved up or down \"my dashboard\"; list every widget id from " +
		"home_get_layout, in the order wanted, with its version."
}

func (t *ArrangeHomeLayoutTool) Prerequisites() []string {
	return []string{toolHomeGetLayout}
}

func (t *ArrangeHomeLayoutTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "widgetIds"},
		"properties": map[string]any{
			"version": versionProperty(),
			"widgetIds": map[string]any{
				"type":        "array",
				schemaItems:   map[string]any{"type": "string"},
				"description": "Every widget id from home_get_layout, first to last.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *ArrangeHomeLayoutTool) edit(params *agenttoolports.ToolExecuteParams) homeEdit {
	return func(
		_ context.Context,
		layout *homelayout.Layout,
		_ *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		order := toolparams.StringSliceParam(params.Params, "widgetIds")
		byID := make(map[string]homelayout.Widget, len(layout.Widgets))
		for _, widget := range layout.Widgets {
			byID[widget.ID] = widget
		}

		arranged := make([]homelayout.Widget, 0, len(order))
		seen := make(map[string]struct{}, len(order))
		for idx, id := range order {
			widget, ok := byID[id]
			if !ok {
				multiErr.Add(fmt.Sprintf("widgetIds[%d]", idx), errortypes.ErrInvalid, fmt.Sprintf(
					"There is no widget %q on this home page; home_get_layout lists them", id,
				))
				continue
			}
			if _, duplicate := seen[id]; duplicate {
				multiErr.Add(fmt.Sprintf("widgetIds[%d]", idx), errortypes.ErrDuplicate,
					fmt.Sprintf("%q is listed twice", id))
				continue
			}
			seen[id] = struct{}{}
			arranged = append(arranged, widget)
		}
		for _, widget := range layout.Widgets {
			if _, listed := seen[widget.ID]; !listed {
				multiErr.Add("widgetIds", errortypes.ErrInvalid, fmt.Sprintf(
					"%q is missing; list every widget, or use home_remove_widget to take one off",
					widget.ID,
				))
			}
		}
		layout.Widgets = arranged
	}
}

func (t *ArrangeHomeLayoutTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, err := t.editor.plan(ctx, &params, t.edit(&params))
	return err
}

func (t *ArrangeHomeLayoutTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, &params, t.edit(&params))
}
