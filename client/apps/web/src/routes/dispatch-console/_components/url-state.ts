import {
  getRangeForDays,
  parseAnchorDate,
  serializeAnchorDate,
  shiftAnchorByDays,
} from "@/lib/timeline/time-scale";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import {
  hasOrganizationCapability,
  OrganizationCapability,
  type OrganizationCapabilities,
} from "@trenova/shared/types/organization-capability";
import { startOfDay } from "date-fns";
import {
  debounce,
  parseAsBoolean,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  useQueryStates,
} from "nuqs";
import { useCallback, useMemo } from "react";
import {
  CAPACITY_FILTERS,
  URGENCY_ORDER,
  type CapacityFilter,
  type UrgencyBucket,
} from "./dispatch-vocabulary";
import { TIMELINE_ZOOM_OPTIONS, type TimelineZoomDays } from "./timeline-metrics";

/**
 * Every cross-cutting bit of console state lives in the URL, so a dispatcher can hand a
 * colleague the exact board they are looking at — window, view, filters, and selection —
 * and a refresh mid-shift lands back on the same screen. Transient interaction state
 * (a drag in flight, a pending pre-flight) is deliberately not here; it belongs to the
 * session and lives in the store instead.
 */

const CENTER_MODES = ["board", "timeline"] as const;
export type CenterMode = (typeof CENTER_MODES)[number];

/**
 * The console's timeline is a grid of driver lanes with loads dropped onto them —
 * the same shape as the capacity rail, laid out horizontally. An organization
 * without its own drivers has no lanes, so the view is withheld and the board is
 * the only centre.
 */
export function isDispatchTimelineAvailable(capabilities: OrganizationCapabilities): boolean {
  return hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations);
}

/**
 * Resolves the requested centre view against what the organization can use. The
 * console's whole state is shareable through the URL, so `?mode=timeline` can
 * outlive the capability behind it and must degrade rather than render lanes
 * that cannot exist.
 */
export function resolveCenterMode(
  mode: CenterMode,
  capabilities: OrganizationCapabilities,
): CenterMode {
  return mode === "timeline" && !isDispatchTimelineAvailable(capabilities) ? "board" : mode;
}

const ZOOM_VALUES: readonly TimelineZoomDays[] = TIMELINE_ZOOM_OPTIONS.map((option) => option.id);
const CAPACITY_FILTER_IDS: readonly CapacityFilter[] = CAPACITY_FILTERS.map((filter) => filter.id);

export const dispatchWindowParsers = {
  at: parseAsString,
  zoom: parseAsNumberLiteral(ZOOM_VALUES).withDefault(1),
  covered: parseAsBoolean.withDefault(false),
};

export const dispatchSelectionParsers = {
  move: parseAsString,
  worker: parseAsString,
};

export const dispatchViewParsers = {
  mode: parseAsStringLiteral(CENTER_MODES).withDefault("board"),
  urgency: parseAsStringLiteral(URGENCY_ORDER),
};

export const dispatchRailParsers = {
  capacity: parseAsStringLiteral(CAPACITY_FILTER_IDS).withDefault("all"),
  q: parseAsString.withDefault(""),
};

// Paging a board back and forth should not bury the browser's back button under history.
const URL_OPTIONS = { history: "replace" } as const;

export function useDispatchWindow() {
  const [{ at, zoom, covered }, setWindow] = useQueryStates(dispatchWindowParsers, URL_OPTIONS);

  const anchor = useMemo(() => parseAnchorDate(at), [at]);
  const range = useMemo(() => getRangeForDays(anchor, zoom), [anchor, zoom]);

  const setAnchor = useCallback(
    (date: Date) => void setWindow({ at: serializeAnchorDate(startOfDay(date)) }),
    [setWindow],
  );
  const shiftWindow = useCallback(
    (direction: 1 | -1) => setAnchor(shiftAnchorByDays(anchor, zoom, direction)),
    [anchor, setAnchor, zoom],
  );
  // Today is the default, so clearing the param is both the jump and the tidier URL.
  const goToday = useCallback(() => void setWindow({ at: null }), [setWindow]);
  const setZoom = useCallback(
    (next: TimelineZoomDays) => void setWindow({ zoom: next }),
    [setWindow],
  );
  const setIncludeCovered = useCallback(
    (next: boolean) => void setWindow({ covered: next }),
    [setWindow],
  );

  return {
    anchor,
    range,
    zoom,
    includeCovered: covered,
    setAnchor,
    shiftWindow,
    goToday,
    setZoom,
    setIncludeCovered,
  };
}

export function useDispatchSelection() {
  const [{ move, worker }, setSelection] = useQueryStates(dispatchSelectionParsers, URL_OPTIONS);

  // A move and a driver are never both selected: the inspector answers one question at a
  // time, and which one is being asked is exactly what the selection encodes.
  const selectMove = useCallback(
    (moveId: string) => void setSelection({ move: moveId, worker: null }),
    [setSelection],
  );
  const selectDriver = useCallback(
    (workerId: string) => void setSelection({ move: null, worker: workerId }),
    [setSelection],
  );
  const clearSelection = useCallback(
    () => void setSelection({ move: null, worker: null }),
    [setSelection],
  );

  return {
    selectedMoveId: move,
    selectedWorkerId: worker,
    selectMove,
    selectDriver,
    clearSelection,
  };
}

export function useDispatchView() {
  const [{ mode, urgency }, setView] = useQueryStates(dispatchViewParsers, URL_OPTIONS);
  const capabilities = useOrgCapabilities();

  const setMode = useCallback((next: CenterMode) => void setView({ mode: next }), [setView]);
  const setUrgencyFocus = useCallback(
    (next: UrgencyBucket | null) => void setView({ urgency: next }),
    [setView],
  );

  return {
    mode: resolveCenterMode(mode, capabilities),
    timelineAvailable: isDispatchTimelineAvailable(capabilities),
    urgencyFocus: urgency,
    setMode,
    setUrgencyFocus,
  };
}

export function useDispatchRail() {
  const [{ capacity, q }, setRail] = useQueryStates(dispatchRailParsers, URL_OPTIONS);

  const setCapacityFilter = useCallback(
    (next: CapacityFilter) => void setRail({ capacity: next }),
    [setRail],
  );
  // The search box reads from state on the keystroke; the URL catches up once typing stops.
  const setDriverSearch = useCallback(
    (next: string) => void setRail({ q: next }, { limitUrlUpdates: debounce(250) }),
    [setRail],
  );

  return { capacityFilter: capacity, driverSearch: q, setCapacityFilter, setDriverSearch };
}
