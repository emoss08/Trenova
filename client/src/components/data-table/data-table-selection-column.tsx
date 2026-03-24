"use no memo";
import type { ColumnDef } from "@tanstack/react-table";
import { Checkbox } from "../animate-ui/components/base/checkbox";

export function createSelectionColumn<
  TData extends Record<string, unknown>,
>(): ColumnDef<TData> {
  return {
    id: "select",
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        indeterminate={table.getIsSomePageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
        nativeButton
        className="translate-y-[2px]"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
        nativeButton
        className="translate-y-[2px]"
      />
    ),
    enableSorting: false,
    enableHiding: false,
    size: 40,
    meta: {
      sortable: false,
      filterable: false,
    },
  };
}
