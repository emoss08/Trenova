import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { EquipmentStatusBadge } from "@/components/status-badge";
import { type Tractor } from "@/types/tractor";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Tractor>[] {
  const columnHelper = createColumnHelper<Tractor>();
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
    createEntityColumn(columnHelper, "code", {
      accessorKey: "code",
      getHeaderText: "Code",
      getId: (tractor) => tractor.id,
      getDisplayText: (tractor) => tractor.code,
    }),
    createEntityRefColumn<Tractor, "equipmentType">(
      columnHelper,
      "equipmentType",
      {
        basePath: "/equipment/configurations/equipment-types",
        getId: (equipType) => equipType.id,
        getDisplayText: (equipType) => equipType.code,
        getHeaderText: "Equipment Type",
        color: {
          getColor: (equipType) => equipType.color,
        },
      },
    ),
    createEntityRefColumn<Tractor, "primaryWorker">(
      columnHelper,
      "primaryWorker",
      {
        basePath: "/dispatch/configurations/workers",
        getHeaderText: "Assigned Workers",
        getId: (worker) => worker.id ?? undefined,
        getDisplayText: (worker) => `${worker.firstName} ${worker.lastName}`,
        getSecondaryInfo: (_, tractor) =>
          tractor.secondaryWorker
            ? {
                label: "Co-Driver",
                entity: tractor.secondaryWorker,
                displayText: `${tractor.secondaryWorker.firstName} ${tractor.secondaryWorker.lastName}`,
              }
            : null,
      },
    ),
  ];
}
