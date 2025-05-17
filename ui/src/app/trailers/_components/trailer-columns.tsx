import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
  createCommonColumns,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { LastInspectionDateBadge } from "@/components/data-table/_components/data-table-components";
import { EquipmentStatusBadge } from "@/components/status-badge";
import type { TrailerSchema } from "@/lib/schemas/trailer-schema";
import { type Trailer } from "@/types/trailer";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<TrailerSchema>[] {
  const columnHelper = createColumnHelper<TrailerSchema>();
  const commonColumns = createCommonColumns<TrailerSchema>();

  return [
    columnHelper.display({
      id: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <EquipmentStatusBadge status={status} />;
      },
    }),
    columnHelper.display({
      id: "code",
      header: "Code",
      cell: ({ row }) => {
        const code = row.original.code;
        return <p>{code}</p>;
      },
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
    columnHelper.display({
      id: "lastInspectionDate",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Last Inspection Date" />
      ),
      cell: ({ row }) => {
        const { lastInspectionDate } = row.original;
        return <LastInspectionDateBadge value={lastInspectionDate} />;
      },
    }),
    commonColumns.createdAt,
  ];
}
