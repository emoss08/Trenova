import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { type FleetCode } from "@/types/fleet-code";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<FleetCode>[] {
  const columnHelper = createColumnHelper<FleetCode>();
  const commonColumns = createCommonColumns<FleetCode>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { color, name } = row.original;
        return <DataTableColorColumn text={name} color={color} />;
      },
    }),
    commonColumns.description,
    {
      id: "manager",
      header: "Manager",
      cell: ({ row }) => {
        const { manager } = row.original;
        if (!manager) return "-";
        return <p>{manager.name}</p>;
      },
    },
    commonColumns.createdAt,
  ];
}
