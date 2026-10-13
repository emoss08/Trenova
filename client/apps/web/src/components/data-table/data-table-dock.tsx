import type { RowData } from "@tanstack/react-table";
import { useDataTable } from "@/contexts/data-table-context";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import { countSelectedRows } from "@trenova/shared/lib/table-features";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { DockAction, Table } from "@trenova/shared/types/data-table";
import { useCallback, useMemo } from "react";
import { SelectionDock } from "@/components/selection-dock";

/** A column the dock totals over the selected rows. */
export type DataTableSelectionTotal<TData extends RowData> = {
  id: string;
  label: string;
  format: "money" | "number";
  getValue: (row: TData) => unknown;
};

type DataTableDockProps<TData extends RowData> = {
  table: Table<TData>;
  actions: DockAction<TData>[];
  totals?: readonly DataTableSelectionTotal<TData>[];
};

const SELECTION_NUMBER_FORMAT = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 });

export function DataTableDock<TData extends RowData>({
  table,
  actions,
  totals,
}: DataTableDockProps<TData>) {
  const { getSelectedRows } = useDataTable<TData, unknown>();
  const selectedCount = useTableAtom(table.atoms.rowSelection, countSelectedRows);
  const selection = useTableAtom(table.atoms.rowSelection);
  const selectionTotals = useMemo(() => {
    // Keyed on the selection itself, not only its size, so a swap re-totals.
    if (!totals || totals.length === 0 || selectedCount === 0 || !selection) return [];
    const rows = getSelectedRows();
    return totals.map((total) => {
      let sum = 0;
      for (const row of rows) {
        const value = Number(total.getValue(row));
        if (Number.isFinite(value)) sum += value;
      }
      return {
        id: total.id,
        label: total.label,
        text: total.format === "money" ? formatCurrency(sum) : SELECTION_NUMBER_FORMAT.format(sum),
      };
    });
  }, [totals, selection, selectedCount, getSelectedRows]);

  const handleClearSelection = useCallback(() => {
    table.setRowSelection({});
  }, [table]);

  return (
    <SelectionDock
      selectedCount={selectedCount}
      totals={selectionTotals}
      actions={actions}
      getSelected={getSelectedRows}
      onClearSelection={handleClearSelection}
    />
  );
}
