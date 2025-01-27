import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import { createCommonColumns } from "@/components/data-table/_components/data-table-column-helpers";
import {
  DataTableColorColumn,
  LastInspectionDateBadge,
} from "@/components/data-table/_components/data-table-components";
import { EquipmentStatusBadge } from "@/components/status-badge";
import { type Trailer } from "@/types/trailer";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Trailer>[] {
  const columnHelper = createColumnHelper<Trailer>();
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
        return <EquipmentStatusBadge status={status} />;
      },
    },
    {
      accessorKey: "code",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Code" />
      ),
    },
    {
      id: "equipmentType",
      accessorKey: "equipmentType",
      header: "Equipment Type",
      cell: ({ row }) => {
        const equipType = row.original.equipmentType;
        const isEquipType = !!equipType;

        return isEquipType ? (
          <DataTableColorColumn
            color={equipType?.color}
            text={equipType?.code ?? ""}
          />
        ) : (
          <p>No equipment type</p>
        );
      },
    },
    {
      accessorKey: "lastInspectionDate",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Last Inspection Date" />
      ),
      cell: ({ row }) => {
        const lastInspectionDate = row.original.lastInspectionDate;
        return <LastInspectionDateBadge value={lastInspectionDate} />;
      },
    },
  ];
}
