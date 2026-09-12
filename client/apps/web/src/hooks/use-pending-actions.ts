import type { RowData } from "@tanstack/react-table";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { useCallback, useMemo, useRef, useState } from "react";

/**
 * Tracks actions whose promise has not settled yet, one per scope (a row, a
 * record). While a scope has an action running, further runs for that scope are
 * ignored, so a double click cannot fire the same request twice.
 *
 * The ref is the guard and the state is only for rendering: two clicks inside a
 * single frame both run before React commits the state update.
 */
export function usePendingActions() {
  const runningRef = useRef(new Map<string, string>());
  const [pending, setPending] = useState<ReadonlyMap<string, string>>(() => new Map());

  const run = useCallback(
    (scope: string, actionId: string, start: () => unknown): Promise<void> | undefined => {
      if (runningRef.current.has(scope)) return undefined;

      const result = start();
      if (!(result instanceof Promise)) return undefined;

      runningRef.current.set(scope, actionId);
      setPending(new Map(runningRef.current));

      const release = () => {
        runningRef.current.delete(scope);
        setPending(new Map(runningRef.current));
      };

      // The action reports its own failure; the guard only needs to know it settled.
      return result.then(release, release);
    },
    [],
  );

  return { pending, run };
}

/**
 * Wraps row actions so an action returning a promise locks its row until the
 * promise settles: every action on that row reads as disabled, and the one that
 * is running reads as pending.
 */
export function useGuardedRowActions<TData extends RowData>(
  actions: RowAction<TData>[],
): RowAction<TData>[] {
  const { pending, run } = usePendingActions();

  return useMemo(
    () =>
      actions.map((action) => ({
        ...action,
        onClick: (row: Row<TData>) => run(row.id, action.id, () => action.onClick(row)),
        disabled: (row: Row<TData>) => pending.has(row.id) || (action.disabled?.(row) ?? false),
        isPending: (row: Row<TData>) => pending.get(row.id) === action.id,
      })),
    [actions, pending, run],
  );
}
