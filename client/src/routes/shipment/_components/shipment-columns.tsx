import {
  EntityRefCell,
  NestedEntityRefCell,
} from "@/components/data-table/_components/entity-ref-link";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { ShipmentStatusBadge } from "@/components/status-badge";
import { shipmentStatusChoices } from "@/lib/choices";
import {
  getDestinationLocation,
  getDestinationStop,
  getOriginLocation,
  getOriginStop,
} from "@/lib/shipment-utils";
import { formatLocation } from "@/lib/utils";
import type { Customer } from "@/types/customer";
import type { Location } from "@/types/location";
import type { Shipment } from "@/types/shipment";
import type { User } from "@/types/user";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<Shipment>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <ShipmentStatusBadge status={row.original.status} />,
      meta: {
        apiField: "status",
        label: "Status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: shipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
    {
      accessorKey: "proNumber",
      header: "PRO Number",
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        label: "PRO Number",
        apiField: "proNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "bol",
      header: "BOL",
      size: 200,
      minSize: 200,
      maxSize: 250,
      meta: {
        label: "BOL",
        apiField: "bol",
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
      size: 250,
      minSize: 200,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Customer, Shipment>
            entity={customer}
            config={{
              basePath: "/billing/configuration-files/customers",
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
        label: "Customer Name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "originLocation",
      accessorKey: "originLocation",
      header: "Orig. Location",
      size: 250,
      minSize: 200,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <NestedEntityRefCell<Location, Shipment>
            getValue={() => {
              return getOriginLocation(row.original);
            }}
            row={row}
            config={{
              getEntity: (shipment) => {
                return getOriginLocation(shipment);
              },
              basePath: "/dispatch/configurations/locations",
              getId: (location) => location.id,
              getDisplayText: (location: Location) => location.name,
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
        label: "Orig. Location Name",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "originScheduledTime",
      accessorKey: "originPickup",
      header: "Orig. Scheduled Time",
      size: 350,
      minSize: 350,
      maxSize: 400,
      cell: ({ row }) => {
        const shipment = row.original;
        const originStop = getOriginStop(shipment);

        return (
          <div className="flex flex-row gap-2">
            <HoverCardTimestamp
              className="font-table tracking-tight"
              timestamp={originStop?.scheduledWindowStart}
            />
            <span className="text-muted-foreground">-</span>
            {originStop?.scheduledWindowEnd && (
              <HoverCardTimestamp
                className="font-table tracking-tight"
                timestamp={originStop.scheduledWindowEnd}
              />
            )}
          </div>
        );
      },
      meta: {
        label: "Orig. Scheduled Time",
        apiField: "originPickup.scheduledWindowStart",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "destinationLocation",
      accessorKey: "destinationLocation",
      header: "Dest. Location",
      size: 250,
      minSize: 200,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <NestedEntityRefCell<Location, Shipment>
            getValue={() => {
              return getDestinationLocation(row.original);
            }}
            row={row}
            config={{
              getEntity: (shipment) => {
                return getDestinationLocation(shipment);
              },
              basePath: "/dispatch/locations",
              getId: (location) => location.id,
              getDisplayText: (location: Location) => location.name,
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
        label: "Dest. Location",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "destinationScheduledTime",
      accessorKey: "destinationPickup",
      header: "Dest. Scheduled Time",
      size: 350,
      minSize: 350,
      maxSize: 400,
      cell: ({ row }) => {
        const shipment = row.original;
        const destinationStop = getDestinationStop(shipment);

        return (
          <div className="flex flex-row gap-2">
            <HoverCardTimestamp
              className="font-table tracking-tight"
              timestamp={destinationStop?.scheduledWindowStart}
            />
            <span className="text-muted-foreground">-</span>
            {destinationStop?.scheduledWindowEnd && (
              <HoverCardTimestamp
                className="font-table tracking-tight"
                timestamp={destinationStop.scheduledWindowEnd}
              />
            )}
          </div>
        );
      },
      meta: {
        label: "Dest. Scheduled Time",
        apiField: "destinationPickup.scheduledWindowStart",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "owner",
      accessorKey: "owner",
      header: "Owner",
      size: 250,
      minSize: 200,
      maxSize: 300,
      cell: ({ row }) => {
        const { owner } = row.original;

        if (!owner) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<User, Shipment>
            entity={owner}
            config={{
              basePath: "/admin/users",
              getId: (user) => user.id,
              getDisplayText: (user) => user.name,
              getHeaderText: "Owner",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        apiField: "owner.name",
        label: "Owner Name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: {
        label: "Created At",
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 300,
      minSize: 250,
      maxSize: 300,
    },
  ];
}
