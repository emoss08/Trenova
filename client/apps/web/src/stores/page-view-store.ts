import type {
  AssistantPageView,
  AssistantPageViewFilter,
  AssistantPageViewKpi,
} from "@/types/assistant";
import { Resource } from "@trenova/shared/types/permission";
import { create } from "zustand";

/**
 * What a table page has registered about itself: the state the list tools
 * take, so a question about "these rows" can be re-run as the same query.
 */
export type RegisteredTableView = {
  resource: string;
  query: string;
  fieldFilters: AssistantPageViewFilter[];
  filterGroups: { filters: AssistantPageViewFilter[] }[];
  sort: { field: string; direction: string }[];
  selectedIds: string[];
  selectionCount: number;
  visibleColumns: string[];
  rowCount: number | null;
};

/** One figure a strip has registered, as the person reads it. */
export type RegisteredKpi = {
  id: string;
  label: string;
  value: string;
  sub?: string;
};

/** The server's bounds on a view (domain/agent/pagecontext.go). */
export const PAGE_VIEW_LIMITS = {
  fieldFilters: 20,
  filterGroups: 5,
  sort: 5,
  selectionIds: 25,
  kpis: 12,
  columns: 40,
  query: 200,
} as const;

const KNOWN_RESOURCES = new Set<string>(Object.values(Resource));

/**
 * The resource a table names, as the permission registry spells it. A table
 * that names itself by a display name is matched on the lower-cased form;
 * one the registry does not know sends no view at all, because the server
 * would refuse the whole message rather than the view.
 */
function knownResource(resource: string): string | null {
  const normalized = resource.trim().toLowerCase();

  return KNOWN_RESOURCES.has(normalized) ? normalized : null;
}

/**
 * Composes what the page registered into the view the server accepts, or
 * null when there is no table to speak of. Figures ride with a table only:
 * a page of figures with no rows behind them is not a query to re-run.
 */
export function composePageView(
  table: RegisteredTableView | null,
  kpis: readonly RegisteredKpi[],
): AssistantPageView | null {
  if (table === null) {
    return null;
  }
  const resource = knownResource(table.resource);
  if (resource === null) {
    return null;
  }

  const view: AssistantPageView = {
    resource,
    query: table.query.trim().slice(0, PAGE_VIEW_LIMITS.query),
    fieldFilters: table.fieldFilters.slice(0, PAGE_VIEW_LIMITS.fieldFilters),
    filterGroups: table.filterGroups.slice(0, PAGE_VIEW_LIMITS.filterGroups),
    sort: table.sort.slice(0, PAGE_VIEW_LIMITS.sort),
    visibleColumns: table.visibleColumns.slice(0, PAGE_VIEW_LIMITS.columns),
  };
  if (table.selectionCount > 0) {
    view.selection = {
      count: table.selectionCount,
      ids: table.selectedIds.slice(0, PAGE_VIEW_LIMITS.selectionIds),
    };
  }
  const figures = kpis
    .filter((kpi) => kpi.label.trim() !== "" && kpi.value.trim() !== "")
    .slice(0, PAGE_VIEW_LIMITS.kpis)
    .map<AssistantPageViewKpi>((kpi) => ({
      label: kpi.label.trim(),
      value: kpi.value.trim(),
      sub: kpi.sub?.trim() ?? "",
    }));
  if (figures.length > 0) {
    view.kpis = figures;
  }
  if (table.rowCount !== null && table.rowCount >= 0) {
    view.rowCount = table.rowCount;
  }

  return view;
}

interface PageViewState {
  /** The one table on the page, or null between pages. */
  table: RegisteredTableView | null;
  /** The figures on the page, by the strip item that registered each. */
  kpis: Record<string, RegisteredKpi>;

  setTable: (table: RegisteredTableView | null) => void;
  setKpi: (kpi: RegisteredKpi) => void;
  removeKpi: (id: string) => void;
}

/**
 * What the page is showing, for the assistant to see. A table registers its
 * state as it changes and clears it on unmount; a figure registers itself
 * while it is on screen. Nothing here persists: it is the page as it is now.
 */
export const usePageViewStore = create<PageViewState>()((set) => ({
  table: null,
  kpis: {},

  setTable: (table) => set({ table }),
  setKpi: (kpi) => set((state) => ({ kpis: { ...state.kpis, [kpi.id]: kpi } })),
  removeKpi: (id) =>
    set((state) => ({
      kpis: Object.fromEntries(Object.entries(state.kpis).filter(([key]) => key !== id)),
    })),
}));

/** Reads the composed view once, at send time. */
export function readPageView(): AssistantPageView | null {
  const { table, kpis } = usePageViewStore.getState();

  return composePageView(table, Object.values(kpis));
}
