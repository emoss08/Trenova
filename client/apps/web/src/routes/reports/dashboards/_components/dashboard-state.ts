import type {
  ReportDashboardFilter,
  ReportDashboardLayout,
  ReportDashboardTile,
  ReportFieldFilter,
  ReportFieldRef,
  ReportFilterGroup,
  ReportIR,
  ReportParameterDef,
} from "@/types/report";
import { DASHBOARD_GRID_COLUMNS } from "@/types/report";
import { nextItemId, packTiles as packGridItems } from "@/components/tile-grid/pack-tiles";

export function refKey(ref: ReportFieldRef): string {
  return `${(ref.path ?? []).join(".")}|${ref.field}`;
}

function hasValue(value: unknown): boolean {
  if (value === undefined || value === null || value === "") return false;
  return !(Array.isArray(value) && value.length === 0);
}

/**
 * Replaces the value of a report's own condition on the same field, rather
 * than adding a second one. A report that already filters "Status = Completed"
 * must not be ANDed with "Status = New" — that yields nothing.
 */
function overrideInGroup(
  group: ReportFilterGroup,
  filter: ReportDashboardFilter,
  value: unknown,
): { group: ReportFilterGroup; matched: boolean } {
  let matched = false;
  const key = refKey(filter.ref);

  const filters = (group.filters ?? []).map((leaf) => {
    // A parameter-bound condition is already driven by something else.
    if (leaf.param || refKey(leaf.ref) !== key) return leaf;
    matched = true;
    return { ...leaf, operator: filter.operator, value };
  });

  const groups = (group.groups ?? []).map((nested) => {
    const result = overrideInGroup(nested, filter, value);
    if (result.matched) matched = true;
    return result.group;
  });

  return { group: { ...group, filters, groups }, matched };
}

/**
 * Folds the dashboard's active filters into one tile's report. A filter
 * applies when the tile's report queries the same entity — anything else has no
 * matching column and is left alone.
 */
export function applyDashboardFilters(
  ir: ReportIR,
  filters: ReportDashboardFilter[],
  values: Record<string, unknown>,
): ReportIR {
  const active = filters.filter(
    (filter) => filter.entity === ir.entity && hasValue(values[filter.id]),
  );
  if (active.length === 0) return ir;

  let group: ReportFilterGroup | null | undefined = ir.filters;
  const added: ReportFieldFilter[] = [];

  for (const filter of active) {
    const value = values[filter.id];
    if (group) {
      const result = overrideInGroup(group, filter, value);
      group = result.group;
      if (result.matched) continue;
    }
    added.push({ ref: filter.ref, operator: filter.operator, value });
  }

  if (added.length === 0) return { ...ir, filters: group };

  // Nesting the report's own filters keeps their and/or grouping intact while
  // the dashboard's new conditions are ANDed on top.
  return {
    ...ir,
    filters: { op: "and", filters: added, groups: group ? [group] : [] },
  };
}

/**
 * Maps the dashboard's shared parameter values onto one tile's report:
 * explicit bindings first, then same-name matches. A tile only receives the
 * parameters its own report declares, so an unrelated filter is ignored rather
 * than rejected by the compiler.
 */
export function resolveTileParams(
  tile: ReportDashboardTile,
  values: Record<string, unknown>,
  reportParams: ReportParameterDef[],
): Record<string, unknown> | undefined {
  if (reportParams.length === 0) return undefined;

  const resolved: Record<string, unknown> = {};
  for (const param of reportParams) {
    const source = tile.paramBindings?.[param.name] ?? param.name;
    const value = values[source];
    if (value !== undefined && value !== null) {
      resolved[param.name] = value;
    }
  }

  return Object.keys(resolved).length > 0 ? resolved : undefined;
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
