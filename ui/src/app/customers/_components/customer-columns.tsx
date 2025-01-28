import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { BooleanBadge } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<CustomerSchema>[] {
  const columnHelper = createColumnHelper<CustomerSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    commonColumns.selection,
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    },
    createEntityColumn(columnHelper, "code", {
      accessorKey: "code",
      getHeaderText: "Code",
      getId: (customer) => customer.id,
      getDisplayText: (customer) => customer.code,
    }),
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
    },
    {
      accessorKey: "autoMarkReadyToBill",
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title="Auto Mark Ready To Bill"
        />
      ),
      cell: ({ row }) => (
        <BooleanBadge value={row.original.autoMarkReadyToBill} />
      ),
    },
  ];
}
