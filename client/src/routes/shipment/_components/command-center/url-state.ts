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
  q: parseAsString.withDefault(""),
};

export function useCommandCenterUrl() {
  return useQueryStates(commandCenterParser, {
    history: "replace",
  });
}
