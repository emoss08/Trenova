import { BOARD_GROUPINGS, type BoardGrouping } from "@/lib/shipment-board/grouping";
import { parseAsQuickFilters } from "@/lib/shipment-board/quick-filters";
import {
  createParser,
  parseAsArrayOf,
  parseAsBoolean,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";

export const BOARD_VIEWS = ["table", "timeline", "map"] as const;
export type BoardView = (typeof BOARD_VIEWS)[number];

export const PANEL_TABS = ["brief", "activity"] as const;
export type PanelTab = (typeof PANEL_TABS)[number];

export const CAPACITY_KINDS = ["Driver", "Carrier"] as const;

/* Links shared before the board had more than one grouping say `group=true`
   or `group=false`; they still open grouped by stage, or ungrouped. */
const LEGACY_GROUPING: Record<string, BoardGrouping> = { true: "stage", false: "none" };

export const parseAsBoardGrouping = createParser<BoardGrouping>({
  parse: (value) =>
    LEGACY_GROUPING[value] ??
    ((BOARD_GROUPINGS as readonly string[]).includes(value) ? (value as BoardGrouping) : null),
  serialize: (value) => value,
});

/**
 * Everything about the board a dispatcher would want to share or come back
 * to lives in the URL: the view, grouping, the quick filters, the collapsed
 * groups and the open row. `expanded` keeps its name because record links
 * (config/record-links.ts) open a shipment through it.
 */
export const shipmentBoardParser = {
  view: parseAsStringLiteral(BOARD_VIEWS).withDefault("table"),
  group: parseAsBoardGrouping.withDefault("stage"),
  panel: parseAsBoolean,
  tab: parseAsStringLiteral(PANEL_TABS).withDefault("brief"),
  qf: parseAsQuickFilters,
  collapsed: parseAsArrayOf(parseAsString).withDefault([]),
  expanded: parseAsString,
  capacity: parseAsStringLiteral(CAPACITY_KINDS),
};

export function useShipmentBoardUrl() {
  return useQueryStates(shipmentBoardParser, { history: "replace" });
}
