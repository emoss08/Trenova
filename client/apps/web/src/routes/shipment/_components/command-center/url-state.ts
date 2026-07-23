import {
  parseAsArrayOf,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import {
  CHIP_FILTERS,
  DEFAULT_VIEW_ID,
  SAVED_VIEWS,
  type ChipFilterId,
  type SavedViewId,
} from "./saved-views";

const SAVED_VIEW_IDS = SAVED_VIEWS.map((v) => v.id) as readonly SavedViewId[];
const CHIP_IDS = CHIP_FILTERS.map((c) => c.id) as readonly ChipFilterId[];
const VIEW_MODES = ["table", "timeline"] as const;
export type CommandCenterViewMode = (typeof VIEW_MODES)[number];
const TIMELINE_ZOOMS = ["day", "3day", "week"] as const;
export type TimelineZoom = (typeof TIMELINE_ZOOMS)[number];
const TIMELINE_SORTS = ["name", "exceptions", "loads"] as const;
export type TimelineSort = (typeof TIMELINE_SORTS)[number];

/**
 * URL state for the Shipments Command Center. Backing every cross-cutting bit
 * of UI with the URL means a dispatcher can share / bookmark a specific
 * workspace (e.g. "at-risk loads delivering today, expanded SHP-1042").
 *
 * Page is 1-indexed in the URL because that's the human convention; we
 * convert to 0-indexed internally for the table.
 */
export const commandCenterParser = {
  view: parseAsStringLiteral(SAVED_VIEW_IDS).withDefault(DEFAULT_VIEW_ID),
  chips: parseAsArrayOf(parseAsStringLiteral(CHIP_IDS)).withDefault([]),
  mode: parseAsStringLiteral(VIEW_MODES).withDefault("table"),
  expanded: parseAsString,
  page: parseAsInteger.withDefault(1),
  size: parseAsInteger.withDefault(10),
  q: parseAsString.withDefault(""),
  at: parseAsString,
  zoom: parseAsStringLiteral(TIMELINE_ZOOMS).withDefault("day"),
  tsort: parseAsStringLiteral(TIMELINE_SORTS).withDefault("name"),
};

export const PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const;
export type CommandCenterPageSize = (typeof PAGE_SIZE_OPTIONS)[number];

export function useCommandCenterUrl() {
  return useQueryStates(commandCenterParser, {
    history: "replace",
  });
}
