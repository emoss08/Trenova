import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<EquipmentManufacturerSchema>[] {
  const columnHelper = createColumnHelper<EquipmentManufacturerSchema>();
  const commonColumns = createCommonColumns<EquipmentManufacturerSchema>();

  return [
    commonColumns.status,
    columnHelper.display({
      id: "name",
      header: "Name",
      cell: ({ row }) => {
        const { name } = row.original;
        return <p>{name}</p>;
      },
    }),
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
