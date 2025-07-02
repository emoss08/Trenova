import {
  createCommonColumns,
  createEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { EquipmentStatusBadge } from "@/components/status-badge";
import type { TractorSchema } from "@/lib/schemas/tractor-schema";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<TractorSchema>[] {
  const columnHelper = createColumnHelper<TractorSchema>();
  const commonColumns = createCommonColumns<TractorSchema>();

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
    createEntityRefColumn<TractorSchema, "equipmentType">(
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
    createEntityRefColumn<TractorSchema, "equipmentManufacturer">(
      columnHelper,
      "equipmentManufacturer",
      {
        basePath: "/equipment/configurations/equipment-manufacturers",
        getId: (equipManufacturer) => equipManufacturer.id,
        getDisplayText: (equipManufacturer) => equipManufacturer.name,
        getHeaderText: "Equipment Manufacturer",
      },
    ),
    createEntityRefColumn<TractorSchema, "primaryWorker">(
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
    createEntityRefColumn<TractorSchema, "fleetCode">(
      columnHelper,
      "fleetCode",
      {
        basePath: "/dispatch/configurations/fleet-codes",
        getId: (fleetCode) => fleetCode.id,
        getDisplayText: (fleetCode) => fleetCode.name,
        getHeaderText: "Fleet Code",
        color: {
          getColor: (fleetCode) => fleetCode.color,
        },
      },
    ),
    commonColumns.createdAt,
  ];
}
