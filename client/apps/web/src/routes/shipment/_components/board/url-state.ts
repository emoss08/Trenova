import { parseAsQuickFilters } from "@/lib/shipment-board/quick-filters";
import {
  parseAsArrayOf,
  parseAsBoolean,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";

export const BOARD_VIEWS = ["table", "timeline", "map"] as const;
export type BoardView = (typeof BOARD_VIEWS)[number];

export const PANEL_TABS = ["brief", "activity"] as const;
export type PanelTab = (typeof PANEL_TABS)[number];

export const CAPACITY_KINDS = ["Driver", "Carrier"] as const;

/**
 * Everything about the board a dispatcher would want to share or come back
 * to lives in the URL: the view, grouping, the quick filters, the collapsed
 * groups and the open row. `expanded` keeps its name because record links
 * (config/record-links.ts) open a shipment through it.
 */
export const shipmentBoardParser = {
  view: parseAsStringLiteral(BOARD_VIEWS).withDefault("table"),
  group: parseAsBoolean.withDefault(true),
  panel: parseAsBoolean,
  tab: parseAsStringLiteral(PANEL_TABS).withDefault("brief"),
  qf: parseAsQuickFilters,
  collapsed: parseAsArrayOf(parseAsInteger).withDefault([]),
  expanded: parseAsString,
  capacity: parseAsStringLiteral(CAPACITY_KINDS),
};

export function useShipmentBoardUrl() {
  return useQueryStates(shipmentBoardParser, { history: "replace" });
}
