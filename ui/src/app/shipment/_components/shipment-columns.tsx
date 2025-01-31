import { DataTableColumnHeader } from "@/components/data-table/_components/data-table-column-header";
import {
    createCommonColumns,
    createEntityColumn,
    createNestedEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { LocationSchema } from "@/lib/schemas/location-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Shipment>[] {
  const columnHelper = createColumnHelper<Shipment>();
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
        return <ShipmentStatusBadge status={status} />;
      },
    },
    createEntityColumn(columnHelper, "proNumber", {
      accessorKey: "proNumber",
      getHeaderText: "Pro Number",
      getId: (shipment) => shipment.id,
      getDisplayText: (shipment) => shipment.proNumber,
    }),
    createNestedEntityRefColumn(columnHelper, {
      basePath: "/dispatch/configurations/locations",
      getHeaderText: "Origin Location",
      getId: (location) => location.id,
      getDisplayText: (location: LocationSchema) => location.name,
      getSecondaryInfo: (location) => {
        return {
          entity: location,
          displayText: formatLocation(location),
          clickable: false,
        };
      },
      getEntity: (shipment) => {
        try {
          return ShipmentLocations.useLocations(shipment).origin;
        } catch {
          throw new Error("Shipment has no origin location");
        }
      },
    }),
    createNestedEntityRefColumn(columnHelper, {
      basePath: "/dispatch/configurations/locations",
      getHeaderText: "Destination Location",
      getId: (location) => location.id,
      getDisplayText: (location: LocationSchema) => location.name,
      getSecondaryInfo: (location) => {
        return {
          entity: location,
          displayText: formatLocation(location),
          clickable: false,
        };
      },
      getEntity: (shipment) => {
        try {
          return ShipmentLocations.useLocations(shipment).destination;
        } catch {
          throw new Error("Shipment has no destination location");
        }
      },
    }),
    commonColumns.createdAt,
  ];
}
