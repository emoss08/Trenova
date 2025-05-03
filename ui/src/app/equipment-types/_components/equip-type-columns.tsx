import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableColorColumn } from "@/components/data-table/_components/data-table-components";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<EquipmentTypeSchema>[] {
  const commonColumns = createCommonColumns<EquipmentTypeSchema>();

  return [
    commonColumns.status,
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        const { color, code } = row.original;
        return <DataTableColorColumn text={code} color={color} />;
      },
    },
    {
      accessorKey: "class",
      header: "Equip. Class",
    },
    commonColumns.description,
    commonColumns.createdAt,
  ];
}
