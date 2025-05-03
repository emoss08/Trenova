import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { type ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentTypeSchema>[] {
  const columnHelper = createColumnHelper<ShipmentTypeSchema>();
  const commonColumns = createCommonColumns();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
