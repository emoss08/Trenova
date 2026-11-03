import { BOARD_GROUPINGS, type BoardGrouping } from "@/lib/shipment-board/grouping";
import { TIMELINE_RANGES, TIMELINE_ROW_MODES } from "@/lib/shipment-board/timeline";
import { applicableQuickFilters, parseAsQuickFilters } from "@/lib/shipment-board/quick-filters";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { useMemo } from "react";
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
 * groups, the open row, and the timeline's range, first day and rows.
 * `expanded` keeps its name because record links (config/record-links.ts) open
 * a shipment through it. `tlday` is an ISO date, so a shared link opens on the
 * same calendar day in the reader's time zone.
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
  tl: parseAsStringLiteral(TIMELINE_RANGES).withDefault("day"),
  tlday: parseAsString,
  tlrows: parseAsStringLiteral(TIMELINE_ROW_MODES).withDefault("shipments"),
};

/**
 * The board's URL state. A shared link can name a quick filter the reader may not apply (low
 * margin, for someone without the margins field group); it is left out rather than sent to a
 * server that would refuse the whole board for it.
 */
export function useShipmentBoardUrl() {
  const [state, setState] = useQueryStates(shipmentBoardParser, { history: "replace" });
  const marginsVisible = useProtectedFieldAccess("Margins");
  const applicable = useMemo(
    () => ({ ...state, qf: applicableQuickFilters(state.qf, marginsVisible) }),
    [state, marginsVisible],
  );

  return [applicable, setState] as const;
}
