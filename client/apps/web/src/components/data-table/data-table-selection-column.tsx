import { useOptionalDataTable } from "@/contexts/data-table-context";
import { useTableAtom } from "@trenova/shared/hooks/use-table-atom";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef, HeaderContext, Row } from "@trenova/shared/types/data-table";
import type { RowSelectionState } from "@tanstack/react-table";
import { Checkbox } from "../animate-ui/components/base/checkbox";

const NONE = 0;
const SOME = 1;
const ALL = 2;

function pageSelectionState(
  rows: readonly { id: string; getCanSelect: () => boolean }[],
  selection: RowSelectionState,
): typeof NONE | typeof SOME | typeof ALL {
  let selectable = 0;
  let selected = 0;
  for (const row of rows) {
    if (!row.getCanSelect()) continue;
    selectable += 1;
    if (selection[row.id]) selected += 1;
  }
  if (selected === 0) return NONE;
  return selected === selectable ? ALL : SOME;
}

/**
 * Follows the selection itself, so ticking a row redraws this box and not the header
 * row. The page's rows come from the table the data table hands down, which changes
 * with every page, so the box also catches up when the page turns under a selection.
 */
function SelectPageCheckbox<TData extends Record<string, unknown>>({
  table,
}: {
  table: HeaderContext<TData>["table"];
}) {
  const pageTable = useOptionalDataTable<TData, unknown>()?.table ?? table;
  const selection = useTableAtom(table.atoms.rowSelection);
  const state = pageSelectionState(pageTable.getRowModel().rows, selection);

  return (
    <Checkbox
      checked={state === ALL}
      indeterminate={state === SOME}
      onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
      aria-label={translate("Select all")}
      nativeButton
      className="translate-y-[2px]"
    />
  );
}

/** Follows only its own row's entry, so ticking one row redraws one box. */
function SelectRowCheckbox<TData extends Record<string, unknown>>({ row }: { row: Row<TData> }) {
  const checked = useTableAtom(row.table.atoms.rowSelection, (selection) => !!selection[row.id]);

  return (
    <Checkbox
      checked={checked}
      onCheckedChange={(value) => row.toggleSelected(!!value)}
      aria-label={translate("Select row")}
      nativeButton
      className="translate-y-[2px]"
    />
  );
}

export function createSelectionColumn<TData extends Record<string, unknown>>(): ColumnDef<TData> {
  return {
    id: "select",
    header: ({ table }) => <SelectPageCheckbox table={table} />,
    cell: ({ row }) => <SelectRowCheckbox row={row} />,
    enableSorting: false,
    enableHiding: false,
    enableResizing: false,
    size: 40,
    meta: {
      sortable: false,
      filterable: false,
    },
  };
}
