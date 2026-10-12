import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

type RowWithId = { id?: string };

type UseDataTableLiveRefreshParams<TData extends RowWithId> = {
  /** Asks the server again on this interval, on top of the changes pushed to it. */
  intervalMs: number | undefined;
  enabled: boolean;
  /** The page's own query, refetched by the interval. */
  queryKey: QueryKey;
  /** Names what the person is looking at: filters, sort, search and page. */
  scopeKey: string;
  /** The page as the server last answered it. */
  results: TData[] | undefined;
  /** True while the previous page stands in for one still loading. */
  isPlaceholderData: boolean;
};

/** Rows whose values changed while on screen, and a counter that replays their glow. */
export type DataTableRowChanges = {
  ids: ReadonlySet<string>;
  version: number;
};

type LiveState<TData> = {
  scopeKey: string;
  /** Whether this scope has had its own answer yet, rather than a stand-in. */
  settled: boolean;
  source: TData[] | undefined;
  shown: TData[] | undefined;
  pending: TData[] | null;
  changes: DataTableRowChanges;
};

const NO_CHANGES: DataTableRowChanges = { ids: new Set(), version: 0 };

function rowId(row: RowWithId): string | undefined {
  return row.id;
}

function sameRowsInSameOrder<TData extends RowWithId>(a: TData[], b: TData[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (rowId(a[i]) !== rowId(b[i])) return false;
  }
  return true;
}

/**
 * The rows that are new or whose values changed. The query keeps an unchanged row
 * as the same object across refetches, so this is an identity check per row, not a
 * comparison of their contents.
 */
function changedRowIds<TData extends RowWithId>(
  before: TData[] | undefined,
  after: TData[],
): Set<string> {
  const previous = new Map<string, TData>();
  for (const row of before ?? []) {
    const id = rowId(row);
    if (id) previous.set(id, row);
  }

  const changed = new Set<string>();
  for (const row of after) {
    const id = rowId(row);
    if (id && previous.get(id) !== row) changed.add(id);
  }
  return changed;
}

function withChanges(
  changes: DataTableRowChanges,
  ids: Set<string>,
): DataTableRowChanges {
  return ids.size === 0 ? NO_CHANGES : { ids, version: changes.version + 1 };
}

function reconcile<TData extends RowWithId>(
  state: LiveState<TData>,
  scopeKey: string,
  results: TData[] | undefined,
  isPlaceholderData: boolean,
): LiveState<TData> {
  const settled = results !== undefined && !isPlaceholderData;
  if (scopeKey !== state.scopeKey || !state.settled || !settled) {
    return {
      scopeKey,
      settled,
      source: results,
      shown: results,
      pending: null,
      changes: NO_CHANGES,
    };
  }

  const shown = state.shown;
  if (!shown || shown.length === 0) {
    return { ...state, source: results, shown: results, pending: null };
  }

  if (sameRowsInSameOrder(shown, results)) {
    return {
      ...state,
      source: results,
      shown: results,
      pending: null,
      changes: withChanges(state.changes, changedRowIds(shown, results)),
    };
  }

  return { ...state, source: results, pending: results };
}

/**
 * Keeps a page current without moving it under the person reading it.
 *
 * A newer answer for the same page (pushed by another person's change, or asked for
 * on the interval) is shown at once when it holds the same rows in the same order,
 * and the rows whose values changed glow once. An answer that adds, removes or
 * reorders rows waits behind the "new updates" pill instead, so a row never jumps
 * away from the cursor or the mouse. Anything the person asks for themselves (a
 * filter, a sort, another page) is a new scope and is shown straight away.
 */
export function useDataTableLiveRefresh<TData extends RowWithId>({
  intervalMs,
  enabled,
  queryKey,
  scopeKey,
  results,
  isPlaceholderData,
}: UseDataTableLiveRefreshParams<TData>) {
  const queryClient = useQueryClient();
  const [state, setState] = useState<LiveState<TData>>(() => ({
    scopeKey,
    settled: results !== undefined && !isPlaceholderData,
    source: results,
    shown: results,
    pending: null,
    changes: NO_CHANGES,
  }));

  let current = state;
  if (state.scopeKey !== scopeKey || state.source !== results) {
    current = reconcile(state, scopeKey, results, isPlaceholderData);
    setState(current);
  }

  const queryKeyRef = useRef(queryKey);
  useEffect(() => {
    queryKeyRef.current = queryKey;
  });

  useEffect(() => {
    if (!enabled || !intervalMs || intervalMs <= 0) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState !== "visible") return;
      void queryClient.invalidateQueries({ queryKey: queryKeyRef.current, exact: true });
    }, intervalMs);
    return () => window.clearInterval(timer);
  }, [enabled, intervalMs, queryClient]);

  const applyStaged = useCallback(() => {
    setState((latest) => {
      if (!latest.pending) return latest;
      return {
        ...latest,
        shown: latest.pending,
        pending: null,
        changes: withChanges(latest.changes, changedRowIds(latest.shown, latest.pending)),
      };
    });
  }, []);

  const dismissStaged = useCallback(() => {
    setState((latest) => (latest.pending ? { ...latest, pending: null } : latest));
  }, []);

  return {
    results: current.shown,
    hasPendingUpdate: current.pending !== null,
    changes: current.changes,
    applyStaged,
    dismissStaged,
  };
}
