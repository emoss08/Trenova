import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityColumn,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { LastInspectionDateBadge } from "@/components/data-table/_components/data-table-components";
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
    createEntityColumn(columnHelper, "code", {
      accessorKey: "code",
      getHeaderText: "Code",
      getId: (trailer) => trailer.id,
      getDisplayText: (trailer) => trailer.code,
    }),
    createEntityRefColumn<Trailer, "equipmentType">(
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
    createEntityRefColumn<Trailer, "equipmentManufacturer">(
      columnHelper,
      "equipmentManufacturer",
      {
        basePath: "/equipment/configurations/equipment-manufacturers",
        getId: (equipManufacturer) => equipManufacturer.id,
        getDisplayText: (equipManufacturer) => equipManufacturer.name,
        getHeaderText: "Equipment Manufacturer",
      },
    ),
    createEntityRefColumn<Trailer, "fleetCode">(columnHelper, "fleetCode", {
      basePath: "/dispatch/configurations/fleet-codes",
      getId: (fleetCode) => fleetCode.id,
      getDisplayText: (fleetCode) => fleetCode.name,
      getHeaderText: "Fleet Code",
      color: {
        getColor: (fleetCode) => fleetCode.color,
      },
    }),
    {
      accessorKey: "lastInspectionDate",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Last Inspection Date" />
      ),
      cell: ({ row }) => {
        const { lastInspectionDate } = row.original;
        return <LastInspectionDateBadge value={lastInspectionDate} />;
      },
    },
    commonColumns.createdAt,
  ];
}
