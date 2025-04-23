/* eslint-disable react-hooks/rules-of-hooks */
import {
  createEntityColumn,
  createEntityRefColumn,
  createNestedEntityRefColumn,
} from "@/components/data-table/_components/data-table-column-helpers";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  generateDateTimeString,
  generateDateTimeStringFromUnixTimestamp,
  toDate,
} from "@/lib/date";
import { LocationSchema } from "@/lib/schemas/location-schema";
import {
  getDestinationStopInfo,
  getOriginStopInfo,
  ShipmentLocations
} from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Shipment>[] {
  const columnHelper = createColumnHelper<Shipment>();

  return [
    {
      id: "select",
      header: ({ table }) => {
        const isAllSelected = table.getIsAllPageRowsSelected();
        const isSomeSelected = table.getIsSomePageRowsSelected();

        return (
          <Checkbox
            data-slot="select-all"
            checked={isAllSelected || (isSomeSelected && "indeterminate")}
            onCheckedChange={(checked) =>
              table.toggleAllPageRowsSelected(!!checked)
            }
            aria-label="Select all"
          />
        );
      },
      cell: ({ row }) => (
        <Checkbox
          data-slot="select-row"
          checked={row.getIsSelected()}
          onCheckedChange={(checked) => row.toggleSelected(!!checked)}
          aria-label="Select row"
        />
      ),
      size: 50,
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status;
        return <ShipmentStatusBadge status={status} />;
      },
      size: 100,
    },
    createEntityColumn(columnHelper, "proNumber", {
      accessorKey: "proNumber",
      getHeaderText: "Pro Number",
      getId: (shipment) => shipment?.id,
      getDisplayText: (shipment) => shipment?.proNumber || "-",
    }),
    createEntityRefColumn<Shipment, "customer">(columnHelper, "customer", {
      basePath: "/billing/configurations/customers",
      getId: (customer) => customer.id,
      getDisplayText: (customer) => customer.name,
      getHeaderText: "Customer",
    }),
    createNestedEntityRefColumn(columnHelper, {
      columnId: "originLocation",
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
    {
      id: "originPickup",
      header: "Origin Date",
      cell: ({ row }) => {
        const shipment = row.original;
        const originStop = getOriginStopInfo(shipment);
        if (!originStop) {
          return <p>-</p>;
        }

        return (
          <p>
            {generateDateTimeStringFromUnixTimestamp(originStop.plannedArrival)}
          </p>
        );
      },
    },
    createNestedEntityRefColumn(columnHelper, {
      columnId: "destinationLocation",
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
    {
      id: "destinationPickup",
      header: "Destination Date",
      cell: ({ row }) => {
        const shipment = row.original;
        const destinationStop = getDestinationStopInfo(shipment);
        if (!destinationStop) {
          return <p>-</p>;
        }

        const arrivalDate = toDate(destinationStop.plannedArrival);
        if (!arrivalDate) {
          return <p>-</p>;
        }

        return <p>{generateDateTimeString(arrivalDate)}</p>;
      },
    },
  ];
}
