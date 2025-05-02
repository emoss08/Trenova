import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentTypeSchema>[] {
  const columnHelper = createColumnHelper<ShipmentTypeSchema>();
  const commonColumns = createCommonColumns(columnHelper);

  return [
    columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <StatusBadge status={status} />;
      },
    }),
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    }),
    columnHelper.display({
      id: "description",
      header: "description",
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    }),
    commonColumns.createdAt,
  ];
}
