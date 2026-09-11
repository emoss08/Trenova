"use no memo";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { Checkbox } from "../animate-ui/components/base/checkbox";

export function createSelectionColumn<TData extends Record<string, unknown>>(): ColumnDef<TData> {
  return {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label={translate("Select all")}
        nativeButton
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label={translate("Select row")}
        nativeButton
        className="translate-y-[2px]"
      />
    ),
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
