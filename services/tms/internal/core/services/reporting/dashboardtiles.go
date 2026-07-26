package reporting

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// ResolveDashboardTilesRequest asks for the concrete report behind every tile
// of a dashboard, with the viewer's filter and parameter choices already
// folded in. The values are keyed the way the dashboard declares them —
// filters by `DashboardFilter.ID`, parameters by name — not by tile.
type ResolveDashboardTilesRequest struct {
	Request

	DashboardID  pulid.ID
	FilterValues map[string]any
	ParamValues  map[string]any
}

// ResolvedTile is one tile's report, ready to compile. Definition is a copy —
// resolving never mutates the stored report, which every other tile shares.
type ResolvedTile struct {
	TileID     string
	Title      string
	Kind       report.TileKind
	ChartID    string
	ColumnID   string
	Definition *report.Definition
	Params     map[string]any
}

// ResolveDashboardTiles derives what each tile actually queries. This is the
// server-side counterpart of the dashboard's on-screen resolution, and exists
// so a dashboard means the same thing without a browser to work it out —
// which is what an export, and later a schedule, depends on.
func (s *Service) ResolveDashboardTiles(
	ctx context.Context,
	req *ResolveDashboardTilesRequest,
) ([]ResolvedTile, error) {
	dashboard, err := s.GetDashboard(ctx, &GetDashboardRequest{
		Request:     req.Request,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}

	layout := dashboard.Layout
	if layout == nil {
		return nil, nil
	}

	resolved := make([]ResolvedTile, 0, len(layout.Tiles))
	for i := range layout.Tiles {
		tile := &layout.Tiles[i]
		// A text tile is a note on the canvas and queries nothing.
		if !tile.Kind.NeedsReport() {
			continue
		}

		definition, title, tileErr := s.tileDefinition(ctx, &req.Request, tile)
		if tileErr != nil {
			return nil, tileErr
		}

		definition = layout.ApplyFilters(definition, req.FilterValues)
		if tile.Limit > 0 {
			// ApplyFilters already returned a copy when anything applied, but
			// it returns the original untouched when nothing did — so copy
			// before writing the limit rather than assuming ownership.
			narrowed := *definition
			narrowed.Limit = tile.Limit
			definition = &narrowed
		}

		resolved = append(resolved, ResolvedTile{
			TileID:     tile.ID,
			Title:      tileTitle(tile, title),
			Kind:       tile.Kind,
			ChartID:    tile.ChartID,
			ColumnID:   tile.ColumnID,
			Definition: definition,
			Params:     tile.ResolveParams(req.ParamValues, definition.Parameters),
		})
	}

	return resolved, nil
}

// tileDefinition resolves a tile's target to its report. A canned tile reads
// the compiled-in library; a saved tile goes through GetDefinition, which is
// what enforces the viewer's access to that report rather than trusting the
// dashboard to have been authored by someone who could see it.
func (s *Service) tileDefinition(
	ctx context.Context,
	base *Request,
	tile *report.DashboardTile,
) (*report.Definition, string, error) {
	if tile.CannedKey != "" {
		entry, ok := s.canned.Get(tile.CannedKey)
		if !ok {
			return nil, "", errortypes.NewNotFoundError(
				fmt.Sprintf("Unknown canned report %q", tile.CannedKey),
			)
		}
		return entry.Definition, entry.Name, nil
	}

	definition, err := s.GetDefinition(ctx, &GetDefinitionRequest{
		Request:      *base,
		DefinitionID: tile.DefinitionID,
	})
	if err != nil {
		return nil, "", err
	}

	return definition.Definition, definition.Name, nil
}

// tileTitle prefers the author's override; a tile left untitled falls back to
// the report's own name so an export never ships an unlabelled sheet.
func tileTitle(tile *report.DashboardTile, reportName string) string {
	if tile.Title != "" {
		return tile.Title
	}
	return reportName
}
