import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import { StatusBadge } from "@/components/status-badge";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { truncateText } from "@/lib/utils";
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
    {
      accessorKey: "code",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Code" />
      ),
      cell: ({ row }) => {
        const isColor = !!row.original.color;
        return isColor ? (
          <div className="flex items-center gap-x-1.5 text-sm font-medium text-foreground">
            <div
              className="size-2 rounded-full"
              style={{
                backgroundColor: row.original.color,
              }}
            />
            <p>{row.original.code}</p>
          </div>
        ) : (
          <p>{row.original.code}</p>
        );
      },
    },
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
      cell: ({ row }) => truncateText(row.original.description ?? "", 40),
    },
  ];
}
