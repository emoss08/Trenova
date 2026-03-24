import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { formatLocation } from "@/lib/utils";
import type { Customer } from "@/types/customer";
import type { DistanceOverride } from "@/types/distance-override";
import type { Location } from "@/types/location";
import type { ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DistanceOverride>[] {
  return [
    {
      accessorKey: "originLocationId",
      header: "Origin Location",
      cell: ({ row }) => {
        const { originLocation } = row.original;
        if (!originLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Location, DistanceOverride>
            entity={originLocation}
            config={{
              basePath: "/dispatch/configuration-files/locations",
              getId: (location) => location.id,
              getDisplayText: (location) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
              getHeaderText: "Origin Location",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: "Origin Location",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "destinationLocationId",
      header: "Destination Location",
      cell: ({ row }) => {
        const { destinationLocation } = row.original;
        if (!destinationLocation) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Location, DistanceOverride>
            entity={destinationLocation}
            config={{
              basePath: "/dispatch/configuration-files/locations",
              getId: (location) => location.id,
              getDisplayText: (location) => location.name,
              getSecondaryInfo: (location) => {
                return {
                  entity: location,
                  displayText: formatLocation(location),
                  clickable: false,
                };
              },
              getHeaderText: "Destination Location",
            }}
            parent={row.original}
          />
        );
      },
      meta: {
        label: "Destination Location",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "distance",
      header: "Distance",
      cell: ({ row }) => row.original.distance,
      meta: {
        label: "Distance",
        apiField: "distance",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
      size: 120,
      minSize: 80,
      maxSize: 180,
    },
    {
      accessorKey: "customerId",
      header: "Customer",
      cell: ({ row }) => {
        const { customer } = row.original;

        if (!customer) {
          return <p className="text-muted-foreground">-</p>;
        }

        return (
          <EntityRefCell<Customer, DistanceOverride>
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
      size: 200,
      minSize: 100,
      maxSize: 250,
      meta: {
        label: "Customer",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "intermediateStops",
      header: "Stops",
      cell: ({ row }) => row.original.intermediateStops?.length ?? 0,
      meta: {
        label: "Stops",
        filterable: false,
        sortable: false,
      },
      size: 80,
      minSize: 70,
      maxSize: 100,
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
