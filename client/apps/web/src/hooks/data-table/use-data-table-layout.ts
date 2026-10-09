import { resetMyTableLayout, saveMyTableLayout } from "@/lib/graphql/table-layout";
import { fromColumnPinningState } from "@/lib/data-table";
import { queries } from "@/lib/queries";
import { stableStringify } from "@/lib/stable-stringify";
import {
  tableLayoutSchema,
  type TableDensity,
  type TableFormatRule,
  type TableLayout,
} from "@/types/table-configuration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { MyTableLayoutQuery } from "@trenova/graphql/generated/graphql";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Table } from "@trenova/shared/types/data-table";
import type { RowData } from "@tanstack/react-table";
import { useCallback, useEffect, useRef } from "react";
import { toast } from "sonner";

/** How long a table sits still before its arrangement is kept. */
const SAVE_DELAY_MS = 1000;
const MIN_COLUMN_WIDTH = 24;
const MAX_COLUMN_WIDTH = 2000;

type UseDataTableLayoutOptions<TData extends RowData> = {
  /** The table's name; one layout is kept per person per name. Empty turns the feature off. */
  resource: string;
  table: Table<TData>;
  density: TableDensity;
  formatRules: TableFormatRule[];
  activeViewId: string | null;
  pinnedRowsCollapsed: boolean;
  hideChangesSinceLastVisit: boolean;
  hideTotals: boolean;
  virtualized: boolean;
};

export type DataTableLayoutState = {
  /** False until the person's saved layout has been asked for and answered (or failed). */
  settled: boolean;
  hasSavedLayout: boolean;
  /** The layout as last saved, read once to restore the table; null when there is none. */
  readSaved: () => TableLayout | null;
  /**
   * Starts keeping changes. Called once the table has been put back the way it was
   * left, so the restore itself is not saved straight back.
   */
  arm: () => void;
  /** Forgets the saved layout; `restoreDefaults` puts the table back before saving resumes. */
  reset: (restoreDefaults: () => void) => Promise<void>;
};

function clampWidths(sizing: Record<string, number>): Record<string, number> {
  const clamped: Record<string, number> = {};
  for (const [id, width] of Object.entries(sizing)) {
    clamped[id] = Math.min(MAX_COLUMN_WIDTH, Math.max(MIN_COLUMN_WIDTH, Math.round(width)));
  }
  return clamped;
}

/**
 * Keeps how one person arranges a table (columns shown, ordered, sized and pinned,
 * density, colour rules and the view it came from) on the server, so the table
 * opens the way they left it on any device.
 *
 * Nothing here renders the table: changes are heard from the table's store and the
 * shell's own state, gathered for a second of quiet, compared with what was last
 * kept and sent only when they differ. Saves and resets go out one after another,
 * so a reset never loses a race to a save sent before it.
 */
export function useDataTableLayout<TData extends RowData>({
  resource,
  table,
  density,
  formatRules,
  activeViewId,
  pinnedRowsCollapsed,
  hideChangesSinceLastVisit,
  hideTotals,
  virtualized,
}: UseDataTableLayoutOptions<TData>): DataTableLayoutState {
  const t = useT();
  const queryClient = useQueryClient();
  const enabled = resource !== "";
  const queryOptions = queries.tableLayout.mine(resource);

  const { data: hasSavedLayout = false, isPending } = useQuery({
    ...queryOptions,
    enabled,
    staleTime: Infinity,
    retry: 1,
    select: (data) => data.myTableLayout != null,
  });

  const armedRef = useRef(false);
  const timerRef = useRef<number | null>(null);
  const lastKeptRef = useRef<string | null>(null);
  const queueRef = useRef<Promise<unknown>>(Promise.resolve());
  const lastErrorRef = useRef<string | null>(null);
  // When the person last had this table open: kept as it was through a visit, so
  // what changed since then stays marked, and moved to now only when they leave.
  const lastSeenAtRef = useRef(0);

  const snapshot = useLatestCallback((): TableLayout => {
    const { atoms } = table;
    const sizing = atoms.columnSizing.get();
    return {
      columnVisibility: atoms.columnVisibility.get(),
      columnOrder: atoms.columnOrder.get(),
      columnSizing: clampWidths(sizing),
      columnPinning: fromColumnPinningState(atoms.columnPinning.get()),
      density,
      formatRules,
      activeViewId,
      pinnedRowIds: atoms.rowPinning.get().top ?? [],
      pinnedRowsCollapsed,
      lastSeenAt: lastSeenAtRef.current,
      hideChangesSinceLastVisit,
      hideTotals,
      virtualized,
    };
  });

  const enqueue = useCallback(<T>(task: () => Promise<T>): Promise<T> => {
    const next = queueRef.current.then(task, task);
    queueRef.current = next.catch(() => undefined);
    return next;
  }, []);

  const reportFailure = useLatestCallback((error: unknown) => {
    const message = error instanceof Error ? error.message : String(error);
    if (lastErrorRef.current === message) return;
    lastErrorRef.current = message;
    toast.error(t("Your table layout was not saved"), { description: message });
  });

  const flush = useLatestCallback((leaving = false) => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    if (!enabled || !armedRef.current) return;

    if (leaving) lastSeenAtRef.current = Math.floor(Date.now() / 1000);
    const layout = snapshot();
    const key = stableStringify(layout);
    if (key === lastKeptRef.current) return;
    lastKeptRef.current = key;

    void enqueue(() => saveMyTableLayout(resource, layout))
      .then((saved) => {
        lastErrorRef.current = null;
        queryClient.setQueryData(queryOptions.queryKey, { myTableLayout: saved });
      })
      .catch((error: unknown) => {
        if (lastKeptRef.current === key) lastKeptRef.current = null;
        reportFailure(error);
      });
  });

  const schedule = useLatestCallback(() => {
    if (!enabled || !armedRef.current) return;
    if (table.atoms.columnResizing.get().isResizingColumn) return;
    if (timerRef.current !== null) window.clearTimeout(timerRef.current);
    timerRef.current = window.setTimeout(() => flush(), SAVE_DELAY_MS);
  });

  const store = table.store;
  useEffect(() => {
    const subscription = store.subscribe(schedule);
    return () => subscription.unsubscribe();
  }, [store, schedule]);

  useEffect(() => {
    schedule();
  }, [
    density,
    formatRules,
    activeViewId,
    pinnedRowsCollapsed,
    hideChangesSinceLastVisit,
    hideTotals,
    virtualized,
    schedule,
  ]);

  useEffect(() => {
    const onPageHide = () => flush(true);
    window.addEventListener("pagehide", onPageHide);
    return () => {
      window.removeEventListener("pagehide", onPageHide);
      flush(true);
    };
  }, [flush]);

  const readSaved = useLatestCallback((): TableLayout | null => {
    const raw = queryClient.getQueryData<MyTableLayoutQuery>(queryOptions.queryKey)?.myTableLayout
      ?.layout;
    if (!raw) return null;
    const parsed = tableLayoutSchema.safeParse(raw);
    if (!parsed.success) return null;
    lastSeenAtRef.current = parsed.data.lastSeenAt;
    return parsed.data;
  });

  const arm = useLatestCallback(() => {
    window.setTimeout(() => {
      lastKeptRef.current = stableStringify(snapshot());
      armedRef.current = true;
    }, 0);
  });

  const reset = useLatestCallback(async (restoreDefaults: () => void) => {
    armedRef.current = false;
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    try {
      await enqueue(() => resetMyTableLayout(resource));
      queryClient.setQueryData(queryOptions.queryKey, { myTableLayout: null });
      restoreDefaults();
    } catch (error) {
      reportFailure(error);
    } finally {
      arm();
    }
  });

  return {
    settled: !enabled || !isPending,
    hasSavedLayout,
    readSaved,
    arm,
    reset,
  };
}
