/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/* eslint-disable react-hooks/rules-of-hooks */
import {
  EntityRefCell,
  NestedEntityRefCell,
} from "@/components/data-table/_components/data-table-column-helpers";
import { HoverCardTimestamp } from "@/components/data-table/_components/data-table-components";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { shipmentStatusChoices } from "@/lib/choices";
import type { CustomerSchema } from "@/lib/schemas/customer-schema";
import { LocationSchema } from "@/lib/schemas/location-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import {
  getDestinationStopInfo,
  getOriginStopInfo,
  ShipmentLocations,
} from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<ShipmentSchema>[] {
  return [
    {
      id: "status",
      accessorKey: "status",
      header: "Status",
      size: 120,
      minSize: 100,
      maxSize: 150,
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
      id: "proNumber",
      accessorKey: "proNumber",
      header: "Pro Number",
      size: 140,
      minSize: 120,
      maxSize: 180,
      cell: ({ row }) => {
        const proNumber = row.original.proNumber;
        return <p>{proNumber}</p>;
      },
      meta: {
        apiField: "proNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "customer",
      accessorKey: "customer",
      header: "Customer",
      size: 200,
      minSize: 150,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<CustomerSchema, ShipmentSchema>
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
      meta: {
        apiField: "customer.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "originLocation",
      accessorKey: "originLocation",
      header: "Origin Location",
      size: 220,
      minSize: 180,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <NestedEntityRefCell<LocationSchema, ShipmentSchema>
            getValue={() => {
              return ShipmentLocations.useLocations(row.original).origin;
            }}
            row={row}
            config={{
              getEntity: (shipment) => {
                return ShipmentLocations.useLocations(shipment).origin;
              },
              basePath: "/dispatch/configurations/locations",
              getId: (location) => location.id,
              getDisplayText: (location: LocationSchema) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
            }}
          />
        );
      },
      meta: {
        apiField: "originLocation.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "originPlannedArrival",
      accessorKey: "originPickup",
      header: "Origin Date",
      size: 150,
      minSize: 130,
      maxSize: 180,
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
    {
      id: "destinationLocation",
      accessorKey: "destinationLocation",
      header: "Destination Location",
      size: 220,
      minSize: 180,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <NestedEntityRefCell<LocationSchema, ShipmentSchema>
            getValue={() => {
              return ShipmentLocations.useLocations(row.original).destination;
            }}
            row={row}
            config={{
              getEntity: (shipment) => {
                return ShipmentLocations.useLocations(shipment).destination;
              },
              basePath: "/dispatch/configurations/locations",
              getId: (location) => location.id,
              getDisplayText: (location: LocationSchema) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
            }}
          />
        );
      },
      meta: {
        apiField: "destinationLocation.name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "destinationPlannedArrival",
      accessorKey: "destinationPickup",
      header: "Destination Date",
      size: 150,
      minSize: 130,
      maxSize: 180,
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
