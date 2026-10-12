import {
  applyCellWrites,
  invertWrites,
  planFillDown,
  planPaste,
  parseClipboardGrid,
  selectedCellGrid,
  type CellWrite,
  type CellWritePlan,
} from "@/lib/data-table-cell-fill";
import {
  buildClipboardGrid,
  clipboardColumns,
  writeClipboardGrid,
} from "@/lib/data-table-clipboard";
import { isTypingTarget } from "@/lib/dom";
import { useQueryClient } from "@tanstack/react-query";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Table } from "@trenova/shared/types/data-table";
import { useEffect, useRef } from "react";
import { toast } from "sonner";

/** The most cells one paste or fill saves; more than this belongs in a bulk edit. */
export const MAX_CELL_WRITES = 500;

type UseDataTableCellWritesParams<TData extends Record<string, any>> = {
  table: Table<TData>;
  queryKey: string;
};

/**
 * Copying, filling and pasting the cells a person has selected. Copy works on any
 * table; fill and paste save through the table's inline edit, a few cells at a time,
 * then refetch once and offer to put every cell back.
 */
export function useDataTableCellWrites<TData extends Record<string, any>>({
  table,
  queryKey,
}: UseDataTableCellWritesParams<TData>) {
  const t = useT();
  const queryClient = useQueryClient();
  const busyRef = useRef(false);

  const currentGrid = () =>
    selectedCellGrid<TData>({
      rows: [...table.getTopRows(), ...table.getCenterRows()],
      selectedIds: new Set(table.getSelectedCellIds()),
    });

  const copySelectedCells = useLatestCallback(async () => {
    const grid = currentGrid();
    if (grid.rows.length === 0) return;
    const copiedRows: TData[] = [];
    for (const cells of grid.rows) {
      const cell = cells.find((entry) => entry !== null);
      if (cell) copiedRows.push(cell.row.original);
    }
    const count = table.getSelectedCellCount();
    try {
      await writeClipboardGrid(
        buildClipboardGrid(copiedRows, grid.columnIds, clipboardColumns(table.getAllLeafColumns())),
      );
      toast.success(t("{0, plural, one {Copied # cell} other {Copied # cells}}", count));
    } catch (error) {
      toast.error(t("The cells were not copied"), {
        description: error instanceof Error ? error.message : undefined,
      });
    }
  });

  const skippedNote = (plan: CellWritePlan<TData>): string | undefined => {
    const notes: string[] = [];
    if (plan.skipped["read-only"] > 0) {
      notes.push(
        t(
          "{0, plural, one {# cell can't be edited} other {# cells can't be edited}}",
          plan.skipped["read-only"],
        ),
      );
    }
    if (plan.skipped.unreadable > 0) {
      notes.push(
        t(
          "{0, plural, one {# value didn't fit its column} other {# values didn't fit their columns}}",
          plan.skipped.unreadable,
        ),
      );
    }
    return notes.length > 0 ? notes.join(" · ") : undefined;
  };

  const write = useLatestCallback(
    async (plan: CellWritePlan<TData>, kind: "fill" | "paste" | "undo") => {
      const commit = table.options.onCellEditCommit;
      if (!commit || !table.options.enableCellEditing) return;
      if (busyRef.current) {
        toast.info(t("Still saving the last change"));
        return;
      }
      if (plan.writes.length === 0) {
        toast.info(t("Nothing to change"), { description: skippedNote(plan) });
        return;
      }
      if (plan.writes.length > MAX_CELL_WRITES) {
        toast.error(t("Too many cells"), {
          description: t(
            "One paste or fill changes at most {0} cells. Use Edit on the selection for more.",
            MAX_CELL_WRITES,
          ),
        });
        return;
      }

      busyRef.current = true;
      const total = plan.writes.length;
      const toastId = toast.loading(t("Saving {0} of {1}", 0, total));
      try {
        const outcome = await applyCellWrites<TData>(
          plan.writes,
          (entry: CellWrite<TData>) => commit({ ...entry, batch: true }),
          (done) => toast.loading(t("Saving {0} of {1}", done, total), { id: toastId }),
        );
        await queryClient.invalidateQueries({ queryKey: [queryKey] });

        const applied = outcome.applied.length;
        const failed = outcome.failed.length;
        const notes = [skippedNote(plan), failed > 0 ? outcome.failed[0].message : undefined]
          .filter(Boolean)
          .join(" · ");
        const undo =
          kind !== "undo" && applied > 0
            ? {
                label: t("Undo"),
                onClick: () =>
                  void write(
                    { writes: invertWrites(outcome.applied), skipped: { "read-only": 0, unreadable: 0 } },
                    "undo",
                  ),
              }
            : undefined;

        if (applied === 0) {
          toast.error(t("No cells were saved"), { id: toastId, description: notes || undefined });
        } else if (failed > 0) {
          toast.warning(
            t("{0} saved, {1} refused", applied.toLocaleString(), failed.toLocaleString()),
            { id: toastId, description: notes || undefined, action: undo },
          );
        } else {
          toast.success(
            kind === "undo"
              ? t("{0, plural, one {# cell put back} other {# cells put back}}", applied)
              : t("{0, plural, one {# cell updated} other {# cells updated}}", applied),
            { id: toastId, description: notes || undefined, action: undo },
          );
        }
      } finally {
        busyRef.current = false;
      }
    },
  );

  const fillDown = useLatestCallback(() => {
    const grid = currentGrid();
    if (grid.rows.length < 2) {
      toast.info(t("Select the cells to fill"), {
        description: t("Select a column of cells; the top one is copied into the rest."),
      });
      return;
    }
    void write(planFillDown(grid), "fill");
  });

  const pasteText = useLatestCallback((text: string) => {
    const grid = currentGrid();
    if (grid.rows.length === 0) return;
    const pasted = parseClipboardGrid(text);
    const rows = [...table.getTopRows(), ...table.getCenterRows()].map((row) =>
      row.getVisibleCells(),
    );
    void write(
      planPaste(pasted, {
        rows,
        anchorRow: grid.anchorRow,
        anchorColumn: grid.anchorColumn,
        selectionRows: grid.rows.length,
        selectionColumns: grid.columnIds.length,
      }),
      "paste",
    );
  });

  const pasteFromClipboard = useLatestCallback(async () => {
    try {
      pasteText(await navigator.clipboard.readText());
    } catch {
      toast.error(t("The clipboard could not be read"), {
        description: t("Press Ctrl+V (⌘V on a Mac) on the selected cells instead."),
      });
    }
  });

  const clearCellSelection = useLatestCallback(() => table.resetCellSelection(true));
  const canWrite = useLatestCallback(
    () => table.options.enableCellEditing === true && table.options.onCellEditCommit !== undefined,
  );
  const cellSelectionAtom = table.atoms.cellSelection;

  useEffect(() => {
    const hasSelection = () => cellSelectionAtom.get().length > 0;

    const onKeyDown = (event: KeyboardEvent) => {
      if (!hasSelection()) return;
      if (event.key === "Escape") {
        clearCellSelection();
        return;
      }
      if (!(event.metaKey || event.ctrlKey) || isTypingTarget(event.target)) return;
      const key = event.key.toLowerCase();
      if (key === "c") {
        if (window.getSelection()?.toString()) return;
        event.preventDefault();
        void copySelectedCells();
      } else if (key === "d" && canWrite()) {
        event.preventDefault();
        fillDown();
      }
    };

    const onPaste = (event: ClipboardEvent) => {
      if (!hasSelection() || !canWrite() || isTypingTarget(event.target)) return;
      const text = event.clipboardData?.getData("text/plain");
      if (!text) return;
      event.preventDefault();
      pasteText(text);
    };

    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("paste", onPaste);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("paste", onPaste);
    };
  }, [cellSelectionAtom, canWrite, clearCellSelection, copySelectedCells, fillDown, pasteText]);

  return { fillDown, pasteFromClipboard };
}
