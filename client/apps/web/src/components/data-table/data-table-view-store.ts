import type {
  ActiveTableView,
  TableConfiguration,
  TableDensity,
  TableFormatRule,
  TableLayout,
} from "@/types/table-configuration";
import type { BulkEditSelectionInput } from "@trenova/graphql/generated/graphql";
import type { RowPinningState, Updater } from "@tanstack/react-table";
import { subscribeWithSelector } from "zustand/middleware";
import { createStore, type Mutate, type StoreApi } from "zustand/vanilla";

const NO_ROW_PINNING: RowPinningState = { top: [], bottom: [] };
const NO_SEEN_ROWS: ReadonlySet<string> = new Set();

/**
 * How one person has a table arranged and what they are doing with it, beside what the address
 * holds (search, filters, sort, page, the open panel) and what the table itself holds (columns
 * shown, ordered, sized and pinned). A link carries the address, never this: it is the reader's
 * own, restored from their saved layout or a saved view.
 */
export interface DataTableViewState {
  /** The saved layout and default view have been applied, so rows may be drawn. */
  ready: boolean;
  density: TableDensity;
  formatRules: TableFormatRule[];
  rowPinning: RowPinningState;
  pinnedRowsCollapsed: boolean;
  hideTotals: boolean;
  virtualized: boolean;
  markChangesSinceLastVisit: boolean;
  /** Rows changed after this moment (seconds) are marked; zero marks nothing. */
  changesSince: number;
  /** Changed rows opened since, which are no longer marked. */
  seenRowIds: ReadonlySet<string>;
  /** The saved view the table was opened in, which the views menu compares it against. */
  activeView: ActiveTableView | null;
  /** Every matching row is being fetched to be selected. */
  selectingAll: boolean;
  /** The rows the bulk editor is open on. */
  bulkEditSelection: BulkEditSelectionInput | null;
  /** The bulk edit that is running, whose progress is shown. */
  bulkEditJobId: string | null;
}

/** What changes the view. One object for the store's life, so reading it never re-renders. */
export interface DataTableViewActions {
  markReady(): void;
  setDensity(density: TableDensity): void;
  setFormatRules(rules: TableFormatRule[]): void;
  setRowPinning(update: Updater<RowPinningState>): void;
  /** Pins a row to the top, or unpins it; a row past `max` pinned is not pinned. */
  togglePinnedRow(rowId: string, max: number): void;
  setPinnedRowsCollapsed(collapsed: boolean): void;
  setHideTotals(hide: boolean): void;
  setVirtualized(virtualized: boolean): void;
  setMarkChangesSinceLastVisit(mark: boolean): void;
  markSeen(rowId: string): void;
  /** Every change so far counts as seen: only what changes from now on is marked. */
  markAllSeen(): void;
  setActiveView(view: ActiveTableView | null): void;
  /** A saved view just created or updated is the one the table is in. */
  keepView(view: TableConfiguration): void;
  /** Leaves the view the table was opened in, if it is `id` (deleted, or found gone). */
  forgetActiveView(id: string): void;
  /** The person's saved arrangement, laid over the table's defaults. */
  applySavedLayout(saved: TableLayout): void;
  /** Back to the table's defaults, leaving any saved view. */
  resetLayout(): void;
  setSelectingAll(selecting: boolean): void;
  openBulkEdit(selection: BulkEditSelectionInput): void;
  closeBulkEdit(): void;
  trackBulkEditJob(jobId: string): void;
  dismissBulkEditJob(): void;
}

export type DataTableViewStoreState = DataTableViewState & { actions: DataTableViewActions };

/** One table's view store, subscribable by slice (`subscribe(selector, listener)`). */
export type DataTableViewStore = Mutate<
  StoreApi<DataTableViewStoreState>,
  [["zustand/subscribeWithSelector", never]]
>;

export const selectViewActions = (state: DataTableViewStoreState) => state.actions;

function defaults(initialDensity: TableDensity) {
  return {
    density: initialDensity,
    formatRules: [] as TableFormatRule[],
    rowPinning: NO_ROW_PINNING,
    pinnedRowsCollapsed: false,
    hideTotals: false,
    virtualized: false,
    markChangesSinceLastVisit: false,
    activeView: null,
  } satisfies Partial<DataTableViewState>;
}

/** One table's view state; each table on screen makes its own. */
export function createDataTableViewStore(initialDensity: TableDensity): DataTableViewStore {
  return createStore<DataTableViewStoreState>()(
    subscribeWithSelector((set) => ({
      ...defaults(initialDensity),
      ready: false,
      changesSince: 0,
      seenRowIds: NO_SEEN_ROWS,
      selectingAll: false,
      bulkEditSelection: null,
      bulkEditJobId: null,

      actions: {
        markReady: () => set({ ready: true }),
        setDensity: (density) => set({ density }),
        setFormatRules: (formatRules) => set({ formatRules }),
        setRowPinning: (update) =>
          set((state) => ({
            rowPinning: typeof update === "function" ? update(state.rowPinning) : update,
          })),
        togglePinnedRow: (rowId, max) =>
          set(({ rowPinning }) => {
            if (rowPinning.top.includes(rowId)) {
              return {
                rowPinning: { ...rowPinning, top: rowPinning.top.filter((id) => id !== rowId) },
              };
            }
            if (rowPinning.top.length >= max) return {};
            return { rowPinning: { ...rowPinning, top: [...rowPinning.top, rowId] } };
          }),
        setPinnedRowsCollapsed: (pinnedRowsCollapsed) => set({ pinnedRowsCollapsed }),
        setHideTotals: (hideTotals) => set({ hideTotals }),
        setVirtualized: (virtualized) => set({ virtualized }),
        setMarkChangesSinceLastVisit: (markChangesSinceLastVisit) =>
          set({ markChangesSinceLastVisit }),
        markSeen: (rowId) =>
          set(({ seenRowIds }) => {
            if (seenRowIds.has(rowId)) return {};
            const next = new Set(seenRowIds);
            next.add(rowId);
            return { seenRowIds: next };
          }),
        markAllSeen: () =>
          set({ changesSince: Math.floor(Date.now() / 1000), seenRowIds: NO_SEEN_ROWS }),
        setActiveView: (activeView) => set({ activeView }),
        keepView: (view) =>
          set({ activeView: { id: view.id, name: view.name, config: view.tableConfig } }),
        forgetActiveView: (id) =>
          set(({ activeView }) => (activeView?.id === id ? { activeView: null } : {})),
        applySavedLayout: (saved) =>
          set((state) => ({
            density: saved.density ?? state.density,
            formatRules: saved.formatRules,
            rowPinning: { top: saved.pinnedRowIds, bottom: [] },
            pinnedRowsCollapsed: saved.pinnedRowsCollapsed,
            changesSince: saved.lastSeenAt,
            markChangesSinceLastVisit: saved.markChangesSinceLastVisit,
            hideTotals: saved.hideTotals,
            virtualized: saved.virtualized,
          })),
        resetLayout: () => set(defaults(initialDensity)),
        setSelectingAll: (selectingAll) => set({ selectingAll }),
        openBulkEdit: (bulkEditSelection) => set({ bulkEditSelection }),
        closeBulkEdit: () => set({ bulkEditSelection: null }),
        trackBulkEditJob: (bulkEditJobId) => set({ bulkEditJobId }),
        dismissBulkEditJob: () => set({ bulkEditJobId: null }),
      },
    })),
  );
}
