/* eslint-disable react-hooks/rules-of-hooks */
import {
  createNestedEntityRefColumn,
  EntityRefCell,
} from "@/components/data-table/_components/data-table-column-helpers";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { shipmentStatusChoices } from "@/lib/choices";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import { LocationSchema } from "@/lib/schemas/location-schema";
import {
  getDestinationStopInfo,
  getOriginStopInfo,
  ShipmentLocations,
} from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Shipment>[] {
  const columnHelper = createColumnHelper<Shipment>();

  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const { status } = row.original;
        return <ShipmentStatusBadge status={status} />;
      },
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: shipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "proNumber",
      header: "Pro Number",
      cell: ({ row }) => {
        const proNumber = row.original.proNumber;
        return <p>{proNumber}</p>;
      },
      enableHiding: false,
    },
    {
      accessorKey: "customer",
      header: "Customer",
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<CustomerSchema, Shipment>
            entity={customer}
            config={{
              basePath: "/billing/configurations/customers",
              getId: (customer) => customer.id,
              getDisplayText: (customer) => customer.name,
              getHeaderText: "Customer",
            }}
            parent={row.original}
          />
        );
      },
    },
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
      accessorKey: "originPickup",
      header: "Origin Date",
      cell: ({ row }) => {
        const shipment = row.original;
        const originStop = getOriginStopInfo(shipment);

        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={originStop?.plannedArrival}
          />
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
      accessorKey: "destinationPickup",
      header: "Destination Date",
      cell: ({ row }) => {
        const shipment = row.original;
        const destinationStop = getDestinationStopInfo(shipment);

        return (
          <HoverCardTimestamp
            className="font-table tracking-tight"
            timestamp={destinationStop?.plannedArrival}
          />
        );
      },
    },
  ];
}
