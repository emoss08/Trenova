import { applyFilters, type FilterValue } from "@trenova/shared/generated/keel-analytics";
import { dashboardFilterToKeel, type KeelDefinition } from "@/lib/report-definition";
import type {
  ReportDashboardFilter,
  ReportDashboardLayout,
  ReportDashboardTile,
  ReportFieldRef,
} from "@/types/report";
import { DASHBOARD_GRID_COLUMNS } from "@/types/report";
import { nextItemId, packTiles as packGridItems } from "@/components/tile-grid/pack-tiles";

export function refKey(ref: ReportFieldRef): string {
  return `${(ref.path ?? []).join(".")}|${ref.field}`;
}

function filterValues(
  filters: ReportDashboardFilter[],
  values: Record<string, unknown>,
): FilterValue[] {
  return filters.map((filter) => ({
    filter: dashboardFilterToKeel(filter),
    value: values[filter.id],
  }));
}

export type TileFilterState = {
  /** The dashboard's field filters, and the value chosen for each. */
  filters: ReportDashboardFilter[];
  filterValues: Record<string, unknown>;
  /** Filters scoped to this tile alone, and the value chosen for each. */
  tileFilters: ReportDashboardFilter[];
  tileFilterValues: Record<string, unknown>;
  /** Values for the dashboard's parameters, by name. */
  params: Record<string, unknown>;
};

/**
 * Folds the dashboard's filters, then the tile's own, and the dashboard's
 * parameters into one tile's report with keel's applyFilters, the function the
 * server's dashboard export runs as ir.ApplyFilters, so a tile on screen and
 * its sheet in the export read the same rows. Each pass replaces the value of
 * a matching condition rather than adding a second one on the same field, so a
 * tile's own filter wins over the dashboard's.
 */
export function applyTileFilters(
  definition: KeelDefinition,
  tile: ReportDashboardTile,
  state: TileFilterState,
): { definition: KeelDefinition; params?: Record<string, unknown> } {
  const shared = applyFilters(definition, filterValues(state.filters, state.filterValues), {
    values: state.params,
    bindings: tile.paramBindings,
  });
  const own = applyFilters(
    shared.definition,
    filterValues(state.tileFilters, state.tileFilterValues),
  );
  return { definition: own.definition, params: shared.params };
}

/**
 * Tiles are authored as an ordered list and flow left-to-right into the grid,
 * so moving a tile is a reorder rather than a coordinate edit. The stored x/y
 * are derived here, which keeps the persisted layout stable for any future
 * free-form editor without asking authors to place tiles by hand.
 */
export function packTiles(tiles: ReportDashboardTile[]): ReportDashboardTile[] {
  return packGridItems(tiles, { columns: DASHBOARD_GRID_COLUMNS });
}

export function nextTileId(tiles: ReportDashboardTile[]): string {
  return nextItemId(tiles, "tile_");
}

export function emptyLayout(): ReportDashboardLayout {
  return { tiles: [], parameters: [] };
}
