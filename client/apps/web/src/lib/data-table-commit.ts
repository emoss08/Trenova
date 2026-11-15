import type { CellEditCommitFn } from "@trenova/shared/lib/cell-editing-feature";

/**
 * Saves one field of a row through the table's own edit commit, as an edited cell does, for a
 * value changed beside a cell rather than in it (a colour picked from a row's dot). The table's
 * commit validates it and refreshes the rows.
 */
export async function commitRowField<TData extends Record<string, unknown>>(
  table: { options: { onCellEditCommit?: CellEditCommitFn<TData> } },
  row: { id: string; original: TData },
  field: keyof TData & string,
  value: unknown,
): Promise<void> {
  await table.options.onCellEditCommit?.({
    rowId: row.id,
    columnId: field,
    value,
    previousValue: row.original[field],
    row: row.original,
  });
}
