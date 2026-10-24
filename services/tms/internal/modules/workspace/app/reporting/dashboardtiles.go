package reporting

import (
	"context"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ResolveDashboardTilesRequest struct {
	Request

	DashboardID  pulid.ID
	FilterValues map[string]any
	ParamValues  map[string]any
}

type ResolvedTile struct {
	TileID     string
	Title      string
	Kind       report.TileKind
	ChartID    string
	ColumnID   string
	Definition *ir.Definition
	Params     map[string]any
}

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

	filters := report.FilterValues(layout.Filters, req.FilterValues)
	resolved := make([]ResolvedTile, 0, len(layout.Tiles))
	for i := range layout.Tiles {
		tile := &layout.Tiles[i]
		if !tile.Kind.NeedsReport() {
			continue
		}

		definition, title, tileErr := s.tileDefinition(ctx, &req.Request, tile)
		if tileErr != nil {
			return nil, tileErr
		}

		applied, params := ir.ApplyFilters(definition, filters, tile.Params(req.ParamValues))
		if tile.Limit > 0 {
			narrowed := *applied
			narrowed.Limit = ir.N(tile.Limit)
			applied = &narrowed
		}

		resolved = append(resolved, ResolvedTile{
			TileID:     tile.ID,
			Title:      tileTitle(tile, title),
			Kind:       tile.Kind,
			ChartID:    tile.ChartID,
			ColumnID:   tile.ColumnID,
			Definition: applied,
			Params:     params,
		})
	}

	return resolved, nil
}

func (s *Service) tileDefinition(
	ctx context.Context,
	base *Request,
	tile *report.DashboardTile,
) (*ir.Definition, string, error) {
	if tile.CannedKey != "" {
		entry, ok := s.canned.Get(tile.CannedKey)
		if !ok {
			return nil, "", errortypes.NewNotFoundError(
				"Unknown canned report \"{0}\"", tile.CannedKey,
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

func tileTitle(tile *report.DashboardTile, reportName string) string {
	if tile.Title != "" {
		return tile.Title
	}
	return reportName
}
