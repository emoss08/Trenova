package dashboardtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramTile = "tile"

var dashboardTileKinds = agenttoolschema.Source("report.tileKind", []report.TileKind{
	report.TileKindTable,
	report.TileKindChart,
	report.TileKindKPI,
	report.TileKindText,
})

type dashboardWriter interface {
	CreateDashboard(
		ctx context.Context,
		req *reportdashboardservice.SaveDashboardRequest,
	) (*report.Dashboard, error)
	UpdateDashboard(
		ctx context.Context,
		req *reportdashboardservice.SaveDashboardRequest,
	) (*report.Dashboard, error)
	GetDashboard(
		ctx context.Context,
		req *reportdashboardservice.GetDashboardRequest,
	) (*report.Dashboard, error)
	UnavailableTileTargets(
		ctx context.Context,
		tenant pagination.TenantInfo,
		tiles []report.DashboardTile,
	) []reportdashboardservice.UnavailableTile
}

type CreateDashboardTool struct {
	dashboards dashboardWriter
}

func NewCreateDashboardTool(dashboards *reportdashboardservice.Service) agenttoolports.AgentTool {
	return &CreateDashboardTool{dashboards: dashboards}
}

func (t *CreateDashboardTool) Name() string { return "report_create_dashboard" }

func (t *CreateDashboardTool) Description() string {
	return "Create a report dashboard: a page of saved or built-in reports under Reports. " +
		"Use it when a person describes a screen of reports they want, such as \"this " +
		"week's revenue, on-time percentage and the unbilled backlog\". Find each tile's " +
		"report with report_list first; never invent an id. Tiles are laid out for you. " +
		"For a dashboard about a measure, lead with its headline figures as kpi tiles when " +
		"the report has a column for them, then the detail table. " +
		"Not for the person's home page — home_get_layout and home_add_widget are for that."
}

func (t *CreateDashboardTool) Prerequisites() []string {
	return []string{toolReportList, toolReportGet}
}

func (t *CreateDashboardTool) Recipe() []string {
	return []string{toolReportList, toolReportGet, "report_create_dashboard"}
}

func (t *CreateDashboardTool) SearchTerms() []string {
	return []string{"report dashboard", "reports page", "board", "build dashboard", "new dashboard"}
}

func (t *CreateDashboardTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"name", "tiles"},
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "What to call it."},
			"description": map[string]any{
				"type":        "string",
				"description": "One line on what the dashboard is for.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional grouping on the dashboards page, such as Operations.",
			},
			"shared": map[string]any{
				"type": "boolean",
				"description": "Whether everyone in the organization sees it. " +
					"Defaults to false.",
			},
			"tiles": map[string]any{
				"type":                 "array",
				"maxItems":             report.MaxDashboardTiles,
				"description":          "What the page shows, in reading order.",
				toolparams.SchemaItems: tileSchema(),
			},
		},
		"additionalProperties": false,
	}
}

func tileSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"kind"},
		"properties": map[string]any{
			"kind": agenttoolschema.Enum(
				"table, chart or kpi draws a report; text shows words.",
				dashboardTileKinds,
			),
			"title": map[string]any{
				"type":        "string",
				"description": "Heading shown on the tile. Defaults to the report's name.",
			},
			"definitionId": agenttoolschema.RecordIDText(
				permission.ResourceReport,
				"A saved report's id, from report_list. Give this or cannedKey, not both.",
			),
			"cannedKey": map[string]any{
				"type":        "string",
				"description": "A built-in report's key, from report_list. Give this or definitionId, not both.",
			},
			"chartId": map[string]any{
				"type":        "string",
				"description": "Which of the report's charts, for a chart tile; report_get lists them.",
			},
			"columnId": map[string]any{
				"type":        "string",
				"description": "The output column a KPI tile reads; report_get lists them.",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "The words a text tile shows.",
			},
			"width": map[string]any{
				"type":    "integer",
				"minimum": 1,
				"maximum": report.MaxTileSpan,
				"description": fmt.Sprintf(
					"How many of the %d columns the tile spans. Defaults to a sensible "+
						"width for the kind.", report.DashboardGridColumns,
				),
			},
			"height": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"maximum":     12,
				"description": "Rows the tile spans. Defaults to a sensible height for the kind.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *CreateDashboardTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDashboard,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Saves a report dashboard colleagues can open; nothing leaves the " +
			"organization.",
	}
}

func (t *CreateDashboardTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	return checkTileTargets(
		ctx,
		t.dashboards,
		toolparams.TenantFrom(params),
		tileParams(params.Params),
		"tiles",
	)
}

func (t *CreateDashboardTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()
	if toolparams.OptionalString(params, "name") == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Give the dashboard a name")
	}

	tiles := tileParams(params)
	if len(tiles) == 0 {
		multiErr.Add("tiles", errortypes.ErrRequired, "A dashboard needs at least one tile")
	}
	if len(tiles) > report.MaxDashboardTiles {
		multiErr.Add("tiles", errortypes.ErrInvalid,
			fmt.Sprintf("A dashboard holds at most %d tiles", report.MaxDashboardTiles))
	}
	validateTiles(tiles, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validateTiles refuses a tile that cannot draw anything.
//
// A report tile with no report and a text tile with no text both render as an
// empty box that looks like a loading failure, and the person who asked for
// the dashboard has no way to tell which it is.
func validateTiles(tiles []map[string]any, multiErr *errortypes.MultiError) {
	for i, tile := range tiles {
		kind := report.TileKind(toolparams.OptionalString(tile, "kind"))
		field := fmt.Sprintf("tiles[%d]", i)

		switch kind {
		case report.TileKindText:
			if toolparams.OptionalString(tile, "text") == "" {
				multiErr.Add(field+".text", errortypes.ErrRequired,
					"A text tile needs something to say")
			}
		case report.TileKindTable, report.TileKindChart, report.TileKindKPI:
			definitionID := toolparams.OptionalString(tile, "definitionId")
			cannedKey := toolparams.OptionalString(tile, "cannedKey")
			switch {
			case definitionID == "" && cannedKey == "":
				multiErr.Add(field+".definitionId", errortypes.ErrRequired,
					"A "+string(kind)+" tile needs a report to draw: a definitionId or "+
						"cannedKey from report_list")
			case definitionID != "" && cannedKey != "":
				multiErr.Add(field, errortypes.ErrInvalid,
					"Give a tile a definitionId or a cannedKey, not both")
			}
			if kind == report.TileKindKPI && toolparams.OptionalString(tile, "columnId") == "" {
				multiErr.Add(field+".columnId", errortypes.ErrRequired,
					"A KPI tile needs the column whose number it shows")
			}
		default:
			multiErr.Add(field+".kind", errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a tile kind", kind))
		}
	}
}

func (t *CreateDashboardTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.dashboards.CreateDashboard(ctx, request)

	return err
}

func (t *CreateDashboardTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*reportdashboardservice.SaveDashboardRequest, error) {
	visibility := report.VisibilityPrivate
	if toolparams.OptionalBool(params.Params, "shared") {
		visibility = report.VisibilityShared
	}

	tiles, err := buildTiles(tileParams(params.Params), 0)
	if err != nil {
		return nil, err
	}

	return &reportdashboardservice.SaveDashboardRequest{
		Request:     reporting.Request{TenantInfo: toolparams.TenantFrom(*params)},
		Name:        toolparams.OptionalString(params.Params, "name"),
		Description: toolparams.OptionalString(params.Params, "description"),
		Category:    toolparams.OptionalString(params.Params, "category"),
		Visibility:  visibility,
		Layout:      &report.DashboardLayout{Tiles: tiles},
	}, nil
}

type AddDashboardTileTool struct {
	dashboards dashboardWriter
}

func NewAddDashboardTileTool(dashboards *reportdashboardservice.Service) agenttoolports.AgentTool {
	return &AddDashboardTileTool{dashboards: dashboards}
}

func (t *AddDashboardTileTool) Name() string { return "report_add_dashboard_tile" }

func (t *AddDashboardTileTool) Recipe() []string {
	return []string{
		"report_list_dashboards",
		toolReportList,
		toolReportGet,
		"report_add_dashboard_tile",
	}
}

func (t *AddDashboardTileTool) Description() string {
	return "Add one tile to a report dashboard that already exists, placed after the ones " +
		"there. Use it when somebody wants one more report on a dashboard they have; " +
		"report_list_dashboards gives the dashboard id and report_list the report. Not for the " +
		"person's home page — home_add_widget is for that."
}

func (t *AddDashboardTileTool) Prerequisites() []string {
	return []string{"report_list_dashboards", toolReportList, toolReportGet}
}

func (t *AddDashboardTileTool) SearchTerms() []string {
	return []string{"report dashboard", "reports page"}
}

func (t *AddDashboardTileTool) ParamSchema() map[string]any {
	tile := tileSchema()
	tile["description"] = "The tile to add."

	return map[string]any{
		"type":     "object",
		"required": []string{"dashboardId", paramTile},
		"properties": map[string]any{
			"dashboardId": agenttoolschema.RecordIDText(
				permission.ResourceDashboard,
				"The dashboard to add to, from report_list_dashboards.",
			),
			paramTile: tile,
		},
		"additionalProperties": false,
	}
}

func (t *AddDashboardTileTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDashboard,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Adds a tile to a saved dashboard inside Trenova.",
	}
}

func (t *AddDashboardTileTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	tenant := toolparams.TenantFrom(params)
	dashboardID, err := pulid.Parse(toolparams.OptionalString(params.Params, "dashboardId"))
	if err == nil {
		_, err = t.dashboards.GetDashboard(ctx, &reportdashboardservice.GetDashboardRequest{
			Request:     reporting.Request{TenantInfo: tenant},
			DashboardID: dashboardID,
		})
	}
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("dashboardId", errortypes.ErrInvalid, fmt.Sprintf(
			"%q is not a dashboard you can open; find it with report_list_dashboards",
			toolparams.OptionalString(params.Params, "dashboardId"),
		))
		return multiErr
	}

	tile, _ := params.Params[paramTile].(map[string]any)

	return checkTileTargets(ctx, t.dashboards, tenant, []map[string]any{tile}, paramTile)
}

func (t *AddDashboardTileTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()
	if toolparams.OptionalString(params, "dashboardId") == "" {
		multiErr.Add("dashboardId", errortypes.ErrRequired, "Name the dashboard to add to")
	}

	tile, ok := params[paramTile].(map[string]any)
	if !ok {
		multiErr.Add(paramTile, errortypes.ErrRequired, "Describe the tile to add")
	} else {
		validateTiles([]map[string]any{tile}, multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *AddDashboardTileTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	request, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.dashboards.UpdateDashboard(ctx, request)

	return err
}

func (t *AddDashboardTileTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*reportdashboardservice.SaveDashboardRequest, *report.Dashboard, error) {
	dashboardID, err := toolparams.RequirePulid(params.Params, "dashboardId")
	if err != nil {
		return nil, nil, err
	}

	tenant := toolparams.TenantFrom(*params)
	existing, err := t.dashboards.GetDashboard(ctx, &reportdashboardservice.GetDashboardRequest{
		Request:     reporting.Request{TenantInfo: tenant},
		DashboardID: dashboardID,
	})
	if err != nil {
		return nil, nil, err
	}
	tiles := make([]report.DashboardTile, 0, report.MaxDashboardTiles)
	layout := &report.DashboardLayout{}
	if existing.Layout != nil {
		tiles = append(tiles, existing.Layout.Tiles...)
		layout.Parameters = existing.Layout.Parameters
		layout.Filters = existing.Layout.Filters
	}
	if len(tiles) >= report.MaxDashboardTiles {
		return nil, existing, errortypes.NewBusinessError(
			"{0} already holds the maximum of {1} tiles",
			existing.Name, report.MaxDashboardTiles,
		)
	}

	tile, _ := params.Params[paramTile].(map[string]any)
	// The new tile is placed below what is already there rather than at the
	// origin, which is what keeps adding one from landing it on top of another.
	added, err := buildTiles([]map[string]any{tile}, nextRow(tiles))
	if err != nil {
		return nil, existing, err
	}
	tiles = append(tiles, added...)
	layout.Tiles = tiles

	return &reportdashboardservice.SaveDashboardRequest{
		Request:     reporting.Request{TenantInfo: tenant},
		DashboardID: dashboardID,
		Name:        existing.Name,
		Description: existing.Description,
		Category:    existing.Category,
		Tags:        existing.Tags,
		Visibility:  existing.Visibility,
		Version:     existing.Version,
		Layout:      layout,
	}, existing, nil
}

func (t *AddDashboardTileTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "dashboardId", permission.ResourceDashboard)
}

func tileParams(params map[string]any) []map[string]any {
	raw, _ := params["tiles"].([]any)
	tiles := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		if tile, ok := entry.(map[string]any); ok {
			tiles = append(tiles, tile)
		}
	}

	return tiles
}

// defaultSpan is how wide a tile of each kind wants to be.
//
// A KPI is one number and a table is a grid of them, so giving both the same
// width produces a page that reads as a mistake. These are what the dashboard
// page's own defaults are, so a described dashboard and a built one look alike.
func defaultSpan(kind report.TileKind) (int, int) {
	switch kind {
	case report.TileKindKPI:
		return 3, 2
	case report.TileKindChart:
		return 6, 4
	case report.TileKindText:
		return 12, 1
	default:
		return 12, 4
	}
}

// buildTiles lays tiles out left to right, wrapping at the grid's width.
//
// The caller never places anything. Asking a model for x and y coordinates
// produces overlapping tiles and gaps, and the person who described the
// dashboard has to go and fix a layout they never chose.
func buildTiles(tiles []map[string]any, startRow int) ([]report.DashboardTile, error) {
	built := make([]report.DashboardTile, 0, len(tiles))
	x, y, rowHeight := 0, startRow, 0

	for _, tile := range tiles {
		kind := report.TileKind(toolparams.OptionalString(tile, "kind"))
		width, height := defaultSpan(kind)
		if requested := int(toolparams.OptionalInt64(tile, "width")); requested > 0 {
			width = min(requested, report.MaxTileSpan)
		}
		if requested := int(toolparams.OptionalInt64(tile, "height")); requested > 0 {
			height = requested
		}

		if x+width > report.DashboardGridColumns {
			x = 0
			y += rowHeight
			rowHeight = 0
		}

		next := report.DashboardTile{
			ID:       pulid.MustNew("tile_").String(),
			Kind:     kind,
			Title:    toolparams.OptionalString(tile, "title"),
			ChartID:  toolparams.OptionalString(tile, "chartId"),
			ColumnID: toolparams.OptionalString(tile, "columnId"),
			Text:     toolparams.OptionalString(tile, "text"),
			X:        x,
			Y:        y,
			W:        width,
			H:        height,
		}
		next.CannedKey = toolparams.OptionalString(tile, "cannedKey")
		if raw := toolparams.OptionalString(tile, "definitionId"); raw != "" {
			definitionID, err := pulid.Parse(raw)
			if err != nil {
				return nil, fmt.Errorf("%q is not a report id", raw)
			}
			next.DefinitionID = definitionID
		}

		built = append(built, next)
		x += width
		rowHeight = max(rowHeight, height)
	}

	return built, nil
}

// nextRow is the first free row under what is already on the dashboard.
func nextRow(tiles []report.DashboardTile) int {
	bottom := 0
	for i := range tiles {
		bottom = max(bottom, tiles[i].Y+tiles[i].H)
	}

	return bottom
}

// checkTileTargets refuses a tile whose report does not exist, before anybody
// is asked to approve the dashboard. A model that cannot find a report writes
// a plausible name where its id belongs; the refusal names the id it wrote
// and the tool that has the real one, while it can still fix the call.
func checkTileTargets(
	ctx context.Context,
	dashboards dashboardWriter,
	tenant pagination.TenantInfo,
	tiles []map[string]any,
	field string,
) error {
	multiErr := errortypes.NewMultiError()
	parsed := make([]report.DashboardTile, 0, len(tiles))
	positions := make([]int, 0, len(tiles))
	for i, tile := range tiles {
		path := tilePath(field, i, len(tiles))
		built, err := buildTiles([]map[string]any{tile}, 0)
		if err != nil {
			multiErr.Add(path+".definitionId", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a report id. Find the report with report_list and use its "+
					"definitionId, or a built-in report's cannedKey",
				toolparams.OptionalString(tile, "definitionId"),
			))
			continue
		}
		parsed = append(parsed, built[0])
		positions = append(positions, i)
	}

	for _, missing := range dashboards.UnavailableTileTargets(ctx, tenant, parsed) {
		path := tilePath(field, positions[missing.Index], len(tiles))
		if missing.Canned {
			multiErr.Add(path+".cannedKey", errortypes.ErrInvalid, fmt.Sprintf(
				"%q is not a built-in report; report_list names the ones that exist",
				parsed[missing.Index].CannedKey,
			))
			continue
		}
		multiErr.Add(path+".definitionId", errortypes.ErrInvalid, fmt.Sprintf(
			"There is no report %s that you can open; find the one you mean with report_list",
			parsed[missing.Index].DefinitionID,
		))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func tilePath(field string, index, count int) string {
	if field == paramTile && count == 1 {
		return field
	}

	return fmt.Sprintf("%s[%d]", field, index)
}
