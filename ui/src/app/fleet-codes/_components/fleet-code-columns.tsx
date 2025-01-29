import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type FleetCode } from "@/types/fleet-code";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FleetCode>[] {
  const columnHelper = createColumnHelper<FleetCode>();
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
    createEntityColumn(columnHelper, "name", {
      accessorKey: "name",
      getHeaderText: "Name",
      getId: (fleetCode) => fleetCode.id,
      getDisplayText: (fleetCode) => fleetCode.name,
      getColor: (fleetCode) => fleetCode.color,
    }),
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    },
    {
      id: "manager",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Manager" />
      ),
      cell: ({ row }) => {
        const { manager } = row.original;
        if (!manager) return "-";
        return <p>{manager.name}</p>;
      },
    },
    commonColumns.createdAt,
  ];
}
