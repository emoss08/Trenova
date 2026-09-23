package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
Building a dashboard by describing it.

Nobody builds a dashboard because the grid is intimidating and every tile is
four decisions — which report, which chart, how big, where. So the reports get
written and then read one at a time, and the thing they were written for never
gets assembled.

These two do the assembly. The layout is not a decision the caller makes: tiles
are placed left to right and wrapped, on the same twelve-column grid the page
uses, so a described dashboard comes out looking like a built one. What a tile
points at is checked against the reports that exist — a tile referencing a
report nobody has is a tile that renders an error forever.
*/

type dashboardWriter interface {
	CreateDashboard(
		ctx context.Context,
		req *reporting.SaveDashboardRequest,
	) (*report.Dashboard, error)
	UpdateDashboard(
		ctx context.Context,
		req *reporting.SaveDashboardRequest,
	) (*report.Dashboard, error)
	GetDashboard(
		ctx context.Context,
		req *reporting.GetDashboardRequest,
	) (*report.Dashboard, error)
	UnavailableTileTargets(
		ctx context.Context,
		tenant pagination.TenantInfo,
		tiles []report.DashboardTile,
	) []reporting.UnavailableTile
}

type createDashboardTool struct {
	dashboards dashboardWriter
}

func newCreateDashboardTool(dashboards *reporting.Service) serviceports.AgentTool {
	return &createDashboardTool{dashboards: dashboards}
}

func (t *createDashboardTool) Name() string { return "create_dashboard" }

func (t *createDashboardTool) Description() string {
	return "Create a report dashboard: a page of saved or built-in reports under Reports. " +
		"Use it when a person describes a screen of reports they want, such as \"this " +
		"week's revenue, on-time percentage and the unbilled backlog\". Find each tile's " +
		"report with list_reports first; never invent an id. Tiles are laid out for you. " +
		"For a dashboard about a measure, lead with its headline figures as kpi tiles when " +
		"the report has a column for them, then the detail table. " +
		"Not for the person's home page — get_my_home_layout and add_home_widget are for that."
}

func (t *createDashboardTool) Prerequisites() []string {
	return []string{"list_reports", "describe_report"}
}

func (t *createDashboardTool) SearchTerms() []string {
	return []string{"report dashboard", "reports page", "board"}
}

func (t *createDashboardTool) ParamSchema() map[string]any {
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
				"type":        "array",
				"maxItems":    report.MaxDashboardTiles,
				"description": "What the page shows, in reading order.",
				"items":       tileSchema(),
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
			"kind": map[string]any{
				"type":        "string",
				"description": "table, chart or kpi draws a report; text shows words.",
				"enum": []string{
					string(report.TileKindTable),
					string(report.TileKindChart),
					string(report.TileKindKPI),
					string(report.TileKindText),
				},
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Heading shown on the tile. Defaults to the report's name.",
			},
			"definitionId": map[string]any{
				"type":        "string",
				"description": "A saved report's id, from list_reports. Give this or cannedKey, not both.",
			},
			"cannedKey": map[string]any{
				"type":        "string",
				"description": "A built-in report's key, from list_reports. Give this or definitionId, not both.",
			},
			"chartId": map[string]any{
				"type":        "string",
				"description": "Which of the report's charts, for a chart tile; describe_report lists them.",
			},
			"columnId": map[string]any{
				"type":        "string",
				"description": "The output column a KPI tile reads; describe_report lists them.",
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

func (t *createDashboardTool) Reversible() bool { return true }

func (t *createDashboardTool) PermissionResource() permission.Resource {
	return permission.ResourceDashboard
}

func (t *createDashboardTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *createDashboardTool) RequiresIdempotencyKey() bool { return false }

func (t *createDashboardTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *createDashboardTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	return checkTileTargets(
		ctx,
		t.dashboards,
		tenantFrom(params),
		tileParams(params.Params),
		"tiles",
	)
}

func (t *createDashboardTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()
	if optionalString(params, "name") == "" {
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
		kind := report.TileKind(optionalString(tile, "kind"))
		field := fmt.Sprintf("tiles[%d]", i)

		switch kind {
		case report.TileKindText:
			if optionalString(tile, "text") == "" {
				multiErr.Add(field+".text", errortypes.ErrRequired,
					"A text tile needs something to say")
			}
		case report.TileKindTable, report.TileKindChart, report.TileKindKPI:
			definitionID := optionalString(tile, "definitionId")
			cannedKey := optionalString(tile, "cannedKey")
			switch {
			case definitionID == "" && cannedKey == "":
				multiErr.Add(field+".definitionId", errortypes.ErrRequired,
					"A "+string(kind)+" tile needs a report to draw: a definitionId or "+
						"cannedKey from list_reports")
			case definitionID != "" && cannedKey != "":
				multiErr.Add(field, errortypes.ErrInvalid,
					"Give a tile a definitionId or a cannedKey, not both")
			}
			if kind == report.TileKindKPI && optionalString(tile, "columnId") == "" {
				multiErr.Add(field+".columnId", errortypes.ErrRequired,
					"A KPI tile needs the column whose number it shows")
			}
		default:
			multiErr.Add(field+".kind", errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a tile kind", kind))
		}
	}
}

func (t *createDashboardTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	visibility := report.VisibilityPrivate
	if optionalBool(params.Params, "shared") {
		visibility = report.VisibilityShared
	}

	tiles, err := buildTiles(tileParams(params.Params), 0)
	if err != nil {
		return err
	}

	_, err = t.dashboards.CreateDashboard(ctx, &reporting.SaveDashboardRequest{
		Request:     reporting.Request{TenantInfo: tenantFrom(params)},
		Name:        optionalString(params.Params, "name"),
		Description: optionalString(params.Params, "description"),
		Category:    optionalString(params.Params, "category"),
		Visibility:  visibility,
		Layout:      &report.DashboardLayout{Tiles: tiles},
	})

	return err
}

type addDashboardTileTool struct {
	dashboards dashboardWriter
}

func newAddDashboardTileTool(dashboards *reporting.Service) serviceports.AgentTool {
	return &addDashboardTileTool{dashboards: dashboards}
}

func (t *addDashboardTileTool) Name() string { return "add_dashboard_tile" }

func (t *addDashboardTileTool) Description() string {
	return "Add one tile to a report dashboard that already exists, placed after the ones " +
		"there. Use it when somebody wants one more report on a dashboard they have; " +
		"list_dashboards gives the dashboard id and list_reports the report. Not for the " +
		"person's home page — add_home_widget is for that."
}

func (t *addDashboardTileTool) Prerequisites() []string {
	return []string{"list_dashboards", "list_reports", "describe_report"}
}

func (t *addDashboardTileTool) SearchTerms() []string {
	return []string{"report dashboard", "reports page"}
}

func (t *addDashboardTileTool) ParamSchema() map[string]any {
	tile := tileSchema()
	tile["description"] = "The tile to add."

	return map[string]any{
		"type":     "object",
		"required": []string{"dashboardId", "tile"},
		"properties": map[string]any{
			"dashboardId": map[string]any{
				"type":        "string",
				"description": "The dashboard to add to, from list_dashboards.",
			},
			"tile": tile,
		},
		"additionalProperties": false,
	}
}

func (t *addDashboardTileTool) Reversible() bool { return true }

func (t *addDashboardTileTool) PermissionResource() permission.Resource {
	return permission.ResourceDashboard
}

func (t *addDashboardTileTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *addDashboardTileTool) RequiresIdempotencyKey() bool { return false }

func (t *addDashboardTileTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *addDashboardTileTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	tenant := tenantFrom(params)
	dashboardID, err := pulid.Parse(optionalString(params.Params, "dashboardId"))
	if err == nil {
		_, err = t.dashboards.GetDashboard(ctx, &reporting.GetDashboardRequest{
			Request:     reporting.Request{TenantInfo: tenant},
			DashboardID: dashboardID,
		})
	}
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("dashboardId", errortypes.ErrInvalid, fmt.Sprintf(
			"%q is not a dashboard you can open; find it with list_dashboards",
			optionalString(params.Params, "dashboardId"),
		))
		return multiErr
	}

	tile, _ := params.Params["tile"].(map[string]any)

	return checkTileTargets(ctx, t.dashboards, tenant, []map[string]any{tile}, "tile")
}

func (t *addDashboardTileTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()
	if optionalString(params, "dashboardId") == "" {
		multiErr.Add("dashboardId", errortypes.ErrRequired, "Name the dashboard to add to")
	}

	tile, ok := params["tile"].(map[string]any)
	if !ok {
		multiErr.Add("tile", errortypes.ErrRequired, "Describe the tile to add")
	} else {
		validateTiles([]map[string]any{tile}, multiErr)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *addDashboardTileTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	dashboardID, err := requirePulid(params.Params, "dashboardId")
	if err != nil {
		return err
	}

	tenant := tenantFrom(params)
	existing, err := t.dashboards.GetDashboard(ctx, &reporting.GetDashboardRequest{
		Request:     reporting.Request{TenantInfo: tenant},
		DashboardID: dashboardID,
	})
	if err != nil {
		return err
	}
	if existing.Layout == nil {
		existing.Layout = &report.DashboardLayout{}
	}
	if len(existing.Layout.Tiles) >= report.MaxDashboardTiles {
		return fmt.Errorf(
			"%q already holds the maximum of %d tiles",
			existing.Name, report.MaxDashboardTiles,
		)
	}

	tile, _ := params.Params["tile"].(map[string]any)
	// The new tile is placed below what is already there rather than at the
	// origin, which is what keeps adding one from landing it on top of another.
	added, err := buildTiles([]map[string]any{tile}, nextRow(existing.Layout.Tiles))
	if err != nil {
		return err
	}

	_, err = t.dashboards.UpdateDashboard(ctx, &reporting.SaveDashboardRequest{
		Request:     reporting.Request{TenantInfo: tenant},
		DashboardID: dashboardID,
		Name:        existing.Name,
		Description: existing.Description,
		Category:    existing.Category,
		Tags:        existing.Tags,
		Visibility:  existing.Visibility,
		Version:     existing.Version,
		Layout: &report.DashboardLayout{
			Tiles:      append(existing.Layout.Tiles, added...),
			Parameters: existing.Layout.Parameters,
			Filters:    existing.Layout.Filters,
		},
	})

	return err
}

func (t *addDashboardTileTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "dashboardId", permission.ResourceDashboard)
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
		kind := report.TileKind(optionalString(tile, "kind"))
		width, height := defaultSpan(kind)
		if requested := int(optionalInt64(tile, "width")); requested > 0 {
			width = min(requested, report.MaxTileSpan)
		}
		if requested := int(optionalInt64(tile, "height")); requested > 0 {
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
			Title:    optionalString(tile, "title"),
			ChartID:  optionalString(tile, "chartId"),
			ColumnID: optionalString(tile, "columnId"),
			Text:     optionalString(tile, "text"),
			X:        x,
			Y:        y,
			W:        width,
			H:        height,
		}
		next.CannedKey = optionalString(tile, "cannedKey")
		if raw := optionalString(tile, "definitionId"); raw != "" {
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
				"%q is not a report id. Find the report with list_reports and use its "+
					"definitionId, or a built-in report's cannedKey",
				optionalString(tile, "definitionId"),
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
				"%q is not a built-in report; list_reports names the ones that exist",
				parsed[missing.Index].CannedKey,
			))
			continue
		}
		multiErr.Add(path+".definitionId", errortypes.ErrInvalid, fmt.Sprintf(
			"There is no report %s that you can open; find the one you mean with list_reports",
			parsed[missing.Index].DefinitionID,
		))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func tilePath(field string, index, count int) string {
	if field == "tile" && count == 1 {
		return field
	}

	return fmt.Sprintf("%s[%d]", field, index)
}
