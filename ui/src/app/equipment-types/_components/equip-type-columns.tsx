import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { StatusBadge } from "@/components/status-badge";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<EquipmentTypeSchema>[] {
  const columnHelper = createColumnHelper<EquipmentTypeSchema>();
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
      getId: (equipmentType) => equipmentType.id,
      getDisplayText: (equipmentType) => equipmentType.code,
      getColor: (equipmentType) => equipmentType.color,
    }),
    {
      accessorKey: "class",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Equip. Class" />
      ),
    },
    {
      accessorKey: "description",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} />
      ),
    },
    commonColumns.createdAt,
  ];
}
