package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/homelayoutservice"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
Changing the person's own home page by describing it.

Asked to build a dashboard, the Homepage Widget Builder could only build a
report dashboard, which the person had not asked for. These three edit the home
page itself, the way its own editor does: they start from the layout the page
shows, keep what is there, and save it as the person's own.

Each is proposed and approved like any change, and each is pinned twice: to the
version of the home page it was read at, so an edit made since is not
overwritten, and to the person it was proposed for, so approving it from
someone else's queue cannot change the approver's home page instead.
*/

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
	) []reporting.UnavailableTile
	GetDashboard(
		ctx context.Context,
		req *reporting.GetDashboardRequest,
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

func homeRequestFor(params serviceports.ToolExecuteParams) *homelayoutservice.Request {
	return &homelayoutservice.Request{
		TenantInfo: tenantFrom(params),
		Principal: serviceports.PrincipalInfo{
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
	params serviceports.ToolExecuteParams,
	edit homeEdit,
) (*homePlan, error) {
	if params.Actor == nil || params.Actor.PrincipalType != serviceports.PrincipalTypeUser {
		return nil, errortypes.NewAuthorizationError("Only a person has a home page to change")
	}
	owner := optionalString(params.Params, serviceports.SelfScopeOwnerParam)
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
	if version := optionalInt64(params.Params, "version"); version != effective.Version {
		multiErr.Add("version", errortypes.ErrVersionMismatch, fmt.Sprintf(
			"The home page has changed since version %d was read; call get_my_home_layout "+
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
	params serviceports.ToolExecuteParams,
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
		"type":        "integer",
		"description": "The version get_my_home_layout returned, so a newer edit is not overwritten.",
	}
}

type addHomeWidgetTool struct {
	homeToolBase
}

func newAddHomeWidgetTool(
	layouts *homelayoutservice.Service,
	reports *reporting.Service,
) serviceports.AgentTool {
	return &addHomeWidgetTool{homeToolBase{editor: homeEditor{layouts: layouts, targets: reports}}}
}

func (t *addHomeWidgetTool) Name() string { return "add_home_widget" }

func (t *addHomeWidgetTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *addHomeWidgetTool) Description() string {
	return "Add a widget to the person's own home page. Use it when they want something on " +
		"\"my dashboard\" or home page; pick the key and metric from list_home_widgets and " +
		"pass the version from get_my_home_layout. A report widget takes a definitionId or " +
		"cannedKey from list_reports, a dashboard widget a dashboardId from list_dashboards. " +
		"Not for report dashboards under Reports; create_dashboard builds those."
}

func (t *addHomeWidgetTool) Prerequisites() []string {
	return []string{"get_my_home_layout", "list_home_widgets", "list_reports", "list_dashboards"}
}

func (t *addHomeWidgetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "key"},
		"properties": map[string]any{
			"version": versionProperty(),
			"key": map[string]any{
				"type":        "string",
				"description": "The widget, by the key list_home_widgets gives.",
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
				"description": "What the widget needs, by its needs in list_home_widgets: metric, " +
					"metrics, windowDays, limit, text, a report or a dashboard.",
				"properties": map[string]any{
					"metric": map[string]any{
						"type":        "string",
						"description": "A metric key from list_home_widgets, for a kpi widget.",
					},
					"metrics": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Metric keys from list_home_widgets, for a kpi row.",
					},
					"definitionId": map[string]any{
						"type":        "string",
						"description": "A saved report's id, from list_reports.",
					},
					"cannedKey": map[string]any{
						"type":        "string",
						"description": "A built-in report's key, from list_reports.",
					},
					"chartId": map[string]any{
						"type":        "string",
						"description": "Which of the report's charts; describe_report lists them.",
					},
					"columnId": map[string]any{
						"type":        "string",
						"description": "Which of the report's columns; describe_report lists them.",
					},
					"dashboardId": map[string]any{
						"type":        "string",
						"description": "A report dashboard's id, from list_dashboards.",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "What an announcement widget says.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "How many rows a queue widget shows.",
					},
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

func (t *addHomeWidgetTool) edit(params serviceports.ToolExecuteParams) homeEdit {
	return func(
		ctx context.Context,
		layout *homelayout.Layout,
		catalog *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		key := optionalString(params.Params, "key")
		definition, offered := offeredWidget(catalog, key)
		if !offered {
			multiErr.Add("key", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a widget this person can add; list_home_widgets names them", key,
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
			Title:  strings.TrimSpace(optionalString(params.Params, "title")),
			W:      definition.DefaultW,
			H:      definition.DefaultH,
			Config: config,
		}
		if width := int(optionalInt64(params.Params, "width")); width > 0 {
			widget.W = width
		}
		if height := int(optionalInt64(params.Params, "height")); height > 0 {
			widget.H = height
		}

		position := len(layout.Widgets)
		if _, given := params.Params["position"]; given {
			position = min(
				max(int(optionalInt64(params.Params, "position")), 0),
				len(layout.Widgets),
			)
		}
		layout.Widgets = slices.Insert(layout.Widgets, position, widget)
	}
}

func (t *addHomeWidgetTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.editor.plan(ctx, params, t.edit(params))
	return err
}

func (t *addHomeWidgetTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, params, t.edit(params))
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
		Metric:     optionalString(raw, "metric"),
		Metrics:    stringSliceParam(raw, "metrics"),
		CannedKey:  optionalString(raw, "cannedKey"),
		ChartID:    optionalString(raw, "chartId"),
		ColumnID:   optionalString(raw, "columnId"),
		Text:       optionalString(raw, "text"),
		Limit:      int(optionalInt64(raw, "limit")),
		WindowDays: int(optionalInt64(raw, "windowDays")),
	}
	if len(config.Metrics) == 0 {
		config.Metrics = nil
	}
	for field, target := range map[string]*pulid.ID{
		"definitionId": &config.DefinitionID,
		"dashboardId":  &config.DashboardID,
	} {
		value := optionalString(raw, field)
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
		return "list_dashboards gives a dashboard's id"
	}

	return "list_reports gives a report's id"
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
				"%q is not a metric this person can show; list_home_widgets names them",
				config.Metric,
			))
		}
	case homelayout.ConfigKindMetricRow:
		for idx, metric := range config.Metrics {
			if !offered(metric) {
				multiErr.Add(fmt.Sprintf("config.metrics[%d]", idx), errortypes.ErrInvalid,
					fmt.Sprintf("%q is not a metric this person can show; list_home_widgets "+
						"names them", metric))
			}
		}
	}
}

// checkWidgetTargets refuses a report or dashboard the person cannot open.
func (e homeEditor) checkWidgetTargets(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
	kind homelayout.ConfigKind,
	config homelayout.WidgetConfig,
	multiErr *errortypes.MultiError,
) {
	tenant := tenantFrom(params)
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
					"%q is not a built-in report; list_reports names the ones that exist",
					config.CannedKey,
				))
				continue
			}
			multiErr.Add("config.definitionId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no report %s that this person can open; find it with list_reports",
				config.DefinitionID,
			))
		}
	case homelayout.ConfigKindDashboard:
		if config.DashboardID.IsNil() {
			return
		}
		if _, err := e.targets.GetDashboard(ctx, &reporting.GetDashboardRequest{
			Request:     reporting.Request{TenantInfo: tenant},
			DashboardID: config.DashboardID,
		}); err != nil {
			multiErr.Add("config.dashboardId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no dashboard %s that this person can open; find it with list_dashboards",
				config.DashboardID,
			))
		}
	}
}

type removeHomeWidgetTool struct {
	homeToolBase
}

func newRemoveHomeWidgetTool(layouts *homelayoutservice.Service) serviceports.AgentTool {
	return &removeHomeWidgetTool{homeToolBase{editor: homeEditor{layouts: layouts}}}
}

func (t *removeHomeWidgetTool) Name() string { return "remove_home_widget" }

func (t *removeHomeWidgetTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *removeHomeWidgetTool) Description() string {
	return "Take one widget off the person's own home page. Use it when they want something " +
		"off \"my dashboard\"; the widget id and version come from get_my_home_layout."
}

func (t *removeHomeWidgetTool) Prerequisites() []string {
	return []string{"get_my_home_layout"}
}

func (t *removeHomeWidgetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "widgetId"},
		"properties": map[string]any{
			"version": versionProperty(),
			"widgetId": map[string]any{
				"type":        "string",
				"description": "The widget's id, from get_my_home_layout.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *removeHomeWidgetTool) edit(params serviceports.ToolExecuteParams) homeEdit {
	return func(
		_ context.Context,
		layout *homelayout.Layout,
		_ *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		id := optionalString(params.Params, "widgetId")
		index := slices.IndexFunc(layout.Widgets, func(widget homelayout.Widget) bool {
			return widget.ID == id
		})
		if index < 0 {
			multiErr.Add("widgetId", errortypes.ErrInvalid, fmt.Sprintf(
				"There is no widget %q on this home page; get_my_home_layout lists them", id,
			))
			return
		}
		layout.Widgets = slices.Delete(layout.Widgets, index, index+1)
	}
}

func (t *removeHomeWidgetTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.editor.plan(ctx, params, t.edit(params))
	return err
}

func (t *removeHomeWidgetTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, params, t.edit(params))
}

type arrangeHomeLayoutTool struct {
	homeToolBase
}

func newArrangeHomeLayoutTool(layouts *homelayoutservice.Service) serviceports.AgentTool {
	return &arrangeHomeLayoutTool{homeToolBase{editor: homeEditor{layouts: layouts}}}
}

func (t *arrangeHomeLayoutTool) Name() string { return "arrange_home_layout" }

func (t *arrangeHomeLayoutTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceHomeLayoutPreset,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeSelf,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressPersonal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Changes only the caller's own home page.",
	}
}

func (t *arrangeHomeLayoutTool) Description() string {
	return "Put the widgets on the person's own home page in a new order. Use it when they " +
		"want something moved up or down \"my dashboard\"; list every widget id from " +
		"get_my_home_layout, in the order wanted, with its version."
}

func (t *arrangeHomeLayoutTool) Prerequisites() []string {
	return []string{"get_my_home_layout"}
}

func (t *arrangeHomeLayoutTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"version", "widgetIds"},
		"properties": map[string]any{
			"version": versionProperty(),
			"widgetIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Every widget id from get_my_home_layout, first to last.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *arrangeHomeLayoutTool) edit(params serviceports.ToolExecuteParams) homeEdit {
	return func(
		_ context.Context,
		layout *homelayout.Layout,
		_ *homelayoutservice.WidgetCatalog,
		multiErr *errortypes.MultiError,
	) {
		order := stringSliceParam(params.Params, "widgetIds")
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
					"There is no widget %q on this home page; get_my_home_layout lists them", id,
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
					"%q is missing; list every widget, or use remove_home_widget to take one off",
					widget.ID,
				))
			}
		}
		layout.Widgets = arranged
	}
}

func (t *arrangeHomeLayoutTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.editor.plan(ctx, params, t.edit(params))
	return err
}

func (t *arrangeHomeLayoutTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	return t.editor.execute(ctx, params, t.edit(params))
}
