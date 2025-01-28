import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentTypeSchema>[] {
  const columnHelper = createColumnHelper<ShipmentTypeSchema>();
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
      getId: (shipmentType) => shipmentType.id,
      getDisplayText: (shipmentType) => shipmentType.code,
      getColor: (shipmentType) => shipmentType.color,
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
  ];
}
